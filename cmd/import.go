package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	
	"goscaffold/internal/models"
	"goscaffold/pkg/backup"
	"goscaffold/pkg/clipboard"
	"goscaffold/pkg/git"
	"goscaffold/pkg/parser"
	"goscaffold/pkg/stats"
	"goscaffold/pkg/ui"
	"goscaffold/pkg/validator"
)

var (
	dryRun       bool
	useClipboard bool
	inputFile    string
	gitCommit    bool
	interactive  bool
	backupFiles  bool
	watchMode    bool
	batchMode    bool
)

var importCmd = &cobra.Command{
	Use:   "import [flags]",
	Short: "Import AI-generated code blocks into your project",
	Long: `Parses code blocks from files, clipboard, or stdin and creates/updates files.
Supports markdown code fences (```) and YAML-style separators (---).

Examples:
  goscaffold import --clipboard
  goscaffold import --input chat.md --git-commit
  cat output.md | goscaffold import --interactive`,
	Aliases: []string{"i", "im"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		
		if watchMode {
			return runWatchMode(ctx)
		}
		
		content, err := getInput(ctx)
		if err != nil {
			return fmt.Errorf("failed to get input: %w", err)
		}
		
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("no input provided")
		}
		
		files := parser.ParseMultiFormat(content)
		if len(files) == 0 {
			return fmt.Errorf("no valid code blocks found")
		}
		
		log.Info(fmt.Sprintf("Found %d files to process", len(files)))
		
		if dryRun {
			return runDryRun(files)
		}
		
		if interactive {
			return runInteractive(files)
		}
		
		return runBatch(ctx, files)
	},
}

func init() {
	importCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Preview operations without writing files")
	importCmd.Flags().BoolVarP(&useClipboard, "clipboard", "c", false, "Read from system clipboard")
	importCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file path (- for stdin)")
	importCmd.Flags().BoolVarP(&gitCommit, "git-commit", "g", false, "Auto-commit changes to git")
	importCmd.Flags().BoolVarP(&interactive, "interactive", "I", false, "Interactive mode with previews")
	importCmd.Flags().BoolVar(&backupFiles, "backup", viper.GetBool("backup.enabled"), "Create backups before overwriting")
	importCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch mode (uses --input file)")
	importCmd.Flags().BoolVar(&batchMode, "batch", false, "Batch mode (no TUI)")
	
	rootCmd.AddCommand(importCmd)
}

func getInput(ctx context.Context) (string, error) {
	if useClipboard {
		return clipboard.Read()
	}
	
	if inputFile != "" {
		if inputFile == "-" {
			return readStdin()
		}
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return "", fmt.Errorf("read file: %w", err)
		}
		return string(data), nil
	}
	
	// Try clipboard as fallback
	if content, _ := clipboard.Read(); content != "" {
		log.Info("Using clipboard content")
		return content, nil
	}
	
	return "", fmt.Errorf("no input source specified")
}

func readStdin() (string, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", fmt.Errorf("no data on stdin")
	}
	
	data, err := os.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return string(data), nil
}

func runDryRun(files []models.File) error {
	log.Info("=== DRY RUN MODE ===")
	
	for _, file := range files {
		log.Info("Would create/update", "file", file.Path, "size", len(file.Code))
		
		if backupFiles {
			if _, err := os.Stat(file.Path); err == nil {
				log.Info("  → Would backup existing file")
			}
		}
		
		// Show first few lines
		lines := strings.Split(file.Code, "\n")
		if len(lines) > 5 {
			log.Info("Preview", "content", strings.Join(lines[:5], "\n")+"...")
		}
	}
	
	return nil
}

func runInteractive(files []models.File) error {
	log.Info("Running in interactive TUI mode...")
	
	p := tea.NewProgram(ui.NewImportModel(files, gitCommit, backupFiles))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	
	return nil
}

func runBatch(ctx context.Context, files []models.File) error {
	s := stats.New()
	bm := backup.NewManager(viper.GetString("backup.retention"))
	
	// Process files with progress bar
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))
	sem := make(chan struct{}, 4) // Max 4 concurrent
	
	for i, file := range files {
		wg.Add(1)
		go func(idx int, f models.File) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			if err := processFile(ctx, f, s, bm); err != nil {
				errChan <- fmt.Errorf("%s: %w", f.Path, err)
			}
		}(i, file)
	}
	
	wg.Wait()
	close(errChan)
	
	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
		log.Error("Processing failed", "error", err)
	}
	
	s.Print()
	
	// Git commit
	if gitCommit && len(s.Languages) > 0 {
		log.Info("Committing changes to git...")
		if err := git.Commit(ctx, getCreatedFiles(files), "chore(scaffold): import AI-generated files"); err != nil {
			log.Error("Git commit failed", "error", err)
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("%d files failed", len(errs))
	}
	
	log.Info("✨ Import completed successfully")
	return nil
}

func processFile(ctx context.Context, file models.File, s *stats.Stats, bm *backup.Manager) error {
	// Backup if exists and enabled
	if backupFiles {
		if err := bm.Backup(file.Path); err != nil {
			log.Warn("Backup failed", "file", file.Path, "error", err)
		}
	}
	
	// Create directory
	dir := filepath.Dir(file.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	
	// Validate if configured
	val, err := validator.GetForFile(file.Path)
	if err == nil {
		if err := val.Validate(ctx, file.Path, file.Code); err != nil {
			log.Warn("Validation warning", "file", file.Path, "error", err)
		}
	}
	
	// Write file
	if err := os.WriteFile(file.Path, []byte(file.Code), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	
	s.AddFile(file.Path, file.Code)
	log.Info("Created", "file", file.Path, "size", len(file.Code))
	return nil
}

func runWatchMode(ctx context.Context) error {
	if inputFile == "" {
		return fmt.Errorf("--watch requires --input file")
	}
	
	log.Info("Starting watch mode", "file", inputFile, "interval", viper.GetString("watch.interval"))
	
	interval, err := time.ParseDuration(viper.GetString("watch.interval"))
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	var lastMod time.Time
	for {
		select {
		case <-ctx.Done():
			log.Info("Watch mode stopped")
			return nil
		case <-ticker.C:
			info, err := os.Stat(inputFile)
			if err != nil {
				continue
			}
			
			if info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
				log.Info("File changed, processing...", "file", inputFile)
				
				// Re-run import
				content, _ := os.ReadFile(inputFile)
				files := parser.ParseMultiFormat(string(content))
				
				if len(files) > 0 {
					if err := runBatch(ctx, files); err != nil {
						log.Error("Watch import failed", "error", err)
					}
				}
			}
		}
	}
}

func getCreatedFiles(files []models.File) []string {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	return paths
}