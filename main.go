package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

// File represents a parsed file from clipboard
type File struct {
	Path string
	Code string
}

func main() {
	// Check if clipboard tool exists
	if !commandExists("pbpaste") && !commandExists("xclip") && !commandExists("wl-paste") {
		color.Red("❌ Clipboard tool not found. Install:")
		color.Yellow("  macOS: Already have pbpaste")
		color.Yellow("  Linux: sudo apt install xclip  # or wl-clipboard for Wayland")
		os.Exit(1)
	}

	color.Cyan("📋 GoScaffold - AI Code to Files")
	fmt.Println("   Press Ctrl+V then Ctrl+D to paste from clipboard...")

	// Read from stdin (piped clipboard)
	reader := bufio.NewReader(os.Stdin)
	var input strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input.WriteString(line)
	}

	if input.Len() == 0 {
		color.Red("❌ No input received!")
		color.Yellow("   Usage: pbpaste | goscaffold    (macOS)")
		color.Yellow("   Usage: xclip -o | goscaffold   (Linux)")
		os.Exit(1)
	}

	// Parse files
	files := parseFiles(input.String())
	if len(files) == 0 {
		color.Red("❌ No code blocks found!")
		fmt.Println("   Make sure your clipboard has:")
		fmt.Println("   --- filename.js ---")
		fmt.Println("   ```code```")
		os.Exit(1)
	}

	// Create files
	color.Green("\n✅ Found %d files to create:", len(files))
	for _, file := range files {
		createFile(file)
	}

	color.Green("\n🎉 All files created successfully!")
}

func parseFiles(text string) []File {
	var files []File
	
	// Regex pattern: --- path ---\n```lang\ncode\n```
	pattern := regexp.MustCompile(`---\s*(.+?)\s*---\s*\n` + "```" + `\w*\n([\s\S]*?)\n` + "```")
	matches := pattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			files = append(files, File{
				Path: strings.TrimSpace(match[1]),
				Code: strings.TrimSpace(match[2]),
			})
		}
	}

	// Fallback: Try without language spec
	if len(files) == 0 {
		pattern = regexp.MustCompile(`---\s*(.+?)\s*---\s*\n` + "```" + `\n([\s\S]*?)\n` + "```")
		matches = pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				files = append(files, File{
					Path: strings.TrimSpace(match[1]),
					Code: strings.TrimSpace(match[2]),
				})
			}
		}
	}

	return files
}

func createFile(file File) {
	// Create directory if needed
	dir := filepath.Dir(file.Path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			color.Red("❌ Failed to create directory %s: %v", dir, err)
			return
		}
	}

	// Write file
	if err := os.WriteFile(file.Path, []byte(file.Code), 0644); err != nil {
		color.Red("❌ Failed to write %s: %v", file.Path, err)
		return
	}

	color.Green("  ✓ Created: %s", file.Path)
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}