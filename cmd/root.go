// Package cmd wires up EnvSync's Cobra command tree.
package cmd

import (
	"fmt"
	"os"

	"github.com/justapileofashes/envsync/internal/api"
	"github.com/justapileofashes/envsync/internal/config"
	"github.com/spf13/cobra"
)

// version is stamped at build time via -ldflags "-X .../cmd.version=...".
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "envsync",
	Short: "EnvSync — zero-knowledge, terminal-first .env synchronization",
	Long: `EnvSync keeps a team's .env files in sync through an end-to-end
encrypted store. Secrets are encrypted on your machine with AES-256-GCM using a
key derived from a shared passphrase that never leaves your computer.

Typical flow:
  envsync login                 # authenticate + set your team passphrase
  envsync init <project_id>     # link this directory to a remote project
  envsync push                  # encrypt and upload the local .env
  envsync pull                  # download and decrypt the latest .env`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version,
}

// Execute runs the root command and converts any error into a clean,
// human-readable terminal message plus a non-zero exit code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		os.Exit(1)
	}
}

// authedClient loads credentials and returns a Client with the JWT applied,
// alongside the loaded credentials for downstream use.
func authedClient() (*api.Client, *config.Credentials, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, nil, err
	}
	if creds.SupabaseURL == "" || creds.AnonKey == "" {
		return nil, nil, fmt.Errorf("credentials are missing the Supabase URL/anon key: re-run `envsync login`")
	}
	client := api.New(creds.SupabaseURL, creds.AnonKey)
	client.SetToken(creds.AccessToken)
	return client, creds, nil
}
