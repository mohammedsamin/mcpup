package output

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrNotInteractive is returned when a prompt cannot run because
// the session is not an interactive terminal.
var ErrNotInteractive = errors.New("not an interactive terminal")

// Confirm prompts the user for a yes/no answer.
// When not interactive it returns defaultYes without prompting.
func Confirm(in io.Reader, out io.Writer, question string, defaultYes bool) (bool, error) {
	if !isTTY || in == nil {
		return defaultYes, nil
	}

	hint := "y/N"
	if defaultYes {
		hint = "Y/n"
	}
	fmt.Fprintf(out, "%s %s %s ", Yellow("?"), Bold(question), Dim("["+hint+"]"))

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		fmt.Fprintln(out)
		return defaultYes, scanner.Err()
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))

	switch answer {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return defaultYes, nil
	}
}

// Select prompts the user to choose from a list using arrow keys.
// Returns the index of the selected option.
// When not interactive it returns ErrNotInteractive.
func Select(in *os.File, out io.Writer, question string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, errors.New("no options to select from")
	}
	if !isTTY || in == nil {
		return -1, ErrNotInteractive
	}

	restore, err := enableRawMode(int(in.Fd()))
	if err != nil {
		return -1, fmt.Errorf("enable raw mode: %w", err)
	}
	defer restore()

	selected := 0
	numLines := 1 + len(options) // question + option lines
	buf := make([]byte, 3)

	// Hide cursor during selection.
	fmt.Fprint(out, "\033[?25l")

	printSelectUI(out, question, options, selected)

	for {
		n, readErr := in.Read(buf)
		if readErr != nil {
			fmt.Fprint(out, "\033[?25h")
			return -1, readErr
		}

		switch {
		case n == 1 && buf[0] == 3: // Ctrl+C
			fmt.Fprint(out, "\033[?25h")
			fmt.Fprintln(out)
			return -1, errors.New("interrupted")

		case n == 1 && (buf[0] == '\r' || buf[0] == '\n'): // Enter
			eraseLines(out, numLines)
			fmt.Fprintf(out, "%s %s %s\r\n", Green(SymbolOK), Bold(question), Cyan(options[selected]))
			fmt.Fprint(out, "\033[?25h")
			return selected, nil

		case n == 3 && buf[0] == '\033' && buf[1] == '[' && buf[2] == 'A': // Up
			if selected > 0 {
				selected--
				eraseLines(out, numLines)
				printSelectUI(out, question, options, selected)
			}

		case n == 3 && buf[0] == '\033' && buf[1] == '[' && buf[2] == 'B': // Down
			if selected < len(options)-1 {
				selected++
				eraseLines(out, numLines)
				printSelectUI(out, question, options, selected)
			}
		}
	}
}

func printSelectUI(w io.Writer, question string, options []string, selected int) {
	fmt.Fprintf(w, "%s %s\r\n", Yellow("?"), Bold(question))
	for i, opt := range options {
		if i == selected {
			fmt.Fprintf(w, "  %s %s\r\n", Cyan(SymbolArrow), Cyan(opt))
		} else {
			fmt.Fprintf(w, "    %s\r\n", Dim(opt))
		}
	}
}

// eraseLines moves the cursor up n lines, clearing each one,
// leaving the cursor at the start of the topmost cleared line.
func eraseLines(w io.Writer, n int) {
	for i := 0; i < n; i++ {
		fmt.Fprint(w, "\033[1A") // move up
		fmt.Fprint(w, "\033[2K") // clear line
	}
	fmt.Fprint(w, "\r")
}
