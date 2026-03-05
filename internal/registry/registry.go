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

// PromptInput describes a user-facing setup input for a template.
type PromptInput struct {
	ID       string `json:"id"`
	Label    string `json:"label,omitempty"`
	Required bool   `json:"required,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// LegacyMatcher identifies a known stale server definition shape.
type LegacyMatcher struct {
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	URL       string   `json:"url,omitempty"`
	Transport string   `json:"transport,omitempty"`
	EnvKeys   []string `json:"envKeys,omitempty"`
}

// Template is a known MCP server definition in the built-in catalog.
type Template struct {
	Name           string            `json:"name"`
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	URL            string            `json:"url,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Transport      string            `json:"transport,omitempty"`
	Tools          []string          `json:"tools,omitempty"`
	EnvVars        []EnvVar          `json:"envVars,omitempty"`
	PromptInputs   []PromptInput     `json:"promptInputs,omitempty"`
	RenderEnv      map[string]string `json:"renderEnv,omitempty"`
	RenderHeaders  map[string]string `json:"renderHeaders,omitempty"`
	LegacyMatchers []LegacyMatcher   `json:"legacyMatchers,omitempty"`
	Description    string            `json:"description"`
	Category       string            `json:"category"`
}

// catalog is the built-in set of known MCP servers.
var catalog = []Template{
	// ── Developer ─────────────────────────────────────────────────────
	{
		Name:        "atlassian",
		URL:         "https://mcp.atlassian.com/v1/mcp",
		Transport:   "streamable-http",
		Description: "Atlassian Jira and Confluence workflows via OAuth",
		Category:    "developer",
	},
	{
		Name:        "bitbucket",
		Command:     "uvx",
		Args:        []string{"bitbucket-mcp-py"},
		Description: "Bitbucket repositories, pull requests, comments, and pipelines",
		Category:    "developer",
	},
	{
		Name:        "github",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-github"},
		Tools:       []string{"search_repositories", "search_code", "search_issues", "search_pull_requests", "create_issue", "create_pull_request"},
		EnvVars:     []EnvVar{{Key: "GITHUB_TOKEN", Required: true, Hint: "https://github.com/settings/tokens"}},
		Description: "GitHub repos, issues, pull requests, and code search",
		Category:    "developer",
	},
	{
		Name:        "gitkraken",
		Command:     "npx",
		Args:        []string{"-y", "@gitkraken/gk"},
		Description: "Manage repos, PRs, and issues across Git providers",
		Category:    "developer",
	},
	{
		Name:        "gitlab",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-gitlab"},
		EnvVars:     []EnvVar{{Key: "GITLAB_TOKEN", Required: true, Hint: "https://gitlab.com/-/user_settings/personal_access_tokens"}},
		Description: "GitLab repos, issues, and merge requests",
		Category:    "developer",
	},
	{
		Name:        "linear",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-linear"},
		EnvVars:     []EnvVar{{Key: "LINEAR_API_KEY", Required: true, Hint: "https://linear.app/settings/api"}},
		Description: "Linear issues, projects, and workflows",
		Category:    "developer",
	},
	{
		Name:        "postman",
		Command:     "npx",
		Args:        []string{"-y", "@postman/postman-mcp-server"},
		EnvVars:     []EnvVar{{Key: "POSTMAN_API_KEY", Required: true, Hint: "https://github.com/postmanlabs/postman-mcp-server"}},
		Description: "A basic MCP server to operate on the Postman API",
		Category:    "developer",
	},
	{
		Name:        "sentry",
		Command:     "npx",
		Args:        []string{"-y", "@sentry/mcp-server"},
		EnvVars:     []EnvVar{{Key: "SENTRY_AUTH_TOKEN", Required: true, Hint: "https://sentry.io/settings/auth-tokens/"}},
		Description: "Sentry errors, traces, and issue triage",
		Category:    "developer",
	},
	{
		Name:        "smartbear",
		Command:     "npx",
		Args:        []string{"-y", "@smartbear/mcp"},
		Description: "MCP server for AI access to SmartBear tools, including BugSnag, Reflect, Swagger, PactFlow",
		Category:    "developer",
	},
	{
		Name:        "teamcity",
		Command:     "npx",
		Args:        []string{"-y", "@daghis/teamcity-mcp"},
		EnvVars:     []EnvVar{{Key: "TEAMCITY_URL", Required: true, Hint: "https://github.com/Daghis/teamcity-mcp"}, {Key: "TEAMCITY_TOKEN", Required: true, Hint: "https://github.com/Daghis/teamcity-mcp"}},
		Description: "MCP server exposing JetBrains TeamCity CI/CD workflows to AI coding assistants",
		Category:    "developer",
	},

	// ── Search ─────────────────────────────────────────────────────
	{
		Name:        "apify",
		URL:         "https://mcp.apify.com/",
		Transport:   "streamable-http",
		Description: "Apify actors for scraping, crawling, and structured data extraction",
		Category:    "search",
	},
	{
		Name:        "brave-search",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-brave-search"},
		EnvVars:     []EnvVar{{Key: "BRAVE_API_KEY", Required: true, Hint: "https://brave.com/search/api/"}},
		Description: "Web and local search via Brave Search API",
		Category:    "search",
	},
	{
		Name:        "context7",
		Command:     "npx",
		Args:        []string{"-y", "@upstash/context7-mcp"},
		Description: "Latest docs and code examples for libraries",
		Category:    "search",
	},
	{
		Name:        "exa",
		URL:         "https://mcp.exa.ai/mcp",
		Transport:   "streamable-http",
		Description: "Exa web search, crawling, and code-context retrieval",
		Category:    "search",
	},
	{
		Name:        "fetch",
		Command:     "uvx",
		Args:        []string{"mcp-server-fetch"},
		Description: "Fetch and convert web pages to markdown",
		Category:    "search",
	},
	{
		Name:        "firecrawl",
		Command:     "npx",
		Args:        []string{"-y", "firecrawl-mcp"},
		Description: "MCP server for Firecrawl web scraping, structured data extraction and web search integration",
		Category:    "search",
	},
	{
		Name:        "perplexity",
		Command:     "npx",
		Args:        []string{"-y", "@perplexity-ai/mcp-server"},
		Description: "Real-time web search, reasoning, and research through Perplexity's API",
		Category:    "search",
	},
	{
		Name:        "pulse-fetch",
		Command:     "npx",
		Args:        []string{"-y", "@pulsemcp/pulse-fetch"},
		Description: "MCP server that extracts clean, structured content from web pages with anti-bot bypass",
		Category:    "search",
	},
	{
		Name:        "tavily",
		Command:     "npx",
		Args:        []string{"-y", "@toolsdk.ai/tavily-mcp"},
		EnvVars:     []EnvVar{{Key: "TAVILY_API_KEY", Required: true, Hint: "https://github.com/Seey215/tavily-mcp"}},
		Description: "MCP server for advanced web search using Tavily API",
		Category:    "search",
	},

	// ── Productivity ─────────────────────────────────────────────────────
	{
		Name:        "airtable",
		Command:     "npx",
		Args:        []string{"-y", "airtable-mcp-server"},
		EnvVars:     []EnvVar{{Key: "AIRTABLE_API_KEY", Required: true, Hint: "https://github.com/domdomegg/airtable-mcp-server.git"}},
		Description: "Read and write access to Airtable database schemas, tables, and records",
		Category:    "productivity",
	},
	{
		Name:        "close",
		URL:         "https://mcp.close.com/mcp",
		Transport:   "streamable-http",
		Description: "Close CRM pipelines, contacts, and sales workflows",
		Category:    "productivity",
	},
	{
		Name:        "google-calendar",
		Command:     "npx",
		Args:        []string{"-y", "google-cal-mcp"},
		EnvVars:     []EnvVar{{Key: "GOOGLE_ACCESS_TOKEN", Required: true, Hint: "https://github.com/domdomegg/google-calendar-mcp.git"}},
		Description: "Allow AI systems to list, create, update, and manage Google Calendar events",
		Category:    "productivity",
	},
	{
		Name:        "google-drive",
		Command:     "npx",
		Args:        []string{"-y", "google-drive-mcp"},
		EnvVars:     []EnvVar{{Key: "GOOGLE_ACCESS_TOKEN", Required: true, Hint: "https://github.com/domdomegg/google-drive-mcp.git"}},
		Description: "Allow AI systems to list, search, upload, download, and manage files and folders in Google",
		Category:    "productivity",
	},
	{
		Name:        "google-maps",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-google-maps"},
		EnvVars:     []EnvVar{{Key: "GOOGLE_MAPS_API_KEY", Required: true, Hint: "https://console.cloud.google.com/apis/credentials"}},
		Description: "Geocoding, places, and route directions",
		Category:    "productivity",
	},
	{
		Name:        "hubspot",
		Command:     "npx",
		Args:        []string{"-y", "@hubspot/mcp-server"},
		EnvVars:     []EnvVar{{Key: "PRIVATE_APP_ACCESS_TOKEN", Required: true, Hint: "https://www.npmjs.com/package/@hubspot/mcp-server"}},
		Description: "Official HubSpot MCP server for CRM objects, associations, workflows, and engagements",
		Category:    "productivity",
	},
	{
		Name:        "mailchimp",
		Command:     "uvx",
		Args:        []string{"mailchimp-mcp-server"},
		EnvVars:     []EnvVar{{Key: "MAILCHIMP_API_KEY", Required: true, Hint: "https://github.com/asklokesh/mailchimp-mcp-server"}},
		Description: "MCP server for Mailchimp API integration",
		Category:    "productivity",
	},
	{
		Name:        "miro",
		URL:         "https://mcp.miro.com/",
		Transport:   "streamable-http",
		Description: "Miro boards, diagrams, and design collaboration",
		Category:    "productivity",
	},
	{
		Name:        "monday",
		URL:         "https://mcp.monday.com/mcp",
		Transport:   "streamable-http",
		Description: "MCP server for monday.com integration",
		Category:    "productivity",
	},
	{
		Name:         "notion",
		Command:      "npx",
		Args:         []string{"-y", "@notionhq/notion-mcp-server"},
		EnvVars:      []EnvVar{{Key: "NOTION_TOKEN", Required: true, Hint: "https://www.notion.so/profile/integrations"}},
		PromptInputs: []PromptInput{{ID: "NOTION_TOKEN", Label: "Notion token", Required: true, Hint: "https://www.notion.so/profile/integrations"}},
		RenderEnv:    map[string]string{"NOTION_TOKEN": "{{NOTION_TOKEN}}"},
		LegacyMatchers: []LegacyMatcher{{
			Command: "npx",
			Args:    []string{"-y", "@notionhq/notion-mcp-server"},
			EnvKeys: []string{"OPENAPI_MCP_HEADERS"},
		}},
		Description: "Notion pages and database operations",
		Category:    "productivity",
	},
	{
		Name:        "salesforce",
		Command:     "npx",
		Args:        []string{"-y", "@salesforce/mcp"},
		Description: "Official Salesforce DX MCP server for org operations, metadata, data, users, and tests",
		Category:    "productivity",
	},
	{
		Name:        "salesforce-cirra",
		URL:         "https://mcp.cirra.ai/sfdc/mcp",
		Transport:   "streamable-http",
		Description: "Salesforce administration and data operations",
		Category:    "productivity",
	},
	{
		Name:        "slack",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-slack"},
		Tools:       []string{"list_channels", "post_message", "read_thread"},
		EnvVars:     []EnvVar{{Key: "SLACK_BOT_TOKEN", Required: true, Hint: "https://api.slack.com/apps"}},
		Description: "Slack channels, messages, and threads",
		Category:    "productivity",
	},
	{
		Name:        "todoist",
		URL:         "https://ai.todoist.net/mcp",
		Transport:   "streamable-http",
		Description: "Official Todoist MCP server for AI assistants to manage tasks, projects, and workflows",
		Category:    "productivity",
	},
	{
		Name:        "trello",
		Command:     "npx",
		Args:        []string{"-y", "trello-mcp"},
		EnvVars:     []EnvVar{{Key: "TRELLO_API_KEY", Required: true, Hint: "https://github.com/stucchi/trello-mcp"}, {Key: "TRELLO_TOKEN", Required: true, Hint: "https://github.com/stucchi/trello-mcp"}},
		Description: "Trello boards, lists, cards, labels, and checklists",
		Category:    "productivity",
	},

	// ── Utility ─────────────────────────────────────────────────────
	{
		Name:        "automem",
		Command:     "npx",
		Args:        []string{"-y", "@verygoodplugins/mcp-automem"},
		Description: "Graph-vector memory for AI assistants using FalkorDB and Qdrant",
		Category:    "utility",
	},
	{
		Name:        "docfork",
		Command:     "npx",
		Args:        []string{"-y", "docfork"},
		Description: "Up-to-date Docs for AI Agents",
		Category:    "utility",
	},
	{
		Name:        "egnyte",
		URL:         "https://mcp-server.egnyte.com/mcp",
		Transport:   "streamable-http",
		Description: "Egnyte's remote MCP server for secure AI access, search, upload and file management in your",
		Category:    "utility",
	},
	{
		Name:    "filesystem",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
		LegacyMatchers: []LegacyMatcher{{
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/dir"},
		}},
		Tools:       []string{"list_directory", "read_file", "write_file", "search_files"},
		Description: "Scoped local filesystem read and write tools",
		Category:    "utility",
	},
	{
		Name:        "memory",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-memory"},
		Description: "Persistent local memory graph for context",
		Category:    "utility",
	},
	{
		Name:        "memphora",
		Command:     "npx",
		Args:        []string{"-y", "memphora-mcp"},
		EnvVars:     []EnvVar{{Key: "MEMPHORA_API_KEY", Required: true, Hint: "https://github.com/Memphora/memphora-mcp"}},
		Description: "Add persistent memory to AI assistants. Store and recall info across conversations",
		Category:    "utility",
	},
	{
		Name:        "nextcloud",
		Command:     "npx",
		Args:        []string{"-y", "aiquila-mcp"},
		EnvVars:     []EnvVar{{Key: "NEXTCLOUD_URL", Required: true, Hint: "https://github.com/elgorro/aiquila.git"}, {Key: "NEXTCLOUD_USER", Required: true, Hint: "https://github.com/elgorro/aiquila.git"}, {Key: "NEXTCLOUD_PASSWORD", Required: true, Hint: "https://github.com/elgorro/aiquila.git"}},
		Description: "MCP server for Nextcloud: file management, search, calendar, contacts, and system status",
		Category:    "utility",
	},
	{
		Name:        "omega-memory",
		Command:     "npx",
		Args:        []string{"-y", "omega-memory"},
		Description: "Persistent memory for AI coding agents. Semantic search, contradiction detection, memory",
		Category:    "utility",
	},
	{
		Name:        "pinecone",
		Command:     "npx",
		Args:        []string{"-y", "@pinecone-database/mcp"},
		EnvVars:     []EnvVar{{Key: "PINECONE_API_KEY", Required: true, Hint: "https://github.com/pinecone-io/pinecone-mcp"}},
		Description: "Official Pinecone MCP server for index operations and Pinecone developer workflows",
		Category:    "utility",
	},
	{
		Name:        "qdrant",
		Command:     "npx",
		Args:        []string{"-y", "@mhalder/qdrant-mcp-server"},
		EnvVars:     []EnvVar{{Key: "QDRANT_URL", Required: true, Hint: "https://github.com/mhalder/qdrant-mcp-server"}, {Key: "QDRANT_API_KEY", Required: true, Hint: "https://github.com/mhalder/qdrant-mcp-server"}},
		Description: "Qdrant MCP server for semantic search with local/remote vector database backends",
		Category:    "utility",
	},
	{
		Name:        "sequential-thinking",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-sequential-thinking"},
		Description: "Step-by-step reasoning helper tools",
		Category:    "utility",
	},
	{
		Name:        "tigris",
		URL:         "https://mcp.storage.dev/mcp",
		Transport:   "streamable-http",
		Description: "Tigris MCP Server seamlessly connects AI agents to Tigris bucket and object management",
		Category:    "utility",
	},

	// ── Database ─────────────────────────────────────────────────────
	{
		Name:        "elasticsearch",
		Command:     "npx",
		Args:        []string{"-y", "@tocharianou/elasticsearch-mcp"},
		EnvVars:     []EnvVar{{Key: "ES_URL", Required: true, Hint: "https://github.com/TocharianOU/elasticsearch-mcp"}},
		Description: "Elasticsearch MCP Server with multi-version support (ES 5.x-9.x) and comprehensive API",
		Category:    "database",
	},
	{
		Name:        "mongodb",
		Command:     "npx",
		Args:        []string{"-y", "mongodb-mcp-server"},
		Description: "MongoDB Model Context Protocol Server",
		Category:    "database",
	},
	{
		Name:        "motherduck",
		Command:     "npx",
		Args:        []string{"-y", "mcp-server-motherduck"},
		Description: "SQL analytics and data engineering for AI Assistants and IDEs",
		Category:    "database",
	},
	{
		Name:        "mysql",
		Command:     "npx",
		Args:        []string{"-y", "@hovecapital/read-only-mysql-mcp-server"},
		Description: "MCP server for read-only MySQL database queries in Claude Desktop",
		Category:    "database",
	},
	{
		Name:        "postgres",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-postgres"},
		EnvVars:     []EnvVar{{Key: "POSTGRES_CONNECTION_STRING", Required: true, Hint: "postgresql://user:pass@host:5432/dbname"}},
		Description: "PostgreSQL querying and schema exploration",
		Category:    "database",
	},
	{
		Name:        "redis",
		Command:     "npx",
		Args:        []string{"-y", "redis-mcp-server"},
		Description: "Natural language interface designed for agentic applications to manage and search data in",
		Category:    "database",
	},
	{
		Name:        "snowflake",
		Command:     "npx",
		Args:        []string{"-y", "snowflake-labs-mcp"},
		Description: "MCP Server for Snowflake from Snowflake Labs",
		Category:    "database",
	},
	{
		Name:        "supabase",
		Command:     "npx",
		Args:        []string{"-y", "@supabase/mcp-server"},
		EnvVars:     []EnvVar{{Key: "SUPABASE_ACCESS_TOKEN", Required: true, Hint: "https://supabase.com/dashboard/account/tokens"}},
		Description: "Supabase projects, SQL, and edge functions",
		Category:    "database",
	},
	{
		Name:        "turso",
		Command:     "npx",
		Args:        []string{"-y", "mcp-turso-cloud"},
		EnvVars:     []EnvVar{{Key: "TURSO_API_TOKEN", Required: true, Hint: "https://github.com/spences10/mcp-turso-cloud"}, {Key: "TURSO_ORGANIZATION", Required: true, Hint: "https://github.com/spences10/mcp-turso-cloud"}},
		Description: "MCP server for integrating Turso with LLMs",
		Category:    "database",
	},

	// ── Automation ─────────────────────────────────────────────────────
	{
		Name:        "browserbase",
		Command:     "npx",
		Args:        []string{"-y", "@browserbasehq/mcp-server-browserbase"},
		EnvVars:     []EnvVar{{Key: "BROWSERBASE_API_KEY", Required: true, Hint: "https://github.com/browserbase/mcp-server-browserbase"}, {Key: "BROWSERBASE_PROJECT_ID", Required: true, Hint: "https://github.com/browserbase/mcp-server-browserbase"}, {Key: "GEMINI_API_KEY", Required: true, Hint: "https://github.com/browserbase/mcp-server-browserbase"}},
		Description: "MCP server for AI web browser automation using Browserbase and Stagehand",
		Category:    "automation",
	},
	{
		Name:        "chrome-devtools",
		Command:     "npx",
		Args:        []string{"-y", "chrome-devtools-mcp"},
		Description: "MCP server for Chrome DevTools",
		Category:    "automation",
	},
	{
		Name:    "playwright",
		Command: "npx",
		Args:    []string{"-y", "@playwright/mcp@latest"},
		LegacyMatchers: []LegacyMatcher{{
			Command: "npx",
			Args:    []string{"-y", "@anthropic/mcp-playwright"},
		}},
		Description: "Browser automation, screenshots, and testing",
		Category:    "automation",
	},
	{
		Name:        "puppeteer",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-puppeteer"},
		Description: "Headless Chrome browser automation",
		Category:    "automation",
	},

	// ── Media ─────────────────────────────────────────────────────
	{
		Name:        "canva",
		Command:     "npx",
		Args:        []string{"-y", "canva-mcp"},
		Description: "MCP server for Canva - create designs, manage assets, use templates, export graphics",
		Category:    "media",
	},
	{
		Name:    "elevenlabs",
		Command: "uvx",
		Args:    []string{"elevenlabs-mcp"},
		EnvVars: []EnvVar{{Key: "ELEVENLABS_API_KEY", Required: true, Hint: "https://elevenlabs.io/app/settings/api-keys"}},
		LegacyMatchers: []LegacyMatcher{{
			Command: "npx",
			Args:    []string{"-y", "@anthropic/mcp-elevenlabs"},
		}},
		Description: "Voice generation and text-to-speech",
		Category:    "media",
	},
	{
		Name:        "everart",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-everart"},
		EnvVars:     []EnvVar{{Key: "EVERART_API_KEY", Required: true, Hint: "https://everart.ai"}},
		Description: "AI image generation and style tools",
		Category:    "media",
	},
	{
		Name:        "figma",
		URL:         "https://mcp.figma.com/mcp",
		Transport:   "streamable-http",
		Description: "Figma design context and assets in AI workflows",
		Category:    "media",
	},

	// ── Cloud ─────────────────────────────────────────────────────
	{
		Name:        "aws",
		Command:     "uvx",
		Args:        []string{"mcp-proxy-for-aws"},
		Description: "A managed MCP server enabling AI agents to access AWS using docs, API calls, and SOP",
		Category:    "cloud",
	},
	{
		Name:        "aws-ecs",
		Command:     "uvx",
		Args:        []string{"mcp-proxy-for-aws"},
		Description: "AI-powered Amazon ECS workload management",
		Category:    "cloud",
	},
	{
		Name:        "aws-eks",
		Command:     "uvx",
		Args:        []string{"mcp-proxy-for-aws"},
		Description: "AI-powered Amazon EKS cluster management and troubleshooting",
		Category:    "cloud",
	},
	{
		Name:        "azure",
		Command:     "Azure.Mcp",
		Description: "All Azure MCP tools to create a seamless connection between AI agents and Azure services",
		Category:    "cloud",
	},
	{
		Name:        "cloudflare",
		Command:     "npx",
		Args:        []string{"-y", "@cloudflare/mcp-server-cloudflare"},
		EnvVars:     []EnvVar{{Key: "CLOUDFLARE_API_TOKEN", Required: true, Hint: "https://dash.cloudflare.com/profile/api-tokens"}},
		Description: "Cloudflare Workers, DNS, KV, R2, and D1",
		Category:    "cloud",
	},
	{
		Name:        "coolify",
		Command:     "npx",
		Args:        []string{"-y", "@masonator/coolify-mcp"},
		EnvVars:     []EnvVar{{Key: "COOLIFY_ACCESS_TOKEN", Required: true, Hint: "https://github.com/StuMason/coolify-mcp"}},
		Description: "38 optimized tools for managing Coolify infrastructure, diagnostics, and docs search",
		Category:    "cloud",
	},
	{
		Name:        "gcp-gemini-cloud-assist",
		Command:     "npx",
		Args:        []string{"-y", "@google-cloud/gemini-cloud-assist-mcp"},
		EnvVars:     []EnvVar{{Key: "GOOGLE_APPLICATION_CREDENTIALS", Required: true, Hint: "https://github.com/GoogleCloudPlatform/gemini-cloud-assist-mcp"}},
		Description: "Google Cloud MCP server for understanding, managing, and troubleshooting GCP environments",
		Category:    "cloud",
	},
	{
		Name:        "kubernetes",
		Command:     "npx",
		Args:        []string{"-y", "kubernetes-mcp-server"},
		Description: "Model Context Protocol (MCP) server for Kubernetes and OpenShift cluster management",
		Category:    "cloud",
	},
	{
		Name:        "netlify",
		Command:     "npx",
		Args:        []string{"-y", "@netlify/mcp"},
		EnvVars:     []EnvVar{{Key: "NETLIFY_PERSONAL_ACCESS_TOKEN", Required: true, Hint: "https://docs.netlify.com/welcome/build-with-ai/netlify-mcp-server/"}},
		Description: "Netlify sites, builds, deploys, and project management",
		Category:    "cloud",
	},
	{
		Name:        "terraform",
		Command:     "docker",
		Args:        []string{"run", "-i", "--rm", "docker.io/hashicorp/terraform-mcp-server:0.4.0"},
		Description: "Terraform and HCP Terraform workflow automation",
		Category:    "cloud",
	},
	{
		Name:        "vercel",
		URL:         "https://mcp.vercel.com",
		Transport:   "streamable-http",
		Description: "Vercel projects, deployments, and logs",
		Category:    "cloud",
	},

	// ── AI ─────────────────────────────────────────────────────
	{
		Name:        "huggingface",
		URL:         "https://huggingface.co/mcp?login",
		Transport:   "streamable-http",
		Description: "Connect to Hugging Face Hub and thousands of Gradio AI Applications",
		Category:    "ai",
	},
	{
		Name:        "litellm",
		Command:     "uvx",
		Args:        []string{"litellm-mcp"},
		Description: "Access 100+ LLMs with one API: GPT-4, Claude, Gemini, Mistral, and more",
		Category:    "ai",
	},
	{
		Name:        "openai-tools",
		URL:         "https://openai-tools.run.mcp.com.ai/mcp",
		Transport:   "streamable-http",
		Description: "OpenAI image and audio generation tools",
		Category:    "ai",
	},

	// ── Communication ─────────────────────────────────────────────────────
	{
		Name:        "discord",
		Command:     "npx",
		Args:        []string{"-y", "@ncodelife/discord-mcp-server"},
		EnvVars:     []EnvVar{{Key: "DISCORD_TOKEN", Required: false, Hint: "https://github.com/ngoctranfire/discord-mcp-server"}, {Key: "DISCORD_GUILD_ID", Required: false, Hint: "https://github.com/ngoctranfire/discord-mcp-server"}},
		Description: "Discord MCP server for messaging, channels, roles, and webhook operations",
		Category:    "communication",
	},
	{
		Name:        "gmail",
		URL:         "https://gmail.mintmcp.com/mcp",
		Transport:   "streamable-http",
		Description: "A MCP server for Gmail that lets you search, read, and draft emails and replies",
		Category:    "communication",
	},
	{
		Name:        "google-calendar-remote",
		URL:         "https://gcal.mintmcp.com/mcp",
		Transport:   "streamable-http",
		Description: "A MCP server that works with Google Calendar to manage event listing, reading, and updates",
		Category:    "communication",
	},
	{
		Name:        "outlook-calendar",
		URL:         "https://outlook-calendar.mintmcp.com/mcp",
		Transport:   "streamable-http",
		Description: "A MCP server that works with Outlook Calendar to manage event listing, reading, and updates",
		Category:    "communication",
	},
	{
		Name:        "outlook-email",
		URL:         "https://outlook-email.mintmcp.com/mcp",
		Transport:   "streamable-http",
		Description: "A MCP server for Outlook email that lets you search, read, and draft emails and replies",
		Category:    "communication",
	},
	{
		Name:        "teams",
		URL:         "https://waystation.ai/teams/mcp",
		Transport:   "streamable-http",
		Description: "Remote Teams MCP integration for chat, collaboration, and meeting workflows",
		Category:    "communication",
	},

	// ── Finance ─────────────────────────────────────────────────────
	{
		Name:        "paypal",
		URL:         "https://mcp.paypal.com/mcp",
		Transport:   "streamable-http",
		Description: "PayPal MCP server provides access to PayPal services and operations for AI assistants",
		Category:    "finance",
	},
	{
		Name:        "quickbooks",
		Command:     "uvx",
		Args:        []string{"quickbooks-mcp-server"},
		EnvVars:     []EnvVar{{Key: "QUICKBOOKS_API_KEY", Required: true, Hint: "https://github.com/asklokesh/quickbooks-mcp-server"}},
		Description: "MCP server for QuickBooks API integration",
		Category:    "finance",
	},
	{
		Name:        "shopify",
		URL:         "https://mcp.gossiper.io/mcp",
		Transport:   "streamable-http",
		Description: "Shopify admin tasks for products, orders, and store ops",
		Category:    "finance",
	},
	{
		Name:        "stripe",
		Command:     "npx",
		Args:        []string{"-y", "@stripe/mcp"},
		EnvVars:     []EnvVar{{Key: "STRIPE_API_KEY", Required: true, Hint: "https://dashboard.stripe.com/apikeys"}},
		Description: "Stripe customers, products, and payment operations",
		Category:    "finance",
	},

	// ── Security ─────────────────────────────────────────────────────
	{
		Name:        "1password",
		Command:     "npx",
		Args:        []string{"-y", "@takescake/1password-mcp"},
		EnvVars:     []EnvVar{{Key: "OP_SERVICE_ACCOUNT_TOKEN", Required: true, Hint: "https://github.com/CakeRepository/1Password-MCP.git"}},
		Description: "MCP server for 1Password service accounts — tools and resources for vaults and credentials",
		Category:    "security",
	},
	{
		Name:        "snyk",
		Command:     "npx",
		Args:        []string{"-y", "snyk"},
		Description: "Easily find and fix security issues in your applications leveraging Snyk platform",
		Category:    "security",
	},
	{
		Name:        "vault",
		Command:     "npx",
		Args:        []string{"-y", "chillai-vault-mcp"},
		Description: "MCP server for credential isolation — bots use passwords and API keys without seeing them",
		Category:    "security",
	},

	// ── Analytics ─────────────────────────────────────────────────────
	{
		Name:        "axiom",
		URL:         "https://mcp.axiom.co/sse",
		Transport:   "sse",
		Description: "Axiom datasets, APL queries, anomalies, and monitoring",
		Category:    "analytics",
	},
	{
		Name:        "datadog",
		Command:     "npx",
		Args:        []string{"-y", "datadog-mcp"},
		EnvVars:     []EnvVar{{Key: "DD_API_KEY", Required: true, Hint: "https://github.com/tantiope/datadog-mcp-server"}, {Key: "DD_APP_KEY", Required: true, Hint: "https://github.com/tantiope/datadog-mcp-server"}},
		Description: "Full Datadog API access: monitors, logs, metrics, traces, dashboards, and observability",
		Category:    "analytics",
	},
	{
		Name:        "grafana",
		Command:     "docker",
		Args:        []string{"run", "-i", "--rm", "docker.io/grafana/mcp-grafana:0.11.2"},
		EnvVars:     []EnvVar{{Key: "GRAFANA_URL", Required: true, Hint: "https://github.com/grafana/mcp-grafana"}},
		Description: "An MCP server giving access to Grafana dashboards, data and more",
		Category:    "analytics",
	},
	{
		Name:        "newrelic",
		URL:         "https://mcp.newrelic.com/mcp",
		Transport:   "streamable-http",
		Description: "Access New Relic observability data through MCP - query metrics, logs, traces, entities",
		Category:    "analytics",
	},
	{
		Name:        "pagerduty",
		Command:     "uvx",
		Args:        []string{"pagerduty-mcp"},
		Description: "PagerDuty's official MCP server which provides tools to interact with your PagerDuty account",
		Category:    "analytics",
	},
	{
		Name:        "prometheus",
		Command:     "npx",
		Args:        []string{"-y", "mcp-prometheus"},
		Description: "A Model Context Protocol (MCP) server for Prometheus monitoring",
		Category:    "analytics",
	},
	{
		Name:        "rootly",
		Command:     "npx",
		Args:        []string{"-y", "rootly-mcp-server"},
		EnvVars:     []EnvVar{{Key: "ROOTLY_API_TOKEN", Required: true, Hint: "https://github.com/Rootly-AI-Labs/Rootly-MCP-server"}},
		Description: "Incident management, on-call scheduling, and intelligent analysis powered by Rootly",
		Category:    "analytics",
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
