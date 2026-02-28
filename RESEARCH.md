# Research Report: MCPilot — Market Validation & Competitive Analysis

## Summary

The pain point is **real and well-documented** — 213 upvotes on a single Claude Code GitHub issue requesting MCP tool filtering, plus multiple forum threads and blog posts about configuration nightmares. However, **the name "MCPilot" is heavily taken** (npm package, 4+ GitHub repos, mcpilot.ai domain, VS Code extension). And there are **more competitors than expected**, including MCP Click (paid product with profiles), MCP Toggle, mcp-hub (444 stars), mcp-manager, and mcp-mux. The opportunity is still real, but positioning and naming need rethinking.

---

## 1. Developer Pain Points — IS THE PROBLEM REAL?

### Verdict: YES — Strongly validated

**GitHub Issues (Hard Evidence):**
- [Claude Code #7328](https://github.com/anthropics/claude-code/issues/7328) — "MCP Tool Filtering: Allow Selective Enable/Disable of Individual Tools" — **213 upvotes, 80 comments**. Users report MCP tool definitions consuming 50k+ tokens, degrading Claude performance. Quote: *"why is it not in claude code? anthropic invented mcp yet the flagship coding product has inadequate support."*
- [Claude Code #24000](https://github.com/anthropics/claude-code/issues/24000) — "MCP Profiles for Rapid Context-Aware Tool Switching" — Users lose 10-15% of their context budget to unnecessary MCP tools. Requests profile-based switching (`claude --mcp-profile coding`).
- [VS Code #254931](https://github.com/microsoft/vscode/issues/254931) — "Enable/disable MCP server via UI/config per user/workspace"
- [Cursor Forum](https://forum.cursor.com/t/mcp-install-config-and-management-suggestions/49283) — Users describe the MCP setup as "unintuitive and inefficient", request toggle switches, config import, better error handling.

**Specific Complaints From Developers:**
1. **Config fragmentation** — Each client uses different JSON format and paths. VS Code uses `{ "mcp": { "servers": {} } }`, Claude Desktop/Cursor use `{ "mcpServers": {} }`. ([Source](https://dev.to/darkmavis1980/understanding-mcp-servers-across-different-platforms-claude-desktop-vs-vs-code-vs-cursor-4opk))
2. **No toggle, only delete** — To disable an MCP server in most clients, you must delete it entirely and re-add later.
3. **Token waste** — MCP tool definitions eat massive chunks of context (50k+ tokens reported for a few servers).
4. **No profiles** — Switching between "coding mode" and "writing mode" requires manual JSON editing.
5. **SSE incompatibility** — Configs that work in Cursor don't work in Claude Desktop due to different transport support.
6. **Behavioral drift** — Claude autonomously uses MCP tools it wasn't intended to (e.g., Slack searches triggered by name mentions).

**Blog Posts & Articles Confirming the Pain:**
- [StackOne](https://www.stackone.com/blog/mcp-where-its-been-where-its-going) — "MCP still has fundamental gaps including multi-tenancy, admin controls, context-aware discovery"
- [MCP Configuration Showdown](https://dredyson.com/mcp-configuration-showdown-i-tested-every-setup-method-for-cursor-and-claude-heres-the-best-approach/) — Author tested every setup method, confirming fragmentation
- [Scott Spence](https://scottspence.com/posts/configuring-mcp-tools-in-claude-code) — "The Better Way" to configure MCP tools, implying the default way is bad

---

## 2. add-mcp — CAN YOU REPLACE IT?

### Verdict: YES — add-mcp is install-only, you'd be a superset

**What add-mcp Actually Does:** ([Source](https://neon.com/blog/add-mcp))
- CLI tool that installs MCP servers across 9 clients with one command
- Auto-detects which agents are configured in your project
- Supports project-level and global installation
- Supports both remote and local MCP servers

**What add-mcp CANNOT Do:**
- ❌ Remove/uninstall MCP servers
- ❌ Disable servers temporarily
- ❌ Toggle individual tools
- ❌ Manage servers after installation
- ❌ Profiles or configuration switching
- ❌ Dashboard or UI

**Your Positioning:**
- add-mcp = "Install once"
- Your tool = "Install, manage, toggle, profile, monitor — forever"
- You would fully subsume add-mcp's functionality and add lifecycle management on top.

---

## 3. COMPETITORS — THE LANDSCAPE IS MORE CROWDED THAN YOU THOUGHT

### Direct Competitors:

| Tool | Stars | What It Does | Limitations |
|------|-------|-------------|-------------|
| **[MCP Click](https://mcp-click.com/)** | N/A (paid product) | Menu bar app, toggle servers, profiles, server store (100+ servers), sync across clients. Supports Claude, Cursor, VS Code, Windsurf, Zed. **$3.99/mo paid tier.** | Closed source, paid for full features, only free for 3 clients + 1 profile |
| **[mcp-hub](https://github.com/ravitemer/mcp-hub)** | **444** | Centralized MCP coordinator, single endpoint for all clients, dynamic start/stop, namespacing, OAuth, hot reload | Gateway approach (proxy), not a management UI for cross-client configs |
| **[MCP Toggle](https://github.com/gabrielbacha/MCP-Manager-GUI)** | 13 | Electron GUI for toggling MCP servers, auto-discovery, import/export | Only Claude + Cursor, no profiles, minimal maintenance (1 commit) |
| **[mcp-manager](https://github.com/MediaPublishing/mcp-manager)** | 57 | Web GUI for managing MCP in Claude + Cursor, toggle on/off, cross-client sync | Only 2 clients, 4 commits, no profiles |
| **[mcp-mux](https://github.com/mcpmux/mcp-mux)** | 0 | Desktop gateway, centralized management, credential security (OS keychain), "Spaces" for isolation, per-client tool permissions | Brand new (v0.3.0, Feb 2026), Rust-based, 0 stars |
| **[add-mcp](https://neon.com/blog/add-mcp)** | N/A | CLI installer across 9 clients | Install-only, no management |

### Key Takeaway:
- **MCP Click** is your closest competitor and is already a paid product with profiles + server store
- **mcp-hub** has the most traction (444 stars) but is a gateway/proxy, not a config manager
- The open-source management tools (MCP Toggle, mcp-manager) are **very early and undermaintained**
- There is a clear gap for a **well-built, open-source, comprehensive management tool** — the existing OSS options are half-baked

---

## 4. NAME CHECK — "MCPilot" IS TAKEN

### Verdict: NAME MUST CHANGE

**The name "MCPilot" is heavily used:**

| Where | What | Link |
|-------|------|------|
| **mcpilot.ai** | Active product — "Your Single Connection To Every MCP Server", FastAPI-based MCP gateway with admin dashboard | [mcpilot.ai](https://mcpilot.ai/) |
| **npm: mcpilot** | Published package (2 weekly downloads) | [Socket.dev analysis](https://socket.dev/npm/package/mcpilot) |
| **GitHub: Xiawpohr/mcpilot** | MetaMask blockchain MCP suite (3 stars) | [GitHub](https://github.com/Xiawpohr/mcpilot) |
| **GitHub: m-rishab/MCPilot** | AI chatbot with MCP servers (Gradio + GROQ) | [GitHub](https://github.com/m-rishab/MCPilot) |
| **GitHub: markushoefinger/MCPilot** | Config management for Claude Desktop + Claude Code with GitHub Gist sync (1 star) | [GitHub](https://github.com/markushoefinger/MCPilot) |
| **VS Code Marketplace** | "McPilot" extension for Terraform + AWS | [Marketplace](https://marketplace.visualstudio.com/items?itemName=mcpilot.mcpilot) |
| **LobeHub** | MCPilot listed as MCP Gateway server | [LobeHub](https://lobehub.com/mcp/ferrary7-mcpilot) |

**The markushoefinger/MCPilot project is especially problematic** — it's literally doing part of what you want to build (config management for Claude), with the exact same name.

---

## 5. ALTERNATIVE NAME SUGGESTIONS

Since "MCPilot" is taken, here are alternatives that capture the same concept:

| Name | Available? | Vibe |
|------|-----------|------|
| **mcpx** | Check needed | Short, dev-friendly, "MCP multiplexer/extended" |
| **mcp-deck** | Check needed | Control deck / dashboard metaphor |
| **mcphub** | Likely taken (mcp-hub exists) | — |
| **toolpilot** | Check needed | Avoids MCP prefix collision |
| **mcpctl** | Check needed | Unix-style control tool (like kubectl) |
| **mcp-switch** | Check needed | Clear what it does |
| **mcpboard** | Check needed | Dashboard metaphor |
| **mcp-central** | Check needed | Central management |

---

## 6. RECOMMENDATIONS

1. **The problem is validated** — 213 upvotes + multiple forum threads + blog posts. Build it.
2. **You CAN replace add-mcp** — it's install-only, you'd be a superset with full lifecycle management.
3. **Change the name** — "MCPilot" has at least 7 conflicting uses including an active product at mcpilot.ai and an npm package.
4. **Your real competitors are MCP Click (paid) and mcp-hub (gateway)** — there's a clear gap for a well-built open-source management tool that's NOT a gateway proxy.
5. **Differentiate on**: open-source, lightweight, profiles, CLI-first with optional web UI, supports ALL major clients, real-time toggle without restart.
