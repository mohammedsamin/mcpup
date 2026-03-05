package store

// CloneConfig returns a deep copy of cfg so callers can safely stage
// mutations before deciding whether to persist them.
func CloneConfig(cfg Config) Config {
	out := Config{
		Version:       cfg.Version,
		Servers:       make(map[string]Server, len(cfg.Servers)),
		Clients:       make(map[string]ClientConfig, len(cfg.Clients)),
		Profiles:      make(map[string]Profile, len(cfg.Profiles)),
		ActiveProfile: cfg.ActiveProfile,
	}

	for name, srv := range cfg.Servers {
		clone := Server{
			Command:     srv.Command,
			Args:        append([]string{}, srv.Args...),
			Env:         cloneStringMap(srv.Env),
			URL:         srv.URL,
			Headers:     cloneStringMap(srv.Headers),
			Transport:   srv.Transport,
			Description: srv.Description,
		}
		out.Servers[name] = clone
	}

	for client, clientCfg := range cfg.Clients {
		clonedClient := ClientConfig{
			Servers: make(map[string]ServerState, len(clientCfg.Servers)),
		}
		for serverName, state := range clientCfg.Servers {
			clonedClient.Servers[serverName] = ServerState{
				Enabled:       state.Enabled,
				EnabledTools:  append([]string{}, state.EnabledTools...),
				DisabledTools: append([]string{}, state.DisabledTools...),
			}
		}
		out.Clients[client] = clonedClient
	}

	for name, prof := range cfg.Profiles {
		clonedProfile := Profile{
			Servers: append([]string{}, prof.Servers...),
		}
		if prof.Tools != nil {
			clonedProfile.Tools = make(map[string]ToolSelection, len(prof.Tools))
			for serverName, selection := range prof.Tools {
				clonedProfile.Tools[serverName] = ToolSelection{
					Enabled:  append([]string{}, selection.Enabled...),
					Disabled: append([]string{}, selection.Disabled...),
				}
			}
		}
		out.Profiles[name] = clonedProfile
	}

	normalizeConfig(&out)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
