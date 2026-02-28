package zed

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/adapters"
	"github.com/mohammedsamin/mcpup/internal/planner"
)

const ClientName = "zed"

// Adapter implements Zed config translation.
// Zed stores MCP servers at top-level key "context_servers".
type Adapter struct{}

// Name returns the adapter client name.
func (a Adapter) Name() string {
	return ClientName
}

// Detect resolves Zed settings path.
func (a Adapter) Detect(workspace string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect zed config: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, ".zed", "settings.json"), nil
	case "windows":
		appData := strings.TrimSpace(os.Getenv("APPDATA"))
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Zed", "settings.json"), nil
	default:
		xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
		if xdg != "" {
			return filepath.Join(xdg, "zed", "settings.json"), nil
		}
		return filepath.Join(home, ".config", "zed", "settings.json"), nil
	}
}

// Read parses current state from Zed settings JSON/JSONC.
func (a Adapter) Read(path string) (planner.ClientState, error) {
	doc, err := readJSONCDocument(path)
	if err != nil {
		return planner.ClientState{}, err
	}
	state, err := adapters.ReadStateFromServerMap(doc, "context_servers", ClientName)
	if err != nil {
		return planner.ClientState{}, err
	}
	return state, nil
}

// Apply computes state diff from current to desired.
func (a Adapter) Apply(current planner.ClientState, desired planner.ClientState) (planner.Plan, error) {
	return planner.Diff(current, desired), nil
}

// Write writes desired state preserving unknown top-level keys.
func (a Adapter) Write(path string, desired planner.ClientState) error {
	doc, err := readJSONCDocument(path)
	if err != nil {
		return err
	}
	return adapters.WriteStateToServerMap(path, doc, "context_servers", desired)
}

// Validate ensures config remains parseable.
func (a Adapter) Validate(path string) error {
	doc, err := readJSONCDocument(path)
	if err != nil {
		return err
	}
	_, err = adapters.ReadStateFromServerMap(doc, "context_servers", ClientName)
	return err
}

func readJSONCDocument(path string) (adapters.JSONDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return adapters.JSONDocument{}, nil
		}
		return nil, err
	}

	doc := adapters.JSONDocument{}
	if err := json.Unmarshal(stripJSONComments(data), &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func stripJSONComments(input []byte) []byte {
	out := make([]byte, 0, len(input))
	inString := false
	escaped := false
	lineComment := false
	blockComment := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if lineComment {
			if ch == '\n' {
				lineComment = false
				out = append(out, ch)
			} else if ch == '\r' {
				lineComment = false
				out = append(out, ch)
			}
			continue
		}

		if blockComment {
			if ch == '*' && i+1 < len(input) && input[i+1] == '/' {
				blockComment = false
				i++
			}
			continue
		}

		if inString {
			out = append(out, ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			out = append(out, ch)
			continue
		}

		if ch == '/' && i+1 < len(input) {
			next := input[i+1]
			if next == '/' {
				lineComment = true
				i++
				continue
			}
			if next == '*' {
				blockComment = true
				i++
				continue
			}
		}

		out = append(out, ch)
	}

	return out
}
