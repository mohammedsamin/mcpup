package adapters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/planner"
)

// JSONDocument preserves unknown top-level keys while manipulating mcpServers.
type JSONDocument map[string]json.RawMessage

// ReadJSONDocument reads a JSON object document from disk.
func ReadJSONDocument(path string) (JSONDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return JSONDocument{}, nil
		}
		return nil, err
	}

	doc := JSONDocument{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// ReadStateFromMCPServers decodes state from a top-level mcpServers key.
func ReadStateFromMCPServers(doc JSONDocument, client string) (planner.ClientState, error) {
	return ReadStateFromServerMap(doc, "mcpServers", client)
}

// ReadStateFromServerMap decodes state from a top-level server object key.
func ReadStateFromServerMap(doc JSONDocument, topLevelKey string, client string) (planner.ClientState, error) {
	state := planner.ClientState{
		Client:  client,
		Servers: map[string]planner.ServerState{},
	}

	raw, ok := doc[topLevelKey]
	if !ok || len(raw) == 0 {
		return state, nil
	}

	rawEntries := map[string]map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &rawEntries); err != nil {
		return planner.ClientState{}, fmt.Errorf("decode %s: %w", topLevelKey, err)
	}

	for serverName, entry := range rawEntries {
		enabled := true
		if rawEnabled, ok := entry["enabled"]; ok {
			if err := json.Unmarshal(rawEnabled, &enabled); err != nil {
				return planner.ClientState{}, fmt.Errorf("decode enabled for server %q: %w", serverName, err)
			}
		}

		var enabledTools []string
		if rawEnabledTools, ok := entry["enabledTools"]; ok {
			if err := json.Unmarshal(rawEnabledTools, &enabledTools); err != nil {
				return planner.ClientState{}, fmt.Errorf("decode enabledTools for server %q: %w", serverName, err)
			}
		}

		var disabledTools []string
		if rawDisabledTools, ok := entry["disabledTools"]; ok {
			if err := json.Unmarshal(rawDisabledTools, &disabledTools); err != nil {
				return planner.ClientState{}, fmt.Errorf("decode disabledTools for server %q: %w", serverName, err)
			}
		}

		state.Servers[serverName] = planner.ServerState{
			Enabled:       enabled,
			EnabledTools:  append([]string{}, enabledTools...),
			DisabledTools: append([]string{}, disabledTools...),
		}
	}

	return planner.NormalizeState(state), nil
}

// WriteStateToMCPServers updates mcpServers while preserving unknown top-level keys.
func WriteStateToMCPServers(path string, doc JSONDocument, desired planner.ClientState) error {
	return WriteStateToServerMap(path, doc, "mcpServers", desired)
}

// WriteStateToServerMap updates a top-level server object while preserving unknown top-level keys.
func WriteStateToServerMap(path string, doc JSONDocument, topLevelKey string, desired planner.ClientState) error {
	existingServers := map[string]map[string]json.RawMessage{}
	if raw, ok := doc[topLevelKey]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &existingServers); err != nil {
			return fmt.Errorf("decode existing %s: %w", topLevelKey, err)
		}
	}

	nextServers := map[string]map[string]json.RawMessage{}
	for serverName, state := range planner.NormalizeState(desired).Servers {
		base := map[string]json.RawMessage{}
		if current, ok := existingServers[serverName]; ok {
			for key, value := range current {
				base[key] = append(json.RawMessage{}, value...)
			}
		}

		rawEnabled, err := json.Marshal(state.Enabled)
		if err != nil {
			return fmt.Errorf("encode enabled for server %q: %w", serverName, err)
		}
		base["enabled"] = rawEnabled

		if len(state.EnabledTools) > 0 {
			rawEnabledTools, err := json.Marshal(state.EnabledTools)
			if err != nil {
				return fmt.Errorf("encode enabledTools for server %q: %w", serverName, err)
			}
			base["enabledTools"] = rawEnabledTools
		} else {
			delete(base, "enabledTools")
		}

		if len(state.DisabledTools) > 0 {
			rawDisabledTools, err := json.Marshal(state.DisabledTools)
			if err != nil {
				return fmt.Errorf("encode disabledTools for server %q: %w", serverName, err)
			}
			base["disabledTools"] = rawDisabledTools
		} else {
			delete(base, "disabledTools")
		}

		if def, ok := desired.ServerDefs[serverName]; ok && strings.TrimSpace(def.Command) != "" {
			rawCmd, err := json.Marshal(def.Command)
			if err != nil {
				return fmt.Errorf("encode command for server %q: %w", serverName, err)
			}
			base["command"] = rawCmd

			if len(def.Args) > 0 {
				rawArgs, err := json.Marshal(def.Args)
				if err != nil {
					return fmt.Errorf("encode args for server %q: %w", serverName, err)
				}
				base["args"] = rawArgs
			} else {
				delete(base, "args")
			}

			if len(def.Env) > 0 {
				rawEnv, err := json.Marshal(def.Env)
				if err != nil {
					return fmt.Errorf("encode env for server %q: %w", serverName, err)
				}
				base["env"] = rawEnv
			} else {
				delete(base, "env")
			}
		}

		nextServers[serverName] = base
	}

	raw, err := json.Marshal(nextServers)
	if err != nil {
		return fmt.Errorf("encode %s: %w", topLevelKey, err)
	}
	doc[topLevelKey] = raw

	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode document: %w", err)
	}
	body = append(body, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

// DeepCopyDocument clones the top-level raw map to avoid mutation surprises.
func DeepCopyDocument(doc JSONDocument) JSONDocument {
	out := make(JSONDocument, len(doc))
	for key, value := range doc {
		out[key] = append(json.RawMessage{}, value...)
	}
	return out
}

// CollectCommandKeys returns command strings from native entries for diagnostics.
func CollectCommandKeys(doc JSONDocument) []string {
	raw, ok := doc["mcpServers"]
	if !ok || len(raw) == 0 {
		return nil
	}

	type entry struct {
		Command string `json:"command"`
	}
	servers := map[string]entry{}
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil
	}

	out := make([]string, 0, len(servers))
	for _, e := range servers {
		if strings.TrimSpace(e.Command) != "" {
			out = append(out, e.Command)
		}
	}
	return out
}
