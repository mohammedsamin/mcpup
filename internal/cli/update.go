package cli

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/mohammedsamin/mcpup/internal/output"
	"github.com/mohammedsamin/mcpup/internal/registry"
	"github.com/mohammedsamin/mcpup/internal/store"
)

type updateCandidate struct {
	Name            string            `json:"name"`
	FromCommand     string            `json:"fromCommand"`
	ToCommand       string            `json:"toCommand"`
	FromArgs        []string          `json:"fromArgs"`
	ToArgs          []string          `json:"toArgs"`
	FromURL         string            `json:"fromURL,omitempty"`
	ToURL           string            `json:"toURL,omitempty"`
	FromHeaders     map[string]string `json:"fromHeaders,omitempty"`
	ToHeaders       map[string]string `json:"toHeaders,omitempty"`
	FromTransport   string            `json:"fromTransport,omitempty"`
	ToTransport     string            `json:"toTransport,omitempty"`
	FromDescription string            `json:"fromDescription"`
	ToDescription   string            `json:"toDescription"`
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
	nextServers := map[string]store.Server{}
	for _, name := range serverNames {
		current := cfg.Servers[name]
		tmpl, ok := registry.Lookup(name)
		if !ok {
			continue
		}
		next := mergeServerWithTemplate(current, tmpl)
		if err := validateRegistryServerDefinition(name, tmpl, next); err != nil {
			return err
		}
		if sameServerDefinition(current, next) {
			continue
		}
		nextServers[name] = next

		candidates = append(candidates, updateCandidate{
			Name:            name,
			FromCommand:     current.Command,
			ToCommand:       next.Command,
			FromArgs:        append([]string{}, current.Args...),
			ToArgs:          append([]string{}, next.Args...),
			FromURL:         current.URL,
			ToURL:           next.URL,
			FromHeaders:     cloneStringMap(current.Headers),
			ToHeaders:       cloneStringMap(next.Headers),
			FromTransport:   current.Transport,
			ToTransport:     next.Transport,
			FromDescription: current.Description,
			ToDescription:   next.Description,
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

	affectedClients := []string{}
	seenClients := map[string]struct{}{}
	for _, candidate := range candidates {
		for _, client := range clientsReferencingServer(cfg, candidate.Name) {
			if _, ok := seenClients[client]; ok {
				continue
			}
			seenClients[client] = struct{}{}
			affectedClients = append(affectedClients, client)
		}
	}
	if err := requireManagedChangeApproval(opts, in, out,
		formatChangeSummary("Overwrite managed server definitions", candidateNames(candidates), affectedClients)); err != nil {
		return err
	}

	for _, c := range candidates {
		cfg.Servers[c.Name] = nextServers[c.Name]
	}

	updatedNames := map[string]struct{}{}
	for _, c := range candidates {
		updatedNames[c.Name] = struct{}{}
	}
	clientResults, err := reconcileClientsTransactional(cfg, affectedClients, "update", opts.DryRun)
	if err != nil {
		return err
	}
	if !opts.DryRun {
		if err := store.SaveConfig(path, cfg); err != nil {
			return err
		}
	}

	message := fmt.Sprintf("%d server definition(s) %s", len(candidates), ternary(opts.DryRun, "would be updated", "updated"))
	return printResult(out, opts, output.Result{
		Command: "update",
		Status:  "ok",
		Message: message,
		Data: map[string]any{
			"updated":       len(candidates),
			"servers":       candidates,
			"clients":       affectedClients,
			"clientResults": clientResults,
			"dryRun":        opts.DryRun,
		},
	})
}

func candidateNames(candidates []updateCandidate) []string {
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		names = append(names, candidate.Name)
	}
	return names
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
