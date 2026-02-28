package core

import (
	"mcpup/internal/adapters"
	"mcpup/internal/adapters/claudecode"
	"mcpup/internal/adapters/claudedesktop"
	"mcpup/internal/adapters/codex"
	"mcpup/internal/adapters/cursor"
	"mcpup/internal/adapters/opencode"
	"mcpup/internal/backup"
	"mcpup/internal/planner"
)

// ReconcileOptions controls reconciliation behavior.
type ReconcileOptions struct {
	Client          string
	Workspace       string
	CommandName     string
	DryRun          bool
	BackupRetention int
}

// ReconcileResult reports planner/reconcile execution.
type ReconcileResult struct {
	Client   string       `json:"client"`
	Path     string       `json:"path"`
	Plan     planner.Plan `json:"plan"`
	DryRun   bool         `json:"dryRun"`
	Changed  bool         `json:"changed"`
	Summary  string       `json:"summary"`
	Restored bool         `json:"restored"`
	Backup   string       `json:"backup,omitempty"`
}

// Reconciler performs adapter writes with planner + backup safety.
type Reconciler struct {
	Registry *adapters.Registry
	Backups  *backup.Manager
}

// NewReconciler constructs a reconciler with default adapters.
func NewReconciler() (*Reconciler, error) {
	registry := adapters.NewRegistry()
	registry.Register(claudecode.Adapter{})
	registry.Register(cursor.Adapter{})
	registry.Register(claudedesktop.Adapter{})
	registry.Register(codex.Adapter{})
	registry.Register(opencode.Adapter{})

	manager, err := backup.NewManager()
	if err != nil {
		return nil, err
	}

	return &Reconciler{
		Registry: registry,
		Backups:  manager,
	}, nil
}

// ReconcileClient applies desired state to one client adapter.
func (r *Reconciler) ReconcileClient(desired planner.ClientState, opts ReconcileOptions) (ReconcileResult, error) {
	if r == nil || r.Registry == nil || r.Backups == nil {
		return ReconcileResult{}, newReconcileError(ExitCodeRuntime, "reconciler is not initialized")
	}

	client := opts.Client
	if client == "" {
		client = desired.Client
	}
	if client == "" {
		return ReconcileResult{}, newReconcileError(ExitCodeRuntime, "client is required")
	}

	adapter, err := r.Registry.Get(client)
	if err != nil {
		return ReconcileResult{}, newReconcileError(ExitCodeRuntime, "%v", err)
	}

	path, err := adapter.Detect(opts.Workspace)
	if err != nil {
		return ReconcileResult{}, newReconcileError(ExitCodeRuntime, "detect %s config: %v", client, err)
	}

	current, err := adapter.Read(path)
	if err != nil {
		return ReconcileResult{}, newReconcileError(ExitCodeValidation, "read %s config: %v", client, err)
	}

	desired.Client = client
	plan, err := adapter.Apply(current, desired)
	if err != nil {
		return ReconcileResult{}, newReconcileError(ExitCodeRuntime, "apply %s plan: %v", client, err)
	}

	commandName := opts.CommandName
	if commandName == "" {
		commandName = "reconcile"
	}
	summary := planner.DryRunSummary(commandName, plan)

	result := ReconcileResult{
		Client:  client,
		Path:    path,
		Plan:    plan,
		DryRun:  opts.DryRun,
		Changed: plan.HasChanges(),
		Summary: summary,
	}

	if opts.DryRun || !plan.HasChanges() {
		return result, nil
	}

	snap, err := r.Backups.SnapshotFile(client, path, commandName)
	if err != nil {
		return ReconcileResult{}, newReconcileError(ExitCodeRuntime, "snapshot %s: %v", client, err)
	}
	result.Backup = snap.Timestamp

	if err := adapter.Write(path, desired); err != nil {
		if restoreErr := r.Backups.Restore(snap); restoreErr == nil {
			result.Restored = true
			return result, newReconcileError(ExitCodePartialRecovered, "write failed and restored from backup %s: %v", snap.Timestamp, err)
		}
		return result, newReconcileError(ExitCodeRuntime, "write %s config failed: %v", client, err)
	}

	if err := adapter.Validate(path); err != nil {
		if restoreErr := r.Backups.Restore(snap); restoreErr == nil {
			result.Restored = true
			return result, newReconcileError(ExitCodePartialRecovered, "validation failed and restored from backup %s: %v", snap.Timestamp, err)
		}
		return result, newReconcileError(ExitCodeValidation, "validate %s config failed: %v", client, err)
	}

	if opts.BackupRetention >= 0 {
		_ = r.Backups.Cleanup(client, opts.BackupRetention)
	}

	return result, nil
}

// ReconcileFromCurrentState applies the state currently requested for a client.
func ReconcileFromCurrentState(cfgServers map[string]struct{}, desired planner.ClientState, requestedServer string) error {
	if requestedServer == "" {
		return nil
	}
	if err := planner.ValidateServerReference(cfgServers, requestedServer); err != nil {
		return newReconcileError(ExitCodeValidation, "%v", err)
	}
	if _, exists := desired.Servers[requestedServer]; !exists {
		return newReconcileError(ExitCodeValidation, "server %q is not part of desired state for client %q", requestedServer, desired.Client)
	}
	return nil
}
