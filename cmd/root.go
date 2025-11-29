package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goscaffold",
	Short: "A cross-platform Go project scaffold generator",
	Long:  `goscaffold creates project structures that work seamlessly on Windows, macOS, and Linux.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add subcommands here
	rootCmd.AddCommand(newCmd)
}
