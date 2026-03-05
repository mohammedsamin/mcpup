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

// reconcileClientsTransactional applies a proposed config to all clients.
// If a client fails after earlier writes succeeded, previous client writes are rolled back.
func reconcileClientsTransactional(cfg store.Config, clients []string, commandName string, dryRun bool) ([]core.ReconcileResult, error) {
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
			if !dryRun {
				rollbackResults(results, reconciler)
			}
			return results, fmt.Errorf("reconcile %q: %w", client, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// applyClientMutations applies a per-client mutation and persists only successful client changes in memory.
func applyClientMutations(
	base store.Config,
	clients []string,
	commandName string,
	dryRun bool,
	mutate func(*store.Config, string) error,
) (store.Config, []core.ReconcileResult, map[string]error, error) {
	if len(clients) == 0 {
		return store.CloneConfig(base), nil, nil, nil
	}

	reconciler, err := core.NewReconciler()
	if err != nil {
		return store.Config{}, nil, nil, err
	}
	workspace, _ := os.Getwd()

	current := store.CloneConfig(base)
	results := make([]core.ReconcileResult, 0, len(clients))
	failures := map[string]error{}

	for _, client := range clients {
		candidate := store.CloneConfig(current)
		if err := mutate(&candidate, client); err != nil {
			failures[client] = err
			continue
		}

		desired, err := planner.DesiredStateForClient(candidate, client)
		if err != nil {
			failures[client] = err
			continue
		}

		result, err := reconciler.ReconcileClient(desired, core.ReconcileOptions{
			Client:          client,
			Workspace:       workspace,
			CommandName:     commandName,
			DryRun:          dryRun,
			BackupRetention: 20,
		})
		if err != nil {
			failures[client] = err
			continue
		}

		current = candidate
		results = append(results, result)
	}

	return current, results, failures, nil
}

func rollbackResults(results []core.ReconcileResult, reconciler *core.Reconciler) {
	if reconciler == nil || reconciler.Backups == nil {
		return
	}
	for i := len(results) - 1; i >= 0; i-- {
		res := results[i]
		if res.Backup == "" {
			continue
		}
		_, _ = reconciler.Backups.Rollback(res.Client, res.Backup)
	}
}
