package stats

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

type Stats struct {
	TotalFiles int
	TotalBytes int
	Languages  map[string]int
}

func New() *Stats {
	return &Stats{
		Languages: make(map[string]int),
	}
}

func (s *Stats) AddFile(path, code string) {
	s.TotalFiles++
	s.TotalBytes += len(code)

	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext == "" {
		ext = "unknown"
	}
	s.Languages[ext]++
}

func (s *Stats) Print() {
	log.Info("=== Statistics ===")
	log.Info(fmt.Sprintf("Files: %d", s.TotalFiles))
	log.Info(fmt.Sprintf("Bytes: %d", s.TotalBytes))
	for lang, count := range s.Languages {
		log.Info(fmt.Sprintf("  %s: %d", lang, count))
	}
}
