package continuedev

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/mohammedsamin/mcpup/internal/adapters"
	"github.com/mohammedsamin/mcpup/internal/planner"
)

const ClientName = "continue"

// Adapter implements Continue MCP config translation.
// mcpup manages a JSON-compatible file in .continue/mcpServers.
type Adapter struct{}

// Name returns the adapter client name.
func (a Adapter) Name() string {
	return ClientName
}

// Detect resolves Continue MCP server config path.
// Preference order:
// 1. <workspace>/.continue/mcpServers/mcpup.json
// 2. ~/.continue/mcpServers/mcpup.json
func (a Adapter) Detect(workspace string) (string, error) {
	if workspace != "" {
		candidate := filepath.Join(workspace, ".continue", "mcpServers", "mcpup.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect continue config: %w", err)
	}
	return filepath.Join(home, ".continue", "mcpServers", "mcpup.json"), nil
}

// Read parses current state from Continue MCP config.
func (a Adapter) Read(path string) (planner.ClientState, error) {
	doc, err := adapters.ReadJSONDocument(path)
	if err != nil {
		return planner.ClientState{}, err
	}
	state, err := adapters.ReadStateFromMCPServers(doc, ClientName)
	if err != nil {
		return planner.ClientState{}, err
	}
	if len(state.Servers) > 0 && len(state.Owned) == 0 {
		state.Owned = map[string]bool{}
		for name := range state.Servers {
			state.Owned[name] = true
		}
		state = planner.NormalizeState(state)
	}
	return state, nil
}

// Apply computes state diff from current to desired.
func (a Adapter) Apply(current planner.ClientState, desired planner.ClientState) (planner.Plan, error) {
	return adapters.ManagedDiff(current, desired), nil
}

// Write writes desired state preserving unknown top-level keys.
// Continue requires explicit "type": "sse" for HTTP servers, so we set
// transport before delegating to the shared helper.
func (a Adapter) Write(path string, desired planner.ClientState) error {
	doc, err := adapters.ReadJSONDocument(path)
	if err != nil {
		return err
	}

	if _, ok := doc["_mcpup"]; !ok {
		current, readErr := adapters.ReadStateFromMCPServers(doc, ClientName)
		if readErr == nil && len(current.Servers) > 0 {
			names := make([]string, 0, len(current.Servers))
			for name := range current.Servers {
				names = append(names, name)
			}
			slices.Sort(names)
			raw, marshalErr := json.Marshal(map[string]any{"managedServers": names})
			if marshalErr == nil {
				doc["_mcpup"] = raw
			}
		}
	}

	// Ensure HTTP servers have a transport set for Continue's "type" field.
	if desired.ServerDefs != nil {
		for name, def := range desired.ServerDefs {
			if def.IsHTTP() && def.Transport == "" {
				def.Transport = "sse"
				desired.ServerDefs[name] = def
			}
		}
	}

	return adapters.WriteStateToMCPServers(path, doc, desired)
}

// Validate ensures config remains parseable.
func (a Adapter) Validate(path string) error {
	doc, err := adapters.ReadJSONDocument(path)
	if err != nil {
		return err
	}
	_, err = adapters.ReadStateFromMCPServers(doc, ClientName)
	return err
}
