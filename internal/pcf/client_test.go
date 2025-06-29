package pcf

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aRustyDev/pcf-mcp/internal/config"
)

// TestNewClient tests the creation of a new PCF client
func TestNewClient(t *testing.T) {
	cfg := config.PCFConfig{
		URL:     "http://localhost:5000",
		APIKey:  "test-key",
		Timeout: 30 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	// Verify client properties
	if client.BaseURL() != cfg.URL {
		t.Errorf("Expected base URL '%s', got '%s'", cfg.URL, client.BaseURL())
	}
}

// TestClientWithInvalidURL tests that invalid URLs are rejected
func TestClientWithInvalidURL(t *testing.T) {
	cfg := config.PCFConfig{
		URL:    "://invalid-url",
		APIKey: "test-key",
	}

	_, err := NewClient(cfg)
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

// TestListProjects tests listing projects from PCF
func TestListProjects(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/api/projects" {
			t.Errorf("Expected path '/api/projects', got '%s'", r.URL.Path)
		}

		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got '%s'", r.Method)
		}

		// Check API key header
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("Expected API key 'test-key', got '%s'", r.Header.Get("X-API-Key"))
		}

		// Send response
		projects := []Project{
			{
				ID:          "proj1",
				Name:        "Test Project 1",
				Description: "First test project",
				CreatedAt:   time.Now().Add(-24 * time.Hour),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          "proj2",
				Name:        "Test Project 2",
				Description: "Second test project",
				CreatedAt:   time.Now().Add(-48 * time.Hour),
				UpdatedAt:   time.Now().Add(-12 * time.Hour),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
	}))
	defer server.Close()

	// Create client
	cfg := config.PCFConfig{
		URL:     server.URL,
		APIKey:  "test-key",
		Timeout: 5 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// List projects
	ctx := context.Background()
	projects, err := client.ListProjects(ctx)
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	// Verify results
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	if projects[0].ID != "proj1" {
		t.Errorf("Expected first project ID 'proj1', got '%s'", projects[0].ID)
	}
}

// TestCreateProject tests creating a new project
func TestCreateProject(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/api/projects" {
			t.Errorf("Expected path '/api/projects', got '%s'", r.URL.Path)
		}

		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got '%s'", r.Method)
		}

		// Parse request body
		var req CreateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Name != "New Project" {
			t.Errorf("Expected project name 'New Project', got '%s'", req.Name)
		}

		// Send response
		project := Project{
			ID:          "new-proj",
			Name:        req.Name,
			Description: req.Description,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(project)
	}))
	defer server.Close()

	// Create client
	cfg := config.PCFConfig{
		URL:     server.URL,
		APIKey:  "test-key",
		Timeout: 5 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create project
	ctx := context.Background()
	req := CreateProjectRequest{
		Name:        "New Project",
		Description: "A new test project",
	}

	project, err := client.CreateProject(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Verify result
	if project.ID != "new-proj" {
		t.Errorf("Expected project ID 'new-proj', got '%s'", project.ID)
	}

	if project.Name != "New Project" {
		t.Errorf("Expected project name 'New Project', got '%s'", project.Name)
	}
}

// TestClientRetry tests that the client retries failed requests
func TestClientRetry(t *testing.T) {
	attempts := 0
	// Create test server that fails first request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Success on second attempt
		projects := []Project{{ID: "test", Name: "Test"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
	}))
	defer server.Close()

	// Create client with retries
	cfg := config.PCFConfig{
		URL:        server.URL,
		APIKey:     "test-key",
		Timeout:    5 * time.Second,
		MaxRetries: 3,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// List projects (should succeed after retry)
	ctx := context.Background()
	projects, err := client.ListProjects(ctx)
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(projects))
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

// TestClientTimeout tests that requests timeout properly
func TestClientTimeout(t *testing.T) {
	// Create test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with short timeout
	cfg := config.PCFConfig{
		URL:     server.URL,
		APIKey:  "test-key",
		Timeout: 100 * time.Millisecond,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// List projects (should timeout)
	ctx := context.Background()
	_, err = client.ListProjects(ctx)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

// TestListHosts tests listing hosts for a project
func TestListHosts(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/api/projects/proj1/hosts" {
			t.Errorf("Expected path '/api/projects/proj1/hosts', got '%s'", r.URL.Path)
		}

		// Send response
		hosts := []Host{
			{
				ID:        "host1",
				ProjectID: "proj1",
				IP:        "192.168.1.100",
				Hostname:  "target1.example.com",
				OS:        "Linux",
				Services:  []string{"ssh", "http", "https"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hosts)
	}))
	defer server.Close()

	// Create client
	cfg := config.PCFConfig{
		URL:     server.URL,
		APIKey:  "test-key",
		Timeout: 5 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// List hosts
	ctx := context.Background()
	hosts, err := client.ListHosts(ctx, "proj1")
	if err != nil {
		t.Fatalf("Failed to list hosts: %v", err)
	}

	// Verify results
	if len(hosts) != 1 {
		t.Errorf("Expected 1 host, got %d", len(hosts))
	}

	if hosts[0].IP != "192.168.1.100" {
		t.Errorf("Expected host IP '192.168.1.100', got '%s'", hosts[0].IP)
	}
}

// TestErrorResponse tests handling of error responses from PCF
func TestErrorResponse(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request",
		})
	}))
	defer server.Close()

	// Create client
	cfg := config.PCFConfig{
		URL:     server.URL,
		APIKey:  "test-key",
		Timeout: 5 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// List projects (should return error)
	ctx := context.Background()
	_, err = client.ListProjects(ctx)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
