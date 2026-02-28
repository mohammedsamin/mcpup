package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// Options controls rendering behavior.
type Options struct {
	JSON    bool
	Verbose bool
	DryRun  bool
}

// Result is the common command response payload.
type Result struct {
	Command string         `json:"command"`
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

// Print writes result in text or JSON format.
func Print(w io.Writer, opts Options, result Result) error {
	if opts.JSON {
		body, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(body))
		return err
	}

	symbol := StatusSymbol(result.Status)
	if opts.DryRun {
		symbol = DryRunSymbol()
	}

	if _, err := fmt.Fprintf(w, "%s %s: %s\n", symbol, result.Command, result.Message); err != nil {
		return err
	}
	if opts.Verbose && len(result.Data) > 0 {
		keys := make([]string, 0, len(result.Data))
		for key := range result.Data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if _, err := fmt.Fprintf(w, "  %s %s: %v\n", Dim(SymbolArrow), Dim(key), result.Data[key]); err != nil {
				return err
			}
		}
	}
	return nil
}
