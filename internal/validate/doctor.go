package validate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mohammedsamin/mcpup/internal/adapters"
	"github.com/mohammedsamin/mcpup/internal/core"
	"github.com/mohammedsamin/mcpup/internal/planner"
	"github.com/mohammedsamin/mcpup/internal/registry"
	"github.com/mohammedsamin/mcpup/internal/store"
)

// CheckStatus is health result type for one diagnostic check.
type CheckStatus string

const (
	StatusPass CheckStatus = "pass"
	StatusWarn CheckStatus = "warn"
	StatusFail CheckStatus = "fail"
)

// Check is one doctor check result.
type Check struct {
	Key        string      `json:"key"`
	Status     CheckStatus `json:"status"`
	Message    string      `json:"message"`
	Suggestion string      `json:"suggestion,omitempty"`
}

// Report is a full doctor response.
type Report struct {
	Checks []Check `json:"checks"`
}

// HasFailures reports whether report contains any failure.
func (r Report) HasFailures() bool {
	for _, check := range r.Checks {
		if check.Status == StatusFail {
			return true
		}
	}
	return false
}

// HasWarnings reports whether report contains any warning.
func (r Report) HasWarnings() bool {
	for _, check := range r.Checks {
		if check.Status == StatusWarn {
			return true
		}
	}
	return false
}

// RunDoctor executes diagnostics across canonical config and client configs.
func RunDoctor(configPath string, workspace string) (Report, error) {
	resolvedPath, err := store.ResolveConfigPath(configPath)
	if err != nil {
		return Report{}, err
	}

	report := Report{Checks: []Check{}}
	cfg := store.NewDefaultConfig()
	canonicalValid := false

	if _, statErr := os.Stat(resolvedPath); statErr != nil {
		report.Checks = append(report.Checks, Check{
			Key:        "config.exists",
			Status:     StatusWarn,
			Message:    fmt.Sprintf("canonical config not found at %s", resolvedPath),
			Suggestion: "run `mcpup init` to create a canonical config file",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Key:     "config.exists",
			Status:  StatusPass,
			Message: fmt.Sprintf("canonical config exists at %s", resolvedPath),
		})
	}

	loadedCfg, loadErr := store.LoadConfig(resolvedPath)
	if loadErr != nil {
		report.Checks = append(report.Checks, Check{
			Key:        "config.schema",
			Status:     StatusFail,
			Message:    fmt.Sprintf("canonical config is invalid: %v", loadErr),
			Suggestion: "fix JSON/schema issues or restore from backup",
		})
	} else {
		cfg = loadedCfg
		canonicalValid = true
		report.Checks = append(report.Checks, Check{
			Key:     "config.schema",
			Status:  StatusPass,
			Message: "canonical config JSON/schema is valid",
		})
	}

	reconciler, recErr := core.NewReconciler()
	if recErr != nil {
		return report, recErr
	}

	for _, client := range store.SupportedClients {
		adapter, err := reconciler.Registry.Get(client)
		if err != nil {
			report.Checks = append(report.Checks, Check{
				Key:        "client.adapter." + client,
				Status:     StatusFail,
				Message:    fmt.Sprintf("adapter not available: %v", err),
				Suggestion: "build with all v1 adapters registered",
			})
			continue
		}

		path, err := adapter.Detect(workspace)
		if err != nil {
			report.Checks = append(report.Checks, Check{
				Key:        "client.path." + client,
				Status:     StatusWarn,
				Message:    fmt.Sprintf("could not detect config path: %v", err),
				Suggestion: "open the client once to generate config files",
			})
			continue
		}

		if _, err := os.Stat(path); err != nil {
			report.Checks = append(report.Checks, Check{
				Key:        "client.exists." + client,
				Status:     StatusWarn,
				Message:    fmt.Sprintf("config file not found: %s", path),
				Suggestion: "launch the client or create its config file",
			})
		} else {
			report.Checks = append(report.Checks, Check{
				Key:     "client.exists." + client,
				Status:  StatusPass,
				Message: fmt.Sprintf("config file exists: %s", path),
			})
		}

		if err := adapter.Validate(path); err != nil {
			report.Checks = append(report.Checks, Check{
				Key:        "client.parse." + client,
				Status:     StatusFail,
				Message:    fmt.Sprintf("config parse/validate failed: %v", err),
				Suggestion: "fix malformed config or restore backup",
			})
		} else {
			report.Checks = append(report.Checks, Check{
				Key:     "client.parse." + client,
				Status:  StatusPass,
				Message: "config parse/validate passed",
			})

			state, readErr := adapter.Read(path)
			if readErr != nil {
				report.Checks = append(report.Checks, Check{
					Key:        "client.read." + client,
					Status:     StatusWarn,
					Message:    fmt.Sprintf("client state read failed: %v", readErr),
					Suggestion: "inspect the client config or restore from backup",
				})
			} else {
				managedCount, unmanagedCount := ownershipCounts(state)
				report.Checks = append(report.Checks, Check{
					Key:     "client.ownership." + client,
					Status:  StatusPass,
					Message: fmt.Sprintf("managed entries: %d, unmanaged entries: %d", managedCount, unmanagedCount),
				})
				if canonicalValid {
					desired, desiredErr := planner.DesiredStateForClient(cfg, client)
					if desiredErr != nil {
						report.Checks = append(report.Checks, Check{
							Key:        "client.drift." + client,
							Status:     StatusFail,
							Message:    fmt.Sprintf("could not build desired state: %v", desiredErr),
							Suggestion: "fix canonical config references or restore from backup",
						})
					} else {
						plan := adapters.ManagedDiff(state, desired)
						if plan.HasChanges() {
							report.Checks = append(report.Checks, Check{
								Key:        "client.drift." + client,
								Status:     StatusWarn,
								Message:    fmt.Sprintf("managed client config drift detected (%d planned change(s))", len(plan.Changes)),
								Suggestion: fmt.Sprintf("run `mcpup setup --client %s` or re-apply the relevant command", client),
							})
						} else {
							report.Checks = append(report.Checks, Check{
								Key:     "client.drift." + client,
								Status:  StatusPass,
								Message: "managed client config matches canonical state",
							})
						}
					}
				}
			}
		}

		if err := checkWriteAccess(path); err != nil {
			report.Checks = append(report.Checks, Check{
				Key:        "client.write." + client,
				Status:     StatusWarn,
				Message:    fmt.Sprintf("write access check failed: %v", err),
				Suggestion: "ensure directory is writable by current user",
			})
		} else {
			report.Checks = append(report.Checks, Check{
				Key:     "client.write." + client,
				Status:  StatusPass,
				Message: "write access check passed",
			})
		}
	}

	for serverName, serverDef := range cfg.Servers {
		enabledAnywhere := serverEnabledAnywhere(cfg, serverName)
		if tmpl, ok := registry.Lookup(serverName); ok {
			missingEnv := missingRequiredEnv(serverDef, tmpl)
			if enabledAnywhere && len(missingEnv) > 0 {
				report.Checks = append(report.Checks, Check{
					Key:        "server.env." + serverName,
					Status:     StatusWarn,
					Message:    fmt.Sprintf("enabled server is missing required env vars: %s", strings.Join(missingEnv, ", ")),
					Suggestion: fmt.Sprintf("set the missing env vars with `mcpup add %s --update --env KEY=value`", serverName),
				})
			} else if enabledAnywhere && len(tmpl.EnvVars) > 0 {
				report.Checks = append(report.Checks, Check{
					Key:     "server.env." + serverName,
					Status:  StatusPass,
					Message: "required env vars are present for enabled server",
				})
			}

			if registryDefinitionChanged(serverDef, tmpl) {
				report.Checks = append(report.Checks, Check{
					Key:        "registry.definition." + serverName,
					Status:     StatusWarn,
					Message:    "server definition differs from built-in registry template",
					Suggestion: fmt.Sprintf("run `mcpup update %s --yes` to refresh the template", serverName),
				})
			} else {
				report.Checks = append(report.Checks, Check{
					Key:     "registry.definition." + serverName,
					Status:  StatusPass,
					Message: "server definition matches the built-in registry template",
				})
			}
		}

		if serverDef.IsHTTP() {
			url := strings.TrimSpace(serverDef.URL)
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				report.Checks = append(report.Checks, Check{
					Key:        "server.url." + serverName,
					Status:     StatusWarn,
					Message:    fmt.Sprintf("server URL does not start with http:// or https://: %s", url),
					Suggestion: "use a fully qualified URL starting with http:// or https://",
				})
			} else {
				report.Checks = append(report.Checks, Check{
					Key:     "server.url." + serverName,
					Status:  StatusPass,
					Message: fmt.Sprintf("server URL is valid: %s", url),
				})
			}
			continue
		}

		command := strings.TrimSpace(serverDef.Command)
		if command == "" {
			report.Checks = append(report.Checks, Check{
				Key:        "command.path." + serverName,
				Status:     StatusWarn,
				Message:    "server command is empty in canonical config",
				Suggestion: "set a valid executable command with `mcpup add` or edit config",
			})
			continue
		}

		executable := command
		fields := strings.Fields(command)
		if len(fields) > 0 {
			executable = fields[0]
		}

		if _, err := exec.LookPath(executable); err != nil {
			report.Checks = append(report.Checks, Check{
				Key:        "command.path." + serverName,
				Status:     StatusWarn,
				Message:    fmt.Sprintf("executable not found in PATH: %s", executable),
				Suggestion: "install executable or update PATH/command",
			})
		} else {
			report.Checks = append(report.Checks, Check{
				Key:     "command.path." + serverName,
				Status:  StatusPass,
				Message: fmt.Sprintf("executable found: %s", executable),
			})
			if shouldProbeRegistryCommand(serverName) {
				if err := probeRegistryCommand(serverDef); err != nil {
					report.Checks = append(report.Checks, Check{
						Key:        "registry.probe." + serverName,
						Status:     StatusWarn,
						Message:    fmt.Sprintf("registry command probe failed: %v", err),
						Suggestion: "verify the registry template or rerun doctor without probe mode",
					})
				} else {
					report.Checks = append(report.Checks, Check{
						Key:     "registry.probe." + serverName,
						Status:  StatusPass,
						Message: "registry command probe passed",
					})
				}
			}
		}
	}

	return report, nil
}

func checkWriteAccess(path string) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	testFile, err := os.CreateTemp(dir, ".mcpup-write-check-*.tmp")
	if err != nil {
		return err
	}
	name := testFile.Name()
	if err := testFile.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}

func ownershipCounts(state planner.ClientState) (managed int, unmanaged int) {
	for name := range state.Servers {
		if state.Owned[name] {
			managed++
			continue
		}
		unmanaged++
	}
	return managed, unmanaged
}

func serverEnabledAnywhere(cfg store.Config, serverName string) bool {
	for _, clientCfg := range cfg.Clients {
		if state, ok := clientCfg.Servers[serverName]; ok && state.Enabled {
			return true
		}
	}
	return false
}

func missingRequiredEnv(server store.Server, tmpl registry.Template) []string {
	var missing []string
	for _, envVar := range tmpl.EnvVars {
		if !envVar.Required {
			continue
		}
		if strings.TrimSpace(server.Env[envVar.Key]) != "" {
			continue
		}
		missing = append(missing, envVar.Key)
	}
	slices.Sort(missing)
	return missing
}

func registryDefinitionChanged(server store.Server, tmpl registry.Template) bool {
	if strings.TrimSpace(server.Description) != strings.TrimSpace(tmpl.Description) {
		return true
	}
	if server.IsHTTP() || strings.TrimSpace(tmpl.URL) != "" {
		return strings.TrimSpace(server.URL) != strings.TrimSpace(tmpl.URL) ||
			strings.TrimSpace(server.Transport) != strings.TrimSpace(tmpl.Transport)
	}
	if strings.TrimSpace(server.Command) != strings.TrimSpace(tmpl.Command) {
		return true
	}
	return !slices.Equal(server.Args, tmpl.Args)
}

func shouldProbeRegistryCommand(serverName string) bool {
	if _, ok := registry.Lookup(serverName); !ok {
		return false
	}
	return os.Getenv("MCPUP_DOCTOR_REGISTRY_PROBE") == "1"
}

func probeRegistryCommand(server store.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := append([]string{}, server.Args...)
	args = append(args, "--help")
	cmd := exec.CommandContext(ctx, firstCommandToken(server.Command), args...)
	cmd.Env = os.Environ()
	for key, value := range server.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timed out")
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func firstCommandToken(command string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
