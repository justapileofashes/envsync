package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/justapileofashes/envsync/internal/api"
	"github.com/justapileofashes/envsync/internal/config"
	"github.com/justapileofashes/envsync/internal/crypto"
	"github.com/justapileofashes/envsync/internal/env"
	"github.com/justapileofashes/envsync/internal/schema"
	"github.com/spf13/cobra"
)

var pushSkipSchema bool

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Encrypt the local .env and upload it as a new version",
	Long: `Reads the local dotenv file, serializes it, encrypts it with a key
derived locally from your passphrase + the org salt, base64-encodes the
ciphertext, and uploads it as the next version in the project's history.

If a .env.schema file is present, the local file is validated against it first;
the push is blocked on missing required keys, prefix/regex violations, or likely
typos of required keys. Use --skip-schema to bypass.`,
	RunE: runPush,
}

func init() {
	pushCmd.Flags().BoolVar(&pushSkipSchema, "skip-schema", false, "skip .env.schema validation")
	rootCmd.AddCommand(pushCmd)
}

func runPush(_ *cobra.Command, _ []string) error {
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

	// 1. Read + serialize the local .env.
	payload, err := env.Read(ws.EnvFile)
	if err != nil {
		return err
	}

	// 1a. Validate against .env.schema if present (typo-squashing + constraints).
	if !pushSkipSchema && schema.Exists(schema.DefaultFile) {
		sc, err := schema.Load(schema.DefaultFile)
		if err != nil {
			return err
		}
		if violations := sc.Validate(payload.Values); len(violations) > 0 {
			fmt.Fprintln(os.Stderr, colorize(ansiRed, fmt.Sprintf("✗ %s validation failed:", schema.DefaultFile)))
			for _, v := range violations {
				fmt.Fprintf(os.Stderr, "  %s %s\n", colorize(ansiRed, "•"), v.String())
			}
			return fmt.Errorf("push blocked by schema validation (%d issue(s)); fix the file or pass --skip-schema", len(violations))
		}
	}

	plaintext, err := payload.Marshal()
	if err != nil {
		return err
	}

	// 2. Derive key and encrypt.
	key, err := deriveKey(creds.Passphrase, ws.Salt)
	if err != nil {
		return err
	}
	blob, err := crypto.Encrypt(key, plaintext)
	if err != nil {
		return err
	}
	ciphertext := base64.StdEncoding.EncodeToString(blob)

	// 3. Determine next version and insert.
	ctx := context.Background()
	nextVersion, err := client.NextVersion(ctx, ws.ProjectID)
	if err != nil {
		return err
	}

	sum := sha256.Sum256(plaintext)
	inserted, err := client.InsertEnvironment(ctx, &api.Environment{
		ProjectID:       ws.ProjectID,
		VersionSequence: nextVersion,
		Ciphertext:      ciphertext,
		Checksum:        hex.EncodeToString(sum[:]),
		CreatedBy:       creds.UserID,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Pushed %d variable(s) as v%d.\n", len(payload.Values), inserted.VersionSequence)
	return nil
}

// deriveKey decodes a base64 salt and derives the AES-256 key.
func deriveKey(passphrase, b64Salt string) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(b64Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid organization salt (not valid base64): %w", err)
	}
	return crypto.DeriveKey(passphrase, salt)
}
