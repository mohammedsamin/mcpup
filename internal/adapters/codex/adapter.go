package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"mcpup/internal/planner"
)

const ClientName = "codex"

const (
	managedStart = "# mcpup:begin"
	managedEnd   = "# mcpup:end"
)

var sectionPattern = regexp.MustCompile(`^\[mcp_servers\.([a-zA-Z0-9_-]+)\]$`)

// Adapter implements Codex config translation for ~/.codex/config.toml.
// It preserves unknown file content and stores managed state in a comment block.
type Adapter struct{}

// Name returns the adapter client name.
func (a Adapter) Name() string {
	return ClientName
}

// Detect resolves Codex config path.
func (a Adapter) Detect(workspace string) (string, error) {
	if workspace != "" {
		candidate := filepath.Join(workspace, ".codex", "config.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect codex config: %w", err)
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

// Read parses managed block first, then falls back to simple TOML section parsing.
func (a Adapter) Read(path string) (planner.ClientState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return planner.ClientState{Client: ClientName, Servers: map[string]planner.ServerState{}}, nil
		}
		return planner.ClientState{}, err
	}

	if state, ok, err := readManagedBlock(data); err != nil {
		return planner.ClientState{}, err
	} else if ok {
		state.Client = ClientName
		return planner.NormalizeState(state), nil
	}

	state, err := parseSimpleTOMLState(data)
	if err != nil {
		return planner.ClientState{}, err
	}
	state.Client = ClientName
	return planner.NormalizeState(state), nil
}

// Apply computes state diff from current to desired.
func (a Adapter) Apply(current planner.ClientState, desired planner.ClientState) (planner.Plan, error) {
	return planner.Diff(current, desired), nil
}

// Write writes desired state as real TOML [mcp_servers.X] sections
// and a managed comment block for mcpup state tracking.
func (a Adapter) Write(path string, desired planner.ClientState) error {
	var data []byte
	existing, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else {
		data = existing
	}

	// Determine all servers managed by mcpup (previous + current).
	managedNames := map[string]bool{}
	if prev, ok, _ := readManagedBlock(data); ok {
		for name := range prev.Servers {
			managedNames[name] = true
		}
	}
	for name := range desired.Servers {
		managedNames[name] = true
	}

	// Remove old mcpup-managed TOML sections and old managed block.
	cleaned := removeTOMLServerSections(data, managedNames)
	cleaned = stripManagedBlock(cleaned)

	// Generate real TOML sections for enabled servers.
	tomlSections := encodeTOMLServers(desired)

	// Generate managed comment block (for mcpup state tracking).
	managed, err := encodeManagedBlock(desired)
	if err != nil {
		return err
	}

	// Assemble final content.
	base := strings.TrimRight(string(cleaned), " \t\n\r")
	var parts []string
	if base != "" {
		parts = append(parts, base)
	}
	if len(tomlSections) > 0 {
		parts = append(parts, strings.TrimRight(string(tomlSections), "\n"))
	}
	parts = append(parts, string(managed))

	result := strings.Join(parts, "\n\n") + "\n"

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(result), 0o644)
}

// Validate ensures file content can be read by this adapter.
func (a Adapter) Validate(path string) error {
	_, err := a.Read(path)
	return err
}

func readManagedBlock(data []byte) (planner.ClientState, bool, error) {
	lines := splitLines(data)
	start := -1
	end := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == managedStart {
			start = i
		}
		if trimmed == managedEnd && start >= 0 && i > start {
			end = i
			break
		}
	}

	if start < 0 || end < 0 {
		return planner.ClientState{}, false, nil
	}

	blockLines := lines[start+1 : end]
	jsonLines := make([]string, 0, len(blockLines))
	for _, line := range blockLines {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimPrefix(trimmed, "#")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			jsonLines = append(jsonLines, trimmed)
		}
	}

	if len(jsonLines) == 0 {
		return planner.ClientState{Servers: map[string]planner.ServerState{}}, true, nil
	}

	var payload struct {
		Servers map[string]planner.ServerState `json:"servers"`
	}
	if err := json.Unmarshal([]byte(strings.Join(jsonLines, "")), &payload); err != nil {
		return planner.ClientState{}, true, fmt.Errorf("decode managed codex block: %w", err)
	}

	return planner.ClientState{
		Servers: payload.Servers,
	}, true, nil
}

type codexServerEntry struct {
	Enabled       bool              `json:"enabled"`
	EnabledTools  []string          `json:"enabledTools,omitempty"`
	DisabledTools []string          `json:"disabledTools,omitempty"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
}

func encodeManagedBlock(state planner.ClientState) ([]byte, error) {
	normalized := planner.NormalizeState(state)
	servers := make(map[string]codexServerEntry, len(normalized.Servers))
	for name, s := range normalized.Servers {
		entry := codexServerEntry{
			Enabled:       s.Enabled,
			EnabledTools:  s.EnabledTools,
			DisabledTools: s.DisabledTools,
		}
		if def, ok := state.ServerDefs[name]; ok {
			entry.Command = def.Command
			entry.Args = def.Args
			entry.Env = def.Env
		}
		servers[name] = entry
	}
	payload := struct {
		Servers map[string]codexServerEntry `json:"servers"`
	}{Servers: servers}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode managed codex block: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(body))
	var out []string
	out = append(out, managedStart)
	for scanner.Scan() {
		out = append(out, "# "+scanner.Text())
	}
	out = append(out, managedEnd)
	return []byte(strings.Join(out, "\n")), nil
}

// encodeTOMLServers generates real TOML [mcp_servers.X] sections for enabled servers.
func encodeTOMLServers(state planner.ClientState) []byte {
	names := make([]string, 0, len(state.Servers))
	for name, s := range state.Servers {
		if s.Enabled {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	if len(names) == 0 {
		return nil
	}

	var sections []string
	for _, name := range names {
		var lines []string
		lines = append(lines, fmt.Sprintf("[mcp_servers.%s]", name))

		if def, ok := state.ServerDefs[name]; ok {
			if def.Command != "" {
				lines = append(lines, fmt.Sprintf("command = %s", quoteTOML(def.Command)))
			}
			if len(def.Args) > 0 {
				lines = append(lines, fmt.Sprintf("args = [%s]", joinTOMLArray(def.Args)))
			}
			if len(def.Env) > 0 {
				envKeys := make([]string, 0, len(def.Env))
				for k := range def.Env {
					envKeys = append(envKeys, k)
				}
				sort.Strings(envKeys)
				lines = append(lines, "")
				lines = append(lines, fmt.Sprintf("[mcp_servers.%s.env]", name))
				for _, k := range envKeys {
					lines = append(lines, fmt.Sprintf("%s = %s", k, quoteTOML(def.Env[k])))
				}
			}
		}

		sections = append(sections, strings.Join(lines, "\n"))
	}

	return []byte(strings.Join(sections, "\n\n"))
}

// removeTOMLServerSections strips [mcp_servers.X] sections where X is in managedNames.
func removeTOMLServerSections(data []byte, managedNames map[string]bool) []byte {
	lines := splitLines(data)
	var result []string
	skip := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") {
			skip = false
			for name := range managedNames {
				prefix := "[mcp_servers." + name
				if trimmed == prefix+"]" || strings.HasPrefix(trimmed, prefix+".") {
					skip = true
					break
				}
			}
		}

		if !skip {
			result = append(result, line)
		}
	}

	return []byte(strings.Join(result, "\n"))
}

// stripManagedBlock removes the mcpup:begin/end comment block from data.
func stripManagedBlock(data []byte) []byte {
	lines := splitLines(data)
	start := -1
	end := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == managedStart {
			start = i
		}
		if trimmed == managedEnd && start >= 0 && i > start {
			end = i
			break
		}
	}

	if start < 0 || end < 0 {
		return data
	}

	var result []string
	result = append(result, lines[:start]...)
	if end+1 < len(lines) {
		result = append(result, lines[end+1:]...)
	}
	return []byte(strings.Join(result, "\n"))
}

func replaceManagedBlock(existing []byte, managed []byte) []byte {
	lines := splitLines(existing)
	start := -1
	end := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == managedStart {
			start = i
		}
		if trimmed == managedEnd && start >= 0 && i >= start {
			end = i
			break
		}
	}

	if start >= 0 && end >= start {
		var result []string
		result = append(result, lines[:start]...)
		result = append(result, splitLines(managed)...)
		if end+1 < len(lines) {
			result = append(result, lines[end+1:]...)
		}
		return []byte(strings.Join(result, "\n"))
	}

	base := strings.TrimRight(string(existing), "\n")
	if base == "" {
		return managed
	}
	return []byte(base + "\n\n" + string(managed))
}

func splitLines(data []byte) []string {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	return strings.Split(text, "\n")
}

func parseSimpleTOMLState(data []byte) (planner.ClientState, error) {
	lines := splitLines(data)
	state := planner.ClientState{
		Servers: map[string]planner.ServerState{},
	}

	currentServer := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if m := sectionPattern.FindStringSubmatch(trimmed); len(m) == 2 {
			currentServer = m[1]
			if _, exists := state.Servers[currentServer]; !exists {
				state.Servers[currentServer] = planner.ServerState{Enabled: true}
			}
			continue
		}

		if currentServer == "" {
			continue
		}

		key, value, ok := splitKV(trimmed)
		if !ok {
			continue
		}
		server := state.Servers[currentServer]
		switch key {
		case "enabled":
			server.Enabled = value == "true"
		case "enabled_tools":
			server.EnabledTools = parseSimpleList(value)
		case "disabled_tools":
			server.DisabledTools = parseSimpleList(value)
		}
		state.Servers[currentServer] = server
	}

	return planner.NormalizeState(state), nil
}

func splitKV(line string) (string, string, bool) {
	idx := strings.Index(line, "=")
	if idx <= 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	return key, value, true
}

func parseSimpleList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 2 || trimmed[0] != '[' || trimmed[len(trimmed)-1] != ']' {
		return nil
	}
	inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	if inner == "" {
		return nil
	}
	parts := strings.Split(inner, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		value = strings.Trim(value, `"`)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func quoteTOML(s string) string {
	if strings.ContainsAny(s, "'\n") {
		// Use double quotes and escape.
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		return `"` + s + `"`
	}
	return "'" + s + "'"
}

func joinTOMLArray(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = quoteTOML(item)
	}
	return strings.Join(quoted, ", ")
}
