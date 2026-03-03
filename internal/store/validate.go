package store

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	serverNamePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	profileNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-_]{0,62}$`)
	envKeyPattern      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

var reservedNames = map[string]struct{}{
	"all":     {},
	"default": {},
	"none":    {},
	"help":    {},
}

var supportedClientSet = map[string]struct{}{
	"claude-code":    {},
	"cursor":         {},
	"claude-desktop": {},
	"codex":          {},
	"opencode":       {},
	"windsurf":       {},
	"zed":            {},
	"continue":       {},
}

// EnvValueMode tells whether an env value is a literal string or an env reference.
type EnvValueMode string

const (
	EnvValueLiteral   EnvValueMode = "literal"
	EnvValueReference EnvValueMode = "reference"
)

// DetectEnvValueMode classifies env values as literal or ${ENV_KEY} reference.
func DetectEnvValueMode(value string) EnvValueMode {
	if IsEnvReference(value) {
		return EnvValueReference
	}
	return EnvValueLiteral
}

// IsEnvReference reports whether value matches ${ENV_KEY}.
func IsEnvReference(value string) bool {
	if !strings.HasPrefix(value, "${") || !strings.HasSuffix(value, "}") {
		return false
	}
	key := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
	return envKeyPattern.MatchString(key)
}

// ValidateServerName validates a server identifier.
func ValidateServerName(name string) error {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return errors.New("server name cannot be empty")
	}
	if _, reserved := reservedNames[normalized]; reserved {
		return fmt.Errorf("server name %q is reserved", normalized)
	}
	if !serverNamePattern.MatchString(normalized) {
		return fmt.Errorf("server name %q is invalid; use lowercase letters, digits, or hyphen", normalized)
	}
	return nil
}

// ValidateProfileName validates a profile identifier.
func ValidateProfileName(name string) error {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return errors.New("profile name cannot be empty")
	}
	if _, reserved := reservedNames[normalized]; reserved {
		return fmt.Errorf("profile name %q is reserved", normalized)
	}
	if !profileNamePattern.MatchString(normalized) {
		return fmt.Errorf("profile name %q is invalid; use lowercase letters, digits, hyphen, or underscore", normalized)
	}
	return nil
}

// ValidateClientName validates that a client is supported in v1.
func ValidateClientName(client string) error {
	if _, ok := supportedClientSet[strings.TrimSpace(client)]; !ok {
		return fmt.Errorf("unsupported client %q", client)
	}
	return nil
}

// ValidateConfigSchema checks core schema shape and references.
func ValidateConfigSchema(cfg Config) error {
	if cfg.Version != CurrentSchemaVersion {
		return fmt.Errorf("unsupported schema version %d", cfg.Version)
	}

	for serverName, def := range cfg.Servers {
		if err := ValidateServerName(serverName); err != nil {
			return fmt.Errorf("servers.%s: %w", serverName, err)
		}
		if strings.TrimSpace(def.Command) == "" && strings.TrimSpace(def.URL) == "" {
			return fmt.Errorf("servers.%s: requires either command or url", serverName)
		}
		if strings.TrimSpace(def.Command) != "" && strings.TrimSpace(def.URL) != "" {
			return fmt.Errorf("servers.%s: cannot have both command and url", serverName)
		}
		if def.Transport != "" {
			switch def.Transport {
			case "stdio", "sse", "streamable-http": // ok
			default:
				return fmt.Errorf("servers.%s.transport: unsupported value %q", serverName, def.Transport)
			}
		}
		for key := range def.Headers {
			if !envKeyPattern.MatchString(key) {
				return fmt.Errorf("servers.%s.headers has invalid key %q", serverName, key)
			}
		}
		for key, value := range def.Env {
			if !envKeyPattern.MatchString(key) {
				return fmt.Errorf("servers.%s.env has invalid key %q", serverName, key)
			}
			mode := DetectEnvValueMode(value)
			if mode == EnvValueReference && !IsEnvReference(value) {
				return fmt.Errorf("servers.%s.env.%s has invalid env reference %q", serverName, key, value)
			}
		}
	}

	for clientName, clientState := range cfg.Clients {
		if err := ValidateClientName(clientName); err != nil {
			return fmt.Errorf("clients.%s: %w", clientName, err)
		}
		for serverName, state := range clientState.Servers {
			if _, ok := cfg.Servers[serverName]; !ok {
				return fmt.Errorf("clients.%s.servers.%s references unknown server", clientName, serverName)
			}
			if err := validateToolLists(state.EnabledTools, state.DisabledTools); err != nil {
				return fmt.Errorf("clients.%s.servers.%s: %w", clientName, serverName, err)
			}
		}
	}

	for profileName, profile := range cfg.Profiles {
		if err := ValidateProfileName(profileName); err != nil {
			return fmt.Errorf("profiles.%s: %w", profileName, err)
		}
		for _, serverName := range profile.Servers {
			if _, ok := cfg.Servers[serverName]; !ok {
				return fmt.Errorf("profiles.%s references unknown server %q", profileName, serverName)
			}
		}
		for serverName, tools := range profile.Tools {
			if _, ok := cfg.Servers[serverName]; !ok {
				return fmt.Errorf("profiles.%s.tools.%s references unknown server", profileName, serverName)
			}
			if err := validateToolLists(tools.Enabled, tools.Disabled); err != nil {
				return fmt.Errorf("profiles.%s.tools.%s: %w", profileName, serverName, err)
			}
		}
	}

	if cfg.ActiveProfile != "" {
		if _, ok := cfg.Profiles[cfg.ActiveProfile]; !ok {
			return fmt.Errorf("activeProfile %q does not exist in profiles", cfg.ActiveProfile)
		}
	}

	return nil
}

func validateToolLists(enabled []string, disabled []string) error {
	if len(enabled) == 0 && len(disabled) == 0 {
		return nil
	}

	enabledSet := make(map[string]struct{}, len(enabled))
	for _, tool := range enabled {
		name := strings.TrimSpace(tool)
		if name == "" {
			return errors.New("enabledTools contains empty value")
		}
		enabledSet[name] = struct{}{}
	}

	for _, tool := range disabled {
		name := strings.TrimSpace(tool)
		if name == "" {
			return errors.New("disabledTools contains empty value")
		}
		if _, exists := enabledSet[name]; exists {
			return fmt.Errorf("tool %q appears in both enabledTools and disabledTools", name)
		}
	}

	return nil
}
