// Package pcf provides a client for interacting with the
// Pentest Collaboration Framework API
package pcf

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/aRustyDev/pcf-mcp/internal/config"
)

// Client represents a PCF API client
type Client struct {
	// baseURL is the base URL of the PCF instance
	baseURL string

	// httpClient is the underlying HTTP client
	httpClient *http.Client

	// apiKey is the authentication key for PCF API
	apiKey string

	// maxRetries is the maximum number of retry attempts
	maxRetries int
}

// Project represents a PCF project
type Project struct {
	// ID is the unique identifier of the project
	ID string `json:"id"`

	// Name is the project name
	Name string `json:"name"`

	// Description provides details about the project
	Description string `json:"description"`

	// CreatedAt is the project creation timestamp
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the last update timestamp
	UpdatedAt time.Time `json:"updated_at"`

	// Status indicates the project status
	Status string `json:"status,omitempty"`

	// Team members associated with the project
	Team []string `json:"team,omitempty"`
}

// Host represents a target host in PCF
type Host struct {
	// ID is the unique identifier of the host
	ID string `json:"id"`

	// ProjectID is the associated project ID
	ProjectID string `json:"project_id"`

	// IP is the host IP address
	IP string `json:"ip"`

	// Hostname is the host's DNS name
	Hostname string `json:"hostname,omitempty"`

	// OS is the operating system
	OS string `json:"os,omitempty"`

	// Services is a list of discovered services
	Services []string `json:"services,omitempty"`

	// Status indicates if the host is active
	Status string `json:"status,omitempty"`
}

// Issue represents a security issue or finding
type Issue struct {
	// ID is the unique identifier of the issue
	ID string `json:"id"`

	// ProjectID is the associated project ID
	ProjectID string `json:"project_id"`

	// HostID is the associated host ID (if applicable)
	HostID string `json:"host_id,omitempty"`

	// Title is the issue title
	Title string `json:"title"`

	// Description provides issue details
	Description string `json:"description"`

	// Severity indicates the issue severity (Critical, High, Medium, Low, Info)
	Severity string `json:"severity"`

	// Status indicates the issue status (Open, In Progress, Resolved, Closed)
	Status string `json:"status"`

	// CVE is the associated CVE identifier (if applicable)
	CVE string `json:"cve,omitempty"`

	// CVSS is the CVSS score (if applicable)
	CVSS float64 `json:"cvss,omitempty"`
}

// Credential represents stored credentials
type Credential struct {
	// ID is the unique identifier
	ID string `json:"id"`

	// ProjectID is the associated project ID
	ProjectID string `json:"project_id"`

	// HostID is the associated host ID (if applicable)
	HostID string `json:"host_id,omitempty"`

	// Type indicates the credential type (password, hash, key, etc.)
	Type string `json:"type"`

	// Username is the username
	Username string `json:"username"`

	// Value is the credential value (encrypted in storage)
	Value string `json:"value"`

	// Service is the associated service
	Service string `json:"service,omitempty"`

	// Notes provides additional context
	Notes string `json:"notes,omitempty"`
}

// CreateProjectRequest represents a request to create a new project
type CreateProjectRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Team        []string `json:"team,omitempty"`
}

// CreateHostRequest represents a request to add a new host
type CreateHostRequest struct {
	IP       string   `json:"ip"`
	Hostname string   `json:"hostname,omitempty"`
	OS       string   `json:"os,omitempty"`
	Services []string `json:"services,omitempty"`
}

// CreateIssueRequest represents a request to create a new issue
type CreateIssueRequest struct {
	HostID      string  `json:"host_id,omitempty"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	CVE         string  `json:"cve,omitempty"`
	CVSS        float64 `json:"cvss,omitempty"`
}

// AddCredentialRequest represents a request to add a new credential
type AddCredentialRequest struct {
	HostID   string `json:"host_id,omitempty"`
	Type     string `json:"type"`
	Username string `json:"username"`
	Value    string `json:"value"`
	Service  string `json:"service,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

// GenerateReportRequest represents a request to generate a report
type GenerateReportRequest struct {
	Format             string   `json:"format"`
	IncludeHosts       bool     `json:"include_hosts"`
	IncludeIssues      bool     `json:"include_issues"`
	IncludeCredentials bool     `json:"include_credentials"`
	Sections           []string `json:"sections,omitempty"`
}

// Report represents a generated report
type Report struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Format    string    `json:"format"`
	Status    string    `json:"status"`
	URL       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Size      int64     `json:"size,omitempty"`
}

// ErrorResponse represents an error response from PCF API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// NewClient creates a new PCF API client
func NewClient(cfg config.PCFConfig) (*Client, error) {
	// Validate URL
	if cfg.URL == "" {
		return nil, fmt.Errorf("PCF URL is required")
	}

	// Parse URL to validate it
	_, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid PCF URL: %w", err)
	}

	// Configure HTTP client
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
	}

	// Configure transport with TLS settings if needed
	if cfg.InsecureSkipVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		httpClient.Transport = transport
	}

	client := &Client{
		baseURL:    cfg.URL,
		httpClient: httpClient,
		apiKey:     cfg.APIKey,
		maxRetries: cfg.MaxRetries,
	}

	return client, nil
}

// BaseURL returns the client's base URL
func (c *Client) BaseURL() string {
	return c.baseURL
}

// ListProjects retrieves all projects from PCF
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	err := c.doRequest(ctx, "GET", "/api/projects", nil, &projects)
	return projects, err
}

// GetProject retrieves a specific project by ID
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	var project Project
	path := fmt.Sprintf("/api/projects/%s", projectID)
	err := c.doRequest(ctx, "GET", path, nil, &project)
	return &project, err
}

// CreateProject creates a new project in PCF
func (c *Client) CreateProject(ctx context.Context, req CreateProjectRequest) (*Project, error) {
	var project Project
	err := c.doRequest(ctx, "POST", "/api/projects", req, &project)
	return &project, err
}

// ListHosts retrieves all hosts for a project
func (c *Client) ListHosts(ctx context.Context, projectID string) ([]Host, error) {
	var hosts []Host
	path := fmt.Sprintf("/api/projects/%s/hosts", projectID)
	err := c.doRequest(ctx, "GET", path, nil, &hosts)
	return hosts, err
}

// AddHost adds a new host to a project
func (c *Client) AddHost(ctx context.Context, projectID string, req CreateHostRequest) (*Host, error) {
	var host Host
	path := fmt.Sprintf("/api/projects/%s/hosts", projectID)
	err := c.doRequest(ctx, "POST", path, req, &host)
	return &host, err
}

// ListIssues retrieves all issues for a project
func (c *Client) ListIssues(ctx context.Context, projectID string) ([]Issue, error) {
	var issues []Issue
	path := fmt.Sprintf("/api/projects/%s/issues", projectID)
	err := c.doRequest(ctx, "GET", path, nil, &issues)
	return issues, err
}

// CreateIssue creates a new issue in a project
func (c *Client) CreateIssue(ctx context.Context, projectID string, req CreateIssueRequest) (*Issue, error) {
	var issue Issue
	path := fmt.Sprintf("/api/projects/%s/issues", projectID)
	err := c.doRequest(ctx, "POST", path, req, &issue)
	return &issue, err
}

// ListCredentials retrieves all credentials for a project
func (c *Client) ListCredentials(ctx context.Context, projectID string) ([]Credential, error) {
	var credentials []Credential
	path := fmt.Sprintf("/api/projects/%s/credentials", projectID)
	err := c.doRequest(ctx, "GET", path, nil, &credentials)
	return credentials, err
}

// AddCredential adds a new credential to a project
func (c *Client) AddCredential(ctx context.Context, projectID string, req AddCredentialRequest) (*Credential, error) {
	var credential Credential
	path := fmt.Sprintf("/api/projects/%s/credentials", projectID)
	err := c.doRequest(ctx, "POST", path, req, &credential)
	return &credential, err
}

// GenerateReport generates a report for a project
func (c *Client) GenerateReport(ctx context.Context, projectID string, req GenerateReportRequest) (*Report, error) {
	var report Report
	path := fmt.Sprintf("/api/projects/%s/report", projectID)
	err := c.doRequest(ctx, "POST", path, req, &report)
	return &report, err
}

// doRequest performs an HTTP request with retries and error handling
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	// Build full URL
	fullURL := c.baseURL + path

	// Prepare request body
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Retry loop
	var lastErr error
	maxRetries := c.maxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Create new request for each attempt
		req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		if c.apiKey != "" {
			req.Header.Set("X-API-Key", c.apiKey)
		}

		// Perform request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			// Retry on network errors
			continue
		}
		defer resp.Body.Close()

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		// Check for errors
		if resp.StatusCode >= 400 {
			var errResp ErrorResponse
			if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != "" {
				lastErr = fmt.Errorf("PCF API error: %s", errResp.Error)
			} else {
				lastErr = fmt.Errorf("PCF API error: %s (status %d)", string(respBody), resp.StatusCode)
			}

			// Retry on 5xx errors
			if resp.StatusCode >= 500 && attempt < maxRetries-1 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}

			return lastErr
		}

		// Parse successful response
		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}

		return nil
	}

	return lastErr
}
