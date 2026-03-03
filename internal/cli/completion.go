package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mohammedsamin/mcpup/internal/registry"
	"github.com/mohammedsamin/mcpup/internal/store"
)

func runCompletion(args []string, out io.Writer) error {
	if len(args) == 0 || hasHelp(args) {
		fmt.Fprintln(out, "Usage: mcpup completion <bash|zsh|fish>")
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("%w: completion requires exactly one positional argument: <shell>", errUsage)
	}

	switch strings.TrimSpace(args[0]) {
	case "bash":
		_, err := io.WriteString(out, bashCompletionScript)
		return err
	case "zsh":
		_, err := io.WriteString(out, zshCompletionScript)
		return err
	case "fish":
		_, err := io.WriteString(out, fishCompletionScript)
		return err
	default:
		return fmt.Errorf("%w: unsupported shell %q (use bash|zsh|fish)", errUsage, args[0])
	}
}

func runInternalComplete(args []string, out io.Writer) error {
	suggestions := completeSuggestions(args)
	for _, s := range suggestions {
		if strings.TrimSpace(s) == "" {
			continue
		}
		if _, err := fmt.Fprintln(out, s); err != nil {
			return err
		}
	}
	return nil
}

func completeSuggestions(words []string) []string {
	if len(words) == 0 {
		return append([]string{}, knownCommands...)
	}

	cmd := strings.TrimSpace(words[0])
	args := words[1:]
	current := ""
	prev := ""
	if len(args) > 0 {
		current = strings.TrimSpace(args[len(args)-1])
	}
	if len(args) > 1 {
		prev = strings.TrimSpace(args[len(args)-2])
	}

	// Some shells omit the trailing empty token when completing after a space.
	if current == "--client" || current == "--servers" || current == "--server" {
		prev = current
	}

	if prev == "--client" {
		return append([]string{}, store.SupportedClients...)
	}
	if prev == "--servers" {
		return configuredServerNames()
	}
	if prev == "--server" {
		if cmd == "setup" {
			return registry.Names()
		}
		return configuredServerNames()
	}

	switch cmd {
	case "enable", "disable":
		return uniqueSortedStrings(append(configuredServerNames(),
			"--client", "--tool", "--dry-run", "--json", "--verbose", "--yes"))
	case "remove":
		return uniqueSortedStrings(append(configuredServerNames(), "--yes", "--dry-run", "--json", "--verbose"))
	case "update":
		return uniqueSortedStrings(append(configuredServerNames(), "--yes", "--dry-run", "--json", "--verbose"))
	case "setup":
		return uniqueSortedStrings(append(registry.Names(),
			"--client", "--server", "--env", "--update", "--dry-run", "--json", "--verbose", "--yes"))
	case "add":
		return uniqueSortedStrings(append(registry.Names(),
			"--command", "--arg", "--env", "--url", "--header", "--transport", "--description", "--update"))
	case "list":
		return []string{"--client", "--json", "--verbose"}
	case "export":
		return uniqueSortedStrings(append(configuredServerNames(), "--servers", "--output", "-o"))
	case "import":
		return []string{"--overwrite", "--yes"}
	case "completion":
		return []string{"bash", "zsh", "fish"}
	case "profile":
		sub := firstNonFlag(args)
		if sub == "" {
			return []string{"create", "apply", "list", "delete"}
		}
		switch sub {
		case "apply", "delete":
			return profileNames()
		case "create":
			return append(configuredServerNames(), "--servers")
		default:
			return nil
		}
	default:
		return append([]string{}, knownCommands...)
	}
}

func configuredServerNames() []string {
	cfg, err := loadCanonicalOrDefault()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func profileNames() []string {
	cfg, err := loadCanonicalOrDefault()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func firstNonFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		part := strings.TrimSpace(args[i])
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "-") {
			if strings.Contains(part, "=") {
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
			continue
		}
		return part
	}
	return ""
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

const bashCompletionScript = `# mcpup bash completion
_mcpup_completions() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local suggestions
  suggestions="$(mcpup __complete "${COMP_WORDS[@]:1}")"
  COMPREPLY=($(compgen -W "${suggestions}" -- "${cur}"))
}
complete -F _mcpup_completions mcpup
`

const zshCompletionScript = `#compdef mcpup
_mcpup_completions() {
  local -a suggestions
  suggestions=("${(@f)$(mcpup __complete "${words[@]:1}")}")
  _describe 'values' suggestions
}
compdef _mcpup_completions mcpup
`

const fishCompletionScript = `function __fish_mcpup_complete
  mcpup __complete (commandline -opc | tail -n +2)
end
complete -c mcpup -f -a "(__fish_mcpup_complete)"
`
