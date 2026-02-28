package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const timestampLayout = "20060102T150405.000000000Z0700"

// Metadata describes a backup snapshot.
type Metadata struct {
	Client        string `json:"client"`
	SourcePath    string `json:"sourcePath"`
	Command       string `json:"command"`
	Timestamp     string `json:"timestamp"`
	SHA256        string `json:"sha256"`
	BackupPath    string `json:"backupPath"`
	SourceMissing bool   `json:"sourceMissing"`
}

// Manager provides snapshot, restore, and retention operations.
type Manager struct {
	RootDir string
}

// NewManager returns a backup manager rooted at ~/.mcpup/backups.
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &Manager{
		RootDir: filepath.Join(home, ".mcpup", "backups"),
	}, nil
}

// SnapshotFile copies sourcePath into backup storage and writes metadata.
func (m *Manager) SnapshotFile(client string, sourcePath string, command string) (Metadata, error) {
	now := time.Now().UTC()
	ts := now.Format(timestampLayout)
	clientDir := filepath.Join(m.RootDir, client)
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		return Metadata{}, err
	}

	meta := Metadata{
		Client:     client,
		SourcePath: sourcePath,
		Command:    command,
		Timestamp:  ts,
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			meta.SourceMissing = true
			data = []byte{}
		} else {
			return Metadata{}, err
		}
	}

	sum := sha256.Sum256(data)
	meta.SHA256 = hex.EncodeToString(sum[:])
	meta.BackupPath = filepath.Join(clientDir, ts+".bak")

	if err := os.WriteFile(meta.BackupPath, data, 0o644); err != nil {
		return Metadata{}, err
	}
	if err := m.writeMetadata(meta); err != nil {
		return Metadata{}, err
	}

	return meta, nil
}

// Restore writes the snapshot data back to sourcePath.
func (m *Manager) Restore(meta Metadata) error {
	data, err := os.ReadFile(meta.BackupPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(meta.SourcePath), 0o755); err != nil {
		return err
	}

	return os.WriteFile(meta.SourcePath, data, 0o644)
}

// SelectBackup picks a backup by timestamp, or latest when targetTimestamp is empty.
func (m *Manager) SelectBackup(client string, targetTimestamp string) (Metadata, error) {
	metas, err := m.ListBackups(client)
	if err != nil {
		return Metadata{}, err
	}
	if len(metas) == 0 {
		return Metadata{}, fmt.Errorf("no backups found for client %q", client)
	}

	if strings.TrimSpace(targetTimestamp) == "" {
		return metas[len(metas)-1], nil
	}

	for _, meta := range metas {
		if meta.Timestamp == targetTimestamp {
			return meta, nil
		}
	}
	return Metadata{}, fmt.Errorf("backup timestamp %q not found for client %q", targetTimestamp, client)
}

// Rollback selects a snapshot and restores it to the source path.
func (m *Manager) Rollback(client string, targetTimestamp string) (Metadata, error) {
	meta, err := m.SelectBackup(client, targetTimestamp)
	if err != nil {
		return Metadata{}, err
	}
	if err := m.Restore(meta); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

// ListBackups returns backups sorted by timestamp ascending.
func (m *Manager) ListBackups(client string) ([]Metadata, error) {
	clientDir := filepath.Join(m.RootDir, client)
	entries, err := os.ReadDir(clientDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Metadata{}, nil
		}
		return nil, err
	}

	metas := make([]Metadata, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".meta.json") {
			continue
		}
		path := filepath.Join(clientDir, entry.Name())
		meta, err := readMetadata(path)
		if err != nil {
			return nil, err
		}
		metas = append(metas, meta)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Timestamp < metas[j].Timestamp
	})
	return metas, nil
}

// Cleanup keeps the newest `keep` backups and removes older snapshots and metadata.
func (m *Manager) Cleanup(client string, keep int) error {
	if keep < 0 {
		return fmt.Errorf("keep must be >= 0")
	}

	metas, err := m.ListBackups(client)
	if err != nil {
		return err
	}
	if len(metas) <= keep {
		return nil
	}

	remove := metas[:len(metas)-keep]
	for _, meta := range remove {
		_ = os.Remove(meta.BackupPath)
		metaPath := filepath.Join(m.RootDir, client, meta.Timestamp+".meta.json")
		_ = os.Remove(metaPath)
	}

	return nil
}

func (m *Manager) writeMetadata(meta Metadata) error {
	metaPath := filepath.Join(m.RootDir, meta.Client, meta.Timestamp+".meta.json")
	body, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(metaPath, body, 0o644)
}

func readMetadata(path string) (Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, err
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}
