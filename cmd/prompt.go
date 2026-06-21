package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// promptLine prints a label and reads a trimmed line from stdin.
func promptLine(label string) (string, error) {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("could not read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// promptSecret reads a line without echoing it to the terminal. It falls back to
// a normal (echoed) read when stdin is not a TTY (e.g. piped input in CI).
func promptSecret(label string) (string, error) {
	fmt.Print(label)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		raw, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("could not read secret input: %w", err)
		}
		return strings.TrimSpace(string(raw)), nil
	}
	// Non-interactive fallback.
	return promptLine("")
}
