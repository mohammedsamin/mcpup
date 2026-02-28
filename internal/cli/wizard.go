package cli

import (
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

	for {
		action, err := w.mainMenu()
		if err != nil {
			return err
		}

		var runErr error
		switch action {
		case 0: // Add Server
			runErr = w.addServer()
		case 1: // Enable / Disable
			runErr = w.enableDisable()
		case 2: // List Servers
			runErr = w.listServers()
		case 3: // Status
			runErr = w.showStatus()
		case 4: // Profiles
			runErr = w.profileMenu()
		case 5: // Doctor
			runErr = w.runDoctor()
		case 6: // Rollback
			runErr = w.rollback()
		case 7: // Exit
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
		"Add a server",
		"Enable / Disable a server",
		"List servers",
		"Status overview",
		"Profiles",
		"Run doctor",
		"Rollback a client",
		"Exit",
	})
}

// ─── Add Server ──────────────────────────────────────────────────────────────

func (w *wizard) addServer() error {
	fmt.Fprintf(w.out, "\n%s\n", output.Bold("Add a new MCP server"))

	name, err := output.Input(w.in, w.out, "Server name:", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	command, err := output.Input(w.in, w.out, "Command (e.g. npx, uvx, docker):", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("command cannot be empty")
	}

	argsStr, err := output.Input(w.in, w.out, "Arguments (space-separated, or empty):", "")
	if err != nil {
		return err
	}
	var args []string
	if strings.TrimSpace(argsStr) != "" {
		args = strings.Fields(argsStr)
	}

	// Env vars loop.
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

	description, err := output.Input(w.in, w.out, "Description (optional):", "")
	if err != nil {
		return err
	}

	// Save to config.
	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	server := store.Server{
		Command:     strings.TrimSpace(command),
		Args:        args,
		Env:         envMap,
		Description: strings.TrimSpace(description),
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

	// Offer to enable on clients.
	enableNow, err := output.Confirm(w.in, w.out, "Enable on clients now?", true)
	if err != nil {
		return err
	}
	if enableNow {
		return w.enableServerOnClients(name)
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

	actionIdx, err := output.Select(w.in, w.out, "Action:", []string{"Enable", "Disable"})
	if err != nil {
		return err
	}
	enable := actionIdx == 0
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

	reconciler, err := core.NewReconciler()
	if err != nil {
		return err
	}
	workspace, _ := os.Getwd()

	successes := 0
	for _, ci := range clientIndices {
		client := store.SupportedClients[ci]

		if err := store.SetClientServerEnabled(&cfg, client, serverName, enable); err != nil {
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
			CommandName:     action,
			BackupRetention: 20,
		})
		if err != nil {
			fmt.Fprintf(w.out, "  %s %s: %v\n", output.Red(output.SymbolErr), client, err)
			continue
		}

		fmt.Fprintf(w.out, "  %s %sd %s on %s\n",
			output.Green(output.SymbolOK), action, output.Bold(serverName), output.Cyan(client))
		successes++
	}

	if successes > 0 {
		if err := store.SaveConfig(path, cfg); err != nil {
			return err
		}
	}

	return nil
}

// enableServerOnClients is the quick-enable flow after adding a server.
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
	tbl := &output.Table{Headers: []string{"SERVER", "COMMAND", "DESCRIPTION"}}
	for _, name := range serverNames {
		srv := cfg.Servers[name]
		desc := srv.Description
		if desc == "" {
			desc = output.Dim("-")
		}
		cmd := srv.Command
		if len(srv.Args) > 0 {
			cmd += " " + strings.Join(srv.Args, " ")
		}
		tbl.AddRow(name, cmd, desc)
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
		ri := len(backups) - 1 - i // reverse order
		label := fmt.Sprintf("%s  %s", b.Timestamp, output.Dim(b.Command))
		options[ri] = label
	}

	backupIdx, err := output.Select(w.in, w.out, "Select a backup to restore:", options)
	if err != nil {
		return err
	}
	// Convert reversed index back.
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

	// Sync mcpup config to match restored state.
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
		return store.NewDefaultConfig(), nil
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
		// Shorten client names for the table.
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
		}
		headers = append(headers, short)
	}
	return headers
}
