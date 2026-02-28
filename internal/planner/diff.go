package planner

import (
	"fmt"
	"slices"
)

// ChangeKind describes one reconciliation action.
type ChangeKind string

const (
	ChangeUpsertServer      ChangeKind = "upsert_server"
	ChangeRemoveServer      ChangeKind = "remove_server"
	ChangeEnableServer      ChangeKind = "enable_server"
	ChangeDisableServer     ChangeKind = "disable_server"
	ChangeEnableTool        ChangeKind = "enable_tool"
	ChangeDisableTool       ChangeKind = "disable_tool"
	ChangeClearEnabledTools ChangeKind = "clear_enabled_tools"
	ChangeClearDisabledTool ChangeKind = "clear_disabled_tools"
)

// Change is one planned mutation from current to desired state.
type Change struct {
	Kind   ChangeKind `json:"kind"`
	Server string     `json:"server"`
	Tool   string     `json:"tool,omitempty"`
	From   any        `json:"from,omitempty"`
	To     any        `json:"to,omitempty"`
}

// Plan is a full diff for one client.
type Plan struct {
	Client  string   `json:"client"`
	Changes []Change `json:"changes"`
}

// HasChanges reports whether this plan mutates anything.
func (p Plan) HasChanges() bool {
	return len(p.Changes) > 0
}

// Diff computes the mutation plan from current to desired state.
func Diff(current ClientState, desired ClientState) Plan {
	currentN := NormalizeState(current)
	desiredN := NormalizeState(desired)

	plan := Plan{
		Client:  desiredN.Client,
		Changes: []Change{},
	}

	for serverName, desiredServer := range desiredN.Servers {
		currentServer, exists := currentN.Servers[serverName]
		if !exists {
			plan.Changes = append(plan.Changes, Change{
				Kind:   ChangeUpsertServer,
				Server: serverName,
				To:     desiredServer,
			})
			continue
		}

		if currentServer.Enabled != desiredServer.Enabled {
			changeKind := ChangeEnableServer
			if !desiredServer.Enabled {
				changeKind = ChangeDisableServer
			}
			plan.Changes = append(plan.Changes, Change{
				Kind:   changeKind,
				Server: serverName,
				From:   currentServer.Enabled,
				To:     desiredServer.Enabled,
			})
		}

		plan.Changes = append(plan.Changes, diffToolLists(serverName, currentServer, desiredServer)...)
	}

	for serverName := range currentN.Servers {
		if _, stillPresent := desiredN.Servers[serverName]; !stillPresent {
			plan.Changes = append(plan.Changes, Change{
				Kind:   ChangeRemoveServer,
				Server: serverName,
			})
		}
	}

	return plan
}

func diffToolLists(serverName string, current ServerState, desired ServerState) []Change {
	changes := []Change{}

	enabledToAdd := difference(desired.EnabledTools, current.EnabledTools)
	enabledToRemove := difference(current.EnabledTools, desired.EnabledTools)
	for _, tool := range enabledToAdd {
		changes = append(changes, Change{
			Kind:   ChangeEnableTool,
			Server: serverName,
			Tool:   tool,
		})
	}
	if len(desired.EnabledTools) == 0 && len(current.EnabledTools) > 0 {
		changes = append(changes, Change{
			Kind:   ChangeClearEnabledTools,
			Server: serverName,
			From:   append([]string{}, current.EnabledTools...),
			To:     []string{},
		})
	} else {
		for _, tool := range enabledToRemove {
			changes = append(changes, Change{
				Kind:   ChangeDisableTool,
				Server: serverName,
				Tool:   tool,
				From:   "enabled",
				To:     "unspecified",
			})
		}
	}

	disabledToAdd := difference(desired.DisabledTools, current.DisabledTools)
	disabledToRemove := difference(current.DisabledTools, desired.DisabledTools)
	for _, tool := range disabledToAdd {
		changes = append(changes, Change{
			Kind:   ChangeDisableTool,
			Server: serverName,
			Tool:   tool,
		})
	}
	if len(desired.DisabledTools) == 0 && len(current.DisabledTools) > 0 {
		changes = append(changes, Change{
			Kind:   ChangeClearDisabledTool,
			Server: serverName,
			From:   append([]string{}, current.DisabledTools...),
			To:     []string{},
		})
	} else {
		for _, tool := range disabledToRemove {
			changes = append(changes, Change{
				Kind:   ChangeEnableTool,
				Server: serverName,
				Tool:   tool,
				From:   "disabled",
				To:     "unspecified",
			})
		}
	}

	return changes
}

// ValidateServerReference ensures requested server exists before planning commands.
func ValidateServerReference(cfgServers map[string]struct{}, serverName string) error {
	if _, ok := cfgServers[serverName]; !ok {
		return fmt.Errorf("unknown server %q", serverName)
	}
	return nil
}

func difference(a []string, b []string) []string {
	if len(a) == 0 {
		return nil
	}
	out := make([]string, 0, len(a))
	for _, value := range a {
		if !slices.Contains(b, value) {
			out = append(out, value)
		}
	}
	return out
}
