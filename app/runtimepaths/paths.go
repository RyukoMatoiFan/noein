package runtimepaths

import (
	"fmt"
	"os"
	"path/filepath"
)

func NoeinRuntimeDir() (string, error) {
	if v := os.Getenv("NOEIN_RUNTIME_DIR"); v != "" {
		if err := os.MkdirAll(v, 0755); err != nil {
			return "", fmt.Errorf("failed to create NOEIN_RUNTIME_DIR: %w", err)
		}
		return v, nil
	}

	if wd, err := os.Getwd(); err == nil && wd != "" {
		base := filepath.Join(wd, "noein")
		if err := os.MkdirAll(base, 0755); err == nil {
			return base, nil
		}
	}

	if exePath, err := os.Executable(); err == nil && exePath != "" {
		exeDir := filepath.Dir(exePath)
		base := filepath.Join(exeDir, "noein")
		if err := os.MkdirAll(base, 0755); err == nil {
			return base, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home dir: %w", err)
	}
	base := filepath.Join(home, "noein")
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", fmt.Errorf("failed to create noein runtime dir: %w", err)
	}
	return base, nil
}
