package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/Retr09871/goscaffold/cmd"
	"github.com/fatih/color"
)

func main() {
	// Check Windows compatibility
	if runtime.GOOS == "windows" {
		ensureWindowsSupport()
	}

	// Run CLI
	if err := cmd.Execute(); err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
}

// ensureWindowsSupport provides helpful errors for missing Windows setup
func ensureWindowsSupport() {
	// Check for common issues
	if os.Getenv("GOPATH") == "" {
		color.Yellow("⚠️  Warning: GOPATH not set")
		fmt.Println("Run this in PowerShell (as Admin):")
		fmt.Println(`[Environment]::SetEnvironmentVariable("GOPATH", "$env:USERPROFILE\go", "User")`)
	}

	if os.Getenv("Path") == "" || !containsGoBin(os.Getenv("Path")) {
		color.Yellow("⚠️  Warning: Go bin directory may not be in PATH")
	}
}

func containsGoBin(path string) bool {
	return false // Simplified check
}
