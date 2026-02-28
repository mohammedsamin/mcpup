package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/output"
	"github.com/mohammedsamin/mcpup/internal/store"
)

type exportPayload struct {
	Version int                     `json:"version"`
	Servers map[string]store.Server `json:"servers"`
}

func runExport(opts GlobalOptions, args []string, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup export [--servers a,b,c] [--output <path>]")
		return nil
	}

	fs := newFlagSet("export")
	servers := fs.String("servers", "", "comma-separated server names")
	outputPath := fs.String("output", "", "output file path")
	fs.StringVar(outputPath, "o", "", "output file path (shorthand)")
	normalized, err := normalizeArgs(args, map[string]bool{
		"--servers": true,
		"--output":  true,
		"-o":        true,
	})
	if err != nil {
		return fmt.Errorf("%w: export: %v", errUsage, err)
	}
	if err := fs.Parse(normalized); err != nil {
		return fmt.Errorf("%w: export: %v", errUsage, err)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: export does not accept positional arguments", errUsage)
	}

	cfg, err := loadCanonicalOrDefault()
	if err != nil {
		return err
	}

	selected := splitCSV(*servers)
	serverNames := selected
	if len(serverNames) == 0 {
		serverNames = make([]string, 0, len(cfg.Servers))
		for name := range cfg.Servers {
			serverNames = append(serverNames, name)
		}
	}
	sort.Strings(serverNames)

	payload := exportPayload{
		Version: store.CurrentSchemaVersion,
		Servers: map[string]store.Server{},
	}
	for _, name := range serverNames {
		srv, ok := cfg.Servers[name]
		if !ok {
			return fmt.Errorf("server %q not found", name)
		}
		payload.Servers[name] = srv
	}

	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	if strings.TrimSpace(*outputPath) == "" {
		_, err = out.Write(body)
		return err
	}

	if err := os.WriteFile(*outputPath, body, 0o644); err != nil {
		return err
	}

	return printResult(out, opts, output.Result{
		Command: "export",
		Status:  "ok",
		Message: fmt.Sprintf("exported %d server(s)", len(payload.Servers)),
		Data: map[string]any{
			"file":        *outputPath,
			"serverCount": len(payload.Servers),
		},
	})
}

func runImport(opts GlobalOptions, args []string, in *os.File, out io.Writer) error {
	if hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup import <file> [--overwrite] [--yes]")
		return nil
	}

	fs := newFlagSet("import")
	overwrite := fs.Bool("overwrite", false, "overwrite existing servers")
	normalized, err := normalizeArgs(args, map[string]bool{
		"--overwrite": false,
	})
	if err != nil {
		return fmt.Errorf("%w: import: %v", errUsage, err)
	}
	if err := fs.Parse(normalized); err != nil {
		return fmt.Errorf("%w: import: %v", errUsage, err)
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: import requires exactly one positional argument: <file>", errUsage)
	}

	body, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}

	servers, err := decodeImportServers(body)
	if err != nil {
		return err
	}
	if len(servers) == 0 {
		return fmt.Errorf("import file has no servers")
	}

	path, cfg, err := store.EnsureConfig("")
	if err != nil {
		return err
	}

	conflicts := 0
	for name := range servers {
		if _, exists := cfg.Servers[name]; exists {
			conflicts++
		}
	}

	if conflicts > 0 && *overwrite && !opts.DryRun && !opts.Yes {
		if in != nil && output.IsTTY() {
			confirmed, confirmErr := output.Confirm(in, out,
				fmt.Sprintf("Overwrite %d existing server(s)?", conflicts), false)
			if confirmErr != nil {
				return confirmErr
			}
			if !confirmed {
				return fmt.Errorf("aborted")
			}
		} else {
			return fmt.Errorf("import would overwrite %d server(s); add --yes to confirm", conflicts)
		}
	}

	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)

	added := 0
	updated := 0
	skipped := 0
	for _, name := range names {
		srv := servers[name]
		if _, exists := cfg.Servers[name]; exists {
			if !*overwrite {
				skipped++
				continue
			}
			if err := store.UpsertServer(&cfg, name, srv); err != nil {
				return err
			}
			updated++
			continue
		}
		if err := store.AddServer(&cfg, name, srv); err != nil {
			return err
		}
		added++
	}

	if !opts.DryRun {
		if err := store.SaveConfig(path, cfg); err != nil {
			return err
		}
	}

	return printResult(out, opts, output.Result{
		Command: "import",
		Status:  "ok",
		Message: fmt.Sprintf("imported servers: %d added, %d updated, %d skipped", added, updated, skipped),
		Data: map[string]any{
			"file":      fs.Arg(0),
			"added":     added,
			"updated":   updated,
			"skipped":   skipped,
			"overwrite": *overwrite,
			"dryRun":    opts.DryRun,
		},
	})
}

func decodeImportServers(body []byte) (map[string]store.Server, error) {
	type withServers struct {
		Servers map[string]store.Server `json:"servers"`
	}
	var payload withServers
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse import file: %w", err)
	}
	if payload.Servers == nil {
		return map[string]store.Server{}, nil
	}
	return payload.Servers, nil
}

func loadCanonicalOrDefault() (store.Config, error) {
	path, err := store.ResolveConfigPath("")
	if err != nil {
		return store.Config{}, err
	}
	cfg, err := store.LoadConfig(path)
	if err != nil {
		var serr *store.StoreError
		if errors.As(err, &serr) && serr.Kind == store.KindNotFound {
			return store.NewDefaultConfig(), nil
		}
		return store.Config{}, err
	}
	return cfg, nil
}
