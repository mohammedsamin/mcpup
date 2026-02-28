package store

import "testing"

func TestValidateConfigSchemaPassesForDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	if err := ValidateConfigSchema(cfg); err != nil {
		t.Fatalf("expected default config valid, got %v", err)
	}
}

func TestValidateConfigSchemaRejectsUnknownClient(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Servers["github"] = Server{Command: "npx github"}
	cfg.Clients["unknown-client"] = ClientConfig{
		Servers: map[string]ServerState{
			"github": {Enabled: true},
		},
	}

	if err := ValidateConfigSchema(cfg); err == nil {
		t.Fatalf("expected unknown client validation error")
	}
}

func TestValidateConfigSchemaRejectsInvalidServerName(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Servers["Bad Name"] = Server{Command: "echo x"}

	if err := ValidateConfigSchema(cfg); err == nil {
		t.Fatalf("expected invalid server name error")
	}
}

func TestValidateConfigSchemaRejectsToolCollision(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Servers["github"] = Server{Command: "npx github"}
	cfg.Clients["cursor"] = ClientConfig{
		Servers: map[string]ServerState{
			"github": {
				Enabled:       true,
				EnabledTools:  []string{"search_issues"},
				DisabledTools: []string{"search_issues"},
			},
		},
	}

	if err := ValidateConfigSchema(cfg); err == nil {
		t.Fatalf("expected tool collision validation error")
	}
}

func TestDetectEnvValueMode(t *testing.T) {
	if mode := DetectEnvValueMode("${GITHUB_TOKEN}"); mode != EnvValueReference {
		t.Fatalf("expected reference mode")
	}
	if mode := DetectEnvValueMode("plain-value"); mode != EnvValueLiteral {
		t.Fatalf("expected literal mode")
	}
}
