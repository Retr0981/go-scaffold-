package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

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
)

// ...existing code...
var importCmd = &cobra.Command{
	Use:     "import [flags]",
	Short:   "Import AI-generated code blocks",
	Long:    `Parse code blocks from input and create files. Supports markdown (```) and clipboard.`,
	Example: `  goscaffold import --clipboard
	goscaffold import --input chat.md --git-commit
	cat output.md | goscaffold import -i -`,
	Aliases:  []string{"i"},
	RunE:     runImport,
}

func init() {
	importCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Preview without writing")
	importCmd.Flags().BoolVarP(&useClipboard, "clipboard", "c", false, "Read from clipboard")
	importCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file (- for stdin)")
	importCmd.Flags().BoolVarP(&gitCommit, "git-commit", "g", false, "Auto-commit")
	importCmd.Flags().BoolVarP(&interactive, "interactive", "I", false, "Interactive mode")
	importCmd.Flags().BoolVar(&backupFiles, "backup", viper.GetBool("backup.enabled"), "Create backups")
	importCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch file for changes")
	
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if watchMode {
		return runWatchMode(ctx)
	}

	content, err := getInput(ctx)
	if err != nil {
		return fmt.Errorf("input error: %w", err)
	}

	files := parser.Parse(content)
	if len(files) == 0 {
		return fmt.Errorf("no valid code blocks found")
	}

	log.Info(fmt.Sprintf("Found %d files", len(files)))

	if dryRun {
		return runDryRun(files)
	}

	if interactive {
		return ui.RunInteractive(files, gitCommit, backupFiles)
	}

	return runBatch(ctx, files)
}

func getInput(ctx context.Context) (string, error) {
	// Priority: clipboard > file > stdin
	if useClipboard {
		return clipboard.Read()
	}

	if inputFile != "" {
		if inputFile == "-" {
			return readStdin()
		}
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// Auto-detect clipboard
	if content, _ := clipboard.Read(); content != "" {
		log.Debug("Using clipboard")
		return content, nil
	}

	return "", fmt.Errorf("no input source specified")
}

func readStdin() (string, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", fmt.Errorf("no stdin data")
	}

	data, err := os.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func runDryRun(files []models.File) error {
	log.Info("=== DRY RUN ===")
	for _, f := range files {
		action := "create"
		if _, err := os.Stat(f.Path); err == nil {
			action = "update"
		}
		log.Info(fmt.Sprintf("Would %s: %s (%d bytes)", action, f.Path, len(f.Code)))
	}
	return nil
}

func runBatch(ctx context.Context, files []models.File) error {
	s := stats.New()
	bm := backup.NewManager(viper.GetString("backup.retention"))

	// Process with concurrency limit
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(4)

	for _, file := range files {
		f := file // capture range variable
		g.Go(func() error {
			return processFile(ctx, f, s, bm)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	s.Print()

	// Git commit
	if gitCommit && s.TotalFiles > 0 {
		log.Info("Committing to git...")
		paths := make([]string, len(files))
		for i, f := range files {
			paths[i] = f.Path
		}
		if err := git.Commit(ctx, paths, "chore(scaffold): import AI files"); err != nil {
			log.Warn("Git commit failed", "error", err)
		}
	}

	log.Info("✨ Import complete")
	return nil
}

func processFile(ctx context.Context, file models.File, s *stats.Stats, bm *backup.Manager) error {
	// Backup
	if backupFiles {
		_ = bm.Backup(file.Path)
	}

	// Ensure directory
	dir := filepath.Dir(file.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	// Validate
	if v, err := validator.Get(file.Path); err == nil {
		if err := v.Validate(ctx, file.Path, file.Code); err != nil {
			log.Warn("Validation warning", "file", file.Path, "error", err)
		}
	}

	// Write
	if err := os.WriteFile(file.Path, []byte(file.Code), 0644); err != nil {
		return fmt.Errorf("write %s: %w", file.Path, err)
	}

	s.AddFile(file.Path, file.Code)
	log.Debug("Created file", "path", file.Path, "size", len(file.Code))
	return nil
}

func runWatchMode(ctx context.Context) error {
	if inputFile == "" {
		return fmt.Errorf("--watch requires --input")
	}

	log.Info("Watching file", "path", inputFile)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(inputFile); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Info("File changed, reprocessing...")
				content, _ := os.ReadFile(inputFile)
				files := parser.Parse(string(content))
				if len(files) > 0 {
					_ = runBatch(ctx, files)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Error("Watch error", "error", err)
		}
	}
}