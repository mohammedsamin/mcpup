package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mohammedsamin/mcpup/internal/adapters"
	"github.com/mohammedsamin/mcpup/internal/planner"
)

const ClientName = "opencode"

// Adapter implements OpenCode config translation.
// OpenCode keeps MCP settings inside top-level "mcp.servers".
type Adapter struct{}

// Name returns adapter client name.
func (a Adapter) Name() string {
	return ClientName
}

// Detect resolves OpenCode config path.
func (a Adapter) Detect(workspace string) (string, error) {
	if workspace != "" {
		candidate := filepath.Join(workspace, "opencode.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect opencode config: %w", err)
	}
	return filepath.Join(home, ".config", "opencode", "opencode.json"), nil
}

// Read parses mcp.servers state.
func (a Adapter) Read(path string) (planner.ClientState, error) {
	doc, err := adapters.ReadJSONDocument(path)
	if err != nil {
		return planner.ClientState{}, err
	}

	state := planner.ClientState{
		Client:  ClientName,
		Servers: map[string]planner.ServerState{},
	}

	rawMCP, ok := doc["mcp"]
	if !ok || len(rawMCP) == 0 {
		return state, nil
	}

	mcpObj := map[string]json.RawMessage{}
	if err := json.Unmarshal(rawMCP, &mcpObj); err != nil {
		return planner.ClientState{}, fmt.Errorf("decode mcp object: %w", err)
	}

	rawServers, ok := mcpObj["servers"]
	if !ok || len(rawServers) == 0 {
		return state, nil
	}

	entries := map[string]map[string]json.RawMessage{}
	if err := json.Unmarshal(rawServers, &entries); err != nil {
		return planner.ClientState{}, fmt.Errorf("decode mcp.servers: %w", err)
	}

	for serverName, entry := range entries {
		server := planner.ServerState{Enabled: true}
		if rawEnabled, ok := entry["enabled"]; ok {
			if err := json.Unmarshal(rawEnabled, &server.Enabled); err != nil {
				return planner.ClientState{}, fmt.Errorf("decode enabled for %s: %w", serverName, err)
			}
		}
		if rawEnabledTools, ok := entry["enabledTools"]; ok {
			if err := json.Unmarshal(rawEnabledTools, &server.EnabledTools); err != nil {
				return planner.ClientState{}, fmt.Errorf("decode enabledTools for %s: %w", serverName, err)
			}
		}
		if rawDisabledTools, ok := entry["disabledTools"]; ok {
			if err := json.Unmarshal(rawDisabledTools, &server.DisabledTools); err != nil {
				return planner.ClientState{}, fmt.Errorf("decode disabledTools for %s: %w", serverName, err)
			}
		}
		state.Servers[serverName] = server
	}

	return planner.NormalizeState(state), nil
}

// Apply computes diff from current to desired.
func (a Adapter) Apply(current planner.ClientState, desired planner.ClientState) (planner.Plan, error) {
	return planner.Diff(current, desired), nil
}

// Write updates mcp.servers while preserving unknown keys.
func (a Adapter) Write(path string, desired planner.ClientState) error {
	doc, err := adapters.ReadJSONDocument(path)
	if err != nil {
		return err
	}

	mcpObj := map[string]json.RawMessage{}
	if rawMCP, ok := doc["mcp"]; ok && len(rawMCP) > 0 {
		if err := json.Unmarshal(rawMCP, &mcpObj); err != nil {
			return fmt.Errorf("decode mcp object: %w", err)
		}
	}

	existingServers := map[string]map[string]json.RawMessage{}
	if rawServers, ok := mcpObj["servers"]; ok && len(rawServers) > 0 {
		if err := json.Unmarshal(rawServers, &existingServers); err != nil {
			return fmt.Errorf("decode mcp.servers: %w", err)
		}
	}

	nextServers := map[string]map[string]json.RawMessage{}
	for serverName, state := range planner.NormalizeState(desired).Servers {
		entry := map[string]json.RawMessage{}
		if current, ok := existingServers[serverName]; ok {
			for key, value := range current {
				entry[key] = append(json.RawMessage{}, value...)
			}
		}

		rawEnabled, _ := json.Marshal(state.Enabled)
		entry["enabled"] = rawEnabled

		if len(state.EnabledTools) > 0 {
			raw, _ := json.Marshal(state.EnabledTools)
			entry["enabledTools"] = raw
		} else {
			delete(entry, "enabledTools")
		}
		if len(state.DisabledTools) > 0 {
			raw, _ := json.Marshal(state.DisabledTools)
			entry["disabledTools"] = raw
		} else {
			delete(entry, "disabledTools")
		}

		// Write server definition fields.
		if def, ok := desired.ServerDefs[serverName]; ok {
			if def.IsHTTP() {
				rawURL, _ := json.Marshal(def.URL)
				entry["url"] = rawURL
				if len(def.Headers) > 0 {
					rawHeaders, _ := json.Marshal(def.Headers)
					entry["headers"] = rawHeaders
				} else {
					delete(entry, "headers")
				}
				if def.Transport != "" {
					rawTransport, _ := json.Marshal(def.Transport)
					entry["transport"] = rawTransport
				}
				delete(entry, "command")
				delete(entry, "args")
				delete(entry, "env")
			} else if def.Command != "" {
				rawCmd, _ := json.Marshal(def.Command)
				entry["command"] = rawCmd
				if len(def.Args) > 0 {
					rawArgs, _ := json.Marshal(def.Args)
					entry["args"] = rawArgs
				} else {
					delete(entry, "args")
				}
				if len(def.Env) > 0 {
					rawEnv, _ := json.Marshal(def.Env)
					entry["env"] = rawEnv
				} else {
					delete(entry, "env")
				}
				delete(entry, "url")
				delete(entry, "headers")
				delete(entry, "transport")
			}
		}

		nextServers[serverName] = entry
	}

	rawServers, err := json.Marshal(nextServers)
	if err != nil {
		return err
	}
	mcpObj["servers"] = rawServers

	rawMCP, err := json.Marshal(mcpObj)
	if err != nil {
		return err
	}
	doc["mcp"] = rawMCP

	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

// Validate checks config parseability for this adapter.
func (a Adapter) Validate(path string) error {
	_, err := a.Read(path)
	return err
}
