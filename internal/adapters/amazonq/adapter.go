package amazonq

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mohammedsamin/mcpup/internal/adapters"
	"github.com/mohammedsamin/mcpup/internal/planner"
)

const ClientName = "amazon-q"

// Adapter implements Amazon Q Developer config translation.
type Adapter struct{}

// Name returns the adapter client name.
func (a Adapter) Name() string {
	return ClientName
}

// Detect resolves Amazon Q MCP config path.
// Preference order:
// 1. <workspace>/.amazonq/mcp.json
// 2. ~/.aws/amazonq/mcp.json
func (a Adapter) Detect(workspace string) (string, error) {
	if workspace != "" {
		candidate := filepath.Join(workspace, ".amazonq", "mcp.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect amazon-q config: %w", err)
	}
	return filepath.Join(home, ".aws", "amazonq", "mcp.json"), nil
}

// Read parses current state from Amazon Q config.
func (a Adapter) Read(path string) (planner.ClientState, error) {
	doc, err := adapters.ReadJSONDocument(path)
	if err != nil {
		return planner.ClientState{}, err
	}
	state, err := adapters.ReadStateFromMCPServers(doc, ClientName)
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
