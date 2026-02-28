package planner

import (
	"strings"
	"testing"

	"mcpup/internal/store"
)

func TestDesiredStateForClient(t *testing.T) {
	cfg := store.NewDefaultConfig()
	cfg.Servers["github"] = store.Server{Command: "npx github"}
	cfg.Clients["cursor"] = store.ClientConfig{
		Servers: map[string]store.ServerState{
			"github": {
				Enabled:       true,
				EnabledTools:  []string{"a", "b"},
				DisabledTools: []string{"c"},
			},
		},
	}

	state, err := DesiredStateForClient(cfg, "cursor")
	if err != nil {
		t.Fatalf("desired state failed: %v", err)
	}
	if !state.Servers["github"].Enabled {
		t.Fatalf("expected github to be enabled")
	}
}

func TestDesiredStateFailsForUnknownServer(t *testing.T) {
	cfg := store.NewDefaultConfig()
	cfg.Clients["cursor"] = store.ClientConfig{
		Servers: map[string]store.ServerState{
			"missing": {Enabled: true},
		},
	}

	_, err := DesiredStateForClient(cfg, "cursor")
	if err == nil {
		t.Fatalf("expected unknown server error")
	}
}

func TestDiffNoop(t *testing.T) {
	current := ClientState{
		Client: "cursor",
		Servers: map[string]ServerState{
			"github": {
				Enabled:       true,
				EnabledTools:  []string{"search"},
				DisabledTools: []string{"delete"},
			},
		},
	}
	desired := current

	plan := Diff(current, desired)
	if plan.HasChanges() {
		t.Fatalf("expected no changes plan")
	}
}

func TestDiffAddsAndRemoves(t *testing.T) {
	current := ClientState{
		Client: "cursor",
		Servers: map[string]ServerState{
			"slack": {Enabled: true},
		},
	}
	desired := ClientState{
		Client: "cursor",
		Servers: map[string]ServerState{
			"github": {
				Enabled: true,
			},
		},
	}

	plan := Diff(current, desired)
	if !plan.HasChanges() {
		t.Fatalf("expected changes")
	}

	summary := DryRunSummary("enable", plan)
	if !strings.Contains(summary, "change(s)") {
		t.Fatalf("unexpected summary: %s", summary)
	}
}
