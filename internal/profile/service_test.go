package profile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"mcpup/internal/adapters"
	"mcpup/internal/backup"
	"mcpup/internal/core"
	"mcpup/internal/planner"
	"mcpup/internal/store"
)

type fakeAdapter struct {
	name         string
	path         string
	current      planner.ClientState
	failWrite    bool
	failValidate bool
}

func (f *fakeAdapter) Name() string                                  { return f.name }
func (f *fakeAdapter) Detect(workspace string) (string, error)       { return f.path, nil }
func (f *fakeAdapter) Read(path string) (planner.ClientState, error) { return f.current, nil }
func (f *fakeAdapter) Apply(current planner.ClientState, desired planner.ClientState) (planner.Plan, error) {
	return planner.Diff(current, desired), nil
}
func (f *fakeAdapter) Write(path string, desired planner.ClientState) error {
	if f.failWrite {
		return errors.New("write failed")
	}
	f.current = desired
	return os.WriteFile(path, []byte(`{"ok":true}`), 0o644)
}
func (f *fakeAdapter) Validate(path string) error {
	if f.failValidate {
		return errors.New("validate failed")
	}
	return nil
}

func TestCreateValidatesServers(t *testing.T) {
	cfg := store.NewDefaultConfig()
	cfg.Servers["github"] = store.Server{Command: "echo github"}

	if err := Create(&cfg, "coding", []string{"github"}); err != nil {
		t.Fatalf("create should pass: %v", err)
	}
	if err := Create(&cfg, "broken", []string{"missing"}); err == nil {
		t.Fatalf("create should fail for unknown server")
	}
}

func TestApplyDryRun(t *testing.T) {
	cfg := store.NewDefaultConfig()
	cfg.Servers["github"] = store.Server{Command: "echo github"}
	cfg.Profiles["coding"] = store.Profile{Servers: []string{"github"}}

	reconciler := buildFakeReconciler(t, "")
	result, err := Apply(&cfg, "coding", reconciler, ApplyOptions{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("apply dry-run failed: %v", err)
	}
	if len(result.Results) != len(store.SupportedClients) {
		t.Fatalf("expected one result per supported client")
	}
}

func TestApplyRollbackOnPartialFailure(t *testing.T) {
	cfg := store.NewDefaultConfig()
	cfg.Servers["github"] = store.Server{Command: "echo github"}
	cfg.Profiles["coding"] = store.Profile{Servers: []string{"github"}}

	reconciler := buildFakeReconciler(t, "codex")
	result, err := Apply(&cfg, "coding", reconciler, ApplyOptions{
		DryRun: false,
	})
	if err == nil {
		t.Fatalf("expected apply error")
	}
	if len(result.RolledBack) == 0 {
		t.Fatalf("expected rollback list to be populated")
	}
}

func buildFakeReconciler(t *testing.T, failClient string) *core.Reconciler {
	t.Helper()
	tmp := t.TempDir()
	reg := adapters.NewRegistry()

	for _, client := range store.SupportedClients {
		path := filepath.Join(tmp, client+".json")
		if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
			t.Fatalf("write fake config: %v", err)
		}

		reg.Register(&fakeAdapter{
			name:         client,
			path:         path,
			current:      planner.ClientState{Client: client, Servers: map[string]planner.ServerState{}},
			failWrite:    client == failClient,
			failValidate: false,
		})
	}

	return &core.Reconciler{
		Registry: reg,
		Backups:  &backup.Manager{RootDir: filepath.Join(tmp, "backups")},
	}
}
