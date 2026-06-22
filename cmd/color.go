package cmd

import (
	"os"

	"golang.org/x/term"
)

// colorEnabled reports whether ANSI color should be emitted: stdout is a TTY and
// NO_COLOR is unset (https://no-color.org).
func colorEnabled() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiDim    = "\033[2m"
)

// colorize wraps s in an ANSI code when color is enabled, else returns s as-is.
func colorize(code, s string) string {
	if !colorEnabled() {
		return s
	}
	return code + s + ansiReset
}
