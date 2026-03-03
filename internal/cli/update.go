package cli

import (
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/core"
	"github.com/mohammedsamin/mcpup/internal/output"
	"github.com/mohammedsamin/mcpup/internal/planner"
	"github.com/mohammedsamin/mcpup/internal/registry"
	"github.com/mohammedsamin/mcpup/internal/store"
)

type updateCandidate struct {
	Name            string `json:"name"`
	FromCommand     string `json:"fromCommand"`
	ToCommand       string `json:"toCommand"`
	FromArgs        []string `json:"fromArgs"`
	ToArgs          []string `json:"toArgs"`
	FromURL         string `json:"fromURL,omitempty"`
	ToURL           string `json:"toURL,omitempty"`
	FromDescription string `json:"fromDescription"`
	ToDescription   string `json:"toDescription"`
}

func runUpdate(opts GlobalOptions, args []string, in *os.File, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup update [server ...] [--yes]")
		return nil
	}

	fs := newFlagSet("update")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: update: %v", errUsage, err)
	}

	selected := normalizeList(fs.Args())
	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	if len(cfg.Servers) == 0 {
		return printResult(out, opts, output.Result{
			Command: "update",
			Status:  "ok",
			Message: "no servers configured",
			Data: map[string]any{
				"updated": 0,
				"dryRun":  opts.DryRun,
			},
		})
	}

	targets := map[string]struct{}{}
	for _, name := range selected {
		if _, exists := cfg.Servers[name]; !exists {
			return fmt.Errorf("server %q not found", name)
		}
		targets[name] = struct{}{}
	}

	serverNames := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		if len(targets) > 0 {
			if _, ok := targets[name]; !ok {
				continue
			}
		}
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	candidates := make([]updateCandidate, 0, len(serverNames))
	for _, name := range serverNames {
		current := cfg.Servers[name]
		tmpl, ok := registry.Lookup(name)
		if !ok {
			continue
		}

		if current.Command == tmpl.Command &&
			slices.Equal(current.Args, tmpl.Args) &&
			current.URL == tmpl.URL &&
			strings.TrimSpace(current.Description) == strings.TrimSpace(tmpl.Description) {
			continue
		}

		candidates = append(candidates, updateCandidate{
			Name:            name,
			FromCommand:     current.Command,
			ToCommand:       tmpl.Command,
			FromArgs:        append([]string{}, current.Args...),
			ToArgs:          append([]string{}, tmpl.Args...),
			FromURL:         current.URL,
			ToURL:           tmpl.URL,
			FromDescription: current.Description,
			ToDescription:   tmpl.Description,
		})
	}

	if len(candidates) == 0 {
		return printResult(out, opts, output.Result{
			Command: "update",
			Status:  "ok",
			Message: "all registry-backed servers are up to date",
			Data: map[string]any{
				"updated": 0,
				"dryRun":  opts.DryRun,
			},
		})
	}

	if !opts.DryRun && !opts.Yes {
		if in != nil && output.IsTTY() {
			confirmed, confirmErr := output.Confirm(in, out,
				fmt.Sprintf("Apply %d update(s) from the server registry?", len(candidates)), true)
			if confirmErr != nil {
				return confirmErr
			}
			if !confirmed {
				return fmt.Errorf("aborted")
			}
		} else {
			return fmt.Errorf("update will modify server definitions; add --yes to confirm")
		}
	}

	for _, c := range candidates {
		srv := cfg.Servers[c.Name]
		srv.Command = c.ToCommand
		srv.Args = append([]string{}, c.ToArgs...)
		srv.URL = c.ToURL
		srv.Description = c.ToDescription
		cfg.Servers[c.Name] = srv
	}

	// Persist canonical desired state before reconciliation to avoid drift if
	// a later client reconcile fails after partially applying changes.
	if !opts.DryRun {
		if err := store.SaveConfig(path, cfg); err != nil {
			return err
		}
	}

	updatedNames := map[string]struct{}{}
	for _, c := range candidates {
		updatedNames[c.Name] = struct{}{}
	}

	reconciler, err := core.NewReconciler()
	if err != nil {
		return err
	}
	workspace, _ := os.Getwd()

	clientResults := []core.ReconcileResult{}
	for _, client := range store.SupportedClients {
		if !clientReferencesAnyUpdatedServer(cfg, client, updatedNames) {
			continue
		}

		desired, desiredErr := planner.DesiredStateForClient(cfg, client)
		if desiredErr != nil {
			return desiredErr
		}
		result, recErr := reconciler.ReconcileClient(desired, core.ReconcileOptions{
			Client:          client,
			Workspace:       workspace,
			CommandName:     "update",
			DryRun:          opts.DryRun,
			BackupRetention: 20,
		})
		if recErr != nil {
			return recErr
		}
		clientResults = append(clientResults, result)
	}

	message := fmt.Sprintf("%d server definition(s) %s", len(candidates), ternary(opts.DryRun, "would be updated", "updated"))
	return printResult(out, opts, output.Result{
		Command: "update",
		Status:  "ok",
		Message: message,
		Data: map[string]any{
			"updated":       len(candidates),
			"servers":       candidates,
			"clientResults": clientResults,
			"dryRun":        opts.DryRun,
		},
	})
}

func clientReferencesAnyUpdatedServer(cfg store.Config, client string, updatedNames map[string]struct{}) bool {
	clientCfg, ok := cfg.Clients[client]
	if !ok {
		return false
	}
	for serverName := range clientCfg.Servers {
		if _, ok := updatedNames[serverName]; ok {
			return true
		}
	}
	return false
}
