package store

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrAlreadyExists is returned when creating an existing resource.
	ErrAlreadyExists = errors.New("already exists")
	// ErrResourceNotFound is returned when a named resource does not exist.
	ErrResourceNotFound = errors.New("not found")
)

// AddServer adds a new server definition.
func AddServer(cfg *Config, name string, server Server) error {
	normalizeConfig(cfg)
	if err := ValidateServerName(name); err != nil {
		return err
	}
	if strings.TrimSpace(server.Command) == "" {
		return errors.New("server command cannot be empty")
	}
	if _, exists := cfg.Servers[name]; exists {
		return fmt.Errorf("server %q: %w", name, ErrAlreadyExists)
	}
	cfg.Servers[name] = server
	return nil
}

// UpsertServer creates or updates a server definition.
func UpsertServer(cfg *Config, name string, server Server) error {
	normalizeConfig(cfg)
	if err := ValidateServerName(name); err != nil {
		return err
	}
	if strings.TrimSpace(server.Command) == "" {
		return errors.New("server command cannot be empty")
	}
	cfg.Servers[name] = server
	return nil
}

// RemoveServer deletes a server and cleans client/profile references.
func RemoveServer(cfg *Config, name string) error {
	normalizeConfig(cfg)
	if _, exists := cfg.Servers[name]; !exists {
		return fmt.Errorf("server %q: %w", name, ErrResourceNotFound)
	}

	delete(cfg.Servers, name)

	for clientName, clientState := range cfg.Clients {
		delete(clientState.Servers, name)
		cfg.Clients[clientName] = clientState
	}

	for profileName, profile := range cfg.Profiles {
		profile.Servers = removeString(profile.Servers, name)
		if profile.Tools != nil {
			delete(profile.Tools, name)
		}
		cfg.Profiles[profileName] = profile
	}

	return nil
}

// CreateProfile creates a new profile.
func CreateProfile(cfg *Config, name string, profile Profile) error {
	normalizeConfig(cfg)
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	if _, exists := cfg.Profiles[name]; exists {
		return fmt.Errorf("profile %q: %w", name, ErrAlreadyExists)
	}
	if err := validateProfileRefs(cfg, profile); err != nil {
		return err
	}
	cfg.Profiles[name] = profile
	return nil
}

// UpsertProfile creates or updates a profile.
func UpsertProfile(cfg *Config, name string, profile Profile) error {
	normalizeConfig(cfg)
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	if err := validateProfileRefs(cfg, profile); err != nil {
		return err
	}
	cfg.Profiles[name] = profile
	return nil
}

// DeleteProfile deletes a profile and clears activeProfile if needed.
func DeleteProfile(cfg *Config, name string) error {
	normalizeConfig(cfg)
	if _, exists := cfg.Profiles[name]; !exists {
		return fmt.Errorf("profile %q: %w", name, ErrResourceNotFound)
	}
	delete(cfg.Profiles, name)
	if cfg.ActiveProfile == name {
		cfg.ActiveProfile = ""
	}
	return nil
}

// SetActiveProfile sets activeProfile to a known profile name.
func SetActiveProfile(cfg *Config, name string) error {
	normalizeConfig(cfg)
	if _, exists := cfg.Profiles[name]; !exists {
		return fmt.Errorf("profile %q: %w", name, ErrResourceNotFound)
	}
	cfg.ActiveProfile = name
	return nil
}

// SetClientServerEnabled sets server-level enabled state for a client.
func SetClientServerEnabled(cfg *Config, client string, server string, enabled bool) error {
	normalizeConfig(cfg)
	if err := ValidateClientName(client); err != nil {
		return err
	}
	if _, exists := cfg.Servers[server]; !exists {
		return fmt.Errorf("server %q: %w", server, ErrResourceNotFound)
	}

	clientState := cfg.Clients[client]
	if clientState.Servers == nil {
		clientState.Servers = map[string]ServerState{}
	}

	serverState := clientState.Servers[server]
	serverState.Enabled = enabled
	clientState.Servers[server] = serverState
	cfg.Clients[client] = clientState

	return nil
}

// SetClientToolEnabled updates per-tool enable/disable state for a client+server.
func SetClientToolEnabled(cfg *Config, client string, server string, tool string, enabled bool) error {
	normalizeConfig(cfg)
	if err := ValidateClientName(client); err != nil {
		return err
	}
	if _, exists := cfg.Servers[server]; !exists {
		return fmt.Errorf("server %q: %w", server, ErrResourceNotFound)
	}
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return errors.New("tool cannot be empty")
	}

	clientState := cfg.Clients[client]
	if clientState.Servers == nil {
		clientState.Servers = map[string]ServerState{}
	}

	serverState := clientState.Servers[server]
	if enabled {
		serverState.Enabled = true
		serverState.DisabledTools = removeString(serverState.DisabledTools, tool)
		serverState.EnabledTools = appendUnique(serverState.EnabledTools, tool)
	} else {
		serverState.EnabledTools = removeString(serverState.EnabledTools, tool)
		serverState.DisabledTools = appendUnique(serverState.DisabledTools, tool)
	}

	clientState.Servers[server] = serverState
	cfg.Clients[client] = clientState
	return nil
}

func validateProfileRefs(cfg *Config, profile Profile) error {
	for _, serverName := range profile.Servers {
		if _, exists := cfg.Servers[serverName]; !exists {
			return fmt.Errorf("profile references unknown server %q", serverName)
		}
	}
	for serverName, tools := range profile.Tools {
		if _, exists := cfg.Servers[serverName]; !exists {
			return fmt.Errorf("profile.tools references unknown server %q", serverName)
		}
		if err := validateToolLists(tools.Enabled, tools.Disabled); err != nil {
			return err
		}
	}
	return nil
}

func appendUnique(values []string, value string) []string {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}

func removeString(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}
