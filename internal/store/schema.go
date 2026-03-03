package store

import "strings"

// CurrentSchemaVersion is the canonical config schema version for mcpup.
const CurrentSchemaVersion = 1

// SupportedClients are the v1 clients managed by mcpup.
var SupportedClients = []string{
	"claude-code",
	"cursor",
	"claude-desktop",
	"codex",
	"opencode",
	"windsurf",
	"zed",
	"continue",
}

// Config is the canonical mcpup state stored in ~/.mcpup/config.json.
type Config struct {
	Version       int                     `json:"version"`
	Servers       map[string]Server       `json:"servers"`
	Clients       map[string]ClientConfig `json:"clients"`
	Profiles      map[string]Profile      `json:"profiles,omitempty"`
	ActiveProfile string                  `json:"activeProfile,omitempty"`
}

// Server describes one MCP server definition in the registry.
type Server struct {
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Transport   string            `json:"transport,omitempty"` // "stdio", "sse", "streamable-http"
	Description string            `json:"description,omitempty"`
}

// IsHTTP reports whether the server is configured via URL rather than command.
func (s Server) IsHTTP() bool {
	return strings.TrimSpace(s.URL) != ""
}

// ClientConfig stores enable/disable state per server for one client.
type ClientConfig struct {
	Servers map[string]ServerState `json:"servers,omitempty"`
}

// ServerState stores server-level and tool-level control for one client.
type ServerState struct {
	Enabled       bool     `json:"enabled"`
	EnabledTools  []string `json:"enabledTools,omitempty"`
	DisabledTools []string `json:"disabledTools,omitempty"`
}

// Profile stores reusable server and tool selections.
type Profile struct {
	Servers []string                 `json:"servers,omitempty"`
	Tools   map[string]ToolSelection `json:"tools,omitempty"`
}

// ToolSelection defines per-server tool preferences inside a profile.
type ToolSelection struct {
	Enabled  []string `json:"enabled,omitempty"`
	Disabled []string `json:"disabled,omitempty"`
}

// NewDefaultConfig returns an initialized empty config at current schema version.
func NewDefaultConfig() Config {
	return Config{
		Version:       CurrentSchemaVersion,
		Servers:       map[string]Server{},
		Clients:       map[string]ClientConfig{},
		Profiles:      map[string]Profile{},
		ActiveProfile: "",
	}
}
