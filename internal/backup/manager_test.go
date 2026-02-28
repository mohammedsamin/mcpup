package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotAndRestore(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "config.json")
	if err := os.WriteFile(source, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	manager := &Manager{RootDir: filepath.Join(tmp, "backups")}
	meta, err := manager.SnapshotFile("cursor", source, "test")
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if meta.BackupPath == "" || meta.Timestamp == "" {
		t.Fatalf("expected metadata to be populated")
	}

	if err := os.WriteFile(source, []byte(`{"a":2}`), 0o644); err != nil {
		t.Fatalf("mutate source: %v", err)
	}
	if err := manager.Restore(meta); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	restored, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read restored source: %v", err)
	}
	if string(restored) != `{"a":1}` {
		t.Fatalf("unexpected restored content: %s", string(restored))
	}
}

func TestSelectLatestAndCleanup(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "config.json")
	if err := os.WriteFile(source, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	manager := &Manager{RootDir: filepath.Join(tmp, "backups")}

	first, err := manager.SnapshotFile("cursor", source, "test-1")
	if err != nil {
		t.Fatalf("snapshot first: %v", err)
	}
	time.Sleep(time.Millisecond)
	second, err := manager.SnapshotFile("cursor", source, "test-2")
	if err != nil {
		t.Fatalf("snapshot second: %v", err)
	}

	latest, err := manager.SelectBackup("cursor", "")
	if err != nil {
		t.Fatalf("select latest: %v", err)
	}
	if latest.Timestamp != second.Timestamp {
		t.Fatalf("expected latest timestamp %q, got %q", second.Timestamp, latest.Timestamp)
	}

	if err := manager.Cleanup("cursor", 1); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	metas, err := manager.ListBackups("cursor")
	if err != nil {
		t.Fatalf("list backups: %v", err)
	}
	if len(metas) != 1 || metas[0].Timestamp != second.Timestamp {
		t.Fatalf("expected only latest backup to remain")
	}
	if _, err := os.Stat(first.BackupPath); !os.IsNotExist(err) {
		t.Fatalf("expected first backup file to be removed")
	}
}

func TestRollbackByTimestamp(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "config.json")
	if err := os.WriteFile(source, []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	manager := &Manager{RootDir: filepath.Join(tmp, "backups")}

	first, err := manager.SnapshotFile("cursor", source, "first")
	if err != nil {
		t.Fatalf("snapshot first: %v", err)
	}

	if err := os.WriteFile(source, []byte(`{"version":2}`), 0o644); err != nil {
		t.Fatalf("write source v2: %v", err)
	}
	if _, err := manager.SnapshotFile("cursor", source, "second"); err != nil {
		t.Fatalf("snapshot second: %v", err)
	}

	if _, err := manager.Rollback("cursor", first.Timestamp); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if string(data) != `{"version":1}` {
		t.Fatalf("unexpected rollback content: %s", string(data))
	}
}
