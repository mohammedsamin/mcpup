package registry

import (
	"slices"
	"testing"

	"github.com/mohammedsamin/mcpup/internal/store"
)

func TestAllSortedByName(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatalf("expected non-empty registry")
	}

	names := make([]string, 0, len(all))
	for _, tmpl := range all {
		names = append(names, tmpl.Name)
	}
	if !slices.IsSorted(names) {
		t.Fatalf("expected templates to be sorted by name")
	}
}

func TestLookupCaseInsensitive(t *testing.T) {
	tmpl, ok := Lookup("GITHUB")
	if !ok {
		t.Fatalf("expected github template to exist")
	}
	if tmpl.Name != "github" {
		t.Fatalf("expected github template, got %q", tmpl.Name)
	}
}

func TestSearchMatchesCategory(t *testing.T) {
	results := Search("database")
	if len(results) == 0 {
		t.Fatalf("expected category search to return results")
	}

	foundPostgres := false
	for _, tmpl := range results {
		if tmpl.Name == "postgres" {
			foundPostgres = true
			break
		}
	}
	if !foundPostgres {
		t.Fatalf("expected postgres template in database search results")
	}
}

func TestCategoriesSortedUnique(t *testing.T) {
	cats := Categories()
	if len(cats) == 0 {
		t.Fatalf("expected categories")
	}
	if !slices.IsSorted(cats) {
		t.Fatalf("expected categories sorted")
	}

	seen := map[string]struct{}{}
	for _, cat := range cats {
		if _, exists := seen[cat]; exists {
			t.Fatalf("duplicate category %q", cat)
		}
		seen[cat] = struct{}{}
	}
}

func TestPlaywrightTemplateUsesOfficialPackage(t *testing.T) {
	tmpl, ok := Lookup("playwright")
	if !ok {
		t.Fatalf("expected playwright template to exist")
	}
	if tmpl.Command != "npx" {
		t.Fatalf("expected playwright command npx, got %q", tmpl.Command)
	}
	expectedArgs := []string{"-y", "@playwright/mcp@latest"}
	if !slices.Equal(tmpl.Args, expectedArgs) {
		t.Fatalf("expected playwright args %v, got %v", expectedArgs, tmpl.Args)
	}
}

func TestFilesystemTemplateUsesRealDefaultPath(t *testing.T) {
	tmpl, ok := Lookup("filesystem")
	if !ok {
		t.Fatalf("expected filesystem template to exist")
	}
	expectedArgs := []string{"-y", "@modelcontextprotocol/server-filesystem", "."}
	if !slices.Equal(tmpl.Args, expectedArgs) {
		t.Fatalf("expected filesystem args %v, got %v", expectedArgs, tmpl.Args)
	}
}

func TestNotionTemplateUsesTokenPrompt(t *testing.T) {
	tmpl, ok := Lookup("notion")
	if !ok {
		t.Fatalf("expected notion template to exist")
	}
	if len(tmpl.PromptInputs) != 1 || tmpl.PromptInputs[0].ID != "NOTION_TOKEN" {
		t.Fatalf("expected notion prompt input NOTION_TOKEN, got %+v", tmpl.PromptInputs)
	}
	if len(tmpl.EnvVars) != 1 || tmpl.EnvVars[0].Key != "NOTION_TOKEN" {
		t.Fatalf("expected notion env var NOTION_TOKEN, got %+v", tmpl.EnvVars)
	}
	if tmpl.RenderEnv["NOTION_TOKEN"] != "{{NOTION_TOKEN}}" {
		t.Fatalf("expected notion render env to map NOTION_TOKEN directly, got %+v", tmpl.RenderEnv)
	}
}

func TestLegacyDefinitionReasonAndMigration(t *testing.T) {
	tests := []struct {
		name     string
		server   store.Server
		wantDesc string
		wantCmd  string
		wantArgs []string
		wantEnv  map[string]string
	}{
		{
			name:     "elevenlabs",
			server:   store.Server{Command: "npx", Args: []string{"-y", "@anthropic/mcp-elevenlabs"}},
			wantDesc: "uses the retired Anthropic ElevenLabs package name",
			wantCmd:  "uvx",
			wantArgs: []string{"elevenlabs-mcp"},
		},
		{
			name:     "playwright",
			server:   store.Server{Command: "npx", Args: []string{"-y", "@anthropic/mcp-playwright"}},
			wantDesc: "uses the retired Anthropic Playwright package name",
			wantCmd:  "npx",
			wantArgs: []string{"-y", "@playwright/mcp@latest"},
		},
		{
			name:     "filesystem",
			server:   store.Server{Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/dir"}},
			wantDesc: "uses the old placeholder filesystem path",
			wantCmd:  "npx",
			wantArgs: []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
		},
		{
			name: "notion",
			server: store.Server{
				Command: "npx",
				Args:    []string{"-y", "@notionhq/notion-mcp-server"},
				Env: map[string]string{
					"OPENAPI_MCP_HEADERS": `{"Authorization":"Bearer ntn_123","Notion-Version":"2022-06-28"}`,
				},
			},
			wantDesc: "uses the old raw OPENAPI_MCP_HEADERS auth shape instead of NOTION_TOKEN",
			wantCmd:  "npx",
			wantArgs: []string{"-y", "@notionhq/notion-mcp-server"},
			wantEnv:  map[string]string{"NOTION_TOKEN": "ntn_123"},
		},
	}

	for _, tt := range tests {
		if got := LegacyDefinitionReason(tt.name, tt.server); got != tt.wantDesc {
			t.Fatalf("%s: got legacy reason %q, want %q", tt.name, got, tt.wantDesc)
		}
		got := MigrateLegacyServerDefinition(tt.name, tt.server)
		if got.Command != tt.wantCmd {
			t.Fatalf("%s: got command %q, want %q", tt.name, got.Command, tt.wantCmd)
		}
		if !slices.Equal(got.Args, tt.wantArgs) {
			t.Fatalf("%s: got args %v, want %v", tt.name, got.Args, tt.wantArgs)
		}
		if tt.wantEnv != nil {
			if len(got.Env) != len(tt.wantEnv) {
				t.Fatalf("%s: got env %v, want %v", tt.name, got.Env, tt.wantEnv)
			}
			for key, value := range tt.wantEnv {
				if got.Env[key] != value {
					t.Fatalf("%s: got env[%s]=%q, want %q", tt.name, key, got.Env[key], value)
				}
			}
		}
	}
}
