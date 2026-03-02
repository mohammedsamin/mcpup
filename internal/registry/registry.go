package registry

import (
	"sort"
	"strings"
)

// EnvVar describes an environment variable a server needs.
type EnvVar struct {
	Key      string `json:"key"`
	Required bool   `json:"required"`
	Hint     string `json:"hint,omitempty"`
}

// Template is a known MCP server definition in the built-in catalog.
type Template struct {
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Tools       []string `json:"tools,omitempty"`
	EnvVars     []EnvVar `json:"envVars,omitempty"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
}

// catalog is the built-in set of known MCP servers.
var catalog = []Template{
	// ── Developer Tools ──────────────────────────────────────────────────
	{
		Name:        "github",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-github"},
		Tools:       []string{"search_repositories", "search_code", "search_issues", "search_pull_requests", "create_issue", "create_pull_request"},
		EnvVars:     []EnvVar{{Key: "GITHUB_TOKEN", Required: true, Hint: "https://github.com/settings/tokens"}},
		Description: "GitHub - search repos, issues, PRs, create branches",
		Category:    "developer",
	},
	{
		Name:        "gitlab",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-gitlab"},
		EnvVars:     []EnvVar{{Key: "GITLAB_TOKEN", Required: true, Hint: "https://gitlab.com/-/user_settings/personal_access_tokens"}},
		Description: "GitLab - manage repos, issues, merge requests",
		Category:    "developer",
	},
	{
		Name:        "linear",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-linear"},
		EnvVars:     []EnvVar{{Key: "LINEAR_API_KEY", Required: true, Hint: "https://linear.app/settings/api"}},
		Description: "Linear - manage issues and projects",
		Category:    "developer",
	},
	{
		Name:        "sentry",
		Command:     "npx",
		Args:        []string{"-y", "@sentry/mcp-server"},
		EnvVars:     []EnvVar{{Key: "SENTRY_AUTH_TOKEN", Required: true, Hint: "https://sentry.io/settings/auth-tokens/"}},
		Description: "Sentry - search errors, get issue details",
		Category:    "developer",
	},

	// ── Data & Search ────────────────────────────────────────────────────
	{
		Name:        "brave-search",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-brave-search"},
		EnvVars:     []EnvVar{{Key: "BRAVE_API_KEY", Required: true, Hint: "https://brave.com/search/api/"}},
		Description: "Brave Search - web and local search",
		Category:    "search",
	},
	{
		Name:        "fetch",
		Command:     "uvx",
		Args:        []string{"mcp-server-fetch"},
		Description: "Fetch - retrieve and convert web pages to markdown",
		Category:    "search",
	},
	{
		Name:        "context7",
		Command:     "npx",
		Args:        []string{"-y", "@upstash/context7-mcp"},
		Description: "Context7 - up-to-date docs for any library",
		Category:    "search",
	},

	// ── Productivity ─────────────────────────────────────────────────────
	{
		Name:        "notion",
		Command:     "npx",
		Args:        []string{"-y", "@notionhq/notion-mcp-server"},
		EnvVars:     []EnvVar{{Key: "OPENAPI_MCP_HEADERS", Required: true, Hint: "JSON: {\"Authorization\":\"Bearer ntn_xxx\",\"Notion-Version\":\"2022-06-28\"}"}},
		Description: "Notion - search and manage pages, databases",
		Category:    "productivity",
	},
	{
		Name:        "slack",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-slack"},
		Tools:       []string{"list_channels", "post_message", "read_thread"},
		EnvVars:     []EnvVar{{Key: "SLACK_BOT_TOKEN", Required: true, Hint: "xoxb-... from https://api.slack.com/apps"}},
		Description: "Slack - read/send messages, manage channels",
		Category:    "productivity",
	},
	{
		Name:        "google-maps",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-google-maps"},
		EnvVars:     []EnvVar{{Key: "GOOGLE_MAPS_API_KEY", Required: true, Hint: "https://console.cloud.google.com/apis/credentials"}},
		Description: "Google Maps - geocode, directions, places",
		Category:    "productivity",
	},

	// ── Utilities ────────────────────────────────────────────────────────
	{
		Name:        "filesystem",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/dir"},
		Tools:       []string{"list_directory", "read_file", "write_file", "search_files"},
		Description: "Filesystem - read, write, search files in allowed directories",
		Category:    "utility",
	},
	{
		Name:        "memory",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-memory"},
		Description: "Memory - persistent knowledge graph for context",
		Category:    "utility",
	},
	{
		Name:        "sequential-thinking",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-sequential-thinking"},
		Description: "Sequential Thinking - step-by-step reasoning helper",
		Category:    "utility",
	},

	// ── Databases ────────────────────────────────────────────────────────
	{
		Name:        "postgres",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-postgres"},
		EnvVars:     []EnvVar{{Key: "POSTGRES_CONNECTION_STRING", Required: true, Hint: "postgresql://user:pass@host:5432/dbname"}},
		Description: "PostgreSQL - query and explore databases",
		Category:    "database",
	},
	{
		Name:        "supabase",
		Command:     "npx",
		Args:        []string{"-y", "@supabase/mcp-server"},
		EnvVars:     []EnvVar{{Key: "SUPABASE_ACCESS_TOKEN", Required: true, Hint: "https://supabase.com/dashboard/account/tokens"}},
		Description: "Supabase - manage projects, tables, edge functions",
		Category:    "database",
	},

	// ── Browser & Automation ─────────────────────────────────────────────
	{
		Name:        "playwright",
		Command:     "npx",
		Args:        []string{"-y", "@anthropic/mcp-playwright"},
		Description: "Playwright - browser automation, screenshots, testing",
		Category:    "automation",
	},
	{
		Name:        "puppeteer",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-puppeteer"},
		Description: "Puppeteer - headless Chrome automation",
		Category:    "automation",
	},

	// ── AI & Media ───────────────────────────────────────────────────────
	{
		Name:        "everart",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-everart"},
		EnvVars:     []EnvVar{{Key: "EVERART_API_KEY", Required: true, Hint: "https://everart.ai"}},
		Description: "EverArt - AI image generation",
		Category:    "media",
	},
	{
		Name:        "elevenlabs",
		Command:     "uvx",
		Args:        []string{"elevenlabs-mcp"},
		EnvVars:     []EnvVar{{Key: "ELEVENLABS_API_KEY", Required: true, Hint: "https://elevenlabs.io/app/settings/api-keys"}},
		Description: "ElevenLabs - text-to-speech, voice generation",
		Category:    "media",
	},
}

// All returns every template in the catalog, sorted by name.
func All() []Template {
	out := make([]Template, len(catalog))
	copy(out, catalog)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// Lookup finds a template by exact name (case-insensitive).
func Lookup(name string) (Template, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, t := range catalog {
		if strings.ToLower(t.Name) == lower {
			return t, true
		}
	}
	return Template{}, false
}

// Search returns templates matching a query string (name or description).
func Search(query string) []Template {
	lower := strings.ToLower(strings.TrimSpace(query))
	if lower == "" {
		return All()
	}
	var matches []Template
	for _, t := range catalog {
		if strings.Contains(strings.ToLower(t.Name), lower) ||
			strings.Contains(strings.ToLower(t.Description), lower) ||
			strings.Contains(strings.ToLower(t.Category), lower) {
			matches = append(matches, t)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})
	return matches
}

// Categories returns sorted unique category names.
func Categories() []string {
	seen := map[string]struct{}{}
	for _, t := range catalog {
		seen[t.Category] = struct{}{}
	}
	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return cats
}

// ByCategory returns templates in a given category.
func ByCategory(category string) []Template {
	lower := strings.ToLower(strings.TrimSpace(category))
	var matches []Template
	for _, t := range catalog {
		if strings.ToLower(t.Category) == lower {
			matches = append(matches, t)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})
	return matches
}

// Names returns all template names sorted.
func Names() []string {
	names := make([]string, len(catalog))
	for i, t := range catalog {
		names[i] = t.Name
	}
	sort.Strings(names)
	return names
}
