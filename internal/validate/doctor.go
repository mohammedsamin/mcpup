package validate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/core"
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
