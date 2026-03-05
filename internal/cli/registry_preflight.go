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
