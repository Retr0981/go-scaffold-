package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
)

func Read() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return readWindows()
	case "darwin":
		return readMac()
	default:
		return readLinux()
	}
}

func readWindows() (string, error) {
	cmd := exec.Command("powershell", "-command", "Get-Clipboard")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("windows clipboard: %w", err)
	}
	return string(out), nil
}

func readMac() (string, error) {
	cmd := exec.Command("pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("mac clipboard: %w", err)
	}
	return string(out), nil
}

func readLinux() (string, error) {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
	out, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("xsel", "--clipboard", "--output")
		out, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("linux clipboard: %w", err)
		}
	}
	return string(out), nil
}
