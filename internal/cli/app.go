package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/backup"
	"github.com/mohammedsamin/mcpup/internal/core"
	"github.com/mohammedsamin/mcpup/internal/output"
	"github.com/mohammedsamin/mcpup/internal/planner"
	"github.com/mohammedsamin/mcpup/internal/profile"
	"github.com/mohammedsamin/mcpup/internal/registry"
	"github.com/mohammedsamin/mcpup/internal/store"
	"github.com/mohammedsamin/mcpup/internal/validate"
)

var errUsage = errors.New("usage error")

// GlobalOptions are root flags accepted by mcpup.
type GlobalOptions struct {
	DryRun  bool `json:"dry_run"`
	JSON    bool `json:"json"`
	Verbose bool `json:"verbose"`
	Yes     bool `json:"yes"`
}

// Run executes the CLI command tree.
// in may be nil when running non-interactively (tests, piped input).
func Run(args []string, in *os.File, out io.Writer, errOut io.Writer) error {
	opts, remaining, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}

	if len(remaining) == 0 {
		// Interactive terminal with no args: launch wizard.
		if in != nil && output.IsTTY() {
			return runWizard(in, out)
		}
		printRootHelp(out)
		return nil
	}
	if remaining[0] == "help" {
		printRootHelp(out)
		return nil
	}

	command := remaining[0]
	commandArgs := stripGlobalFlags(&opts, remaining[1:])

	switch command {
	case "init":
		return runInit(opts, commandArgs, out)
	case "add":
		return runAdd(opts, commandArgs, out)
	case "remove":
		return runRemove(opts, commandArgs, in, out)
	case "enable":
		return runToggle(opts, "enable", commandArgs, in, out)
	case "disable":
		return runToggle(opts, "disable", commandArgs, in, out)
	case "list":
		return runList(opts, commandArgs, out)
	case "status":
		return runStatus(opts, commandArgs, out)
	case "profile":
		return runProfile(opts, commandArgs, in, out)
	case "clients":
		return runClients(opts, commandArgs, out)
	case "doctor":
		return runDoctor(opts, commandArgs, out)
	case "rollback":
		return runRollback(opts, commandArgs, out)
	case "registry":
		return runRegistry(opts, commandArgs, out)
	default:
		if suggestion := suggestCommand(command); suggestion != "" {
			return fmt.Errorf("%w: unknown command %q — did you mean %q?", errUsage, command, suggestion)
		}
		return fmt.Errorf("%w: unknown command %q", errUsage, command)
	}
}

func parseGlobalFlags(args []string) (GlobalOptions, []string, error) {
	var opts GlobalOptions
	remaining := args

	for len(remaining) > 0 {
		current := remaining[0]
		if current == "--" {
			remaining = remaining[1:]
			break
		}
		if !strings.HasPrefix(current, "-") {
			break
		}

		switch current {
		case "-h", "--help":
			return opts, []string{"help"}, nil
		case "--dry-run":
			opts.DryRun = true
		case "--json":
			opts.JSON = true
		case "--verbose":
			opts.Verbose = true
		case "--yes":
			opts.Yes = true
		default:
			return GlobalOptions{}, nil, fmt.Errorf("%w: unknown global flag %q", errUsage, current)
		}

		remaining = remaining[1:]
	}

	return opts, remaining, nil
}

// stripGlobalFlags removes global flags from command args and merges them into opts.
// This allows users to pass --dry-run, --json, --verbose, --yes after the command name.
func stripGlobalFlags(opts *GlobalOptions, args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--dry-run":
			opts.DryRun = true
		case "--json":
			opts.JSON = true
		case "--verbose":
			opts.Verbose = true
		case "--yes":
			opts.Yes = true
		default:
			out = append(out, arg)
		}
	}
	return out
}

func runInit(opts GlobalOptions, args []string, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup init [--import]")
		return nil
	}

	fs := newFlagSet("init")
	importExisting := fs.Bool("import", false, "import existing client server states")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: init: %v", errUsage, err)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: init does not accept positional arguments", errUsage)
	}

	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	importedClients := 0
	importedServers := 0
	if *importExisting {
		reconciler, recErr := core.NewReconciler()
		if recErr != nil {
			return recErr
		}
		workspace, _ := os.Getwd()
		clients, servers, importErr := importClientStates(&cfg, reconciler, workspace)
		if importErr != nil {
			return importErr
		}
		importedClients = clients
		importedServers = servers
		if !opts.DryRun {
			if err := store.SaveConfig(path, cfg); err != nil {
				return err
			}
		}
	}

	return printResult(out, opts, output.Result{
		Command: "init",
		Status:  "ok",
		Message: "canonical config is ready",
		Data: map[string]any{
			"path":            path,
			"importedClients": importedClients,
			"importedServers": importedServers,
			"dryRun":          opts.DryRun,
		},
	})
}

func runAdd(opts GlobalOptions, args []string, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup add <name> --command <cmd> [--arg <value>]... [--env KEY=VALUE]... [--description <text>]")
		return nil
	}

	fs := newFlagSet("add")
	command := fs.String("command", "", "server command")
	description := fs.String("description", "", "server description")
	update := fs.Bool("update", false, "update server if it already exists")
	var cmdArgs repeatedFlag
	var envVars repeatedFlag
	fs.Var(&cmdArgs, "arg", "server argument (repeatable)")
	fs.Var(&envVars, "env", "environment variable KEY=VALUE (repeatable)")

	normalized, err := normalizeArgs(args, map[string]bool{
		"--command":     true,
		"--arg":         true,
		"--env":         true,
		"--description": true,
		"--update":      false,
	})
	if err != nil {
		return fmt.Errorf("%w: add: %v", errUsage, err)
	}
	if err := fs.Parse(normalized); err != nil {
		return fmt.Errorf("%w: add: %v", errUsage, err)
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: add requires exactly one positional argument: <name>", errUsage)
	}
	// When --command is not provided, check the built-in registry.
	if strings.TrimSpace(*command) == "" {
		tmpl, found := registry.Lookup(fs.Arg(0))
		if !found {
			return fmt.Errorf("%w: add requires --command (server %q is not in the built-in registry)", errUsage, fs.Arg(0))
		}
		*command = tmpl.Command
		if len(cmdArgs) == 0 {
			cmdArgs = repeatedFlag(tmpl.Args)
		}
		if strings.TrimSpace(*description) == "" {
			*description = tmpl.Description
		}
		// Validate that required env vars are provided.
		for _, ev := range tmpl.EnvVars {
			if ev.Required {
				found := false
				for _, kv := range envVars {
					key, _, _ := strings.Cut(kv, "=")
					if strings.TrimSpace(key) == ev.Key {
						found = true
						break
					}
				}
				if !found {
					hint := ""
					if ev.Hint != "" {
						hint = fmt.Sprintf(" (get it from %s)", ev.Hint)
					}
					return fmt.Errorf("%w: server %q requires --%s %s=<value>%s", errUsage, fs.Arg(0), "env", ev.Key, hint)
				}
			}
		}
	}

	envMap := map[string]string{}
	for _, kv := range envVars {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return fmt.Errorf("%w: invalid --env value %q, expected KEY=VALUE", errUsage, kv)
		}
		envMap[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	server := store.Server{
		Command:     strings.TrimSpace(*command),
		Args:        normalizeList([]string(cmdArgs)),
		Env:         envMap,
		Description: strings.TrimSpace(*description),
	}
	existed := false
	if _, ok := cfg.Servers[fs.Arg(0)]; ok {
		existed = true
	}
	if *update {
		if err := store.UpsertServer(&cfg, fs.Arg(0), server); err != nil {
			return err
		}
	} else {
		if err := store.AddServer(&cfg, fs.Arg(0), server); err != nil {
			if errors.Is(err, store.ErrAlreadyExists) {
				return fmt.Errorf("server %q already exists; use --update to overwrite", fs.Arg(0))
			}
			return err
		}
	}

	if !opts.DryRun {
		if err := store.SaveConfig(path, cfg); err != nil {
			return err
		}
	}

	return printResult(out, opts, output.Result{
		Command: "add",
		Status:  "ok",
		Message: fmt.Sprintf("server %q %s", fs.Arg(0), ternary(existed, "updated", "added")),
		Data: map[string]any{
			"name":        fs.Arg(0),
			"command":     server.Command,
			"args":        server.Args,
			"envCount":    len(server.Env),
			"description": server.Description,
			"updated":     existed,
			"dryRun":      opts.DryRun,
		},
	})
}

func runRemove(opts GlobalOptions, args []string, in *os.File, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup remove <name> [--yes]")
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("%w: remove requires exactly one positional argument: <name>", errUsage)
	}

	// Interactive confirmation unless --yes.
	if !opts.Yes && in != nil && output.IsTTY() {
		confirmed, err := output.Confirm(in, out, fmt.Sprintf("Remove server %q? This cannot be undone.", args[0]), false)
		if err != nil {
			return err
		}
		if !confirmed {
			return fmt.Errorf("aborted")
		}
	}

	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	if err := store.RemoveServer(&cfg, args[0]); err != nil {
		return err
	}

	if !opts.DryRun {
		if err := store.SaveConfig(path, cfg); err != nil {
			return err
		}
	}

	return printResult(out, opts, output.Result{
		Command: "remove",
		Status:  "ok",
		Message: fmt.Sprintf("server %q removed", args[0]),
		Data: map[string]any{
			"name":   args[0],
			"dryRun": opts.DryRun,
		},
	})
}

func runToggle(opts GlobalOptions, action string, args []string, in *os.File, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintf(out, "Usage: mcpup %s <name> --client <client> [--tool <tool>]...\n", action)
		return nil
	}

	fs := newFlagSet(action)
	client := fs.String("client", "", "target client")
	var tools repeatedFlag
	fs.Var(&tools, "tool", "tool name for per-tool control (repeatable)")

	normalized, err := normalizeArgs(args, map[string]bool{
		"--client": true,
		"--tool":   true,
	})
	if err != nil {
		return fmt.Errorf("%w: %s: %v", errUsage, action, err)
	}
	if err := fs.Parse(normalized); err != nil {
		return fmt.Errorf("%w: %s: %v", errUsage, action, err)
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: %s requires exactly one positional argument: <name>", errUsage, action)
	}

	// If --client not provided, show interactive selector.
	if strings.TrimSpace(*client) == "" {
		if in != nil && output.IsTTY() {
			idx, selectErr := output.Select(in, out, "Select a client:", store.SupportedClients)
			if selectErr != nil {
				return fmt.Errorf("%w: %s requires --client", errUsage, action)
			}
			*client = store.SupportedClients[idx]
		} else {
			return fmt.Errorf("%w: %s requires --client", errUsage, action)
		}
	}

	serverName := fs.Arg(0)
	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	if _, ok := cfg.Servers[serverName]; !ok {
		return fmt.Errorf("server %q not found", serverName)
	}

	toolList := normalizeList([]string(tools))
	enableAction := action == "enable"
	if len(toolList) > 0 {
		for _, tool := range toolList {
			if err := store.SetClientToolEnabled(&cfg, *client, serverName, tool, enableAction); err != nil {
				return err
			}
		}
	} else {
		if err := store.SetClientServerEnabled(&cfg, *client, serverName, enableAction); err != nil {
			return err
		}
	}

	desired, err := planner.DesiredStateForClient(cfg, *client)
	if err != nil {
		return err
	}

	serverSet := map[string]struct{}{}
	for name := range cfg.Servers {
		serverSet[name] = struct{}{}
	}
	if err := core.ReconcileFromCurrentState(serverSet, desired, serverName); err != nil {
		return err
	}

	reconciler, err := core.NewReconciler()
	if err != nil {
		return err
	}
	workspace, _ := os.Getwd()
	reconcileResult, err := reconciler.ReconcileClient(desired, core.ReconcileOptions{
		Client:          *client,
		Workspace:       workspace,
		CommandName:     action,
		DryRun:          opts.DryRun,
		BackupRetention: 20,
	})
	if err != nil {
		return err
	}

	if !opts.DryRun {
		if err := store.SaveConfig(path, cfg); err != nil {
			return err
		}
	}

	var msg string
	if opts.DryRun {
		msg = fmt.Sprintf("dry-run: would %s server %q on client %q", action, serverName, *client)
	} else {
		msg = fmt.Sprintf("%s completed for server %q on client %q", action, serverName, *client)
	}
	data := map[string]any{
		"name":        serverName,
		"client":      *client,
		"tools":       toolList,
		"changed":     reconcileResult.Changed,
		"changeCount": len(reconcileResult.Plan.Changes),
		"summary":     reconcileResult.Summary,
		"backup":      reconcileResult.Backup,
		"dryRun":      opts.DryRun,
	}
	return printResult(out, opts, output.Result{
		Command: action,
		Status:  "ok",
		Message: msg,
		Data:    data,
	})
}

func runList(opts GlobalOptions, args []string, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup list [--client <client>]")
		return nil
	}

	fs := newFlagSet("list")
	client := fs.String("client", "", "optional client filter")
	normalized, err := normalizeArgs(args, map[string]bool{
		"--client": true,
	})
	if err != nil {
		return fmt.Errorf("%w: list: %v", errUsage, err)
	}
	if err := fs.Parse(normalized); err != nil {
		return fmt.Errorf("%w: list: %v", errUsage, err)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: list does not accept positional arguments", errUsage)
	}

	path, err := store.ResolveConfigPath("")
	if err != nil {
		return err
	}
	cfg, err := store.LoadConfig(path)
	if err != nil {
		var serr *store.StoreError
		if errors.As(err, &serr) && serr.Kind == store.KindNotFound {
			cfg = store.NewDefaultConfig()
		} else {
			return err
		}
	}

	serverNames := make([]string, 0, len(cfg.Servers))
	for serverName := range cfg.Servers {
		serverNames = append(serverNames, serverName)
	}
	sort.Strings(serverNames)

	filter := strings.TrimSpace(*client)

	// JSON mode: use the standard result output.
	if opts.JSON {
		serversData := make([]map[string]any, 0, len(serverNames))
		for _, serverName := range serverNames {
			entry := map[string]any{
				"name":    serverName,
				"command": cfg.Servers[serverName].Command,
			}
			if filter != "" {
				if state, ok := cfg.Clients[filter].Servers[serverName]; ok {
					entry["enabled"] = state.Enabled
					entry["enabledTools"] = state.EnabledTools
					entry["disabledTools"] = state.DisabledTools
				} else {
					entry["enabled"] = false
				}
			}
			serversData = append(serversData, entry)
		}

		return printResult(out, opts, output.Result{
			Command: "list",
			Status:  "ok",
			Message: "server list loaded",
			Data: map[string]any{
				"clientFilter": filter,
				"serverCount":  len(serverNames),
				"servers":      serversData,
			},
		})
	}

	// Text mode: render a table.
	if len(serverNames) == 0 {
		fmt.Fprintf(out, "%s No servers configured. Run %s to add one.\n",
			output.Dim(output.SymbolArrow), output.Bold("mcpup add"))
		return nil
	}

	tbl := &output.Table{}
	if filter != "" {
		tbl.Headers = []string{"NAME", "COMMAND", "ENABLED"}
	} else {
		tbl.Headers = []string{"NAME", "COMMAND"}
	}

	for _, serverName := range serverNames {
		cmd := cfg.Servers[serverName].Command
		if filter != "" {
			enabled := false
			if state, ok := cfg.Clients[filter].Servers[serverName]; ok {
				enabled = state.Enabled
			}
			tbl.AddRow(serverName, cmd, output.EnabledSymbol(enabled))
		} else {
			tbl.AddRow(serverName, cmd)
		}
	}

	tbl.Render(out)
	return nil
}

func runStatus(opts GlobalOptions, args []string, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup status")
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("%w: status does not accept positional arguments", errUsage)
	}

	path, err := store.ResolveConfigPath("")
	if err != nil {
		return err
	}
	cfg, err := store.LoadConfig(path)
	if err != nil {
		var serr *store.StoreError
		if errors.As(err, &serr) && serr.Kind == store.KindNotFound {
			cfg = store.NewDefaultConfig()
		} else {
			return err
		}
	}

	// JSON mode: standard result output.
	if opts.JSON {
		clientStatus := map[string]map[string]any{}
		for _, client := range store.SupportedClients {
			state := cfg.Clients[client]
			enabledCount := 0
			for _, serverState := range state.Servers {
				if serverState.Enabled {
					enabledCount++
				}
			}
			clientStatus[client] = map[string]any{
				"serverCount":  len(state.Servers),
				"enabledCount": enabledCount,
			}
		}

		return printResult(out, opts, output.Result{
			Command: "status",
			Status:  "ok",
			Message: "status loaded",
			Data: map[string]any{
				"activeProfile": cfg.ActiveProfile,
				"serverCount":   len(cfg.Servers),
				"profileCount":  len(cfg.Profiles),
				"clients":       clientStatus,
			},
		})
	}

	// Text mode: summary + client table.
	profileName := cfg.ActiveProfile
	if profileName == "" {
		profileName = output.Dim("(none)")
	}
	fmt.Fprintf(out, "%s  %s\n", output.Bold("Profile:"), profileName)
	fmt.Fprintf(out, "%s  %d servers, %d profiles\n\n", output.Bold("Config:"), len(cfg.Servers), len(cfg.Profiles))

	tbl := &output.Table{Headers: []string{"CLIENT", "SERVERS", "ENABLED"}}
	for _, client := range store.SupportedClients {
		state := cfg.Clients[client]
		enabledCount := 0
		for _, serverState := range state.Servers {
			if serverState.Enabled {
				enabledCount++
			}
		}
		tbl.AddRow(client, fmt.Sprintf("%d", len(state.Servers)), fmt.Sprintf("%d", enabledCount))
	}
	tbl.Render(out)
	return nil
}

func runProfile(opts GlobalOptions, args []string, in *os.File, out io.Writer) error {
	if len(args) == 0 || hasHelp(args) {
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintln(out, "  mcpup profile create <name> --servers a,b,c")
		fmt.Fprintln(out, "  mcpup profile apply <name>")
		fmt.Fprintln(out, "  mcpup profile list")
		fmt.Fprintln(out, "  mcpup profile delete <name>")
		return nil
	}

	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "create":
		fs := newFlagSet("profile create")
		servers := fs.String("servers", "", "comma-separated server names")
		normalized, err := normalizeArgs(actionArgs, map[string]bool{
			"--servers": true,
		})
		if err != nil {
			return fmt.Errorf("%w: profile create: %v", errUsage, err)
		}
		if err := fs.Parse(normalized); err != nil {
			return fmt.Errorf("%w: profile create: %v", errUsage, err)
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("%w: profile create requires exactly one positional argument: <name>", errUsage)
		}

		path, cfg, err := store.EnsureConfig("")
		if err != nil {
			return err
		}
		if err := profile.Create(&cfg, fs.Arg(0), splitCSV(*servers)); err != nil {
			return err
		}
		if !opts.DryRun {
			if err := store.SaveConfig(path, cfg); err != nil {
				return err
			}
		}

		return printResult(out, opts, output.Result{
			Command: "profile create",
			Status:  "ok",
			Message: fmt.Sprintf("profile %q created", fs.Arg(0)),
			Data: map[string]any{
				"name":    fs.Arg(0),
				"servers": splitCSV(*servers),
				"dryRun":  opts.DryRun,
			},
		})

	case "apply":
		fs := newFlagSet("profile apply")
		var clientFlags repeatedFlag
		fs.Var(&clientFlags, "client", "apply only to this client (repeatable)")
		normalized, err := normalizeArgs(actionArgs, map[string]bool{
			"--client": true,
		})
		if err != nil {
			return fmt.Errorf("%w: profile apply: %v", errUsage, err)
		}
		if err := fs.Parse(normalized); err != nil {
			return fmt.Errorf("%w: profile apply: %v", errUsage, err)
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("%w: profile apply requires exactly one positional argument: <name>", errUsage)
		}
		clients := normalizeList([]string(clientFlags))

		// When applying to all clients, require confirmation.
		if len(clients) == 0 && !opts.Yes {
			if in != nil && output.IsTTY() {
				confirmed, confirmErr := output.Confirm(in, out,
					fmt.Sprintf("Apply profile %q to all clients?", fs.Arg(0)), false)
				if confirmErr != nil {
					return confirmErr
				}
				if !confirmed {
					return fmt.Errorf("aborted")
				}
			} else {
				return fmt.Errorf("profile apply will update all clients; add --yes to confirm or use --client to target one client")
			}
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
		result, err := profile.Apply(&cfg, fs.Arg(0), reconciler, profile.ApplyOptions{
			DryRun:          opts.DryRun,
			Workspace:       workspace,
			BackupRetention: 20,
			Clients:         clients,
		})
		if err != nil {
			return err
		}

		if !opts.DryRun {
			if err := store.SaveConfig(path, cfg); err != nil {
				return err
			}
		}

		changes := 0
		for _, r := range result.Results {
			changes += len(r.Plan.Changes)
		}
		return printResult(out, opts, output.Result{
			Command: "profile apply",
			Status:  "ok",
			Message: fmt.Sprintf("profile %q applied", fs.Arg(0)),
			Data: map[string]any{
				"name":          fs.Arg(0),
				"dryRun":        opts.DryRun,
				"clients":       len(result.Results),
				"totalChanges":  changes,
				"rolledBack":    result.RolledBack,
				"clientResults": result.Results,
			},
		})

	case "list":
		if len(actionArgs) != 0 {
			return fmt.Errorf("%w: profile list does not accept positional arguments", errUsage)
		}
		path, err := store.ResolveConfigPath("")
		if err != nil {
			return err
		}
		cfg, err := store.LoadConfig(path)
		if err != nil {
			var serr *store.StoreError
			if errors.As(err, &serr) && serr.Kind == store.KindNotFound {
				cfg = store.NewDefaultConfig()
			} else {
				return err
			}
		}

		if opts.JSON {
			return printResult(out, opts, output.Result{
				Command: "profile list",
				Status:  "ok",
				Message: "profiles listed",
				Data: map[string]any{
					"activeProfile": cfg.ActiveProfile,
					"profiles":      profile.List(cfg),
				},
			})
		}

		profiles := profile.List(cfg)
		if len(profiles) == 0 {
			fmt.Fprintf(out, "%s No profiles configured. Run %s to create one.\n",
				output.Dim(output.SymbolArrow), output.Bold("mcpup profile create"))
			return nil
		}

		tbl := &output.Table{Headers: []string{"NAME", "ACTIVE", "SERVERS"}}
		for _, p := range profiles {
			active := ""
			if p.Active {
				active = output.Green(output.SymbolOK)
			}
			tbl.AddRow(p.Name, active, strings.Join(p.ServerIDs, ", "))
		}
		tbl.Render(out)
		return nil

	case "delete":
		if len(actionArgs) != 1 {
			return fmt.Errorf("%w: profile delete requires exactly one positional argument: <name>", errUsage)
		}
		path, cfg, err := store.EnsureConfig("")
		if err != nil {
			return err
		}
		deleted := true
		if err := profile.Delete(&cfg, actionArgs[0]); err != nil {
			if errors.Is(err, store.ErrResourceNotFound) {
				deleted = false
			} else {
				return err
			}
		}
		if !opts.DryRun {
			if err := store.SaveConfig(path, cfg); err != nil {
				return err
			}
		}
		return printResult(out, opts, output.Result{
			Command: "profile delete",
			Status:  "ok",
			Message: fmt.Sprintf("profile %q %s", actionArgs[0], ternary(deleted, "deleted", "already absent")),
			Data: map[string]any{
				"name":    actionArgs[0],
				"deleted": deleted,
				"dryRun":  opts.DryRun,
			},
		})

	default:
		return fmt.Errorf("%w: unknown profile subcommand %q", errUsage, action)
	}
}

func runClients(opts GlobalOptions, args []string, out io.Writer) error {
	if len(args) == 0 || hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup clients list")
		return nil
	}
	if args[0] != "list" {
		return fmt.Errorf("%w: unknown clients subcommand %q", errUsage, args[0])
	}
	if len(args) != 1 {
		return fmt.Errorf("%w: clients list does not accept positional arguments", errUsage)
	}

	return printResult(out, opts, output.Result{
		Command: "clients list",
		Status:  "ok",
		Message: "supported clients",
		Data: map[string]any{
			"supported": store.SupportedClients,
		},
	})
}

func runDoctor(opts GlobalOptions, args []string, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup doctor")
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("%w: doctor does not accept positional arguments", errUsage)
	}

	workspace, _ := os.Getwd()
	report, err := validate.RunDoctor("", workspace)
	if err != nil {
		return err
	}

	// JSON mode: standard result output.
	if opts.JSON {
		status := "ok"
		message := "all checks passed"
		if report.HasFailures() {
			status = "err"
			message = "some checks failed"
		} else if report.HasWarnings() {
			status = "warn"
			message = "all checks passed with warnings"
		}

		if err := printResult(out, opts, output.Result{
			Command: "doctor",
			Status:  status,
			Message: message,
			Data: map[string]any{
				"checks": report.Checks,
			},
		}); err != nil {
			return err
		}

		if report.HasFailures() {
			return &core.ReconcileError{
				Code: core.ExitCodeValidation,
				Err:  fmt.Errorf("doctor detected failure checks"),
			}
		}
		return nil
	}

	// Text mode: render each check with ✓/✗/⊘.
	for _, check := range report.Checks {
		sym := doctorSymbol(check.Status)
		fmt.Fprintf(out, "%s %s\n", sym, check.Message)
		if check.Suggestion != "" && check.Status != validate.StatusPass {
			fmt.Fprintf(out, "  %s %s\n", output.Dim(output.SymbolArrow), output.Dim(check.Suggestion))
		}
	}

	fmt.Fprintln(out)
	if report.HasFailures() {
		fmt.Fprintf(out, "%s some checks failed\n", output.StatusSymbol("err"))
		return &core.ReconcileError{
			Code: core.ExitCodeValidation,
			Err:  fmt.Errorf("doctor detected failure checks"),
		}
	}
	if report.HasWarnings() {
		fmt.Fprintf(out, "%s all checks passed with warnings\n", output.StatusSymbol("warn"))
		return nil
	}
	fmt.Fprintf(out, "%s all checks passed\n", output.StatusSymbol("ok"))
	return nil
}

func doctorSymbol(status validate.CheckStatus) string {
	switch status {
	case validate.StatusPass:
		return output.StatusSymbol("ok")
	case validate.StatusWarn:
		return output.StatusSymbol("warn")
	default:
		return output.StatusSymbol("err")
	}
}

func runRollback(opts GlobalOptions, args []string, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup rollback --client <client> [--to <timestamp>]")
		return nil
	}

	fs := newFlagSet("rollback")
	client := fs.String("client", "", "target client")
	target := fs.String("to", "", "backup timestamp to restore")
	normalized, err := normalizeArgs(args, map[string]bool{
		"--client": true,
		"--to":     true,
	})
	if err != nil {
		return fmt.Errorf("%w: rollback: %v", errUsage, err)
	}
	if err := fs.Parse(normalized); err != nil {
		return fmt.Errorf("%w: rollback: %v", errUsage, err)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: rollback does not accept positional arguments", errUsage)
	}
	if strings.TrimSpace(*client) == "" {
		return fmt.Errorf("%w: rollback requires --client", errUsage)
	}

	manager, err := backup.NewManager()
	if err != nil {
		return err
	}

	meta, err := manager.Rollback(*client, strings.TrimSpace(*target))
	if err != nil {
		return err
	}

	// Sync mcpup's config to match the restored client file.
	reconciler, recErr := core.NewReconciler()
	if recErr == nil {
		if adapter, adErr := reconciler.Registry.Get(*client); adErr == nil {
			if restored, readErr := adapter.Read(meta.SourcePath); readErr == nil {
				cfgPath, cfg, ensureErr := store.EnsureConfig("")
				if ensureErr == nil {
					clientCfg := cfg.Clients[*client]
					if clientCfg.Servers == nil {
						clientCfg.Servers = map[string]store.ServerState{}
					}
					// Reset this client's state to match the restored file.
					for name := range clientCfg.Servers {
						delete(clientCfg.Servers, name)
					}
					for name, srv := range restored.Servers {
						clientCfg.Servers[name] = store.ServerState{
							Enabled:       srv.Enabled,
							EnabledTools:  srv.EnabledTools,
							DisabledTools: srv.DisabledTools,
						}
					}
					cfg.Clients[*client] = clientCfg
					_ = store.SaveConfig(cfgPath, cfg)
				}
			}
		}
	}

	return printResult(out, opts, output.Result{
		Command: "rollback",
		Status:  "ok",
		Message: fmt.Sprintf("rollback restored client %q from backup %s", *client, meta.Timestamp),
		Data: map[string]any{
			"client":    *client,
			"timestamp": meta.Timestamp,
			"source":    meta.SourcePath,
		},
	})
}

func runRegistry(opts GlobalOptions, args []string, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup registry [search-term]")
		return nil
	}

	query := ""
	if len(args) > 0 {
		query = strings.Join(args, " ")
	}

	templates := registry.Search(query)

	if opts.JSON {
		return printResult(out, opts, output.Result{
			Command: "registry",
			Status:  "ok",
			Message: fmt.Sprintf("%d servers found", len(templates)),
			Data: map[string]any{
				"query":   query,
				"count":   len(templates),
				"servers": templates,
			},
		})
	}

	if len(templates) == 0 {
		fmt.Fprintf(out, "%s No servers found matching %q\n", output.Dim(output.SymbolArrow), query)
		return nil
	}

	tbl := &output.Table{Headers: []string{"NAME", "CATEGORY", "DESCRIPTION"}}
	for _, t := range templates {
		tbl.AddRow(t.Name, t.Category, t.Description)
	}
	tbl.Render(out)
	fmt.Fprintf(out, "\n%s Add with: %s\n", output.Dim(output.SymbolArrow), output.Bold("mcpup add <name>"))
	return nil
}

func printRootHelp(out io.Writer) {
	fmt.Fprintf(out, "%s - MCP configuration manager\n", output.Bold("mcpup"))
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", output.Bold("Usage:"))
	fmt.Fprintf(out, "  mcpup [--dry-run] [--json] [--verbose] [--yes] <command> [args]\n")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", output.Bold("Commands:"))
	helpLine(out, "init", "Initialize canonical config")
	helpLine(out, "add", "Add an MCP server definition")
	helpLine(out, "remove", "Remove a server definition")
	helpLine(out, "enable", "Enable a server on a client")
	helpLine(out, "disable", "Disable a server on a client")
	helpLine(out, "list", "List configured servers")
	helpLine(out, "status", "Show overall status")
	helpLine(out, "profile", "Manage profiles (create|apply|list|delete)")
	helpLine(out, "clients", "List supported clients")
	helpLine(out, "doctor", "Run diagnostics")
	helpLine(out, "rollback", "Restore from backup")
	helpLine(out, "registry", "Browse built-in server catalog")
}

func helpLine(out io.Writer, cmd, desc string) {
	fmt.Fprintf(out, "  %-12s %s\n", output.Bold(cmd), output.Dim(desc))
}

func printResult(out io.Writer, opts GlobalOptions, result output.Result) error {
	return output.Print(out, output.Options{
		JSON:    opts.JSON,
		Verbose: opts.Verbose,
		DryRun:  opts.DryRun,
	}, result)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func hasHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func splitCSV(raw string) []string {
	return normalizeList(strings.Split(raw, ","))
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
	sort.Strings(out)
	return out
}

// normalizeArgs reorders flags before positionals for flag.FlagSet parsing.
// valueFlags maps flag name to whether it takes a value:
//
//	true  = flag takes a value (e.g. --client cursor)
//	false = boolean flag, no value consumed (e.g. --update)
//
// Any flag not present in the map causes an error.
func normalizeArgs(args []string, valueFlags map[string]bool) ([]string, error) {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		token := args[i]
		if token == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}

		if !strings.HasPrefix(token, "-") {
			positionals = append(positionals, token)
			continue
		}

		if token == "-h" || token == "--help" {
			flags = append(flags, token)
			continue
		}

		name := token
		inlineValue := false
		if idx := strings.Index(token, "="); idx > 0 {
			name = token[:idx]
			inlineValue = true
		}

		takesValue, known := valueFlags[name]
		if !known {
			return nil, fmt.Errorf("unknown flag %q", name)
		}

		flags = append(flags, token)
		if inlineValue || !takesValue {
			continue
		}

		if i+1 >= len(args) {
			return nil, fmt.Errorf("flag %q requires a value", name)
		}
		next := args[i+1]
		if strings.HasPrefix(next, "-") {
			nextName := next
			if idx := strings.Index(nextName, "="); idx > 0 {
				nextName = nextName[:idx]
			}
			if _, known := valueFlags[nextName]; known || nextName == "-h" || nextName == "--help" {
				return nil, fmt.Errorf("flag %q requires a value", name)
			}
		}
		flags = append(flags, next)
		i++
	}

	return append(flags, positionals...), nil
}

func importClientStates(cfg *store.Config, reconciler *core.Reconciler, workspace string) (int, int, error) {
	importedClients := 0
	importedServers := 0

	for _, client := range store.SupportedClients {
		adapter, err := reconciler.Registry.Get(client)
		if err != nil {
			return importedClients, importedServers, err
		}
		path, err := adapter.Detect(workspace)
		if err != nil {
			continue
		}
		state, err := adapter.Read(path)
		if err != nil {
			continue
		}
		if len(state.Servers) == 0 {
			continue
		}

		importedClients++
		for serverName, serverState := range state.Servers {
			if _, exists := cfg.Servers[serverName]; !exists {
				cfg.Servers[serverName] = store.Server{
					Command: "unknown",
				}
				importedServers++
			}
			clientState := cfg.Clients[client]
			if clientState.Servers == nil {
				clientState.Servers = map[string]store.ServerState{}
			}
			clientState.Servers[serverName] = store.ServerState{
				Enabled:       serverState.Enabled,
				EnabledTools:  append([]string{}, serverState.EnabledTools...),
				DisabledTools: append([]string{}, serverState.DisabledTools...),
			}
			cfg.Clients[client] = clientState
		}
	}

	return importedClients, importedServers, nil
}

type repeatedFlag []string

func (r *repeatedFlag) String() string {
	if r == nil {
		return ""
	}
	return strings.Join(*r, ",")
}

func (r *repeatedFlag) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func currentWorkspace() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		return wd
	}
	return abs
}

var knownCommands = []string{
	"init", "add", "remove", "enable", "disable",
	"list", "status", "profile", "clients", "doctor", "rollback", "registry",
}

// suggestCommand returns the closest known command to input, or "" if none is close enough.
func suggestCommand(input string) string {
	best := ""
	bestDist := len(input)/2 + 1 // must be within roughly half the length
	for _, cmd := range knownCommands {
		d := levenshtein(input, cmd)
		if d < bestDist {
			bestDist = d
			best = cmd
		}
	}
	return best
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = min(del, min(ins, sub))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func ternary(condition bool, yes string, no string) string {
	if condition {
		return yes
	}
	return no
}
