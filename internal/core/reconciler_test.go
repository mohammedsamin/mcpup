package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"mcpup/internal/adapters"
	"mcpup/internal/backup"
	"mcpup/internal/planner"
)

type fakeAdapter struct {
	name            string
	path            string
	current         planner.ClientState
	failWrite       bool
	failValidate    bool
	writeInvoked    bool
	validateInvoked bool
}

func (f *fakeAdapter) Name() string                                  { return f.name }
func (f *fakeAdapter) Detect(workspace string) (string, error)       { return f.path, nil }
func (f *fakeAdapter) Read(path string) (planner.ClientState, error) { return f.current, nil }
func (f *fakeAdapter) Apply(current planner.ClientState, desired planner.ClientState) (planner.Plan, error) {
	return planner.Diff(current, desired), nil
}
func (f *fakeAdapter) Write(path string, desired planner.ClientState) error {
	f.writeInvoked = true
	if f.failWrite {
		return errors.New("write failed")
	}
	f.current = desired
	return os.WriteFile(path, []byte(`{"ok":true}`), 0o644)
}
func (f *fakeAdapter) Validate(path string) error {
	f.validateInvoked = true
	if f.failValidate {
		return errors.New("validate failed")
	}
	return nil
}

func TestReconcileDryRun(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cursor.json")
	if err := os.WriteFile(path, []byte(`{"base":1}`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	reg := adapters.NewRegistry()
	adapter := &fakeAdapter{
		name: "cursor",
		path: path,
		current: planner.ClientState{
			Client:  "cursor",
			Servers: map[string]planner.ServerState{},
		},
	}
	reg.Register(adapter)
	rec := &Reconciler{
		Registry: reg,
		Backups:  &backup.Manager{RootDir: filepath.Join(tmp, "backups")},
	}

	desired := planner.ClientState{
		Client: "cursor",
		Servers: map[string]planner.ServerState{
			"github": {Enabled: true},
		},
	}

	result, err := rec.ReconcileClient(desired, ReconcileOptions{
		Client:      "cursor",
		CommandName: "enable",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("dry-run reconcile failed: %v", err)
	}
	if !result.Plan.HasChanges() {
		t.Fatalf("expected changes in dry-run plan")
	}
	if adapter.writeInvoked {
		t.Fatalf("dry-run should not call write")
	}
}

func TestReconcileValidationFailureRestoresBackup(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cursor.json")
	original := []byte(`{"before":"state"}`)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	reg := adapters.NewRegistry()
	adapter := &fakeAdapter{
		name: "cursor",
		path: path,
		current: planner.ClientState{
			Client:  "cursor",
			Servers: map[string]planner.ServerState{},
		},
		failValidate: true,
	}
	reg.Register(adapter)

	rec := &Reconciler{
		Registry: reg,
		Backups:  &backup.Manager{RootDir: filepath.Join(tmp, "backups")},
	}

	desired := planner.ClientState{
		Client: "cursor",
		Servers: map[string]planner.ServerState{
			"github": {Enabled: true},
		},
	}

	result, err := rec.ReconcileClient(desired, ReconcileOptions{
		Client:      "cursor",
		CommandName: "enable",
		DryRun:      false,
	})
	if err == nil {
		t.Fatalf("expected reconcile error")
	}

	var recErr *ReconcileError
	if !errors.As(err, &recErr) {
		t.Fatalf("expected ReconcileError, got %T", err)
	}
	if recErr.Code != ExitCodePartialRecovered {
		t.Fatalf("expected partial-recovered code %d, got %d", ExitCodePartialRecovered, recErr.Code)
	}
	if !result.Restored {
		t.Fatalf("expected restored=true")
	}

	current, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read restored file: %v", readErr)
	}
	if string(current) != string(original) {
		t.Fatalf("expected source to be restored")
	}
}

func TestReconcileWriteFailureRestoresBackup(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cursor.json")
	original := []byte(`{"before":"state"}`)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	reg := adapters.NewRegistry()
	adapter := &fakeAdapter{
		name: "cursor",
		path: path,
		current: planner.ClientState{
			Client:  "cursor",
			Servers: map[string]planner.ServerState{},
		},
		failWrite: true,
	}
	reg.Register(adapter)

	rec := &Reconciler{
		Registry: reg,
		Backups:  &backup.Manager{RootDir: filepath.Join(tmp, "backups")},
	}

	desired := planner.ClientState{
		Client: "cursor",
		Servers: map[string]planner.ServerState{
			"github": {Enabled: true},
		},
	}

	result, err := rec.ReconcileClient(desired, ReconcileOptions{
		Client:      "cursor",
		CommandName: "enable",
		DryRun:      false,
	})
	if err == nil {
		t.Fatalf("expected reconcile error")
	}

	var recErr *ReconcileError
	if !errors.As(err, &recErr) {
		t.Fatalf("expected ReconcileError, got %T", err)
	}
	if recErr.Code != ExitCodePartialRecovered {
		t.Fatalf("expected partial recovered code, got %d", recErr.Code)
	}
	if !result.Restored {
		t.Fatalf("expected restored=true")
	}

	current, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read restored file: %v", readErr)
	}
	if string(current) != string(original) {
		t.Fatalf("expected file content restored after write failure")
	}
}

func TestReconcileFromCurrentStateConflict(t *testing.T) {
	cfgServers := map[string]struct{}{
		"github": {},
	}
	desired := planner.ClientState{
		Client: "cursor",
		Servers: map[string]planner.ServerState{
			"github": {Enabled: true},
		},
	}

	if err := ReconcileFromCurrentState(cfgServers, desired, "missing"); err == nil {
		t.Fatalf("expected missing server conflict")
	}
}
