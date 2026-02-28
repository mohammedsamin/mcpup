package output

import (
	"fmt"
	"io"
	"strings"
)

// Table renders column-aligned output.
type Table struct {
	Headers []string
	Rows    [][]string
}

// AddRow appends a row to the table.
func (t *Table) AddRow(cells ...string) {
	t.Rows = append(t.Rows, cells)
}

// Render writes the table to w with aligned columns and styled headers.
func (t *Table) Render(w io.Writer) {
	if len(t.Headers) == 0 {
		return
	}

	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = displayWidth(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) {
				if dw := displayWidth(cell); dw > widths[i] {
					widths[i] = dw
				}
			}
		}
	}

	// Header row.
	for i, h := range t.Headers {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprint(w, BoldUnderline(padRight(h, widths[i])))
	}
	fmt.Fprintln(w)

	// Data rows.
	for _, row := range t.Rows {
		for i := range t.Headers {
			if i > 0 {
				fmt.Fprint(w, "  ")
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			fmt.Fprint(w, padRight(cell, widths[i]))
		}
		fmt.Fprintln(w)
	}
}

// padRight pads s with spaces to reach the given visible width.
func padRight(s string, width int) string {
	dw := displayWidth(s)
	if dw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-dw)
}

// displayWidth returns the visible width of s, ignoring ANSI escape sequences.
func displayWidth(s string) int {
	width := 0
	inEscape := false
	for _, r := range s {
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		if r == '\033' {
			inEscape = true
			continue
		}
		width++
	}
	return width
}
