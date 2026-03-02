package cli

import (
	"fmt"
	"os"

	"github.com/mohammedsamin/mcpup/internal/core"
	"github.com/mohammedsamin/mcpup/internal/planner"
	"github.com/mohammedsamin/mcpup/internal/store"
)

// clientsReferencingServer returns supported clients that currently track the server.
func clientsReferencingServer(cfg store.Config, serverName string) []string {
	clients := make([]string, 0, len(store.SupportedClients))
	for _, client := range store.SupportedClients {
		clientCfg, ok := cfg.Clients[client]
		if !ok {
			continue
		}
		if _, exists := clientCfg.Servers[serverName]; exists {
			clients = append(clients, client)
		}
	}
	return clients
}

// reconcileClients applies desired state for each provided client.
func reconcileClients(cfg store.Config, clients []string, commandName string, dryRun bool) ([]core.ReconcileResult, error) {
	if len(clients) == 0 {
		return nil, nil
	}

	reconciler, err := core.NewReconciler()
	if err != nil {
		return nil, err
	}
	workspace, _ := os.Getwd()

	results := make([]core.ReconcileResult, 0, len(clients))
	for _, client := range clients {
		desired, err := planner.DesiredStateForClient(cfg, client)
		if err != nil {
			return results, fmt.Errorf("build desired state for %q: %w", client, err)
		}
		result, err := reconciler.ReconcileClient(desired, core.ReconcileOptions{
			Client:          client,
			Workspace:       workspace,
			CommandName:     commandName,
			DryRun:          dryRun,
			BackupRetention: 20,
		})
		if err != nil {
			return results, fmt.Errorf("reconcile %q: %w", client, err)
		}
		results = append(results, result)
	}

	return results, nil
}
