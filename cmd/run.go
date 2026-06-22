package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/justapileofashes/envsync/internal/config"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run -- <command> [args...]",
	Short: "Run a command with secrets injected into memory (zero-disk)",
	Long: `Decrypts the latest environment and injects the variables directly into
the child process's environment — the secrets are NEVER written to disk. They
exist only in RAM for the lifetime of the command and vanish when it exits.

Examples:
  envsync run -- npm run dev
  envsync run -- go run main.go
  envsync run -- printenv DATABASE_URL`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE:               runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	// Strip a leading "--" separator if present.
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return fmt.Errorf("no command given: usage `envsync run -- <command> [args...]`")
	}
	// Allow `envsync run --help` to behave normally.
	if args[0] == "-h" || args[0] == "--help" {
		return cmd.Help()
	}

	client, creds, err := authedClient()
	if err != nil {
		return err
	}
	ws, err := config.LoadWorkspace()
	if err != nil {
		return err
	}

	ctx := context.Background()
	payload, latest, err := fetchLatestPayload(ctx, client, creds, ws)
	if err != nil {
		return err
	}

	// Build the child environment: inherit the parent, then layer the decrypted
	// secrets on top (overriding any same-named parent vars).
	childEnv := mergeEnv(os.Environ(), payload.Values)

	fmt.Fprintf(os.Stderr, "envsync: injected %d variable(s) from v%d into `%s` (zero-disk)\n",
		len(payload.Values), latest.VersionSequence, args[0])

	proc := exec.Command(args[0], args[1:]...)
	proc.Env = childEnv
	proc.Stdin = os.Stdin
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr

	if err := proc.Run(); err != nil {
		// Propagate the child's exit code transparently.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("command not found: %s", args[0])
		}
		return fmt.Errorf("failed to run %q: %w", args[0], err)
	}
	return nil
}

// mergeEnv overlays key=value pairs onto a copy of base, replacing any existing
// definition of the same key (case-sensitive, matching OS env semantics).
func mergeEnv(base []string, overlay map[string]string) []string {
	merged := make(map[string]string, len(base)+len(overlay))
	for _, kv := range base {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				merged[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	for k, v := range overlay {
		merged[k] = v
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(merged))
	for _, k := range keys {
		out = append(out, k+"="+merged[k])
	}
	return out
}
