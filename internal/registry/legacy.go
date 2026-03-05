package registry

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/store"
)

// LegacyDefinitionReason reports whether server matches a known stale mcpup-generated shape.
func LegacyDefinitionReason(name string, server store.Server) string {
	switch name {
	case "elevenlabs":
		if strings.TrimSpace(server.Command) == "npx" && slices.Equal(server.Args, []string{"-y", "@anthropic/mcp-elevenlabs"}) {
			return "uses the retired Anthropic ElevenLabs package name"
		}
	case "playwright":
		if strings.TrimSpace(server.Command) == "npx" && slices.Equal(server.Args, []string{"-y", "@anthropic/mcp-playwright"}) {
			return "uses the retired Anthropic Playwright package name"
		}
	case "filesystem":
		if strings.TrimSpace(server.Command) == "npx" && slices.Equal(server.Args, []string{"-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/dir"}) {
			return "uses the old placeholder filesystem path"
		}
	case "notion":
		if strings.TrimSpace(server.Command) == "npx" && slices.Equal(server.Args, []string{"-y", "@notionhq/notion-mcp-server"}) &&
			strings.TrimSpace(server.Env["NOTION_TOKEN"]) == "" && strings.TrimSpace(server.Env["OPENAPI_MCP_HEADERS"]) != "" {
			return "uses the old raw OPENAPI_MCP_HEADERS auth shape instead of NOTION_TOKEN"
		}
	}
	return ""
}

// MigrateLegacyServerDefinition rewrites known stale definitions to the current supported shape.
func MigrateLegacyServerDefinition(name string, server store.Server) store.Server {
	switch name {
	case "elevenlabs":
		if LegacyDefinitionReason(name, server) != "" {
			server.Command = "uvx"
			server.Args = []string{"elevenlabs-mcp"}
		}
	case "playwright":
		if LegacyDefinitionReason(name, server) != "" {
			server.Command = "npx"
			server.Args = []string{"-y", "@playwright/mcp@latest"}
		}
	case "filesystem":
		if LegacyDefinitionReason(name, server) != "" {
			server.Command = "npx"
			server.Args = []string{"-y", "@modelcontextprotocol/server-filesystem", "."}
		}
	case "notion":
		if token := extractNotionToken(server.Env["OPENAPI_MCP_HEADERS"]); token != "" && strings.TrimSpace(server.Env["NOTION_TOKEN"]) == "" {
			if server.Env == nil {
				server.Env = map[string]string{}
			}
			server.Env["NOTION_TOKEN"] = token
			delete(server.Env, "OPENAPI_MCP_HEADERS")
		}
	}
	return server
}

func extractNotionToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	headers := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		return ""
	}
	auth := strings.TrimSpace(headers["Authorization"])
	if len(auth) < 7 || !strings.EqualFold(auth[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(auth[7:])
}
