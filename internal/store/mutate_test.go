package store

import "testing"

func TestServerCRUD(t *testing.T) {
	cfg := NewDefaultConfig()

	server := Server{
		Command: "npx -y @modelcontextprotocol/server-github",
		Env:     map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	}

	if err := AddServer(&cfg, "github", server); err != nil {
		t.Fatalf("add server failed: %v", err)
	}
	if err := UpsertServer(&cfg, "github", Server{Command: "npx github-v2"}); err != nil {
		t.Fatalf("upsert server failed: %v", err)
	}
	if cfg.Servers["github"].Command != "npx github-v2" {
		t.Fatalf("expected server command to be updated")
	}
	if err := RemoveServer(&cfg, "github"); err != nil {
		t.Fatalf("remove server failed: %v", err)
	}
	if _, ok := cfg.Servers["github"]; ok {
		t.Fatalf("expected server to be removed")
	}
}

func TestProfileCRUD(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Servers["github"] = Server{Command: "npx github"}
	cfg.Servers["postgres"] = Server{Command: "npx postgres"}

	if err := CreateProfile(&cfg, "coding", Profile{Servers: []string{"github"}}); err != nil {
		t.Fatalf("create profile failed: %v", err)
	}

	if err := UpsertProfile(&cfg, "coding", Profile{Servers: []string{"github", "postgres"}}); err != nil {
		t.Fatalf("upsert profile failed: %v", err)
	}
	if len(cfg.Profiles["coding"].Servers) != 2 {
		t.Fatalf("expected profile servers to be updated")
	}

	if err := SetActiveProfile(&cfg, "coding"); err != nil {
		t.Fatalf("set active profile failed: %v", err)
	}
	if cfg.ActiveProfile != "coding" {
		t.Fatalf("expected active profile to be set")
	}

	if err := DeleteProfile(&cfg, "coding"); err != nil {
		t.Fatalf("delete profile failed: %v", err)
	}
	if cfg.ActiveProfile != "" {
		t.Fatalf("expected active profile to be cleared")
	}
}

func TestAddServerWithURL(t *testing.T) {
	cfg := NewDefaultConfig()

	server := Server{
		URL:     "https://api.example.com/mcp",
		Headers: map[string]string{"Authorization": "Bearer sk-xxx"},
	}
	if err := AddServer(&cfg, "remote", server); err != nil {
		t.Fatalf("add HTTP server failed: %v", err)
	}
	if !cfg.Servers["remote"].IsHTTP() {
		t.Fatalf("expected server to be HTTP")
	}
	if cfg.Servers["remote"].URL != "https://api.example.com/mcp" {
		t.Fatalf("expected URL to be set")
	}
}

func TestAddServerRejectsEmptyCommandAndURL(t *testing.T) {
	cfg := NewDefaultConfig()
	if err := AddServer(&cfg, "empty", Server{}); err == nil {
		t.Fatalf("expected error for empty command and url")
	}
}

func TestUpsertServerWithURL(t *testing.T) {
	cfg := NewDefaultConfig()

	server := Server{URL: "https://example.com/mcp"}
	if err := UpsertServer(&cfg, "remote", server); err != nil {
		t.Fatalf("upsert HTTP server failed: %v", err)
	}
	if cfg.Servers["remote"].URL != "https://example.com/mcp" {
		t.Fatalf("expected URL to be set")
	}

	// Update to command-based.
	if err := UpsertServer(&cfg, "remote", Server{Command: "npx server"}); err != nil {
		t.Fatalf("upsert to command server failed: %v", err)
	}
	if cfg.Servers["remote"].Command != "npx server" {
		t.Fatalf("expected command to be set")
	}
}

func TestClientServerAndToolUpdates(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Servers["github"] = Server{Command: "npx github"}

	if err := SetClientServerEnabled(&cfg, "cursor", "github", true); err != nil {
		t.Fatalf("set client server enabled failed: %v", err)
	}
	if !cfg.Clients["cursor"].Servers["github"].Enabled {
		t.Fatalf("expected server enabled")
	}

	if err := SetClientToolEnabled(&cfg, "cursor", "github", "search_issues", true); err != nil {
		t.Fatalf("enable tool failed: %v", err)
	}
	if err := SetClientToolEnabled(&cfg, "cursor", "github", "delete_issue", false); err != nil {
		t.Fatalf("disable tool failed: %v", err)
	}

	state := cfg.Clients["cursor"].Servers["github"]
	if len(state.EnabledTools) != 1 || state.EnabledTools[0] != "search_issues" {
		t.Fatalf("unexpected enabled tools: %v", state.EnabledTools)
	}
	if len(state.DisabledTools) != 1 || state.DisabledTools[0] != "delete_issue" {
		t.Fatalf("unexpected disabled tools: %v", state.DisabledTools)
	}
}
