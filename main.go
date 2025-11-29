package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"

	"goscaffold/internal/stats"
	"goscaffold/internal/ui"
	"goscaffold/pkg/config"
	"goscaffold/pkg/git"
	"goscaffold/pkg/parser"
	"goscaffold/pkg/validator"
)

var (
	dryRun       bool
	useClipboard bool
	inputFile    string
	gitCommit    bool
	interactive  bool
	backup       bool
	watchMode    bool
)

func init() {
	importCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Preview without creating files")
	importCmd.Flags().BoolVarP(&useClipboard, "clipboard", "c", false, "Read from clipboard")
	importCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file path")
	importCmd.Flags().BoolVarP(&gitCommit, "git-commit", "g", false, "Auto-commit to git")
	importCmd.Flags().BoolVarP(&interactive, "interactive", "I", false, "Prompt for each file")
	importCmd.Flags().BoolVar(&backup, "backup", true, "Create .goscaffold-backup/")
	importCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch clipboard for changes")
	rootCmd.AddCommand(importCmd)
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import code blocks from AI chat",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.Load()

		if watchMode {
			runWatchMode(cfg)
			return
		}

		content := getInput()
		if strings.TrimSpace(content) == "" {
			color.Yellow("⚠️ No input provided!")
			return
		}

		files := parser.ParseMultiFormat(content)
		if len(files) == 0 {
			color.Red("❌ No valid code blocks found!")
			return
		}

		color.Cyan("📦 Found %d files to create", len(files))

		stats := stats.New()
		bar := progressbar.NewOptions(len(files),
			progressbar.OptionSetTheme(progressbar.Theme{Saucer: "█", SaucerPadding: "░"}),
			progressbar.OptionShowBytes(false),
			progressbar.OptionSetWidth(30),
		)

		for _, file := range files {
			if interactive {
				ui.ShowFilePreview(file.Path, file.Code)
				if !ui.PromptUser(fmt.Sprintf("Create %s?", file.Path)) {
					bar.Add(1)
					continue
				}
			}

			if err := processFile(file, cfg, stats); err != nil {
				color.Red("❌ %s", err)
			}
			bar.Add(1)
		}

		stats.Print()

		if gitCommit && len(stats.Languages) > 0 {
			color.Cyan("\n📝 Committing to git...")
			if err := git.CommitFiles(getCreatedFiles(files), "chore: scaffold AI-generated files"); err != nil {
				color.Red("Git error: %v", err)
			}
		}

		color.Green("\n🎉 All done!")
	},
}

func runWatchMode(cfg *config.Config) {
	color.Cyan("👀 Watching clipboard every 5 seconds (Ctrl+C to stop)...")

	lastContent := ""
	for {
		content := getClipboardContent()
		if content != "" && content != lastContent && containsCodeBlock(content) {
			color.Green("\n📋 New content detected!")
			files := parser.ParseMultiFormat(content)
			if len(files) > 0 {
				color.Cyan("Will create %d files (simulating - run without --watch to execute)", len(files))
			}
			lastContent = content
		}
		time.Sleep(5 * time.Second)
	}
}

func containsCodeBlock(s string) bool {
	return strings.Contains(s, "```") || strings.Contains(s, "---")
}

func getClipboardContent() string {
	cmd := exec.Command("powershell", "-command", "Get-Clipboard")
	out, _ := cmd.Output()
	return string(out)
}

func processFile(file parser.File, cfg *config.Config, s *stats.Stats) error {
	// Backup existing file
	if backup {
		if _, err := os.Stat(file.Path); err == nil {
			os.MkdirAll(".goscaffold-backup", 0755)
			backupPath := filepath.Join(".goscaffold-backup", file.Path+".backup")
			os.MkdirAll(filepath.Dir(backupPath), 0755)
			os.Rename(file.Path, backupPath)
		}
	}

	// Create directory
	os.MkdirAll(filepath.Dir(file.Path), 0755)

	// Validate syntax if validator exists
	for _, v := range cfg.Validators {
		if strings.HasSuffix(file.Path, v.Extension) {
			validator := validator.New(v.Extension, v.Command)
			if err := validator.Validate(file.Path, file.Code); err != nil {
				color.Yellow("⚠️ Validation warning for %s: %v", file.Path, err)
			}
		}
	}

	// Write file
	if err := os.WriteFile(file.Path, []byte(file.Code), 0644); err != nil {
		return fmt.Errorf("write %s: %w", file.Path, err)
	}

	s.AddFile(file.Path, file.Code)
	color.Green("  ✓ %s (%d bytes)", file.Path, len(file.Code))
	return nil
}
