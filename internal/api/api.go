// Package api is a thin, defensively-programmed wrapper over the Supabase REST
// and Auth APIs used by EnvSync.
//
// It speaks two protocols:
//
//   - GoTrue auth   : POST /auth/v1/token?grant_type=password
//   - PostgREST     : /rest/v1/<table> with apikey + bearer JWT
//
// Every payload that crosses the wire is already encrypted; this layer never
// sees plaintext secrets and never transmits the cryptographic passphrase.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to a single Supabase project.
type Client struct {
	baseURL    string
	anonKey    string
	token      string // user JWT; empty for unauthenticated calls
	httpClient *http.Client
}

// New constructs a Client. baseURL is the Supabase project URL
// (e.g. https://abc.supabase.co). anonKey is the public anon API key.
func New(baseURL, anonKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		anonKey:    anonKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetToken sets the bearer JWT used for authenticated PostgREST calls.
func (c *Client) SetToken(token string) { c.token = token }

// apiError renders a human-readable error from a non-2xx Supabase response.
func apiError(action string, status int, body []byte) error {
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = http.StatusText(status)
	}
	// Try to surface PostgREST/GoTrue structured messages cleanly.
	var structured struct {
		Message          string `json:"message"`
		Msg              string `json:"msg"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Hint             string `json:"hint"`
	}
	if json.Unmarshal(body, &structured) == nil {
		for _, candidate := range []string{
			structured.ErrorDescription, structured.Message, structured.Msg, structured.Error,
		} {
			if candidate != "" {
				msg = candidate
				if structured.Hint != "" {
					msg += " (" + structured.Hint + ")"
				}
				break
			}
		}
	}
	return fmt.Errorf("%s failed [HTTP %d]: %s", action, status, msg)
}

// -----------------------------------------------------------------------------
// Auth
// -----------------------------------------------------------------------------

// Session is the subset of a GoTrue token response EnvSync persists.
type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

// Login authenticates with email/password against GoTrue and returns a Session.
func (c *Client) Login(ctx context.Context, email, password string) (*Session, error) {
	endpoint := c.baseURL + "/auth/v1/token?grant_type=password"
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("login: cannot build request: %w", err)
	}
	req.Header.Set("apikey", c.anonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login: network error contacting Supabase: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apiError("login", resp.StatusCode, raw)
	}
	var s Session
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("login: malformed token response: %w", err)
	}
	if s.AccessToken == "" {
		return nil, fmt.Errorf("login: Supabase returned an empty access token")
	}
	return &s, nil
}

// -----------------------------------------------------------------------------
// PostgREST helpers
// -----------------------------------------------------------------------------

// rest performs a PostgREST request against /rest/v1/<path>.
func (c *Client) rest(ctx context.Context, method, path string, query url.Values, body io.Reader, prefer string) ([]byte, error) {
	endpoint := c.baseURL + "/rest/v1/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("request build error: %w", err)
	}
	req.Header.Set("apikey", c.anonKey)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error contacting Supabase: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apiError(method+" "+path, resp.StatusCode, raw)
	}
	return raw, nil
}

// -----------------------------------------------------------------------------
// Domain models
// -----------------------------------------------------------------------------

// Project is the subset of the projects table EnvSync reads.
type Project struct {
	ID    string `json:"id"`
	OrgID string `json:"org_id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
}

// Organization is the subset of the organizations table EnvSync reads.
type Organization struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	CryptographicSalt string `json:"cryptographic_salt"`
}

// Environment is one versioned encrypted blob row.
type Environment struct {
	ID              string `json:"id"`
	ProjectID       string `json:"project_id"`
	VersionSequence int    `json:"version_sequence"`
	Ciphertext      string `json:"ciphertext"`
	Checksum        string `json:"checksum,omitempty"`
	CreatedBy       string `json:"created_by,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
}

// GetProject fetches a single project by id.
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	q := url.Values{}
	q.Set("id", "eq."+projectID)
	q.Set("select", "id,org_id,name,slug")
	q.Set("limit", "1")

	raw, err := c.rest(ctx, http.MethodGet, "projects", q, nil, "")
	if err != nil {
		return nil, err
	}
	var projects []Project
	if err := json.Unmarshal(raw, &projects); err != nil {
		return nil, fmt.Errorf("get project: malformed response: %w", err)
	}
	if len(projects) == 0 {
		return nil, fmt.Errorf("project %q not found or you lack access", projectID)
	}
	return &projects[0], nil
}

// GetOrganization fetches a single organization by id (for its salt).
func (c *Client) GetOrganization(ctx context.Context, orgID string) (*Organization, error) {
	q := url.Values{}
	q.Set("id", "eq."+orgID)
	q.Set("select", "id,name,cryptographic_salt")
	q.Set("limit", "1")

	raw, err := c.rest(ctx, http.MethodGet, "organizations", q, nil, "")
	if err != nil {
		return nil, err
	}
	var orgs []Organization
	if err := json.Unmarshal(raw, &orgs); err != nil {
		return nil, fmt.Errorf("get organization: malformed response: %w", err)
	}
	if len(orgs) == 0 {
		return nil, fmt.Errorf("organization %q not found or you lack access", orgID)
	}
	return &orgs[0], nil
}

// GetLatestEnvironment returns the highest-versioned environment for a project,
// or (nil, nil) when none exists yet.
func (c *Client) GetLatestEnvironment(ctx context.Context, projectID string) (*Environment, error) {
	q := url.Values{}
	q.Set("project_id", "eq."+projectID)
	q.Set("select", "id,project_id,version_sequence,ciphertext,checksum,created_by,created_at")
	q.Set("order", "version_sequence.desc")
	q.Set("limit", "1")

	raw, err := c.rest(ctx, http.MethodGet, "environments", q, nil, "")
	if err != nil {
		return nil, err
	}
	var envs []Environment
	if err := json.Unmarshal(raw, &envs); err != nil {
		return nil, fmt.Errorf("get latest environment: malformed response: %w", err)
	}
	if len(envs) == 0 {
		return nil, nil
	}
	return &envs[0], nil
}

// InsertEnvironment appends a new encrypted version row and returns it.
func (c *Client) InsertEnvironment(ctx context.Context, env *Environment) (*Environment, error) {
	payload := map[string]any{
		"project_id":       env.ProjectID,
		"version_sequence": env.VersionSequence,
		"ciphertext":       env.Ciphertext,
	}
	if env.Checksum != "" {
		payload["checksum"] = env.Checksum
	}
	if env.CreatedBy != "" {
		payload["created_by"] = env.CreatedBy
	}
	body, _ := json.Marshal(payload)

	raw, err := c.rest(ctx, http.MethodPost, "environments", nil, bytes.NewReader(body), "return=representation")
	if err != nil {
		return nil, err
	}
	var inserted []Environment
	if err := json.Unmarshal(raw, &inserted); err != nil {
		return nil, fmt.Errorf("insert environment: malformed response: %w", err)
	}
	if len(inserted) == 0 {
		return nil, fmt.Errorf("insert environment: server returned no row (RLS may have blocked the write)")
	}
	return &inserted[0], nil
}

// NextVersion computes the next version_sequence for a project (latest+1, or 1).
func (c *Client) NextVersion(ctx context.Context, projectID string) (int, error) {
	latest, err := c.GetLatestEnvironment(ctx, projectID)
	if err != nil {
		return 0, err
	}
	if latest == nil {
		return 1, nil
	}
	return latest.VersionSequence + 1, nil
}
