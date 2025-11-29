package clipboard

import (
	"fmt"

	"golang.design/x/clipboard"
)

func init() {
	if err := clipboard.Init(); err != nil {
		// Fallback to platform-specific methods
	}
}

func Read() (string, error) {
	data := clipboard.Read(clipboard.FmtText)
	if len(data) == 0 {
		return "", fmt.Errorf("clipboard empty or not available")
	}
	return string(data), nil
}

func Write(content string) error {
	clipboard.Write(clipboard.FmtText, []byte(content))
	return nil
}

func IsAvailable() bool {
	_, err := Read()
	return err == nil
}
