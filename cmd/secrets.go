package cmd

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/justapileofashes/envsync/internal/api"
	"github.com/justapileofashes/envsync/internal/config"
	"github.com/justapileofashes/envsync/internal/crypto"
	"github.com/justapileofashes/envsync/internal/env"
)

// fetchLatestPayload pulls the highest-versioned encrypted blob for the
// workspace's project and decrypts it locally into an env.Payload. It is shared
// by `pull` (writes to disk) and `run` (injects into memory only).
func fetchLatestPayload(ctx context.Context, client *api.Client, creds *config.Credentials, ws *config.Workspace) (*env.Payload, *api.Environment, error) {
	if creds.Passphrase == "" {
		return nil, nil, fmt.Errorf("no cryptographic passphrase set: re-run `envsync login`")
	}

	latest, err := client.GetLatestEnvironment(ctx, ws.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if latest == nil {
		return nil, nil, fmt.Errorf("nothing to pull: this project has no pushed versions yet")
	}

	blob, err := base64.StdEncoding.DecodeString(latest.Ciphertext)
	if err != nil {
		return nil, nil, fmt.Errorf("server returned malformed ciphertext (invalid base64): %w", err)
	}
	key, err := deriveKey(creds.Passphrase, ws.Salt)
	if err != nil {
		return nil, nil, err
	}
	plaintext, err := crypto.Decrypt(key, blob)
	if err != nil {
		return nil, nil, err
	}
	payload, err := env.Unmarshal(plaintext)
	if err != nil {
		return nil, nil, err
	}
	return payload, latest, nil
}
