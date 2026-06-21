package cmd

import (
	"context"
	"fmt"

	"github.com/justapileofashes/envsync/internal/api"
	"github.com/justapileofashes/envsync/internal/config"
	"github.com/spf13/cobra"
)

var (
	loginURL      string
	loginAnonKey  string
	loginEmail    string
	loginNonInter bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Supabase and set your team passphrase",
	Long: `Authenticates against Supabase with email/password and stores the
returned JWT in ~/.envsync/credentials.json (mode 0600).

You will also be asked for the team's Cryptographic Passphrase. This passphrase
is the input to local key derivation and is stored ONLY on this machine — it is
never transmitted to the server.`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().StringVar(&loginURL, "url", "", "Supabase project URL (e.g. https://abc.supabase.co)")
	loginCmd.Flags().StringVar(&loginAnonKey, "anon-key", "", "Supabase anon API key")
	loginCmd.Flags().StringVar(&loginEmail, "email", "", "account email (prompted if omitted)")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, _ []string) error {
	// Carry over previously-saved URL/anon key so users need not retype them.
	existing, _ := config.LoadCredentials()

	url := firstNonEmpty(loginURL, valueOr(existing, func(c *config.Credentials) string { return c.SupabaseURL }))
	if url == "" {
		var err error
		if url, err = promptLine("Supabase URL: "); err != nil {
			return err
		}
	}
	anonKey := firstNonEmpty(loginAnonKey, valueOr(existing, func(c *config.Credentials) string { return c.AnonKey }))
	if anonKey == "" {
		var err error
		if anonKey, err = promptSecret("Supabase anon key: "); err != nil {
			return err
		}
	}

	email := loginEmail
	if email == "" {
		var err error
		if email, err = promptLine("Email: "); err != nil {
			return err
		}
	}
	password, err := promptSecret("Password: ")
	if err != nil {
		return err
	}
	if email == "" || password == "" {
		return fmt.Errorf("email and password are required")
	}

	client := api.New(url, anonKey)
	fmt.Println("Authenticating…")
	session, err := client.Login(context.Background(), email, password)
	if err != nil {
		return err
	}

	fmt.Println("Authenticated as", session.User.Email)
	fmt.Println()
	fmt.Println("Now set the team Cryptographic Passphrase.")
	fmt.Println("This is used to derive your encryption key locally and is NEVER sent to the server.")
	passphrase, err := promptSecret("Cryptographic passphrase: ")
	if err != nil {
		return err
	}
	if passphrase == "" {
		return fmt.Errorf("the cryptographic passphrase must not be empty")
	}
	confirm, err := promptSecret("Confirm passphrase: ")
	if err != nil {
		return err
	}
	if passphrase != confirm {
		return fmt.Errorf("passphrases do not match")
	}

	creds := &config.Credentials{
		SupabaseURL:  url,
		AnonKey:      anonKey,
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		UserID:       session.User.ID,
		Email:        session.User.Email,
		Passphrase:   passphrase,
	}
	if err := config.SaveCredentials(creds); err != nil {
		return err
	}

	fmt.Println("\nLogged in. Credentials saved to ~/.envsync/credentials.json (mode 0600).")
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func valueOr(c *config.Credentials, f func(*config.Credentials) string) string {
	if c == nil {
		return ""
	}
	return f(c)
}
