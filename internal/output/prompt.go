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

// Input prompts the user for free-text input with an optional default value.
// Returns the default when not interactive or the user presses enter without typing.
func Input(in io.Reader, out io.Writer, question string, defaultVal string) (string, error) {
	if !isTTY || in == nil {
		return defaultVal, nil
	}

	hint := ""
	if defaultVal != "" {
		hint = " " + Dim("("+defaultVal+")")
	}
	fmt.Fprintf(out, "%s %s%s ", Yellow("?"), Bold(question), hint)

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		fmt.Fprintln(out)
		return defaultVal, scanner.Err()
	}
	answer := strings.TrimSpace(scanner.Text())
	if answer == "" {
		return defaultVal, nil
	}
	return answer, nil
}

// MultiSelect prompts the user to toggle multiple options with space and confirm with enter.
// preSelected sets which options start checked (nil means all unchecked).
// Returns indices of selected options.
func MultiSelect(in *os.File, out io.Writer, question string, options []string, preSelected []bool) ([]int, error) {
	if len(options) == 0 {
		return nil, errors.New("no options to select from")
	}
	if !isTTY || in == nil {
		return nil, ErrNotInteractive
	}

	restore, err := enableRawMode(int(in.Fd()))
	if err != nil {
		return nil, fmt.Errorf("enable raw mode: %w", err)
	}
	defer restore()

	cursor := 0
	checked := make([]bool, len(options))
	if preSelected != nil {
		copy(checked, preSelected)
	}
	numLines := 1 + len(options) + 1 // question + options + hint
	buf := make([]byte, 3)

	fmt.Fprint(out, "\033[?25l")
	printMultiSelectUI(out, question, options, checked, cursor)

	for {
		n, readErr := in.Read(buf)
		if readErr != nil {
			fmt.Fprint(out, "\033[?25h")
			return nil, readErr
		}

		switch {
		case n == 1 && buf[0] == 3: // Ctrl+C
			fmt.Fprint(out, "\033[?25h")
			fmt.Fprintln(out)
			return nil, errors.New("interrupted")

		case n == 1 && buf[0] == ' ': // Space toggles
			checked[cursor] = !checked[cursor]
			eraseLines(out, numLines)
			printMultiSelectUI(out, question, options, checked, cursor)

		case n == 1 && buf[0] == 'a': // 'a' toggles all
			allChecked := true
			for _, c := range checked {
				if !c {
					allChecked = false
					break
				}
			}
			for i := range checked {
				checked[i] = !allChecked
			}
			eraseLines(out, numLines)
			printMultiSelectUI(out, question, options, checked, cursor)

		case n == 1 && (buf[0] == '\r' || buf[0] == '\n'): // Enter confirms
			eraseLines(out, numLines)
			var selected []int
			var names []string
			for i, c := range checked {
				if c {
					selected = append(selected, i)
					names = append(names, options[i])
				}
			}
			summary := strings.Join(names, ", ")
			if summary == "" {
				summary = Dim("(none)")
			}
			fmt.Fprintf(out, "%s %s %s\r\n", Green(SymbolOK), Bold(question), Cyan(summary))
			fmt.Fprint(out, "\033[?25h")
			return selected, nil

		case n == 3 && buf[0] == '\033' && buf[1] == '[' && buf[2] == 'A': // Up
			if cursor > 0 {
				cursor--
				eraseLines(out, numLines)
				printMultiSelectUI(out, question, options, checked, cursor)
			}

		case n == 3 && buf[0] == '\033' && buf[1] == '[' && buf[2] == 'B': // Down
			if cursor < len(options)-1 {
				cursor++
				eraseLines(out, numLines)
				printMultiSelectUI(out, question, options, checked, cursor)
			}
		}
	}
}

func printMultiSelectUI(w io.Writer, question string, options []string, checked []bool, cursor int) {
	fmt.Fprintf(w, "%s %s\r\n", Yellow("?"), Bold(question))
	for i, opt := range options {
		box := "[ ]"
		if checked[i] {
			box = "[" + Green(SymbolOK) + "]"
		}
		if i == cursor {
			fmt.Fprintf(w, "  %s %s %s\r\n", Cyan(SymbolArrow), box, Cyan(opt))
		} else {
			fmt.Fprintf(w, "    %s %s\r\n", box, Dim(opt))
		}
	}
	fmt.Fprintf(w, "  %s\r\n", Dim("space: toggle  a: all  enter: confirm"))
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
