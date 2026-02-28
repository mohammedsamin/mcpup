package planner

import (
	"fmt"
	"strings"
)

// DryRunSummary renders a stable text summary for --dry-run output.
func DryRunSummary(command string, plan Plan) string {
	if !plan.HasChanges() {
		return fmt.Sprintf("%s dry-run: no changes for client %s", command, plan.Client)
	}

	lines := []string{
		fmt.Sprintf("%s dry-run: %d change(s) for client %s", command, len(plan.Changes), plan.Client),
	}

	for _, change := range plan.Changes {
		line := fmt.Sprintf("- %s server=%s", change.Kind, change.Server)
		if change.Tool != "" {
			line += fmt.Sprintf(" tool=%s", change.Tool)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
