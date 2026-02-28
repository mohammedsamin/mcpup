package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mcpup/internal/adapters"
	"mcpup/internal/planner"
)

func TestAdapterDryRunDoesNotWriteFile(t *testing.T) {
	tempPath := copyFixture(t, "normal.toml")
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
	tempPath := copyFixture(t, "normal.toml")

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

	if _, err := adapters.RunHarness(Adapter{}, tempPath, desired, false); err != nil {
		t.Fatalf("real-write harness failed: %v", err)
	}

	content := string(mustRead(t, tempPath))
	if !strings.Contains(content, `[core]`) {
		t.Fatalf("expected unknown core section to be preserved")
	}
	if !strings.Contains(content, managedStart) || !strings.Contains(content, managedEnd) {
		t.Fatalf("expected managed block markers")
	}

	state, err := Adapter{}.Read(tempPath)
	if err != nil {
		t.Fatalf("read after write failed: %v", err)
	}
	if _, ok := state.Servers["github"]; !ok {
		t.Fatalf("expected github server in state")
	}
}

func TestAdapterWriteFromEdgeFixture(t *testing.T) {
	tempPath := copyFixture(t, "edge.toml")
	desired := planner.ClientState{
		Client: ClientName,
		Servers: map[string]planner.ServerState{
			"context7": {Enabled: true},
		},
	}

	if _, err := adapters.RunHarness(Adapter{}, tempPath, desired, false); err != nil {
		t.Fatalf("write from edge fixture failed: %v", err)
	}

	content := string(mustRead(t, tempPath))
	if !strings.Contains(content, managedStart) {
		t.Fatalf("expected managed block to be appended")
	}
}

func TestDetectPrefersWorkspaceFile(t *testing.T) {
	workspace := t.TempDir()
	workspaceDir := filepath.Join(workspace, ".codex")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	workspaceFile := filepath.Join(workspaceDir, "config.toml")
	if err := os.WriteFile(workspaceFile, []byte(""), 0o644); err != nil {
		t.Fatalf("write workspace file: %v", err)
	}

	path, err := Adapter{}.Detect(workspace)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if path != workspaceFile {
		t.Fatalf("expected workspace path %q, got %q", workspaceFile, path)
	}
}

func copyFixture(t *testing.T, name string) string {
	t.Helper()
	source := filepath.Join("..", "..", "..", "testdata", "fixtures", "codex", name)
	body := mustRead(t, source)

	targetDir := filepath.Join(t.TempDir(), ".codex")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir temp fixture dir: %v", err)
	}
	target := filepath.Join(targetDir, "config.toml")
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
