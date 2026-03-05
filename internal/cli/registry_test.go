package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mohammedsamin/mcpup/internal/store"
)

func TestAddUsesRegistryTemplateWhenCommandMissing(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")
	runCLI(t, env, "add", "github", "--env", "GITHUB_TOKEN=test-token")

	cfg, err := store.LoadConfig(env.configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	srv, ok := cfg.Servers["github"]
	if !ok {
		t.Fatalf("expected github server in config")
	}
	if srv.Command != "npx" {
		t.Fatalf("expected command npx from registry template, got %q", srv.Command)
	}
	if len(srv.Args) < 2 || srv.Args[0] != "-y" {
		t.Fatalf("expected registry args to be populated, got %v", srv.Args)
	}
	if srv.Env["GITHUB_TOKEN"] != "test-token" {
		t.Fatalf("expected env var from CLI input to be saved")
	}
}

func TestAddRegistryTemplateRequiresEnv(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")

	var stderr bytes.Buffer
	err := Run([]string{"add", "github"}, nil, &bytes.Buffer{}, &stderr)
	if err == nil {
		t.Fatalf("expected add github without required env to fail")
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN is required for github") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistryCommandJSONOutput(t *testing.T) {
	env := setupTestEnv(t)
	out := runCLI(t, env, "--json", "registry", "github")
	payload := parseJSONResult(t, out)
	if payload.Command != "registry" {
		t.Fatalf("expected registry command, got %q", payload.Command)
	}
	if payload.Status != "ok" {
		t.Fatalf("expected ok status, got %q", payload.Status)
	}

	serversAny, ok := payload.Data["servers"]
	if !ok {
		t.Fatalf("expected servers list in payload data")
	}
	servers, ok := serversAny.([]any)
	if !ok || len(servers) == 0 {
		t.Fatalf("expected non-empty servers list, got %T", serversAny)
	}
}
