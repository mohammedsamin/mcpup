package zed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mohammedsamin/mcpup/internal/adapters"
	"github.com/mohammedsamin/mcpup/internal/planner"
)

func TestAdapterDryRunDoesNotWriteFile(t *testing.T) {
	tempPath := copyFixture(t, "normal.json")
	original := mustRead(t, tempPath)

	desired := planner.ClientState{
		Client: ClientName,
		Servers: map[string]planner.ServerState{
			"github": {
				Enabled:       true,
				EnabledTools:  []string{"search_issues"},
				DisabledTools: []string{"delete_issue"},
			},
		},
	}

	result, err := adapters.RunHarness(Adapter{}, tempPath, desired, true)
	if err != nil {
		t.Fatalf("dry-run harness failed: %v", err)
	}
	if !result.Plan.HasChanges() {
		t.Fatalf("expected dry-run plan to include changes")
	}

	current := mustRead(t, tempPath)
	if string(current) != string(original) {
		t.Fatalf("dry-run modified file contents")
	}
}

func TestAdapterRealWritePreservesUnknownKeys(t *testing.T) {
	tempPath := copyFixture(t, "normal.json")

	desired := planner.ClientState{
		Client: ClientName,
		Servers: map[string]planner.ServerState{
			"github": {
				Enabled:       true,
				EnabledTools:  []string{"search_issues"},
				DisabledTools: []string{"delete_issue"},
			},
		},
	}

	result, err := adapters.RunHarness(Adapter{}, tempPath, desired, false)
	if err != nil {
		t.Fatalf("real-write harness failed: %v", err)
	}
	if !result.Plan.HasChanges() {
		t.Fatalf("expected real-write plan to include changes")
	}

	doc, err := adapters.ReadJSONDocument(tempPath)
	if err != nil {
		t.Fatalf("read written doc: %v", err)
	}
	if _, ok := doc["assistant"]; !ok {
		t.Fatalf("expected assistant top-level key to be preserved")
	}

	rawServers := map[string]map[string]json.RawMessage{}
	if err := json.Unmarshal(doc["context_servers"], &rawServers); err != nil {
		t.Fatalf("decode context_servers: %v", err)
	}
	if _, ok := rawServers["github"]["zedOnly"]; !ok {
		t.Fatalf("expected github.zedOnly to be preserved")
	}
	if _, ok := rawServers["notion"]; !ok {
		t.Fatalf("expected unmanaged notion server to be preserved")
	}
}

func TestAdapterWriteFromEdgeFixture(t *testing.T) {
	tempPath := copyFixture(t, "edge.json")

	desired := planner.ClientState{
		Client: ClientName,
		Servers: map[string]planner.ServerState{
			"context7": {Enabled: true},
		},
	}

	if _, err := adapters.RunHarness(Adapter{}, tempPath, desired, false); err != nil {
		t.Fatalf("write from edge fixture failed: %v", err)
	}

	doc, err := adapters.ReadJSONDocument(tempPath)
	if err != nil {
		t.Fatalf("read written doc: %v", err)
	}
	if _, ok := doc["theme"]; !ok {
		t.Fatalf("expected theme key to be preserved")
	}
}

func TestDetectReturnsExpectedOSPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config-custom"))

	got, err := Adapter{}.Detect("")
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}

	var want string
	switch runtime.GOOS {
	case "darwin":
		want = filepath.Join(home, ".zed", "settings.json")
	case "windows":
		want = filepath.Join(home, "AppData", "Roaming", "Zed", "settings.json")
	default:
		want = filepath.Join(home, ".config-custom", "zed", "settings.json")
	}
	if got != want {
		t.Fatalf("expected detect path %q, got %q", want, got)
	}
}

func copyFixture(t *testing.T, name string) string {
	t.Helper()
	source := filepath.Join("..", "..", "..", "testdata", "fixtures", "zed", name)
	body := mustRead(t, source)

	targetDir := filepath.Join(t.TempDir(), ".config", "zed")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir temp fixture dir: %v", err)
	}
	target := filepath.Join(targetDir, "settings.json")
	if err := os.WriteFile(target, body, 0o644); err != nil {
		t.Fatalf("write temp fixture: %v", err)
	}
	return target
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	return data
}
