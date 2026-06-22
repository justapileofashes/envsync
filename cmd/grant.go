package cmd

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/justapileofashes/envsync/internal/api"
	"github.com/justapileofashes/envsync/internal/config"
	"github.com/spf13/cobra"
)

var (
	grantRole    string
	grantExpires string
	grantProject string
)

var grantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Mint a time-limited, scoped access token for a contractor",
	Long: `Generates a scoped, time-limited access token for the current project —
ideal for handing a freelancer temporary access that auto-expires.

Only the SHA-256 hash of the token is stored server-side; the raw token is shown
to you exactly once here. After --expires elapses (or you revoke it), the token
stops working automatically — no need to remember to revoke access manually.

Examples:
  envsync grant --role read-only --expires 48h
  envsync grant --role developer --expires 7d`,
	RunE: runGrant,
}

func init() {
	grantCmd.Flags().StringVar(&grantRole, "role", "read-only", "grant role: read-only or developer")
	grantCmd.Flags().StringVar(&grantExpires, "expires", "48h", "lifetime, e.g. 90m, 48h, 7d")
	grantCmd.Flags().StringVar(&grantProject, "project", "", "project id (default: this workspace's project)")
	rootCmd.AddCommand(grantCmd)
}

func runGrant(_ *cobra.Command, _ []string) error {
	if grantRole != "read-only" && grantRole != "developer" {
		return fmt.Errorf("invalid --role %q: must be read-only or developer", grantRole)
	}
	lifetime, err := parseDuration(grantExpires)
	if err != nil {
		return err
	}

	client, creds, err := authedClient()
	if err != nil {
		return err
	}

	projectID := grantProject
	if projectID == "" {
		ws, err := config.LoadWorkspace()
		if err != nil {
			return err
		}
		projectID = ws.ProjectID
	}

	// Generate a high-entropy token; store only its hash.
	rawToken, err := newToken()
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(rawToken))
	expiresAt := time.Now().UTC().Add(lifetime)

	grant, err := client.InsertGrant(context.Background(), &api.Grant{
		ProjectID: projectID,
		TokenHash: hex.EncodeToString(sum[:]),
		Role:      grantRole,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		CreatedBy: creds.UserID,
	})
	if err != nil {
		return err
	}

	fmt.Println(colorize(ansiGreen, "✓ Access grant created"))
	fmt.Printf("  role    : %s\n", grant.Role)
	fmt.Printf("  expires : %s (in %s)\n", expiresAt.Format(time.RFC3339), grantExpires)
	fmt.Printf("  project : %s\n", projectID)
	fmt.Println()
	fmt.Println(colorize(ansiYellow, "  Share this token now — it is shown only once:"))
	fmt.Printf("    %s\n", rawToken)
	fmt.Println()
	fmt.Println(colorize(ansiDim, "  Server enforces expiry/revocation via redeem_access_grant(); revoke early"))
	fmt.Println(colorize(ansiDim, "  by setting revoked=true on the row in the dashboard."))
	return nil
}

// newToken returns a URL-safe, 256-bit random token.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("could not generate token: %w", err)
	}
	return "envg_" + base64.RawURLEncoding.EncodeToString(b), nil
}

// parseDuration extends time.ParseDuration with a "d" (days) suffix.
func parseDuration(s string) (time.Duration, error) {
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s[:len(s)-1], "%d", &days); err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid --expires %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid --expires %q (try 90m, 48h, 7d)", s)
	}
	return d, nil
}
