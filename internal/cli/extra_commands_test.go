package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mohammedsamin/mcpup/internal/store"
)

func TestUpdateRefreshesRegistryBackedServer(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")
	runCLI(t, env, "add", "github", "--command", "echo", "--arg", "old", "--env", "GITHUB_TOKEN=test-token", "--description", "old")

	out := runCLI(t, env, "--json", "update", "github", "--yes")
	payload := parseJSONResult(t, out)
	if payload.Command != "update" || payload.Status != "ok" {
		t.Fatalf("expected successful update response, got command=%q status=%q", payload.Command, payload.Status)
	}

	cfg, err := store.LoadConfig(env.configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	srv := cfg.Servers["github"]
	if srv.Command != "npx" {
		t.Fatalf("expected updated command npx, got %q", srv.Command)
	}
	if len(srv.Args) < 2 || srv.Args[0] != "-y" {
		t.Fatalf("expected registry args after update, got %v", srv.Args)
	}
	if srv.Env["GITHUB_TOKEN"] != "test-token" {
		t.Fatalf("expected existing env values to be preserved")
	}
}

func TestUpdateDoesNotPersistCanonicalChangeWhenClientReconcileFails(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")
	runCLI(t, env, "add", "github", "--command", "echo", "--arg", "old", "--env", "GITHUB_TOKEN=test-token")
	runCLI(t, env, "enable", "github", "--client", "cursor")

	cursorPath := filepath.Join(env.home, ".cursor", "mcp.json")
	if err := os.WriteFile(cursorPath, []byte(`{"mcpServers":`), 0o644); err != nil {
		t.Fatalf("corrupt cursor config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{"update", "github", "--yes"}, nil, &stdout, &stderr)
	if err == nil {
		t.Fatalf("expected update to fail when client reconcile fails")
	}

	cfg, loadErr := store.LoadConfig(env.configPath)
	if loadErr != nil {
		t.Fatalf("load config after failed update: %v", loadErr)
	}
	if cfg.Servers["github"].Command != "echo" {
		t.Fatalf("expected canonical command to remain unchanged after failed update, got %q", cfg.Servers["github"].Command)
	}
}

func TestUpdateMigratesLegacyNotionHeadersToToken(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")
	runCLI(t, env,
		"add", "notion",
		"--command", "npx",
		"--arg", "-y",
		"--arg", "@notionhq/notion-mcp-server",
		"--env", `OPENAPI_MCP_HEADERS={"Authorization":"Bearer ntn_123","Notion-Version":"2022-06-28"}`,
	)

	runCLI(t, env, "update", "notion", "--yes")

	cfg, err := store.LoadConfig(env.configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	srv := cfg.Servers["notion"]
	if srv.Env["NOTION_TOKEN"] != "ntn_123" {
		t.Fatalf("expected NOTION_TOKEN after migration, got %+v", srv.Env)
	}
	if _, ok := srv.Env["OPENAPI_MCP_HEADERS"]; ok {
		t.Fatalf("expected OPENAPI_MCP_HEADERS to be removed after migration")
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	envA := setupTestEnv(t)
	runCLI(t, envA, "init")
	runCLI(t, envA, "add", "github", "--command", "echo", "--arg", "gh", "--env", "GITHUB_TOKEN=test-token")

	exportPath := filepath.Join(t.TempDir(), "servers.json")
	runCLI(t, envA, "export", "--output", exportPath)

	envB := setupTestEnv(t)
	runCLI(t, envB, "init")
	runCLI(t, envB, "import", exportPath)

	cfg, err := store.LoadConfig(envB.configPath)
	if err != nil {
		t.Fatalf("load imported config: %v", err)
	}
	srv, ok := cfg.Servers["github"]
	if !ok {
		t.Fatalf("expected github to be imported")
	}
	if srv.Command != "echo" {
		t.Fatalf("expected imported command echo, got %q", srv.Command)
	}
	if len(srv.Args) != 1 || srv.Args[0] != "gh" {
		t.Fatalf("expected imported args [gh], got %v", srv.Args)
	}
}

func TestCompletionScriptsAndSuggestions(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")
	runCLI(t, env, "add", "github", "--command", "echo", "--arg", "gh")

	bash := runCLI(t, env, "completion", "bash")
	if !strings.Contains(bash, "complete -F _mcpup_completions mcpup") {
		t.Fatalf("bash completion output missing complete directive")
	}
	zsh := runCLI(t, env, "completion", "zsh")
	if !strings.Contains(zsh, "compdef _mcpup_completions mcpup") {
		t.Fatalf("zsh completion output missing compdef directive")
	}
	fish := runCLI(t, env, "completion", "fish")
	if !strings.Contains(fish, "complete -c mcpup") {
		t.Fatalf("fish completion output missing complete directive")
	}

	suggestTop := runCLI(t, env, "__complete")
	if !strings.Contains(suggestTop, "update") || !strings.Contains(suggestTop, "completion") {
		t.Fatalf("top-level suggestions missing new commands: %q", suggestTop)
	}

	suggestEnable := runCLI(t, env, "__complete", "enable")
	if !strings.Contains(suggestEnable, "github") || !strings.Contains(suggestEnable, "--client") {
		t.Fatalf("enable suggestions should include server and --client: %q", suggestEnable)
	}
}
