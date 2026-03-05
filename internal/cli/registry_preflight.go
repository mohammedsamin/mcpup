package cli

import (
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/output"
	"github.com/mohammedsamin/mcpup/internal/registry"
	"github.com/mohammedsamin/mcpup/internal/store"
)

func mergeServerWithTemplate(current store.Server, tmpl registry.Template) store.Server {
	current = registry.MigrateLegacyServerDefinition(tmpl.Name, current)
	next := serverFromTemplate(tmpl, current.Env)
	if len(current.Headers) > 0 {
		if next.Headers == nil {
			next.Headers = map[string]string{}
		}
		for key, value := range current.Headers {
			if _, exists := next.Headers[key]; !exists {
				next.Headers[key] = value
			}
		}
	}
	if next.Transport == "" {
		next.Transport = current.Transport
	}
	return next
}

func sameServerDefinition(a store.Server, b store.Server) bool {
	return strings.TrimSpace(a.Command) == strings.TrimSpace(b.Command) &&
		slices.Equal(a.Args, b.Args) &&
		sameStringKeySet(a.Env, b.Env) &&
		strings.TrimSpace(a.URL) == strings.TrimSpace(b.URL) &&
		strings.TrimSpace(a.Transport) == strings.TrimSpace(b.Transport) &&
		strings.TrimSpace(a.Description) == strings.TrimSpace(b.Description) &&
		maps.Equal(a.Headers, b.Headers)
}

func validateRegistryServerDefinition(name string, tmpl registry.Template, server store.Server) error {
	missingEnv := make([]string, 0, len(tmpl.EnvVars))
	for _, ev := range tmpl.EnvVars {
		if ev.Required && strings.TrimSpace(server.Env[ev.Key]) == "" {
			missingEnv = append(missingEnv, ev.Key)
		}
	}
	if len(missingEnv) > 0 {
		slices.Sort(missingEnv)
		return fmt.Errorf("registry server %q is missing required env vars: %s", name, strings.Join(missingEnv, ", "))
	}

	if server.IsHTTP() {
		return nil
	}

	executable := firstCommandToken(server.Command)
	if executable == "" {
		return fmt.Errorf("registry server %q has an empty command", name)
	}
	if _, err := exec.LookPath(executable); err != nil {
		return fmt.Errorf("registry server %q requires executable %q in PATH", name, executable)
	}
	return nil
}

func requireManagedChangeApproval(opts GlobalOptions, in *os.File, out io.Writer, summary string) error {
	if opts.DryRun || strings.TrimSpace(summary) == "" || opts.Yes {
		return nil
	}
	if in != nil && output.IsTTY() {
		confirmed, err := output.Confirm(in, out, summary, false)
		if err != nil {
			return err
		}
		if !confirmed {
			return fmt.Errorf("aborted")
		}
		return nil
	}
	return fmt.Errorf("%s; add --yes to confirm", summary)
}

func formatChangeSummary(prefix string, servers []string, clients []string) string {
	serverText := previewNames(servers, 4)
	clientText := previewNames(clients, 4)
	if len(clients) == 0 {
		return fmt.Sprintf("%s: %s", prefix, serverText)
	}
	return fmt.Sprintf("%s: %s across %d client(s) (%s)", prefix, serverText, len(clients), clientText)
}

func previewNames(names []string, limit int) string {
	if len(names) == 0 {
		return "none"
	}
	copyNames := append([]string{}, names...)
	slices.Sort(copyNames)
	if len(copyNames) <= limit {
		return strings.Join(copyNames, ", ")
	}
	return fmt.Sprintf("%s, +%d more", strings.Join(copyNames[:limit], ", "), len(copyNames)-limit)
}

func firstCommandToken(command string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func sameStringKeySet(a map[string]string, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if _, ok := b[key]; !ok {
			return false
		}
	}
	return true
}

func applyTemplateInputs(
	server *store.Server,
	tmpl registry.Template,
	overrides map[string]string,
	interactive bool,
	in *os.File,
	out io.Writer,
) error {
	if server.Env == nil {
		server.Env = map[string]string{}
	}
	for key, value := range overrides {
		server.Env[key] = value
	}
	*server = registry.MigrateLegacyServerDefinition(tmpl.Name, *server)

	promptInputs := templatePromptInputs(tmpl)
	if len(promptInputs) > 0 && !requiredTemplateEnvSatisfied(tmpl, server.Env) {
		values := map[string]string{}
		for _, input := range promptInputs {
			value := strings.TrimSpace(server.Env[input.ID])
			if value == "" {
				value = strings.TrimSpace(os.Getenv(input.ID))
			}
			if value == "" && interactive {
				label := input.Label
				if label == "" {
					label = input.ID
				}
				if input.Hint != "" {
					label += " " + output.Dim("("+input.Hint+")")
				}
				if input.Required {
					label += output.Red(" [required]")
				}
				answer, err := output.Input(in, out, label+":", "")
				if err != nil {
					return err
				}
				value = strings.TrimSpace(answer)
			}
			if value == "" && input.Required {
				label := input.Label
				if label == "" {
					label = input.ID
				}
				return fmt.Errorf("%s is required for %s", label, tmpl.Name)
			}
			if value != "" {
				values[input.ID] = value
			}
		}

		renderedEnv, renderedHeaders := renderTemplateInputs(tmpl, values)
		for key, value := range renderedEnv {
			server.Env[key] = value
		}
		if len(renderedHeaders) > 0 {
			if server.Headers == nil {
				server.Headers = map[string]string{}
			}
			for key, value := range renderedHeaders {
				server.Headers[key] = value
			}
		}
		*server = registry.MigrateLegacyServerDefinition(tmpl.Name, *server)
	}

	return validateRegistryServerDefinition(tmpl.Name, tmpl, *server)
}

func templatePromptInputs(tmpl registry.Template) []registry.PromptInput {
	if len(tmpl.PromptInputs) > 0 {
		return append([]registry.PromptInput{}, tmpl.PromptInputs...)
	}
	out := make([]registry.PromptInput, 0, len(tmpl.EnvVars))
	for _, envVar := range tmpl.EnvVars {
		out = append(out, registry.PromptInput{
			ID:       envVar.Key,
			Label:    envVar.Key,
			Required: envVar.Required,
			Hint:     envVar.Hint,
		})
	}
	return out
}

func requiredTemplateEnvSatisfied(tmpl registry.Template, env map[string]string) bool {
	for _, envVar := range tmpl.EnvVars {
		if envVar.Required && strings.TrimSpace(env[envVar.Key]) == "" {
			return false
		}
	}
	return true
}

func renderTemplateInputs(tmpl registry.Template, values map[string]string) (map[string]string, map[string]string) {
	if len(tmpl.RenderEnv) == 0 && len(tmpl.RenderHeaders) == 0 {
		env := map[string]string{}
		for key, value := range values {
			env[key] = value
		}
		return env, nil
	}

	env := map[string]string{}
	for key, value := range tmpl.RenderEnv {
		env[key] = renderTemplateString(value, values)
	}
	headers := map[string]string{}
	for key, value := range tmpl.RenderHeaders {
		headers[key] = renderTemplateString(value, values)
	}
	if len(headers) == 0 {
		headers = nil
	}
	return env, headers
}

func renderTemplateString(template string, values map[string]string) string {
	out := template
	for key, value := range values {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}
