package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new [project-name]",
	Short: "Create a new Go project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]

		// Windows-safe path handling
		projectPath := filepath.Join(".", projectName)
		if runtime.GOOS == "windows" {
			projectPath = filepath.Clean(projectPath)
		}

		fmt.Printf("Creating project %s at %s\n", projectName, projectPath)

		// Create directory structure
		dirs := []string{
			projectPath,
			filepath.Join(projectPath, "cmd"),
			filepath.Join(projectPath, "internal"),
			filepath.Join(projectPath, "pkg"),
		}

		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create %s: %w", dir, err)
			}
			color.Green("✓ Created %s", dir)
		}

		// Create main.go with Windows clipboard fix
		mainContent := `package main

import "fmt"

func main() {
    fmt.Println("Welcome to %s!")
}
`
		mainPath := filepath.Join(projectPath, "cmd", "main.go")
		if err := os.WriteFile(mainPath, []byte(fmt.Sprintf(mainContent, projectName)), 0644); err != nil {
			return err
		}

		// Create go.mod
		modContent := fmt.Sprintf(`module %s

go 1.22
`, projectName)

		modPath := filepath.Join(projectPath, "go.mod")
		if err := os.WriteFile(modPath, []byte(modContent), 0644); err != nil {
			return err
		}

		color.Cyan("\n✨ Project %s created successfully!", projectName)
		color.Yellow("Next steps:")
		fmt.Printf("  cd %s\n", projectName)
		fmt.Println("  go mod tidy")
		return nil
	},
}
