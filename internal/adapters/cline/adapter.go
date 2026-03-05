package cline

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mohammedsamin/mcpup/internal/adapters"
	"github.com/mohammedsamin/mcpup/internal/planner"
)

const ClientName = "cline"

// Adapter implements Cline config translation.
type Adapter struct{}

// Name returns the adapter client name.
func (a Adapter) Name() string {
	return ClientName
}

// Detect resolves Cline MCP settings path.
// Cline stores config in VS Code extension global storage (no workspace override).
func (a Adapter) Detect(workspace string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect cline config: %w", err)
	}

	const relPath = "Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json"

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", relPath), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA is not set")
		}
		return filepath.Join(appData, relPath), nil
	default:
		return filepath.Join(home, ".config", relPath), nil
	}
}

// Read parses current state from Cline config.
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
