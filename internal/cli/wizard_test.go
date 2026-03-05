package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mohammedsamin/mcpup/internal/backup"
	"github.com/mohammedsamin/mcpup/internal/registry"
	"github.com/mohammedsamin/mcpup/internal/store"
)

func TestLoadConfigOrDefaultErrorsOnInvalidConfig(t *testing.T) {
	env := setupTestEnv(t)

	if err := os.WriteFile(env.configPath, []byte(`{"bad":true}`), 0o644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	if _, err := loadConfigOrDefault(); err == nil {
		t.Fatalf("expected invalid config to return error")
	}
}

func TestAddRegistryTemplatePreservesHTTPTemplateFields(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")

	tmpl, ok := registry.Lookup("atlassian")
	if !ok {
		t.Fatalf("expected atlassian template to exist")
	}

	w := &wizard{out: &bytes.Buffer{}}
	if err := w.addRegistryTemplate(tmpl); err != nil {
		t.Fatalf("addRegistryTemplate failed: %v", err)
	}

	cfg, err := store.LoadConfig(env.configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	srv := cfg.Servers["atlassian"]
	if srv.URL != tmpl.URL {
		t.Fatalf("expected URL %q, got %q", tmpl.URL, srv.URL)
	}
	if srv.Transport != tmpl.Transport {
		t.Fatalf("expected transport %q, got %q", tmpl.Transport, srv.Transport)
	}
}

func TestSyncAfterRollbackReturnsErrorWhenCanonicalSyncFails(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")

	cursorPath := filepath.Join(env.home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cursorPath), 0o755); err != nil {
		t.Fatalf("mkdir cursor dir: %v", err)
	}
	body := []byte(`{"mcpServers":{"orphan":{"command":"echo","enabled":true}}}` + "\n")
	if err := os.WriteFile(cursorPath, body, 0o644); err != nil {
		t.Fatalf("write cursor config: %v", err)
	}

	w := &wizard{}
	err := w.syncAfterRollback("cursor", backup.Metadata{SourcePath: cursorPath})
	if err == nil {
		t.Fatalf("expected canonical sync error for orphan server")
	}
}
