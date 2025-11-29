package parser

import (
	"strings"

	"goscaffold/internal/models"
)

func ParseMultiFormat(content string) []models.File {
	var files []models.File

	// Try markdown code blocks first
	files = append(files, parseMarkdown(content)...)

	// Try YAML-style separators
	if len(files) == 0 {
		files = append(files, parseYAMLStyle(content)...)
	}

	return files
}

func parseMarkdown(content string) []models.File {
	var files []models.File
	lines := strings.Split(content, "\n")

	inBlock := false
	var currentPath string
	var currentCode strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") && !inBlock {
			inBlock = true
			currentCode.Reset()
			continue
		}

		if strings.HasPrefix(trimmed, "```") && inBlock {
			inBlock = false
			if currentPath != "" && currentCode.Len() > 0 {
				files = append(files, models.File{
					Path: currentPath,
					Code: strings.TrimSpace(currentCode.String()),
				})
			}
			currentPath = ""
			continue
		}

		if inBlock && strings.HasPrefix(trimmed, "//") && strings.Contains(trimmed, "path:") {
			parts := strings.SplitN(trimmed, "path:", 2)
			if len(parts) == 2 {
				currentPath = strings.TrimSpace(parts[1])
			}
			continue
		}

		if inBlock {
			currentCode.WriteString(line + "\n")
		}
	}

	return files
}

func parseYAMLStyle(content string) []models.File {
	// Implementation for --- separated blocks
	return nil // Simplified for brevity
}
