package adapters

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/planner"
	"github.com/mohammedsamin/mcpup/internal/store"
)

// JSONDocument preserves unknown top-level keys while manipulating mcpServers.
type JSONDocument map[string]json.RawMessage

const metaKey = "_mcpup"

type ownershipMetadata struct {
	ManagedServers []string `json:"managedServers,omitempty"`
}

// ReadJSONDocument reads a JSON object document from disk.
func ReadJSONDocument(path string) (JSONDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return JSONDocument{}, nil
		}
		return nil, err
	}

	doc := JSONDocument{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// ReadStateFromMCPServers decodes state from a top-level mcpServers key.
func ReadStateFromMCPServers(doc JSONDocument, client string) (planner.ClientState, error) {
	return ReadStateFromServerMap(doc, "mcpServers", client)
}

// ReadStateFromServerMap decodes state from a top-level server object key.
func ReadStateFromServerMap(doc JSONDocument, topLevelKey string, client string) (planner.ClientState, error) {
	state := planner.ClientState{
		Client:     client,
		Servers:    map[string]planner.ServerState{},
		Owned:      managedServersFromDoc(doc),
		ServerDefs: map[string]store.Server{},
	}

	raw, ok := doc[topLevelKey]
	if !ok || len(raw) == 0 {
		return state, nil
	}

	rawEntries := map[string]map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &rawEntries); err != nil {
		return planner.ClientState{}, fmt.Errorf("decode %s: %w", topLevelKey, err)
	}

	for serverName, entry := range rawEntries {
		enabled := true
		if rawEnabled, ok := entry["enabled"]; ok {
			if err := json.Unmarshal(rawEnabled, &enabled); err != nil {
				return planner.ClientState{}, fmt.Errorf("decode enabled for server %q: %w", serverName, err)
			}
		}

		var enabledTools []string
		if rawEnabledTools, ok := entry["enabledTools"]; ok {
			if err := json.Unmarshal(rawEnabledTools, &enabledTools); err != nil {
				return planner.ClientState{}, fmt.Errorf("decode enabledTools for server %q: %w", serverName, err)
			}
		}

		var disabledTools []string
		if rawDisabledTools, ok := entry["disabledTools"]; ok {
			if err := json.Unmarshal(rawDisabledTools, &disabledTools); err != nil {
				return planner.ClientState{}, fmt.Errorf("decode disabledTools for server %q: %w", serverName, err)
			}
		}

		state.Servers[serverName] = planner.ServerState{
			Enabled:       enabled,
			EnabledTools:  append([]string{}, enabledTools...),
			DisabledTools: append([]string{}, disabledTools...),
		}
		def, err := DecodeServerDefinitionEntry(entry)
		if err != nil {
			return planner.ClientState{}, fmt.Errorf("decode server definition for %q: %w", serverName, err)
		}
		state.ServerDefs[serverName] = def
	}

	return planner.NormalizeState(state), nil
}

// WriteStateToMCPServers updates mcpServers while preserving unknown top-level keys.
func WriteStateToMCPServers(path string, doc JSONDocument, desired planner.ClientState) error {
	return WriteStateToServerMap(path, doc, "mcpServers", desired)
}

// WriteStateToServerMap updates a top-level server object while preserving unknown top-level keys.
func WriteStateToServerMap(path string, doc JSONDocument, topLevelKey string, desired planner.ClientState) error {
	normalized := planner.NormalizeState(desired)
	existingServers := map[string]map[string]json.RawMessage{}
	if raw, ok := doc[topLevelKey]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &existingServers); err != nil {
			return fmt.Errorf("decode existing %s: %w", topLevelKey, err)
		}
	}

	managedBefore := managedServersFromDoc(doc)
	nextServers := map[string]map[string]json.RawMessage{}
	for serverName, current := range existingServers {
		if managedBefore[serverName] {
			continue
		}
		if _, managedNow := normalized.Servers[serverName]; managedNow {
			continue
		}
		base := map[string]json.RawMessage{}
		for key, value := range current {
			base[key] = append(json.RawMessage{}, value...)
		}
		nextServers[serverName] = base
	}

	for serverName, state := range normalized.Servers {
		base := map[string]json.RawMessage{}
		if current, ok := existingServers[serverName]; ok {
			for key, value := range current {
				base[key] = append(json.RawMessage{}, value...)
			}
		}

		rawEnabled, err := json.Marshal(state.Enabled)
		if err != nil {
			return fmt.Errorf("encode enabled for server %q: %w", serverName, err)
		}
		base["enabled"] = rawEnabled

		if len(state.EnabledTools) > 0 {
			rawEnabledTools, err := json.Marshal(state.EnabledTools)
			if err != nil {
				return fmt.Errorf("encode enabledTools for server %q: %w", serverName, err)
			}
			base["enabledTools"] = rawEnabledTools
		} else {
			delete(base, "enabledTools")
		}

		if len(state.DisabledTools) > 0 {
			rawDisabledTools, err := json.Marshal(state.DisabledTools)
			if err != nil {
				return fmt.Errorf("encode disabledTools for server %q: %w", serverName, err)
			}
			base["disabledTools"] = rawDisabledTools
		} else {
			delete(base, "disabledTools")
		}

		if def, ok := desired.ServerDefs[serverName]; ok {
			if def.IsHTTP() {
				rawURL, err := json.Marshal(def.URL)
				if err != nil {
					return fmt.Errorf("encode url for server %q: %w", serverName, err)
				}
				base["url"] = rawURL

				if len(def.Headers) > 0 {
					rawHeaders, err := json.Marshal(def.Headers)
					if err != nil {
						return fmt.Errorf("encode headers for server %q: %w", serverName, err)
					}
					base["headers"] = rawHeaders
				} else {
					delete(base, "headers")
				}

				if def.Transport != "" {
					rawTransport, err := json.Marshal(def.Transport)
					if err != nil {
						return fmt.Errorf("encode transport for server %q: %w", serverName, err)
					}
					base["transport"] = rawTransport
				}

				// Remove stdio-only fields.
				delete(base, "command")
				delete(base, "args")
				delete(base, "env")
			} else if strings.TrimSpace(def.Command) != "" {
				rawCmd, err := json.Marshal(def.Command)
				if err != nil {
					return fmt.Errorf("encode command for server %q: %w", serverName, err)
				}
				base["command"] = rawCmd

				if len(def.Args) > 0 {
					rawArgs, err := json.Marshal(def.Args)
					if err != nil {
						return fmt.Errorf("encode args for server %q: %w", serverName, err)
					}
					base["args"] = rawArgs
				} else {
					delete(base, "args")
				}

				if len(def.Env) > 0 {
					rawEnv, err := json.Marshal(def.Env)
					if err != nil {
						return fmt.Errorf("encode env for server %q: %w", serverName, err)
					}
					base["env"] = rawEnv
				} else {
					delete(base, "env")
				}

				// Remove HTTP-only fields.
				delete(base, "url")
				delete(base, "headers")
				delete(base, "transport")
			}
		}

		nextServers[serverName] = base
	}

	raw, err := json.Marshal(nextServers)
	if err != nil {
		return fmt.Errorf("encode %s: %w", topLevelKey, err)
	}
	doc[topLevelKey] = raw
	setManagedServersInDoc(doc, normalized.Servers)

	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode document: %w", err)
	}
	body = append(body, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

// ManagedDiff compares desired state against the managed subset of current state.
func ManagedDiff(current planner.ClientState, desired planner.ClientState) planner.Plan {
	filtered := filterManagedState(current, desired)
	plan := planner.Diff(filtered, desired)
	for serverName := range desired.Servers {
		desiredDef, ok := desired.ServerDefs[serverName]
		if !ok {
			continue
		}
		currentDef, currentOK := filtered.ServerDefs[serverName]
		if currentOK && sameServerDefinition(currentDef, desiredDef) {
			continue
		}
		plan.Changes = append(plan.Changes, planner.Change{
			Kind:   planner.ChangeUpsertServer,
			Server: serverName,
			From:   currentDef,
			To:     desiredDef,
		})
	}
	return plan
}

// DeepCopyDocument clones the top-level raw map to avoid mutation surprises.
func DeepCopyDocument(doc JSONDocument) JSONDocument {
	out := make(JSONDocument, len(doc))
	for key, value := range doc {
		out[key] = append(json.RawMessage{}, value...)
	}
	return out
}

// CollectCommandKeys returns command strings from native entries for diagnostics.
func CollectCommandKeys(doc JSONDocument) []string {
	raw, ok := doc["mcpServers"]
	if !ok || len(raw) == 0 {
		return nil
	}

	type entry struct {
		Command string `json:"command"`
	}
	servers := map[string]entry{}
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil
	}

	out := make([]string, 0, len(servers))
	for _, e := range servers {
		if strings.TrimSpace(e.Command) != "" {
			out = append(out, e.Command)
		}
	}
	return out
}

func filterManagedState(current planner.ClientState, desired planner.ClientState) planner.ClientState {
	out := planner.ClientState{
		Client:     current.Client,
		Servers:    map[string]planner.ServerState{},
		Owned:      map[string]bool{},
		ServerDefs: map[string]store.Server{},
	}

	for serverName, state := range current.Servers {
		if current.Owned[serverName] {
			out.Servers[serverName] = state
			out.Owned[serverName] = true
			if def, ok := current.ServerDefs[serverName]; ok {
				out.ServerDefs[serverName] = def
			}
			continue
		}
		if _, ok := desired.Servers[serverName]; ok {
			out.Servers[serverName] = state
			if def, ok := current.ServerDefs[serverName]; ok {
				out.ServerDefs[serverName] = def
			}
		}
	}

	return planner.NormalizeState(out)
}

// DecodeServerDefinitionEntry extracts a store.Server definition from one native server entry.
func DecodeServerDefinitionEntry(entry map[string]json.RawMessage) (store.Server, error) {
	def := store.Server{}
	if raw, ok := entry["command"]; ok {
		if err := json.Unmarshal(raw, &def.Command); err != nil {
			return store.Server{}, err
		}
	}
	if raw, ok := entry["args"]; ok {
		if err := json.Unmarshal(raw, &def.Args); err != nil {
			return store.Server{}, err
		}
	}
	if raw, ok := entry["env"]; ok {
		if err := json.Unmarshal(raw, &def.Env); err != nil {
			return store.Server{}, err
		}
	}
	if raw, ok := entry["url"]; ok {
		if err := json.Unmarshal(raw, &def.URL); err != nil {
			return store.Server{}, err
		}
	}
	if raw, ok := entry["headers"]; ok {
		if err := json.Unmarshal(raw, &def.Headers); err != nil {
			return store.Server{}, err
		}
	}
	if raw, ok := entry["transport"]; ok {
		if err := json.Unmarshal(raw, &def.Transport); err != nil {
			return store.Server{}, err
		}
	}
	return def, nil
}

func sameServerDefinition(a store.Server, b store.Server) bool {
	return strings.TrimSpace(a.Command) == strings.TrimSpace(b.Command) &&
		slices.Equal(a.Args, b.Args) &&
		maps.Equal(a.Env, b.Env) &&
		strings.TrimSpace(a.URL) == strings.TrimSpace(b.URL) &&
		maps.Equal(a.Headers, b.Headers) &&
		strings.TrimSpace(a.Transport) == strings.TrimSpace(b.Transport) &&
		strings.TrimSpace(a.Description) == strings.TrimSpace(b.Description)
}

func managedServersFromDoc(doc JSONDocument) map[string]bool {
	raw, ok := doc[metaKey]
	if !ok || len(raw) == 0 {
		return nil
	}

	var meta ownershipMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil
	}

	out := make(map[string]bool, len(meta.ManagedServers))
	for _, name := range meta.ManagedServers {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func setManagedServersInDoc(doc JSONDocument, servers map[string]planner.ServerState) {
	if len(servers) == 0 {
		delete(doc, metaKey)
		return
	}

	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	names = normalizeManagedNames(names)

	raw, err := json.Marshal(ownershipMetadata{ManagedServers: names})
	if err != nil {
		return
	}
	doc[metaKey] = raw
}

func normalizeManagedNames(names []string) []string {
	set := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := set[name]; ok {
			continue
		}
		set[name] = struct{}{}
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}
