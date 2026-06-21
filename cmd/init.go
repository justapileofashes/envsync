package cmd

import (
	"context"
	"fmt"

	"github.com/envsync/envsync/internal/config"
	"github.com/spf13/cobra"
)

var initEnvFile string

var initCmd = &cobra.Command{
	Use:   "init <project_id>",
	Short: "Link the current directory to a remote project",
	Long: `Creates a .envsync.json file in the current directory mapping it to the
given Supabase project_id. It also fetches and caches the organization's
cryptographic_salt, which is required for local key derivation during push/pull.`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initEnvFile, "env-file", ".env", "path to the dotenv file this project syncs")
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, args []string) error {
	projectID := args[0]

	client, _, err := authedClient()
	if err != nil {
		return err
	}
	ctx := context.Background()

	fmt.Printf("Resolving project %s…\n", projectID)
	project, err := client.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	org, err := client.GetOrganization(ctx, project.OrgID)
	if err != nil {
		return err
	}
	if org.CryptographicSalt == "" {
		return fmt.Errorf("organization %q has no cryptographic_salt configured", org.ID)
	}

	ws := &config.Workspace{
		ProjectID: project.ID,
		OrgID:     project.OrgID,
		Salt:      org.CryptographicSalt,
		EnvFile:   initEnvFile,
	}
	if err := config.SaveWorkspace(ws); err != nil {
		return err
	}

	fmt.Printf("Initialized %s\n", config.WorkspaceFileName)
	fmt.Printf("  project : %s (%s)\n", project.Name, project.ID)
	fmt.Printf("  org     : %s\n", org.Name)
	fmt.Printf("  env file: %s\n", initEnvFile)
	fmt.Println("\nNext: `envsync push` to upload, or `envsync pull` to download.")
	return nil
}
