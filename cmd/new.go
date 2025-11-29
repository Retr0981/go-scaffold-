package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	templateName string
	modules      []string
	overwrite    bool
	initGit      bool
)

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create new Go project",
	Args:  cobra.ExactArgs(1),
	Example: `  goscaffold new myapp
  goscaffold new myapi --modules=gin,zerolog`,
	RunE: runNew,
}

func init() {
	newCmd.Flags().StringVarP(&templateName, "template", "t", "default", "Project template")
	newCmd.Flags().StringSliceVarP(&modules, "modules", "m", []string{}, "Go modules to init")
	newCmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing")
	newCmd.Flags().BoolVar(&initGit, "git", false, "Initialize git repo")

	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]
	path := filepath.Join(".", name)

	if _, err := os.Stat(path); err == nil && !overwrite {
		return fmt.Errorf("directory %s already exists (use --overwrite)", name)
	}

	log.Info("Creating project", "name", name, "path", path)

	// Create structure
	dirs := []string{
		path,
		filepath.Join(path, "cmd"),
		filepath.Join(path, "internal"),
		filepath.Join(path, "pkg"),
		filepath.Join(path, "api"),
		filepath.Join(path, "configs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		log.Debug("Created directory", "path", dir)
	}

	// Create main.go
	mainTmpl := `package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello from {{.Name}}!")
}
`
	if err := writeTemplate(filepath.Join(path, "cmd", "main.go"), mainTmpl, map[string]string{"Name": name}); err != nil {
		return err
	}

	// Create go.mod
	modContent := fmt.Sprintf(`module %s

go 1.22
`, name)

	if err := os.WriteFile(filepath.Join(path, "go.mod"), []byte(modContent), 0644); err != nil {
		return err
	}

	// Create .gitignore
	gitignore := `.env
*.log
.goscaffold-backup/
`
	os.WriteFile(filepath.Join(path, ".gitignore"), []byte(gitignore), 0644)

	// Init git
	if initGit || viper.GetBool("git.auto_init") {
		if err := initGitRepo(path); err != nil {
			log.Warn("Git init failed", "error", err)
		}
	}

	log.Info("✨ Project created", "name", name, "path", path)
	return nil
}

func writeTemplate(path, tmpl string, data interface{}) error {
	t, err := template.New("file").Parse(tmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}

func initGitRepo(path string) error {
	// Simplified git init
	return nil
}
