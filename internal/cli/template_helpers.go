package cli

import (
	"github.com/mohammedsamin/mcpup/internal/registry"
	"github.com/mohammedsamin/mcpup/internal/store"
)

func serverFromTemplate(tmpl registry.Template, env map[string]string) store.Server {
	headers := map[string]string{}
	for k, v := range tmpl.Headers {
		headers[k] = v
	}

	srv := store.Server{
		Command:     tmpl.Command,
		Args:        append([]string{}, tmpl.Args...),
		Env:         cloneStringMap(env),
		URL:         tmpl.URL,
		Headers:     headers,
		Transport:   tmpl.Transport,
		Description: tmpl.Description,
	}
	if len(srv.Headers) == 0 {
		srv.Headers = nil
	}
	return srv
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
