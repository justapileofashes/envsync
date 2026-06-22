package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/justapileofashes/envsync/internal/config"
	"github.com/justapileofashes/envsync/internal/env"
	"github.com/spf13/cobra"
)

var (
	diffShowValues bool
	diffFile       string
	diffExitCode   bool
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare the local env file against the latest remote version",
	Long: `Compares your local environment file against the latest pushed (remote)
version and prints a color-coded diff:

  + key   present locally but missing on remote
  - key   present on remote but missing locally
  ~ key   present on both but the value differs

Secret values are masked by default; pass --values to reveal them. Use
--exit-code to make the command exit non-zero when differences exist (useful in
CI to catch drift between environments).`,
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().BoolVar(&diffShowValues, "values", false, "reveal secret values in the diff (default: masked)")
	diffCmd.Flags().StringVar(&diffFile, "file", "", "local file to compare (default: the init env file)")
	diffCmd.Flags().BoolVar(&diffExitCode, "exit-code", false, "exit non-zero if there are differences")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(_ *cobra.Command, _ []string) error {
	client, creds, err := authedClient()
	if err != nil {
		return err
	}
	ws, err := config.LoadWorkspace()
	if err != nil {
		return err
	}

	localFile := diffFile
	if localFile == "" {
		localFile = ws.EnvFile
	}
	local, err := env.Read(localFile)
	if err != nil {
		return err
	}

	ctx := context.Background()
	remote, latest, err := fetchLatestPayload(ctx, client, creds, ws)
	if err != nil {
		return err
	}

	d := env.Diff(local.Values, remote.Values)

	fmt.Printf("%s  (local: %s   vs   remote: v%d)\n\n",
		colorize(ansiDim, "envsync diff"), localFile, latest.VersionSequence)

	if d.Identical() {
		fmt.Println(colorize(ansiGreen, "✓ in sync — no differences"))
		return nil
	}

	for _, k := range d.OnlyLeft {
		fmt.Println(colorize(ansiGreen, "+ "+render(k, d.LeftVals[k])) + colorize(ansiDim, "   (local only)"))
	}
	for _, k := range d.OnlyRight {
		fmt.Println(colorize(ansiRed, "- "+render(k, d.RightVals[k])) + colorize(ansiDim, "   (missing locally)"))
	}
	for _, k := range d.Changed {
		if diffShowValues {
			fmt.Printf("%s\n", colorize(ansiYellow, "~ "+k))
			fmt.Printf("    local : %s\n", d.LeftVals[k])
			fmt.Printf("    remote: %s\n", d.RightVals[k])
		} else {
			fmt.Println(colorize(ansiYellow, "~ "+k) + colorize(ansiDim, "   (value differs)"))
		}
	}

	fmt.Printf("\n%d added, %d missing, %d changed, %d unchanged\n",
		len(d.OnlyLeft), len(d.OnlyRight), len(d.Changed), len(d.Same))

	if diffExitCode {
		os.Exit(2)
	}
	return nil
}

// render shows "KEY=value" when values are revealed, else "KEY" with a mask.
func render(key, value string) string {
	if diffShowValues {
		return key + "=" + value
	}
	return key + "=" + mask(value)
}

// mask hides a secret's content while hinting at its length.
func mask(v string) string {
	if v == "" {
		return "∅"
	}
	if len(v) <= 4 {
		return "••••"
	}
	return v[:2] + "••••" + v[len(v)-2:]
}
