package cli

import (
	"fmt"
	"slices"

	"github.com/mohammedsamin/mcpup/internal/planner"
	"github.com/mohammedsamin/mcpup/internal/store"
)

func syncManagedClientState(cfg *store.Config, client string, restored planner.ClientState) ([]string, error) {
	clientCfg := cfg.Clients[client]
	currentManaged := map[string]bool{}
	for name := range clientCfg.Servers {
		currentManaged[name] = true
	}

	nextServers := map[string]store.ServerState{}
	skipped := []string{}

	for name, srv := range restored.Servers {
		if !restored.Owned[name] && !currentManaged[name] {
			skipped = append(skipped, name)
			continue
		}
		if _, ok := cfg.Servers[name]; !ok {
			return skipped, fmt.Errorf("restored managed server %q is not defined in canonical config", name)
		}
		nextServers[name] = store.ServerState{
			Enabled:       srv.Enabled,
			EnabledTools:  append([]string{}, srv.EnabledTools...),
			DisabledTools: append([]string{}, srv.DisabledTools...),
		}
	}

	clientCfg.Servers = nextServers
	cfg.Clients[client] = clientCfg
	slices.Sort(skipped)
	return skipped, nil
}
