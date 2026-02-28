package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"mcpup/internal/adapters"
	"mcpup/internal/planner"
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

	if _, ok := doc["extraTopLevel"]; !ok {
		t.Fatalf("expected extraTopLevel key to be preserved")
	}

	rawServers := map[string]map[string]json.RawMessage{}
	if err := json.Unmarshal(doc["mcpServers"], &rawServers); err != nil {
		t.Fatalf("decode mcpServers: %v", err)
	}
	if _, ok := rawServers["github"]["customField"]; !ok {
		t.Fatalf("expected github.customField to be preserved")
	}
	if _, ok := rawServers["slack"]; ok {
		t.Fatalf("expected slack server to be removed from desired state")
	}
}

func TestAdapterWriteFromEdgeFixture(t *testing.T) {
	tempPath := copyFixture(t, "edge.json")

	desired := planner.ClientState{
		Client: ClientName,
		Servers: map[string]planner.ServerState{
			"github": {
				Enabled: true,
			},
		},
	}

	if _, err := adapters.RunHarness(Adapter{}, tempPath, desired, false); err != nil {
		t.Fatalf("write from edge fixture failed: %v", err)
	}

	doc, err := adapters.ReadJSONDocument(tempPath)
	if err != nil {
		t.Fatalf("read written doc: %v", err)
	}
	if _, ok := doc["logging"]; !ok {
		t.Fatalf("expected logging key to be preserved")
	}

	state, err := adapters.ReadStateFromMCPServers(doc, ClientName)
	if err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if _, ok := state.Servers["github"]; !ok {
		t.Fatalf("expected github server to exist after write")
	}
}

func TestDetectPrefersWorkspaceFile(t *testing.T) {
	workspace := t.TempDir()
	workspaceFile := filepath.Join(workspace, ".mcp.json")
	if err := os.WriteFile(workspaceFile, []byte(`{}`), 0o644); err != nil {
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
	source := filepath.Join("..", "..", "..", "testdata", "fixtures", "claudecode", name)
	body := mustRead(t, source)

	target := filepath.Join(t.TempDir(), ".mcp.json")
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
