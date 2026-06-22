package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/justapileofashes/envsync/internal/config"
	"github.com/justapileofashes/envsync/internal/detect"
	"github.com/justapileofashes/envsync/internal/env"
	"github.com/spf13/cobra"
)

var (
	pullOut        string
	pullNoDetect   bool
	pullNoOverride bool
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download and decrypt the latest .env version",
	Long: `Fetches the highest-versioned encrypted blob for this project,
decrypts it locally with a key derived from your passphrase + the org salt,
backs up any existing target file to <file>.bak, and writes the result.

By default the output filename is auto-detected from the project's framework
(e.g. .env.local for Next.js, .env.development for Vite). Override with --out,
or disable detection with --no-detect to use the file set during init.`,
	RunE: runPull,
}

func init() {
	pullCmd.Flags().StringVar(&pullOut, "out", "", "explicit output file (overrides auto-detection)")
	pullCmd.Flags().BoolVar(&pullNoDetect, "no-detect", false, "skip framework auto-detection; use the init env file")
	pullCmd.Flags().BoolVar(&pullNoOverride, "no-override", false, "do not merge local "+env.DefaultOverrideFile)
	rootCmd.AddCommand(pullCmd)
}

func runPull(_ *cobra.Command, _ []string) error {
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

	// Merge personal local overrides (never pushed back).
	if !pullNoOverride {
		n, err := payload.MergeOverride(env.DefaultOverrideFile)
		if err != nil {
			return err
		}
		if n > 0 {
			fmt.Printf("Merged %d local override(s) from %s\n", n, env.DefaultOverrideFile)
		}
	}

	// Resolve the target filename.
	target := resolvePullTarget(ws)

	// Back up existing file then write.
	backup, err := env.Backup(target)
	if err != nil {
		return err
	}
	if backup != "" {
		fmt.Printf("Backed up existing %s -> %s\n", target, backup)
	}
	if err := env.Write(target, payload); err != nil {
		return err
	}

	fmt.Printf("Pulled v%d (%d variable(s)) into %s.\n", latest.VersionSequence, len(payload.Values), target)
	return nil
}

// resolvePullTarget decides where to write the decrypted env file, honoring (in
// order): --out, --no-detect, framework auto-detection, then the workspace file.
func resolvePullTarget(ws *config.Workspace) string {
	if pullOut != "" {
		return pullOut
	}
	if pullNoDetect {
		return ws.EnvFile
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ws.EnvFile
	}
	res := detect.Detect(cwd)
	if res.Framework != "" {
		fmt.Printf("Detected %s — writing %s\n", res.Framework, res.EnvFile)
		return res.EnvFile
	}
	return ws.EnvFile
}
