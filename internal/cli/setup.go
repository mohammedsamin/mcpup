package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/core"
	"github.com/mohammedsamin/mcpup/internal/output"
	"github.com/mohammedsamin/mcpup/internal/registry"
	"github.com/mohammedsamin/mcpup/internal/store"
)

func runSetup(opts GlobalOptions, args []string, in *os.File, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup setup [--client <client> ...] [--server <name> ...] [--env KEY=VALUE ...] [--update]")
		return nil
	}

	fs := newFlagSet("setup")
	var clientFlags repeatedFlag
	var serverFlags repeatedFlag
	var envFlags repeatedFlag
	updateDefs := fs.Bool("update", false, "update existing server definitions from registry")
	fs.Var(&clientFlags, "client", "target client (repeatable)")
	fs.Var(&serverFlags, "server", "registry server name (repeatable)")
	fs.Var(&envFlags, "env", "environment variable KEY=VALUE (repeatable)")

	normalized, err := normalizeArgs(args, map[string]bool{
		"--client": true,
		"--server": true,
		"--env":    true,
		"--update": false,
	})
	if err != nil {
		return fmt.Errorf("%w: setup: %v", errUsage, err)
	}
	if err := fs.Parse(normalized); err != nil {
		return fmt.Errorf("%w: setup: %v", errUsage, err)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: setup does not accept positional arguments", errUsage)
	}

	envOverrides, err := parseSetupEnvOverrides([]string(envFlags))
	if err != nil {
		return fmt.Errorf("%w: setup: %v", errUsage, err)
	}

	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	reconciler, err := core.NewReconciler()
	if err != nil {
		return err
	}
	workspace, _ := os.Getwd()
	interactive := in != nil && output.IsTTY()

	clients := normalizeList([]string(clientFlags))
	if len(clients) == 0 {
		if interactive {
			picked, pickErr := pickSetupClients(in, out, reconciler, workspace)
			if pickErr != nil {
				return pickErr
			}
			clients = picked
		} else {
			clients = append([]string{}, store.SupportedClients...)
		}
	}
	for _, client := range clients {
		if err := store.ValidateClientName(client); err != nil {
			return err
		}
	}

	serverNames := normalizeList([]string(serverFlags))
	if len(serverNames) == 0 {
		if interactive {
			picked, pickErr := pickSetupServers(in, out)
			if pickErr != nil {
				return pickErr
			}
			serverNames = picked
		} else {
			return fmt.Errorf("%w: setup requires at least one --server in non-interactive mode", errUsage)
		}
	}
	if len(serverNames) == 0 {
		return fmt.Errorf("no servers selected")
	}

	templates := make([]registry.Template, 0, len(serverNames))
	for _, name := range serverNames {
		tmpl, ok := registry.Lookup(name)
		if !ok {
			return fmt.Errorf("%w: unknown registry server %q", errUsage, name)
		}
		templates = append(templates, tmpl)
	}

	added := 0
	updated := 0
	reused := 0
	updatedNames := []string{}
	for _, tmpl := range templates {
		result, setupErr := setupServerFromTemplate(&cfg, tmpl, *updateDefs, envOverrides, interactive, in, out)
		if setupErr != nil {
			return setupErr
		}
		switch result {
		case "added":
			added++
		case "updated":
			updated++
			updatedNames = append(updatedNames, tmpl.Name)
		default:
			reused++
		}
	}

	for _, client := range clients {
		for _, tmpl := range templates {
			if err := store.SetClientServerEnabled(&cfg, client, tmpl.Name, true); err != nil {
				return err
			}
		}
	}

	affectedClients := append([]string{}, clients...)
	if len(updatedNames) > 0 {
		seen := map[string]struct{}{}
		for _, client := range affectedClients {
			seen[client] = struct{}{}
		}
		for _, name := range updatedNames {
			for _, client := range clientsReferencingServer(cfg, name) {
				if _, ok := seen[client]; ok {
					continue
				}
				seen[client] = struct{}{}
				affectedClients = append(affectedClients, client)
			}
		}
		if err := requireManagedChangeApproval(opts, in, out,
			formatChangeSummary("Overwrite managed server definitions", updatedNames, affectedClients)); err != nil {
			return err
		}
	}

	clientResults, err := reconcileClientsTransactional(cfg, affectedClients, "setup", opts.DryRun)
	if err != nil {
		return err
	}
	if !opts.DryRun {
		if err := store.SaveConfig(path, cfg); err != nil {
			return err
		}
	}

	return printResult(out, opts, output.Result{
		Command: "setup",
		Status:  "ok",
		Message: fmt.Sprintf("setup %s for %d server(s) on %d client(s)", ternary(opts.DryRun, "planned", "completed"), len(templates), len(clients)),
		Data: map[string]any{
			"clients":       clients,
			"affected":      affectedClients,
			"servers":       serverNames,
			"added":         added,
			"updated":       updated,
			"reused":        reused,
			"clientResults": clientResults,
			"dryRun":        opts.DryRun,
		},
	})
}

func parseSetupEnvOverrides(values []string) (map[string]string, error) {
	out := map[string]string{}
	for _, kv := range values {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid --env value %q, expected KEY=VALUE", kv)
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out, nil
}

func pickSetupClients(in *os.File, out io.Writer, reconciler *core.Reconciler, workspace string) ([]string, error) {
	preSelected := make([]bool, len(store.SupportedClients))
	preCount := 0
	for i, client := range store.SupportedClients {
		adapter, err := reconciler.Registry.Get(client)
		if err != nil {
			continue
		}
		path, err := adapter.Detect(workspace)
		if err != nil {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			preSelected[i] = true
			preCount++
		}
	}
	if preCount == 0 {
		preSelected = nil
	}

	indices, err := output.SearchMultiSelect(in, out, "Select clients to configure:", store.SupportedClients, preSelected)
	if err != nil {
		return nil, err
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("no clients selected")
	}

	clients := make([]string, 0, len(indices))
	for _, idx := range indices {
		clients = append(clients, store.SupportedClients[idx])
	}
	return clients, nil
}

func pickSetupServers(in *os.File, out io.Writer) ([]string, error) {
	templates := registry.All()
	options := make([]string, len(templates))
	for i, t := range templates {
		options[i] = formatSetupRegistryOption(t)
	}

	indices, err := output.SearchMultiSelect(in, out, "Select servers to install:", options, nil)
	if err != nil {
		return nil, err
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("no servers selected")
	}

	names := make([]string, 0, len(indices))
	for _, idx := range indices {
		names = append(names, templates[idx].Name)
	}
	sort.Strings(names)
	return names, nil
}

func setupServerFromTemplate(
	cfg *store.Config,
	tmpl registry.Template,
	updateDefs bool,
	envOverrides map[string]string,
	interactive bool,
	in *os.File,
	out io.Writer,
) (string, error) {
	existing, exists := cfg.Servers[tmpl.Name]
	server := existing
	if !exists || updateDefs {
		server.Command = tmpl.Command
		server.Args = append([]string{}, tmpl.Args...)
		server.URL = tmpl.URL
		if tmpl.Headers != nil {
			server.Headers = make(map[string]string, len(tmpl.Headers))
			for k, v := range tmpl.Headers {
				server.Headers[k] = v
			}
		}
		server.Transport = tmpl.Transport
		server.Description = tmpl.Description
	}
	if server.Env == nil {
		server.Env = map[string]string{}
	}

	for key, value := range envOverrides {
		server.Env[key] = value
	}

	for _, ev := range tmpl.EnvVars {
		if strings.TrimSpace(server.Env[ev.Key]) != "" {
			continue
		}

		defaultVal := strings.TrimSpace(os.Getenv(ev.Key))
		if interactive {
			label := fmt.Sprintf("%s for %s", ev.Key, tmpl.Name)
			if ev.Hint != "" {
				label += " " + output.Dim("("+ev.Hint+")")
			}
			if ev.Required {
				label += output.Red(" [required]")
			}
			value, err := output.Input(in, out, label+":", defaultVal)
			if err != nil {
				return "", err
			}
			value = strings.TrimSpace(value)
			if value == "" {
				if ev.Required {
					return "", fmt.Errorf("%s is required for %s", ev.Key, tmpl.Name)
				}
				continue
			}
			server.Env[ev.Key] = value
			continue
		}

		if defaultVal != "" {
			server.Env[ev.Key] = defaultVal
			continue
		}
		if ev.Required {
			hint := ""
			if ev.Hint != "" {
				hint = fmt.Sprintf(" (get it from %s)", ev.Hint)
			}
			return "", fmt.Errorf("%w: server %q requires --env %s=<value>%s", errUsage, tmpl.Name, ev.Key, hint)
		}
	}
	if err := validateRegistryServerDefinition(tmpl.Name, tmpl, server); err != nil {
		return "", err
	}

	if !exists {
		if err := store.AddServer(cfg, tmpl.Name, server); err != nil {
			return "", err
		}
		return "added", nil
	}
	if sameServerDefinition(existing, server) {
		return "reused", nil
	}

	if err := store.UpsertServer(cfg, tmpl.Name, server); err != nil {
		return "", err
	}
	if updateDefs {
		return "updated", nil
	}
	return "reused", nil
}
