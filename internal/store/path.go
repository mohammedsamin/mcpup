package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultConfigDirName  = ".mcpup"
	defaultConfigFileName = "config.json"
	envConfigPath         = "MCPUP_CONFIG"
)

// ResolveConfigPath returns an absolute config path.
// If inputPath is empty, ~/.mcpup/config.json is returned.
func ResolveConfigPath(inputPath string) (string, error) {
	trimmed := strings.TrimSpace(inputPath)
	if trimmed != "" {
		absPath, err := filepath.Abs(trimmed)
		if err != nil {
			return "", fmt.Errorf("resolve config path: %w", err)
		}
		return absPath, nil
	}

	if envPath := strings.TrimSpace(os.Getenv(envConfigPath)); envPath != "" {
		absPath, err := filepath.Abs(envPath)
		if err != nil {
			return "", fmt.Errorf("resolve config path: %w", err)
		}
		return absPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}
	return filepath.Join(home, defaultConfigDirName, defaultConfigFileName), nil
}
