package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
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

// wizard holds state for the interactive menu session.
type wizard struct {
	in  *os.File
	out io.Writer
}

// runWizard is the main interactive loop.
func runWizard(in *os.File, out io.Writer) error {
	w := &wizard{in: in, out: out}
	w.printBanner()

	// First-run: offer to init + import if no config exists.
	if err := w.maybeInit(); err != nil {
		return err
	}

	for {
		action, err := w.mainMenu()
		if err != nil {
			return err
		}

		var runErr error
		switch action {
		case 0: // Quick Setup
			runErr = w.quickSetup()
		case 1: // Add Server
			runErr = w.addServer()
		case 2: // Remove Server
			runErr = w.removeServer()
		case 3: // Enable / Disable
			runErr = w.enableDisable()
		case 4: // List Servers
			runErr = w.listServers()
		case 5: // Browse Registry
			runErr = w.browseRegistry()
		case 6: // Status
			runErr = w.showStatus()
		case 7: // Profiles
			runErr = w.profileMenu()
		case 8: // Doctor
			runErr = w.runDoctor()
		case 9: // Rollback
			runErr = w.rollback()
		case 10: // Exit
			fmt.Fprintf(out, "\n%s Goodbye!\n", output.Dim(output.SymbolArrow))
			return nil
		}

		if runErr != nil {
			fmt.Fprintf(out, "\n%s %s\n", output.Red(output.SymbolErr), runErr.Error())
		}
		fmt.Fprintln(out)
	}
}

func (w *wizard) printBanner() {
	fmt.Fprintln(w.out)
	fmt.Fprintf(w.out, "  %s  %s\n",
		output.Bold(output.Cyan("mcpup")),
		output.Dim("MCP configuration manager"))
	fmt.Fprintln(w.out)
}

func (w *wizard) mainMenu() (int, error) {
	return output.Select(w.in, w.out, "What would you like to do?", []string{
		"Quick setup (recommended)",
		"Add a server",
		"Remove a server",
		"Enable / Disable a server",
		"List servers",
		"Browse server registry",
		"Status overview",
		"Profiles",
		"Run doctor",
		"Rollback a client",
		"Exit",
	})
}

func (w *wizard) quickSetup() error {
	return runSetup(GlobalOptions{}, nil, w.in, w.out)
}

// ─── Init ────────────────────────────────────────────────────────────────────

func (w *wizard) maybeInit() error {
	path, err := store.ResolveConfigPath("")
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(path); statErr == nil {
		return nil // config exists, skip init
	}

	fmt.Fprintf(w.out, "  %s No config found. Let's set things up.\n\n", output.Yellow(output.SymbolWarn))

	importExisting, err := output.Confirm(w.in, w.out, "Import servers from your existing AI clients?", true)
	if err != nil {
		return err
	}

	_, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	if importExisting {
		reconciler, recErr := core.NewReconciler()
		if recErr != nil {
			return recErr
		}
		workspace, _ := os.Getwd()
		clients, servers, importErr := importClientStates(&cfg, reconciler, workspace)
		if importErr != nil {
			return importErr
		}
		if err := store.SaveConfig(path, cfg); err != nil {
			return err
		}
		if servers > 0 {
			fmt.Fprintf(w.out, "\n  %s Imported %d servers from %d clients\n\n",
				output.Green(output.SymbolOK), servers, clients)
		} else {
			fmt.Fprintf(w.out, "\n  %s No existing servers found — you can add them below\n\n",
				output.Dim(output.SymbolArrow))
		}
	} else {
		fmt.Fprintf(w.out, "\n  %s Config initialized at %s\n\n",
			output.Green(output.SymbolOK), output.Dim(path))
	}
	return nil
}

// ─── Add Server ──────────────────────────────────────────────────────────────

func (w *wizard) addServer() error {
	fmt.Fprintf(w.out, "\n%s\n", output.Bold("Add a new MCP server"))

	modeIdx, err := output.Select(w.in, w.out, "How would you like to add?", []string{
		"Pick from registry",
		"Custom server",
	})
	if err != nil {
		return err
	}

	if modeIdx == 0 {
		return w.addFromRegistry()
	}
	return w.addCustomServer()
}

func (w *wizard) addFromRegistry() error {
	templates := registry.All()
	options := make([]string, len(templates))
	for i, t := range templates {
		options[i] = fmt.Sprintf("%-18s %s", t.Name, output.Dim(t.Description))
	}

	idx, err := output.Select(w.in, w.out, "Select a server:", options)
	if err != nil {
		return err
	}
	tmpl := templates[idx]

	// Collect required env vars.
	envMap := map[string]string{}
	for _, ev := range tmpl.EnvVars {
		label := ev.Key
		if ev.Hint != "" {
			label += output.Dim("  " + ev.Hint)
		}
		if ev.Required {
			label += output.Red(" (required)")
		}
		val, err := output.Input(w.in, w.out, label+":", "")
		if err != nil {
			return err
		}
		if strings.TrimSpace(val) != "" {
			envMap[ev.Key] = strings.TrimSpace(val)
		} else if ev.Required {
			return fmt.Errorf("%s is required for %s", ev.Key, tmpl.Name)
		}
	}

	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	server := store.Server{
		Command:     tmpl.Command,
		Args:        tmpl.Args,
		Env:         envMap,
		Description: tmpl.Description,
	}

	if err := store.AddServer(&cfg, tmpl.Name, server); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			confirmed, cErr := output.Confirm(w.in, w.out, fmt.Sprintf("Server %q already exists. Overwrite?", tmpl.Name), false)
			if cErr != nil {
				return cErr
			}
			if !confirmed {
				return fmt.Errorf("aborted")
			}
			if err := store.UpsertServer(&cfg, tmpl.Name, server); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if err := store.SaveConfig(path, cfg); err != nil {
		return err
	}

	fmt.Fprintf(w.out, "\n%s Server %s added\n", output.Green(output.SymbolOK), output.Bold(tmpl.Name))

	enableNow, err := output.Confirm(w.in, w.out, "Enable on clients now?", false)
	if err != nil {
		return err
	}
	if enableNow {
		return w.enableServerOnClients(tmpl.Name)
	}
	return nil
}

func (w *wizard) addCustomServer() error {
	name, err := output.Input(w.in, w.out, "Server name:", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	transportIdx, err := output.Select(w.in, w.out, "Transport type:", []string{
		"stdio (command-based)",
		"HTTP/SSE (url-based)",
	})
	if err != nil {
		return err
	}

	var server store.Server

	if transportIdx == 1 {
		// HTTP/SSE server.
		url, urlErr := output.Input(w.in, w.out, "Server URL:", "")
		if urlErr != nil {
			return urlErr
		}
		if strings.TrimSpace(url) == "" {
			return fmt.Errorf("url cannot be empty")
		}

		headerMap := map[string]string{}
		for {
			headerStr, headerErr := output.Input(w.in, w.out, "HTTP header (Key:Value, or empty to skip):", "")
			if headerErr != nil {
				return headerErr
			}
			if strings.TrimSpace(headerStr) == "" {
				break
			}
			key, value, ok := strings.Cut(headerStr, ":")
			if !ok || strings.TrimSpace(key) == "" {
				fmt.Fprintf(w.out, "  %s expected Key:Value format, try again\n", output.Yellow(output.SymbolWarn))
				continue
			}
			headerMap[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}

		description, descErr := output.Input(w.in, w.out, "Description (optional):", "")
		if descErr != nil {
			return descErr
		}

		server = store.Server{
			URL:         strings.TrimSpace(url),
			Headers:     headerMap,
			Description: strings.TrimSpace(description),
		}
	} else {
		// stdio server.
		command, cmdErr := output.Input(w.in, w.out, "Command (e.g. npx, uvx, docker):", "")
		if cmdErr != nil {
			return cmdErr
		}
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("command cannot be empty")
		}

		argsStr, argsErr := output.Input(w.in, w.out, "Arguments (space-separated, or empty):", "")
		if argsErr != nil {
			return argsErr
		}
		var args []string
		if strings.TrimSpace(argsStr) != "" {
			args = strings.Fields(argsStr)
		}

		envMap := map[string]string{}
		for {
			envStr, envErr := output.Input(w.in, w.out, "Environment variable (KEY=VALUE, or empty to skip):", "")
			if envErr != nil {
				return envErr
			}
			if strings.TrimSpace(envStr) == "" {
				break
			}
			key, value, ok := strings.Cut(envStr, "=")
			if !ok || strings.TrimSpace(key) == "" {
				fmt.Fprintf(w.out, "  %s expected KEY=VALUE format, try again\n", output.Yellow(output.SymbolWarn))
				continue
			}
			envMap[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}

		description, descErr := output.Input(w.in, w.out, "Description (optional):", "")
		if descErr != nil {
			return descErr
		}

		server = store.Server{
			Command:     strings.TrimSpace(command),
			Args:        args,
			Env:         envMap,
			Description: strings.TrimSpace(description),
		}
	}

	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	if err := store.AddServer(&cfg, name, server); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			confirmed, cErr := output.Confirm(w.in, w.out, fmt.Sprintf("Server %q already exists. Overwrite?", name), false)
			if cErr != nil {
				return cErr
			}
			if !confirmed {
				return fmt.Errorf("aborted")
			}
			if err := store.UpsertServer(&cfg, name, server); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if err := store.SaveConfig(path, cfg); err != nil {
		return err
	}

	fmt.Fprintf(w.out, "\n%s Server %s added\n", output.Green(output.SymbolOK), output.Bold(name))

	enableNow, err := output.Confirm(w.in, w.out, "Enable on clients now?", false)
	if err != nil {
		return err
	}
	if enableNow {
		return w.enableServerOnClients(name)
	}
	return nil
}

// ─── Remove Server ───────────────────────────────────────────────────────────

func (w *wizard) removeServer() error {
	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	serverNames := sortedServerNames(cfg)
	if len(serverNames) == 0 {
		return fmt.Errorf("no servers to remove")
	}

	fmt.Fprintf(w.out, "\n%s\n", output.Bold("Remove a server"))

	idx, err := output.Select(w.in, w.out, "Select a server to remove:", serverNames)
	if err != nil {
		return err
	}
	name := serverNames[idx]
	affectedClients := clientsReferencingServer(cfg, name)

	confirmed, err := output.Confirm(w.in, w.out, fmt.Sprintf("Remove %q? This cannot be undone.", name), false)
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("aborted")
	}

	if err := store.RemoveServer(&cfg, name); err != nil {
		return err
	}
	if err := store.SaveConfig(path, cfg); err != nil {
		return err
	}
	results, err := reconcileClients(cfg, affectedClients, "remove", false)
	if err != nil {
		return err
	}

	fmt.Fprintf(w.out, "\n%s Server %s removed\n", output.Green(output.SymbolOK), output.Bold(name))
	if len(results) > 0 {
		fmt.Fprintf(w.out, "  %s Synced removal to %d client(s)\n", output.Green(output.SymbolOK), len(results))
	}
	return nil
}

// ─── Enable / Disable ───────────────────────────────────────────────────────

func (w *wizard) enableDisable() error {
	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	serverNames := sortedServerNames(cfg)
	if len(serverNames) == 0 {
		return fmt.Errorf("no servers configured — add one first")
	}

	fmt.Fprintf(w.out, "\n%s\n", output.Bold("Enable / Disable a server"))

	serverIdx, err := output.Select(w.in, w.out, "Select a server:", serverNames)
	if err != nil {
		return err
	}
	serverName := serverNames[serverIdx]

	actionIdx, err := output.Select(w.in, w.out, "Action:", []string{
		"Enable server",
		"Disable server",
		"Enable tools",
		"Disable tools",
	})
	if err != nil {
		return err
	}
	toolMode := actionIdx >= 2
	enable := actionIdx == 0 || actionIdx == 2
	action := "enable"
	if !enable {
		action = "disable"
	}

	clientIndices, err := output.MultiSelect(w.in, w.out, "Select clients:", store.SupportedClients, nil)
	if err != nil {
		return err
	}
	if len(clientIndices) == 0 {
		return fmt.Errorf("no clients selected")
	}

	var selectedTools []string
	if toolMode {
		toolOptions := wizardToolOptions(cfg, serverName, clientIndices)
		if len(toolOptions) > 0 {
			selected, selErr := output.MultiSelect(w.in, w.out, "Select tools:", toolOptions, nil)
			if selErr != nil {
				return selErr
			}
			for _, idx := range selected {
				selectedTools = append(selectedTools, toolOptions[idx])
			}
		} else {
			raw, inputErr := output.Input(w.in, w.out, "Tool names (comma-separated):", "")
			if inputErr != nil {
				return inputErr
			}
			selectedTools = splitCSV(raw)
		}
		if len(selectedTools) == 0 {
			return fmt.Errorf("no tools selected")
		}
	}

	reconciler, err := core.NewReconciler()
	if err != nil {
		return err
	}
	workspace, _ := os.Getwd()

	successes := 0
	for _, ci := range clientIndices {
		client := store.SupportedClients[ci]

		if toolMode {
			toolErr := false
			for _, tool := range selectedTools {
				if err := store.SetClientToolEnabled(&cfg, client, serverName, tool, enable); err != nil {
					fmt.Fprintf(w.out, "  %s %s: %v\n", output.Red(output.SymbolErr), client, err)
					toolErr = true
					break
				}
			}
			if toolErr {
				continue
			}
		} else {
			if err := store.SetClientServerEnabled(&cfg, client, serverName, enable); err != nil {
				fmt.Fprintf(w.out, "  %s %s: %v\n", output.Red(output.SymbolErr), client, err)
				continue
			}
		}

		desired, err := planner.DesiredStateForClient(cfg, client)
		if err != nil {
			fmt.Fprintf(w.out, "  %s %s: %v\n", output.Red(output.SymbolErr), client, err)
			continue
		}

		_, err = reconciler.ReconcileClient(desired, core.ReconcileOptions{
			Client:          client,
			Workspace:       workspace,
			CommandName:     action,
			BackupRetention: 20,
		})
		if err != nil {
			fmt.Fprintf(w.out, "  %s %s: %v\n", output.Red(output.SymbolErr), client, err)
			continue
		}

		if toolMode {
			fmt.Fprintf(w.out, "  %s %sd tools (%s) on %s (%s)\n",
				output.Green(output.SymbolOK), action, strings.Join(selectedTools, ", "),
				output.Bold(serverName), output.Cyan(client))
		} else {
			fmt.Fprintf(w.out, "  %s %sd %s on %s\n",
				output.Green(output.SymbolOK), action, output.Bold(serverName), output.Cyan(client))
		}
		successes++
	}

	if successes > 0 {
		if err := store.SaveConfig(path, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (w *wizard) enableServerOnClients(serverName string) error {
	clientIndices, err := output.MultiSelect(w.in, w.out, "Select clients to enable on:", store.SupportedClients, nil)
	if err != nil {
		return err
	}
	if len(clientIndices) == 0 {
		return nil
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

	for _, ci := range clientIndices {
		client := store.SupportedClients[ci]

		if err := store.SetClientServerEnabled(&cfg, client, serverName, true); err != nil {
			fmt.Fprintf(w.out, "  %s %s: %v\n", output.Red(output.SymbolErr), client, err)
			continue
		}

		desired, err := planner.DesiredStateForClient(cfg, client)
		if err != nil {
			fmt.Fprintf(w.out, "  %s %s: %v\n", output.Red(output.SymbolErr), client, err)
			continue
		}

		_, err = reconciler.ReconcileClient(desired, core.ReconcileOptions{
			Client:          client,
			Workspace:       workspace,
			CommandName:     "enable",
			BackupRetention: 20,
		})
		if err != nil {
			fmt.Fprintf(w.out, "  %s %s: %v\n", output.Red(output.SymbolErr), client, err)
			continue
		}

		fmt.Fprintf(w.out, "  %s enabled on %s\n", output.Green(output.SymbolOK), output.Cyan(client))
	}

	if err := store.SaveConfig(path, cfg); err != nil {
		return err
	}
	return nil
}

// ─── List Servers ────────────────────────────────────────────────────────────

func (w *wizard) listServers() error {
	cfg, err := loadConfigOrDefault()
	if err != nil {
		return err
	}

	serverNames := sortedServerNames(cfg)
	if len(serverNames) == 0 {
		fmt.Fprintf(w.out, "\n%s No servers configured. Use %s to add one.\n",
			output.Dim(output.SymbolArrow), output.Bold("Add a server"))
		return nil
	}

	fmt.Fprintln(w.out)
	tbl := &output.Table{Headers: []string{"SERVER", "COMMAND/URL", "DESCRIPTION"}}
	for _, name := range serverNames {
		srv := cfg.Servers[name]
		desc := srv.Description
		if desc == "" {
			desc = output.Dim("-")
		}
		var target string
		if srv.IsHTTP() {
			target = srv.URL
		} else {
			target = srv.Command
			if len(srv.Args) > 0 {
				target += " " + strings.Join(srv.Args, " ")
			}
		}
		tbl.AddRow(name, target, desc)
	}
	tbl.Render(w.out)

	// Show per-client enable status.
	fmt.Fprintln(w.out)
	tbl2 := &output.Table{Headers: clientTableHeaders()}
	for _, name := range serverNames {
		row := []string{name}
		for _, client := range store.SupportedClients {
			enabled := false
			if state, ok := cfg.Clients[client].Servers[name]; ok {
				enabled = state.Enabled
			}
			row = append(row, output.EnabledSymbol(enabled))
		}
		tbl2.AddRow(row...)
	}
	tbl2.Render(w.out)

	return nil
}

// ─── Browse Registry ─────────────────────────────────────────────────────────

func (w *wizard) browseRegistry() error {
	fmt.Fprintf(w.out, "\n%s\n", output.Bold("Server Registry"))

	categories := registry.Categories()
	options := append([]string{"All servers"}, categories...)

	catIdx, err := output.Select(w.in, w.out, "Browse by category:", options)
	if err != nil {
		return err
	}

	var templates []registry.Template
	if catIdx == 0 {
		templates = registry.All()
	} else {
		templates = registry.ByCategory(categories[catIdx-1])
	}

	if len(templates) == 0 {
		fmt.Fprintf(w.out, "\n%s No servers in this category\n", output.Dim(output.SymbolArrow))
		return nil
	}

	fmt.Fprintln(w.out)
	tbl := &output.Table{Headers: []string{"NAME", "CATEGORY", "DESCRIPTION"}}
	for _, t := range templates {
		tbl.AddRow(t.Name, t.Category, t.Description)
	}
	tbl.Render(w.out)

	addOne, err := output.Confirm(w.in, w.out, "Add one of these servers?", true)
	if err != nil {
		return err
	}
	if !addOne {
		return nil
	}

	return w.addFromRegistry()
}

// ─── Status ──────────────────────────────────────────────────────────────────

func (w *wizard) showStatus() error {
	cfg, err := loadConfigOrDefault()
	if err != nil {
		return err
	}

	fmt.Fprintln(w.out)
	profileName := cfg.ActiveProfile
	if profileName == "" {
		profileName = output.Dim("(none)")
	}
	fmt.Fprintf(w.out, "  %s  %s\n", output.Bold("Profile:"), profileName)
	fmt.Fprintf(w.out, "  %s  %d servers, %d profiles\n\n",
		output.Bold("Config:"), len(cfg.Servers), len(cfg.Profiles))

	tbl := &output.Table{Headers: []string{"CLIENT", "SERVERS", "ENABLED"}}
	for _, client := range store.SupportedClients {
		state := cfg.Clients[client]
		enabledCount := 0
		for _, ss := range state.Servers {
			if ss.Enabled {
				enabledCount++
			}
		}
		tbl.AddRow(client, fmt.Sprintf("%d", len(state.Servers)), fmt.Sprintf("%d", enabledCount))
	}
	tbl.Render(w.out)
	return nil
}

// ─── Profiles ────────────────────────────────────────────────────────────────

func (w *wizard) profileMenu() error {
	for {
		fmt.Fprintf(w.out, "\n%s\n", output.Bold("Profiles"))

		idx, err := output.Select(w.in, w.out, "Profile action:", []string{
			"Create a profile",
			"Apply a profile",
			"List profiles",
			"Delete a profile",
			"Back to main menu",
		})
		if err != nil {
			return err
		}

		var runErr error
		switch idx {
		case 0:
			runErr = w.profileCreate()
		case 1:
			runErr = w.profileApply()
		case 2:
			runErr = w.profileList()
		case 3:
			runErr = w.profileDelete()
		case 4:
			return nil
		}

		if runErr != nil {
			fmt.Fprintf(w.out, "\n%s %s\n", output.Red(output.SymbolErr), runErr.Error())
		}
	}
}

func (w *wizard) profileCreate() error {
	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	serverNames := sortedServerNames(cfg)
	if len(serverNames) == 0 {
		return fmt.Errorf("no servers configured — add servers first")
	}

	name, err := output.Input(w.in, w.out, "Profile name:", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	selected, err := output.MultiSelect(w.in, w.out, "Select servers for this profile:", serverNames, nil)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return fmt.Errorf("no servers selected")
	}

	var servers []string
	for _, i := range selected {
		servers = append(servers, serverNames[i])
	}

	if err := profile.Create(&cfg, name, servers); err != nil {
		return err
	}

	if err := store.SaveConfig(path, cfg); err != nil {
		return err
	}

	fmt.Fprintf(w.out, "\n%s Profile %s created with %d servers\n",
		output.Green(output.SymbolOK), output.Bold(name), len(servers))
	return nil
}

func (w *wizard) profileApply() error {
	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	profiles := profile.List(cfg)
	if len(profiles) == 0 {
		return fmt.Errorf("no profiles configured — create one first")
	}

	profileNames := make([]string, len(profiles))
	for i, p := range profiles {
		label := p.Name
		if p.Active {
			label += output.Dim(" (active)")
		}
		label += output.Dim(fmt.Sprintf(" — %s", strings.Join(p.ServerIDs, ", ")))
		profileNames[i] = label
	}

	idx, err := output.Select(w.in, w.out, "Select a profile to apply:", profileNames)
	if err != nil {
		return err
	}
	selectedProfile := profiles[idx].Name

	confirmed, err := output.Confirm(w.in, w.out,
		fmt.Sprintf("Apply profile %q to all clients?", selectedProfile), true)
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("aborted")
	}

	reconciler, err := core.NewReconciler()
	if err != nil {
		return err
	}
	workspace, _ := os.Getwd()

	result, err := profile.Apply(&cfg, selectedProfile, reconciler, profile.ApplyOptions{
		Workspace:       workspace,
		BackupRetention: 20,
	})
	if err != nil {
		return err
	}

	if err := store.SaveConfig(path, cfg); err != nil {
		return err
	}

	changes := 0
	for _, r := range result.Results {
		changes += len(r.Plan.Changes)
	}
	fmt.Fprintf(w.out, "\n%s Profile %s applied (%d clients, %d changes)\n",
		output.Green(output.SymbolOK), output.Bold(selectedProfile),
		len(result.Results), changes)
	return nil
}

func (w *wizard) profileList() error {
	cfg, err := loadConfigOrDefault()
	if err != nil {
		return err
	}

	profiles := profile.List(cfg)
	if len(profiles) == 0 {
		fmt.Fprintf(w.out, "\n%s No profiles configured.\n", output.Dim(output.SymbolArrow))
		return nil
	}

	fmt.Fprintln(w.out)
	tbl := &output.Table{Headers: []string{"NAME", "ACTIVE", "SERVERS"}}
	for _, p := range profiles {
		active := ""
		if p.Active {
			active = output.Green(output.SymbolOK)
		}
		tbl.AddRow(p.Name, active, strings.Join(p.ServerIDs, ", "))
	}
	tbl.Render(w.out)
	return nil
}

func (w *wizard) profileDelete() error {
	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	profiles := profile.List(cfg)
	if len(profiles) == 0 {
		return fmt.Errorf("no profiles to delete")
	}

	profileNames := make([]string, len(profiles))
	for i, p := range profiles {
		profileNames[i] = p.Name
	}

	idx, err := output.Select(w.in, w.out, "Select a profile to delete:", profileNames)
	if err != nil {
		return err
	}
	name := profileNames[idx]

	confirmed, err := output.Confirm(w.in, w.out, fmt.Sprintf("Delete profile %q?", name), false)
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("aborted")
	}

	if err := profile.Delete(&cfg, name); err != nil {
		return err
	}
	if err := store.SaveConfig(path, cfg); err != nil {
		return err
	}

	fmt.Fprintf(w.out, "\n%s Profile %s deleted\n", output.Green(output.SymbolOK), output.Bold(name))
	return nil
}

// ─── Doctor ──────────────────────────────────────────────────────────────────

func (w *wizard) runDoctor() error {
	fmt.Fprintf(w.out, "\n%s\n", output.Bold("Running diagnostics..."))

	workspace, _ := os.Getwd()
	report, err := validate.RunDoctor("", workspace)
	if err != nil {
		return err
	}

	fmt.Fprintln(w.out)
	for _, check := range report.Checks {
		sym := doctorSymbol(check.Status)
		fmt.Fprintf(w.out, "  %s %s\n", sym, check.Message)
		if check.Suggestion != "" && check.Status != validate.StatusPass {
			fmt.Fprintf(w.out, "    %s %s\n", output.Dim(output.SymbolArrow), output.Dim(check.Suggestion))
		}
	}

	fmt.Fprintln(w.out)
	if report.HasFailures() {
		fmt.Fprintf(w.out, "  %s some checks failed\n", output.Red(output.SymbolErr))
	} else if report.HasWarnings() {
		fmt.Fprintf(w.out, "  %s all checks passed with warnings\n", output.Yellow(output.SymbolWarn))
	} else {
		fmt.Fprintf(w.out, "  %s all checks passed\n", output.Green(output.SymbolOK))
	}
	return nil
}

// ─── Rollback ────────────────────────────────────────────────────────────────

func (w *wizard) rollback() error {
	fmt.Fprintf(w.out, "\n%s\n", output.Bold("Rollback a client"))

	clientIdx, err := output.Select(w.in, w.out, "Select a client:", store.SupportedClients)
	if err != nil {
		return err
	}
	client := store.SupportedClients[clientIdx]

	manager, err := backup.NewManager()
	if err != nil {
		return err
	}

	backups, err := manager.ListBackups(client)
	if err != nil {
		return err
	}
	if len(backups) == 0 {
		return fmt.Errorf("no backups found for %s", client)
	}

	// Show most recent first.
	options := make([]string, len(backups))
	for i, b := range backups {
		ri := len(backups) - 1 - i
		label := fmt.Sprintf("%s  %s", b.Timestamp, output.Dim(b.Command))
		options[ri] = label
	}

	backupIdx, err := output.Select(w.in, w.out, "Select a backup to restore:", options)
	if err != nil {
		return err
	}
	actualIdx := len(backups) - 1 - backupIdx
	selectedBackup := backups[actualIdx]

	confirmed, err := output.Confirm(w.in, w.out,
		fmt.Sprintf("Restore %s from %s?", client, selectedBackup.Timestamp), false)
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("aborted")
	}

	meta, err := manager.Rollback(client, selectedBackup.Timestamp)
	if err != nil {
		return err
	}

	w.syncAfterRollback(client, meta)

	fmt.Fprintf(w.out, "\n%s Restored %s from backup %s\n",
		output.Green(output.SymbolOK), output.Cyan(client), output.Dim(meta.Timestamp))
	return nil
}

func (w *wizard) syncAfterRollback(client string, meta backup.Metadata) {
	reconciler, err := core.NewReconciler()
	if err != nil {
		return
	}
	adapter, err := reconciler.Registry.Get(client)
	if err != nil {
		return
	}
	restored, err := adapter.Read(meta.SourcePath)
	if err != nil {
		return
	}
	cfgPath, cfg, err := store.EnsureConfig("")
	if err != nil {
		return
	}

	clientCfg := cfg.Clients[client]
	if clientCfg.Servers == nil {
		clientCfg.Servers = map[string]store.ServerState{}
	}
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
	cfg.Clients[client] = clientCfg
	_ = store.SaveConfig(cfgPath, cfg)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func loadConfigOrDefault() (store.Config, error) {
	path, err := store.ResolveConfigPath("")
	if err != nil {
		return store.Config{}, err
	}
	cfg, err := store.LoadConfig(path)
	if err != nil {
		var serr *store.StoreError
		if errors.As(err, &serr) && serr.Kind == store.KindNotFound {
			return store.NewDefaultConfig(), nil
		}
		return store.Config{}, err
	}
	return cfg, nil
}

func sortedServerNames(cfg store.Config) []string {
	names := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func clientTableHeaders() []string {
	headers := []string{"SERVER"}
	for _, c := range store.SupportedClients {
		short := c
		switch c {
		case "claude-code":
			short = "CODE"
		case "cursor":
			short = "CURSOR"
		case "claude-desktop":
			short = "DESKTOP"
		case "codex":
			short = "CODEX"
		case "opencode":
			short = "OPENCODE"
		case "windsurf":
			short = "WINDSURF"
		case "zed":
			short = "ZED"
		case "continue":
			short = "CONTINUE"
		}
		headers = append(headers, short)
	}
	return headers
}

func wizardToolOptions(cfg store.Config, serverName string, clientIndices []int) []string {
	seen := map[string]struct{}{}

	if tmpl, ok := registry.Lookup(serverName); ok {
		for _, tool := range tmpl.Tools {
			tool = strings.TrimSpace(tool)
			if tool != "" {
				seen[tool] = struct{}{}
			}
		}
	}

	for _, ci := range clientIndices {
		if ci < 0 || ci >= len(store.SupportedClients) {
			continue
		}
		client := store.SupportedClients[ci]
		state, ok := cfg.Clients[client].Servers[serverName]
		if !ok {
			continue
		}
		for _, tool := range state.EnabledTools {
			tool = strings.TrimSpace(tool)
			if tool != "" {
				seen[tool] = struct{}{}
			}
		}
		for _, tool := range state.DisabledTools {
			tool = strings.TrimSpace(tool)
			if tool != "" {
				seen[tool] = struct{}{}
			}
		}
	}

	options := make([]string, 0, len(seen))
	for tool := range seen {
		options = append(options, tool)
	}
	sort.Strings(options)
	return options
}
