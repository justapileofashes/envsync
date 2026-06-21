package cmd

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/justapileofashes/envsync/internal/config"
	"github.com/justapileofashes/envsync/internal/crypto"
	"github.com/justapileofashes/envsync/internal/env"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download and decrypt the latest .env version",
	Long: `Fetches the highest-versioned encrypted blob for this project,
decrypts it locally with a key derived from your passphrase + the org salt,
backs up any existing .env to .env.bak, and writes the decrypted result.`,
	RunE: runPull,
}

func init() {
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
	if creds.Passphrase == "" {
		return fmt.Errorf("no cryptographic passphrase set: re-run `envsync login`")
	}

	// 1. Fetch latest version.
	ctx := context.Background()
	latest, err := client.GetLatestEnvironment(ctx, ws.ProjectID)
	if err != nil {
		return err
	}
	if latest == nil {
		return fmt.Errorf("nothing to pull: this project has no pushed versions yet")
	}

	// 2. Base64 decode + decrypt.
	blob, err := base64.StdEncoding.DecodeString(latest.Ciphertext)
	if err != nil {
		return fmt.Errorf("server returned malformed ciphertext (invalid base64): %w", err)
	}
	key, err := deriveKey(creds.Passphrase, ws.Salt)
	if err != nil {
		return err
	}
	plaintext, err := crypto.Decrypt(key, blob)
	if err != nil {
		return err
	}
	payload, err := env.Unmarshal(plaintext)
	if err != nil {
		return err
	}

	// 3. Back up existing .env then write.
	backup, err := env.Backup(ws.EnvFile)
	if err != nil {
		return err
	}
	if backup != "" {
		fmt.Printf("Backed up existing %s -> %s\n", ws.EnvFile, backup)
	}
	if err := env.Write(ws.EnvFile, payload); err != nil {
		return err
	}

	fmt.Printf("Pulled v%d (%d variable(s)) into %s.\n", latest.VersionSequence, len(payload.Values), ws.EnvFile)
	return nil
}
