package planner

import (
	"fmt"
	"slices"
	"strings"

	"mcpup/internal/store"
)

// ClientState is the normalized desired/current state for one client.
type ClientState struct {
	Client     string                  `json:"client"`
	Servers    map[string]ServerState  `json:"servers"`
	ServerDefs map[string]store.Server `json:"-"` // populated by DesiredStateForClient, used by adapters for writing command/args/env
}

// ServerState contains server-level and per-tool controls.
type ServerState struct {
	Enabled       bool     `json:"enabled"`
	EnabledTools  []string `json:"enabledTools,omitempty"`
	DisabledTools []string `json:"disabledTools,omitempty"`
}

// DesiredStateForClient builds desired state from canonical config for one client.
func DesiredStateForClient(cfg store.Config, client string) (ClientState, error) {
	if err := store.ValidateClientName(client); err != nil {
		return ClientState{}, err
	}

	serverDefs := make(map[string]store.Server, len(cfg.Servers))
	for name, srv := range cfg.Servers {
		serverDefs[name] = srv
	}

	result := ClientState{
		Client:     client,
		Servers:    map[string]ServerState{},
		ServerDefs: serverDefs,
	}

	clientCfg, ok := cfg.Clients[client]
	if !ok {
		return result, nil
	}

	for serverName, state := range clientCfg.Servers {
		if _, exists := cfg.Servers[serverName]; !exists {
			return ClientState{}, fmt.Errorf("client %q references unknown server %q", client, serverName)
		}

		result.Servers[serverName] = normalizeServerState(ServerState{
			Enabled:       state.Enabled,
			EnabledTools:  append([]string{}, state.EnabledTools...),
			DisabledTools: append([]string{}, state.DisabledTools...),
		})
	}

	return result, nil
}

// NormalizeState canonicalizes list ordering and empty map handling.
func NormalizeState(state ClientState) ClientState {
	out := ClientState{
		Client:  strings.TrimSpace(state.Client),
		Servers: map[string]ServerState{},
	}
	for serverName, serverState := range state.Servers {
		out.Servers[strings.TrimSpace(serverName)] = normalizeServerState(serverState)
	}
	return out
}

// EquivalentState reports whether two client states are equal after normalization.
func EquivalentState(a ClientState, b ClientState) bool {
	na := NormalizeState(a)
	nb := NormalizeState(b)

	if na.Client != nb.Client || len(na.Servers) != len(nb.Servers) {
		return false
	}

	for serverName, aState := range na.Servers {
		bState, ok := nb.Servers[serverName]
		if !ok {
			return false
		}
		if aState.Enabled != bState.Enabled {
			return false
		}
		if !slices.Equal(aState.EnabledTools, bState.EnabledTools) {
			return false
		}
		if !slices.Equal(aState.DisabledTools, bState.DisabledTools) {
			return false
		}
	}

	return true
}

func normalizeServerState(state ServerState) ServerState {
	normalized := ServerState{
		Enabled:       state.Enabled,
		EnabledTools:  uniqueSorted(state.EnabledTools),
		DisabledTools: uniqueSorted(state.DisabledTools),
	}

	if len(normalized.EnabledTools) == 0 {
		normalized.EnabledTools = nil
	}
	if len(normalized.DisabledTools) == 0 {
		normalized.DisabledTools = nil
	}
	return normalized
}

func uniqueSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := set[trimmed]; exists {
			continue
		}
		set[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	slices.Sort(out)
	return out
}
