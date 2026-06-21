// Package config manages EnvSync's local state on disk.
//
// Two distinct stores exist:
//
//   - Global credentials  (~/.envsync/credentials.json): the Supabase JWT and
//     the team cryptographic passphrase. Written with 0600 permissions. The
//     passphrase NEVER leaves this file / the local machine.
//   - Workspace state     (./.envsync.json): maps the current directory to a
//     remote project, and caches the org's key-derivation salt.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	credentialsDirName  = ".envsync"
	credentialsFileName = "credentials.json"
	// WorkspaceFileName is the per-directory state file created by `envsync init`.
	WorkspaceFileName = ".envsync.json"
)

// Credentials is the global, machine-local auth + crypto material.
type Credentials struct {
	// SupabaseURL and AnonKey identify the backend this user authenticated against.
	SupabaseURL  string `json:"supabase_url"`
	AnonKey      string `json:"anon_key"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	// Passphrase is the team's cryptographic passphrase. Local only.
	Passphrase string `json:"passphrase"`
}

// Workspace is the per-project state recorded by `envsync init`.
type Workspace struct {
	ProjectID string `json:"project_id"`
	OrgID     string `json:"org_id"`
	// Salt is base64-encoded; cached so push/pull need not re-fetch it.
	Salt    string `json:"cryptographic_salt"`
	EnvFile string `json:"env_file"`
}

// ErrNotLoggedIn signals that no credentials file exists yet.
var ErrNotLoggedIn = errors.New("not logged in: run `envsync login` first")

// ErrNoWorkspace signals that the current directory is not initialized.
var ErrNoWorkspace = errors.New("no .envsync.json found: run `envsync init <project_id>` here first")

// credentialsPath resolves ~/.envsync/credentials.json.
func credentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: cannot resolve home directory: %w", err)
	}
	return filepath.Join(home, credentialsDirName, credentialsFileName), nil
}

// SaveCredentials writes the global credentials file with 0600 permissions,
// creating ~/.envsync if needed.
func SaveCredentials(c *Credentials) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: cannot create credentials directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: cannot encode credentials: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("config: cannot write credentials: %w", err)
	}
	return nil
}

// LoadCredentials reads the global credentials file.
func LoadCredentials() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotLoggedIn
		}
		return nil, fmt.Errorf("config: cannot read credentials: %w", err)
	}
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config: corrupt credentials file: %w", err)
	}
	return &c, nil
}

// SaveWorkspace writes ./.envsync.json in the current directory.
func SaveWorkspace(w *Workspace) error {
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return fmt.Errorf("config: cannot encode workspace: %w", err)
	}
	if err := os.WriteFile(WorkspaceFileName, data, 0o644); err != nil {
		return fmt.Errorf("config: cannot write %s: %w", WorkspaceFileName, err)
	}
	return nil
}

// LoadWorkspace reads ./.envsync.json from the current directory.
func LoadWorkspace() (*Workspace, error) {
	data, err := os.ReadFile(WorkspaceFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoWorkspace
		}
		return nil, fmt.Errorf("config: cannot read %s: %w", WorkspaceFileName, err)
	}
	var w Workspace
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("config: corrupt %s: %w", WorkspaceFileName, err)
	}
	if w.EnvFile == "" {
		w.EnvFile = ".env"
	}
	return &w, nil
}
