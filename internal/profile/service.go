package profile

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/core"
	"github.com/mohammedsamin/mcpup/internal/planner"
	"github.com/mohammedsamin/mcpup/internal/store"
)

// ApplyOptions controls profile apply behavior.
type ApplyOptions struct {
	DryRun          bool
	Workspace       string
	BackupRetention int
	Clients         []string // if non-empty, apply only to these clients; otherwise all supported clients
}

// ApplyResult reports reconciliation output for a profile apply operation.
type ApplyResult struct {
	Name       string                 `json:"name"`
	Results    []core.ReconcileResult `json:"results"`
	RolledBack []string               `json:"rolledBack,omitempty"`
}

// Summary describes one saved profile.
type Summary struct {
	Name      string   `json:"name"`
	Active    bool     `json:"active"`
	ServerIDs []string `json:"serverIds"`
}

// Create stores a new profile after validating server references.
func Create(cfg *store.Config, name string, servers []string) error {
	serverList := normalizeList(servers)
	if len(serverList) == 0 {
		return fmt.Errorf("profile create requires at least one server")
	}
	for _, server := range serverList {
		if _, ok := cfg.Servers[server]; !ok {
			return fmt.Errorf("profile references unknown server %q", server)
		}
	}
	return store.UpsertProfile(cfg, name, store.Profile{
		Servers: serverList,
	})
}

// Delete removes a profile and clears activeProfile if needed.
func Delete(cfg *store.Config, name string) error {
	return store.DeleteProfile(cfg, strings.TrimSpace(name))
}

// List returns ordered profile summaries with active marker.
func List(cfg store.Config) []Summary {
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	slices.Sort(names)

	out := make([]Summary, 0, len(names))
	for _, name := range names {
		prof := cfg.Profiles[name]
		out = append(out, Summary{
			Name:      name,
			Active:    cfg.ActiveProfile == name,
			ServerIDs: append([]string{}, prof.Servers...),
		})
	}
	return out
}

// Apply applies a profile across all supported clients.
// On non-dry-run partial failure, previously changed clients are rolled back.
func Apply(cfg *store.Config, name string, reconciler *core.Reconciler, opts ApplyOptions) (ApplyResult, error) {
	prof, ok := cfg.Profiles[name]
	if !ok {
		return ApplyResult{}, fmt.Errorf("profile %q not found", name)
	}
	if reconciler == nil {
		return ApplyResult{}, fmt.Errorf("reconciler is required")
	}

	applyProfileToConfig(cfg, prof)
	cfg.ActiveProfile = name

	result := ApplyResult{
		Name:    name,
		Results: []core.ReconcileResult{},
	}

	clients := opts.Clients
	if len(clients) == 0 {
		clients = store.SupportedClients
	}

	for _, client := range clients {
		desired, err := planner.DesiredStateForClient(*cfg, client)
		if err != nil {
			return result, err
		}

		r, err := reconciler.ReconcileClient(desired, core.ReconcileOptions{
			Client:          client,
			Workspace:       opts.Workspace,
			CommandName:     "profile apply",
			DryRun:          opts.DryRun,
			BackupRetention: opts.BackupRetention,
		})
		if err != nil {
			if !opts.DryRun {
				result.RolledBack = rollbackApplied(result.Results, reconciler)
			}
			return result, err
		}
		result.Results = append(result.Results, r)
	}

	return result, nil
}

func applyProfileToConfig(cfg *store.Config, prof store.Profile) {
	profileServers := make(map[string]struct{}, len(prof.Servers))
	for _, server := range normalizeList(prof.Servers) {
		profileServers[server] = struct{}{}
	}

	for _, client := range store.SupportedClients {
		state := cfg.Clients[client]
		if state.Servers == nil {
			state.Servers = map[string]store.ServerState{}
		}

		for serverName := range cfg.Servers {
			serverState := state.Servers[serverName]
			_, enabled := profileServers[serverName]
			serverState.Enabled = enabled
			if !enabled {
				serverState.EnabledTools = nil
				serverState.DisabledTools = nil
			}

			if selection, ok := prof.Tools[serverName]; ok {
				serverState.EnabledTools = normalizeList(selection.Enabled)
				serverState.DisabledTools = normalizeList(selection.Disabled)
				if len(serverState.EnabledTools) > 0 || len(serverState.DisabledTools) > 0 {
					serverState.Enabled = true
				}
			}

			state.Servers[serverName] = serverState
		}
		cfg.Clients[client] = state
	}
}

func rollbackApplied(results []core.ReconcileResult, reconciler *core.Reconciler) []string {
	if reconciler == nil || reconciler.Backups == nil {
		return nil
	}
	rolled := []string{}
	for i := len(results) - 1; i >= 0; i-- {
		res := results[i]
		if strings.TrimSpace(res.Backup) == "" {
			continue
		}
		if _, err := reconciler.Backups.Rollback(res.Client, res.Backup); err != nil {
			rolled = append(rolled, res.Client+":rollback_failed")
			continue
		}
		rolled = append(rolled, res.Client)
	}
	return rolled
}

func normalizeList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	slices.Sort(out)
	return out
}
