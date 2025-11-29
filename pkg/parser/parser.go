package parser

import (
	"regexp"
	"strings"
)

type File struct {
	Path string
	Code string
}

func Parse(content string) []File {
	var files []File

	// Pattern for: ```go path:filename.go
	re := regexp.MustCompile("(?s)```(?:[\\w\\+]+)?\\s+path:(\\S+)\\n(.*?)```")
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			files = append(files, File{
				Path: strings.TrimSpace(match[1]),
				Code: strings.TrimSpace(match[2]),
			})
		}
	}

	return files
}
