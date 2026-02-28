package output

import "os"

// ANSI escape sequences for colored terminal output.
const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiDim       = "\033[2m"
	ansiUnderline = "\033[4m"
	ansiGreen     = "\033[32m"
	ansiRed       = "\033[31m"
	ansiYellow    = "\033[33m"
	ansiCyan      = "\033[36m"
)

// Symbols used in terminal output.
const (
	SymbolOK    = "\u2713" // ✓
	SymbolErr   = "\u2717" // ✗
	SymbolWarn  = "\u2298" // ⊘
	SymbolArrow = "\u2192" // →
)

// isTTY is true when stdout is connected to a terminal.
var isTTY = isTerminal(os.Stdout.Fd())

// IsTTY reports whether stdout is connected to a terminal.
func IsTTY() bool { return isTTY }

// SetTTY overrides terminal detection for testing.
func SetTTY(v bool) { isTTY = v }

// styled wraps text in ANSI codes when output is a terminal.
func styled(codes, text string) string {
	if !isTTY {
		return text
	}
	return codes + text + ansiReset
}

// StatusSymbol returns the appropriate colored symbol for a result status.
func StatusSymbol(status string) string {
	switch status {
	case "ok":
		return styled(ansiBold+ansiGreen, SymbolOK)
	case "warn":
		return styled(ansiBold+ansiYellow, SymbolWarn)
	default:
		return styled(ansiBold+ansiRed, SymbolErr)
	}
}

// DryRunSymbol returns the dry-run symbol (yellow ⊘).
func DryRunSymbol() string {
	return styled(ansiBold+ansiYellow, SymbolWarn)
}

// EnabledSymbol returns ✓ (green) for true, ✗ (red) for false.
func EnabledSymbol(enabled bool) string {
	if enabled {
		return styled(ansiGreen, SymbolOK)
	}
	return styled(ansiRed, SymbolErr)
}

// Green returns green-colored text.
func Green(s string) string { return styled(ansiGreen, s) }

// Red returns red-colored text.
func Red(s string) string { return styled(ansiRed, s) }

// Yellow returns yellow-colored text.
func Yellow(s string) string { return styled(ansiYellow, s) }

// Cyan returns cyan-colored text.
func Cyan(s string) string { return styled(ansiCyan, s) }

// Dim returns dimmed text.
func Dim(s string) string { return styled(ansiDim, s) }

// Bold returns bold text.
func Bold(s string) string { return styled(ansiBold, s) }

// BoldUnderline returns bold and underlined text.
func BoldUnderline(s string) string { return styled(ansiBold+ansiUnderline, s) }
