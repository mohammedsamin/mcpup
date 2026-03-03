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

func TestValidateConfigSchemaAcceptsHTTPServer(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Servers["remote"] = Server{
		URL:       "https://api.example.com/mcp",
		Headers:   map[string]string{"Authorization": "Bearer token"},
		Transport: "sse",
	}

	if err := ValidateConfigSchema(cfg); err != nil {
		t.Fatalf("expected HTTP server to be valid, got %v", err)
	}
}

func TestValidateConfigSchemaRejectsCommandAndURL(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Servers["bad"] = Server{
		Command: "npx server",
		URL:     "https://example.com/mcp",
	}

	if err := ValidateConfigSchema(cfg); err == nil {
		t.Fatalf("expected error for server with both command and url")
	}
}

func TestValidateConfigSchemaRejectsInvalidTransport(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Servers["bad"] = Server{
		URL:       "https://example.com/mcp",
		Transport: "invalid-transport",
	}

	if err := ValidateConfigSchema(cfg); err == nil {
		t.Fatalf("expected error for invalid transport")
	}
}

func TestValidateConfigSchemaRejectsEmptyCommandAndURL(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Servers["empty"] = Server{}

	if err := ValidateConfigSchema(cfg); err == nil {
		t.Fatalf("expected error for server with no command or url")
	}
}

func TestValidateConfigSchemaRejectsInvalidHeaderKey(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Servers["bad"] = Server{
		URL:     "https://example.com/mcp",
		Headers: map[string]string{"Invalid-Key!": "value"},
	}

	if err := ValidateConfigSchema(cfg); err == nil {
		t.Fatalf("expected error for invalid header key")
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
