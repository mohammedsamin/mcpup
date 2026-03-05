package vscode

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mohammedsamin/mcpup/internal/adapters"
	"github.com/mohammedsamin/mcpup/internal/planner"
)

const ClientName = "vscode"

// Adapter implements VS Code Copilot config translation.
// VS Code stores MCP servers at top-level key "servers".
type Adapter struct{}

// Name returns the adapter client name.
func (a Adapter) Name() string {
	return ClientName
}

// Detect resolves VS Code MCP config path.
// Preference order:
// 1. <workspace>/.vscode/mcp.json
// 2. ~/.vscode/mcp.json
func (a Adapter) Detect(workspace string) (string, error) {
	if workspace != "" {
		candidate := filepath.Join(workspace, ".vscode", "mcp.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect vscode config: %w", err)
	}
	return filepath.Join(home, ".vscode", "mcp.json"), nil
}

// Read parses current state from VS Code config.
func (a Adapter) Read(path string) (planner.ClientState, error) {
	doc, err := adapters.ReadJSONDocument(path)
	if err != nil {
		return planner.ClientState{}, err
	}
	state, err := adapters.ReadStateFromServerMap(doc, "servers", ClientName)
	if err != nil {
		return planner.ClientState{}, err
	}
	return state, nil
}

// Apply computes state diff from current to desired.
func (a Adapter) Apply(current planner.ClientState, desired planner.ClientState) (planner.Plan, error) {
	return adapters.ManagedDiff(current, desired), nil
}

// Write writes desired state preserving unknown top-level keys.
func (a Adapter) Write(path string, desired planner.ClientState) error {
	doc, err := adapters.ReadJSONDocument(path)
	if err != nil {
		return err
	}
	return adapters.WriteStateToServerMap(path, doc, "servers", desired)
}

// Validate ensures config remains parseable.
func (a Adapter) Validate(path string) error {
	doc, err := adapters.ReadJSONDocument(path)
	if err != nil {
		return err
	}
	_, err = adapters.ReadStateFromServerMap(doc, "servers", ClientName)
	return err
}
