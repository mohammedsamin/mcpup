package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mohammedsamin/mcpup/internal/store"
)

func TestRunDoctorWithValidCanonicalConfig(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	t.Setenv("HOME", home)
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("MCPUP_CONFIG", configPath)

	cfg := store.NewDefaultConfig()
	cfg.Servers["echo"] = store.Server{Command: "echo hello"}
	if err := store.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	report, err := RunDoctor("", tmp)
	if err != nil {
		t.Fatalf("run doctor: %v", err)
	}
	if len(report.Checks) == 0 {
		t.Fatalf("expected checks in report")
	}
	if _, err := os.Stat(filepath.Join(home, ".cursor")); !os.IsNotExist(err) {
		t.Fatalf("doctor should not create client config directories, got err=%v", err)
	}
}

func TestRunDoctorFailsSchemaCheckForInvalidConfig(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	t.Setenv("HOME", home)
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("MCPUP_CONFIG", configPath)

	if err := os.WriteFile(configPath, []byte(`{"bad":true}`), 0o644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	report, err := RunDoctor("", tmp)
	if err != nil {
		t.Fatalf("run doctor: %v", err)
	}

	foundFailure := false
	for _, check := range report.Checks {
		if check.Key == "config.schema" && check.Status == StatusFail {
			foundFailure = true
			break
		}
	}
	if !foundFailure {
		t.Fatalf("expected config.schema failure")
	}
}

func TestRunDoctorReportsLegacyRegistryDefinitions(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	t.Setenv("HOME", home)
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("MCPUP_CONFIG", configPath)

	cfg := store.NewDefaultConfig()
	cfg.Servers["notion"] = store.Server{
		Command: "npx",
		Args:    []string{"-y", "@notionhq/notion-mcp-server"},
		Env: map[string]string{
			"OPENAPI_MCP_HEADERS": `{"Authorization":"Bearer ntn_legacy","Notion-Version":"2022-06-28"}`,
		},
	}
	cfg.Clients["cursor"] = store.ClientConfig{
		Servers: map[string]store.ServerState{
			"notion": {Enabled: true},
		},
	}
	if err := store.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	report, err := RunDoctor("", tmp)
	if err != nil {
		t.Fatalf("run doctor: %v", err)
	}

	for _, check := range report.Checks {
		if check.Key == "registry.legacy.notion" {
			if check.Status != StatusWarn {
				t.Fatalf("expected legacy notion warning, got %s", check.Status)
			}
			return
		}
	}
	t.Fatalf("expected registry.legacy.notion check")
}
