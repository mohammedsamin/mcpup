package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EnsureConfig resolves the path and ensures a valid config exists.
// If the config file does not exist, it creates a new default config.
func EnsureConfig(inputPath string) (string, Config, error) {
	resolvedPath, err := ResolveConfigPath(inputPath)
	if err != nil {
		return "", Config{}, err
	}

	cfg, err := LoadConfig(resolvedPath)
	if err == nil {
		return resolvedPath, cfg, nil
	}

	var storeErr *StoreError
	if errors.As(err, &storeErr) && storeErr.Kind == KindNotFound {
		cfg = NewDefaultConfig()
		if writeErr := SaveConfig(resolvedPath, cfg); writeErr != nil {
			return "", Config{}, writeErr
		}
		return resolvedPath, cfg, nil
	}

	return "", Config{}, err
}

// LoadConfig reads and validates config from disk using strict JSON decode rules.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, newStoreError("read", path, KindNotFound, err)
		}
		return Config{}, newStoreError("read", path, KindIO, err)
	}

	var cfg Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, newStoreError("decode", path, KindDecode, err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("unexpected trailing JSON value")
		}
		return Config{}, newStoreError("decode", path, KindDecode, err)
	}

	normalizeConfig(&cfg)
	if err := ValidateConfigSchema(cfg); err != nil {
		return Config{}, newStoreError("validate", path, KindValidation, err)
	}

	return cfg, nil
}

// SaveConfig validates and writes config with atomic replace semantics.
func SaveConfig(path string, cfg Config) error {
	normalizeConfig(&cfg)
	if err := ValidateConfigSchema(cfg); err != nil {
		return newStoreError("validate", path, KindValidation, err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return newStoreError("mkdir", dir, KindIO, err)
	}

	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return newStoreError("encode", path, KindIO, err)
	}
	body = append(body, '\n')

	tempFile, err := os.CreateTemp(dir, ".mcpup-config-*.tmp")
	if err != nil {
		return newStoreError("create-temp", dir, KindIO, err)
	}
	tempPath := tempFile.Name()

	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(body); err != nil {
		_ = tempFile.Close()
		return newStoreError("write-temp", tempPath, KindIO, err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return newStoreError("sync-temp", tempPath, KindIO, err)
	}
	if err := tempFile.Close(); err != nil {
		return newStoreError("close-temp", tempPath, KindIO, err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return newStoreError("rename", fmt.Sprintf("%s -> %s", tempPath, path), KindIO, err)
	}

	return nil
}

func normalizeConfig(cfg *Config) {
	if cfg.Version == 0 {
		cfg.Version = CurrentSchemaVersion
	}
	if cfg.Servers == nil {
		cfg.Servers = map[string]Server{}
	}
	if cfg.Clients == nil {
		cfg.Clients = map[string]ClientConfig{}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	for clientName, state := range cfg.Clients {
		if state.Servers == nil {
			state.Servers = map[string]ServerState{}
		}
		cfg.Clients[clientName] = state
	}
}
