// +build integration

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/analyst/pcf-mcp/internal/config"
	"github.com/analyst/pcf-mcp/internal/mcp"
	"github.com/analyst/pcf-mcp/internal/mcp/tools"
	"github.com/analyst/pcf-mcp/internal/observability"
	"github.com/analyst/pcf-mcp/internal/pcf"
)

// MockPCFServer creates a mock PCF API server for testing
type MockPCFServer struct {
	*httptest.Server
	projects    map[string]*pcf.Project
	hosts       map[string][]pcf.Host
	issues      map[string][]pcf.Issue
	credentials map[string][]pcf.Credential
}

// NewMockPCFServer creates a new mock PCF server
func NewMockPCFServer() *MockPCFServer {
	m := &MockPCFServer{
		projects:    make(map[string]*pcf.Project),
		hosts:       make(map[string][]pcf.Host),
		issues:      make(map[string][]pcf.Issue),
		credentials: make(map[string][]pcf.Credential),
	}
	
	// Initialize with some test data
	m.projects["test-project"] = &pcf.Project{
		ID:          "test-project",
		Name:        "Test Project",
		Description: "Integration test project",
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	m.hosts["test-project"] = []pcf.Host{
		{
			ID:        "host-1",
			ProjectID: "test-project",
			IP:        "192.168.1.100",
			Hostname:  "test-host-1",
			OS:        "Linux",
			Services:  []string{"ssh", "http"},
			Status:    "active",
		},
	}
	
	m.issues["test-project"] = []pcf.Issue{
		{
			ID:          "issue-1",
			ProjectID:   "test-project",
			Title:       "Test Issue",
			Description: "Test issue description",
			Severity:    "High",
			Status:      "Open",
		},
	}
	
	// Create HTTP server
	mux := http.NewServeMux()
	
	// Projects endpoints
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			projects := make([]pcf.Project, 0, len(m.projects))
			for _, p := range m.projects {
				projects = append(projects, *p)
			}
			json.NewEncoder(w).Encode(projects)
		case http.MethodPost:
			var req pcf.CreateProjectRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			project := &pcf.Project{
				ID:          fmt.Sprintf("proj-%d", len(m.projects)+1),
				Name:        req.Name,
				Description: req.Description,
				Team:        req.Team,
				Status:      "active",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			m.projects[project.ID] = project
			json.NewEncoder(w).Encode(project)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	
	// Hosts endpoints
	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		// Extract project ID and resource from path
		path := r.URL.Path[len("/api/projects/"):]
		parts := bytes.Split([]byte(path), []byte("/"))
		
		if len(parts) < 2 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		
		projectID := string(parts[0])
		resource := string(parts[1])
		
		switch resource {
		case "hosts":
			switch r.Method {
			case http.MethodGet:
				hosts := m.hosts[projectID]
				if hosts == nil {
					hosts = []pcf.Host{}
				}
				json.NewEncoder(w).Encode(hosts)
			case http.MethodPost:
				var req pcf.CreateHostRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				host := pcf.Host{
					ID:        fmt.Sprintf("host-%d", len(m.hosts[projectID])+1),
					ProjectID: projectID,
					IP:        req.IP,
					Hostname:  req.Hostname,
					OS:        req.OS,
					Services:  req.Services,
					Status:    "active",
				}
				m.hosts[projectID] = append(m.hosts[projectID], host)
				json.NewEncoder(w).Encode(&host)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			
		case "issues":
			switch r.Method {
			case http.MethodGet:
				issues := m.issues[projectID]
				if issues == nil {
					issues = []pcf.Issue{}
				}
				json.NewEncoder(w).Encode(issues)
			case http.MethodPost:
				var req pcf.CreateIssueRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				issue := pcf.Issue{
					ID:          fmt.Sprintf("issue-%d", len(m.issues[projectID])+1),
					ProjectID:   projectID,
					HostID:      req.HostID,
					Title:       req.Title,
					Description: req.Description,
					Severity:    req.Severity,
					Status:      "Open",
					CVE:         req.CVE,
					CVSS:        req.CVSS,
				}
				m.issues[projectID] = append(m.issues[projectID], issue)
				json.NewEncoder(w).Encode(&issue)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			
		case "credentials":
			switch r.Method {
			case http.MethodGet:
				creds := m.credentials[projectID]
				if creds == nil {
					creds = []pcf.Credential{}
				}
				json.NewEncoder(w).Encode(creds)
			case http.MethodPost:
				var req pcf.AddCredentialRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				cred := pcf.Credential{
					ID:        fmt.Sprintf("cred-%d", len(m.credentials[projectID])+1),
					ProjectID: projectID,
					HostID:    req.HostID,
					Type:      req.Type,
					Username:  req.Username,
					Value:     "***encrypted***",
					Service:   req.Service,
					Notes:     req.Notes,
				}
				m.credentials[projectID] = append(m.credentials[projectID], cred)
				json.NewEncoder(w).Encode(&cred)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			
		case "report":
			if r.Method == http.MethodPost {
				var req pcf.GenerateReportRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				report := pcf.Report{
					ID:        fmt.Sprintf("report-%d", time.Now().Unix()),
					ProjectID: projectID,
					Format:    req.Format,
					Status:    "completed",
					URL:       fmt.Sprintf("http://mock-pcf/reports/%s.%s", projectID, req.Format),
					CreatedAt: time.Now(),
					Size:      1024 * 1024, // 1MB
				}
				json.NewEncoder(w).Encode(&report)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	})
	
	m.Server = httptest.NewServer(mux)
	return m
}

// TestFullIntegration tests the complete MCP server with all tools
func TestFullIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Integration tests not enabled. Set INTEGRATION_TESTS=true to run.")
	}
	
	// Start mock PCF server
	mockPCF := NewMockPCFServer()
	defer mockPCF.Close()
	
	// Create configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Transport:    "http",
			Host:         "localhost",
			Port:         0, // Use any available port
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		PCF: config.PCFConfig{
			URL:     mockPCF.URL,
			APIKey:  "test-api-key",
			Timeout: 10 * time.Second,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
			Port:    0, // Use any available port
		},
		Tracing: config.TracingConfig{
			Enabled: false,
		},
	}
	
	// Initialize logging
	logger, err := observability.NewLogger(cfg.Logging)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	observability.SetGlobalLogger(logger)
	
	// Initialize metrics
	metrics, err := observability.InitMetrics(cfg.Metrics)
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	
	// Create PCF client
	pcfClient, err := pcf.NewClient(cfg.PCF)
	if err != nil {
		t.Fatalf("Failed to create PCF client: %v", err)
	}
	
	// Create MCP server
	mcpServer, err := mcp.NewServer(cfg.Server)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}
	
	// Set metrics
	mcpServer.SetMetrics(metrics)
	
	// Register all tools
	if err := tools.RegisterAllTools(mcpServer, pcfClient); err != nil {
		t.Fatalf("Failed to register tools: %v", err)
	}
	
	// Start HTTP server
	handler := mcpServer.HTTPHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()
	
	// Test all endpoints
	t.Run("Health Check", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to get health: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		
		var health map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}
		
		if health["status"] != "healthy" {
			t.Errorf("Expected healthy status, got %v", health["status"])
		}
	})
	
	t.Run("Server Info", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/info")
		if err != nil {
			t.Fatalf("Failed to get info: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		
		var info map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			t.Fatalf("Failed to decode info response: %v", err)
		}
		
		if info["name"] != "pcf-mcp" {
			t.Errorf("Expected name 'pcf-mcp', got %v", info["name"])
		}
	})
	
	t.Run("List Tools", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/tools")
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode tools response: %v", err)
		}
		
		tools, ok := result["tools"].([]interface{})
		if !ok {
			t.Fatal("Tools should be an array")
		}
		
		if len(tools) != 9 {
			t.Errorf("Expected 9 tools, got %d", len(tools))
		}
	})
	
	// Test each tool
	testCases := []struct {
		name   string
		tool   string
		params map[string]interface{}
		check  func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "List Projects",
			tool: "list_projects",
			params: map[string]interface{}{},
			check: func(t *testing.T, result map[string]interface{}) {
				projects, ok := result["projects"].([]interface{})
				if !ok {
					t.Fatal("Expected projects array")
				}
				if len(projects) < 1 {
					t.Error("Expected at least one project")
				}
			},
		},
		{
			name: "Create Project",
			tool: "create_project",
			params: map[string]interface{}{
				"name":        "Integration Test Project",
				"description": "Created by integration test",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				project, ok := result["project"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected project object")
				}
				if project["name"] != "Integration Test Project" {
					t.Errorf("Expected project name 'Integration Test Project', got %v", project["name"])
				}
			},
		},
		{
			name: "List Hosts",
			tool: "list_hosts",
			params: map[string]interface{}{
				"project_id": "test-project",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				hosts, ok := result["hosts"].([]interface{})
				if !ok {
					t.Fatal("Expected hosts array")
				}
				if len(hosts) < 1 {
					t.Error("Expected at least one host")
				}
			},
		},
		{
			name: "Add Host",
			tool: "add_host",
			params: map[string]interface{}{
				"project_id": "test-project",
				"ip":         "192.168.1.200",
				"hostname":   "new-host",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				host, ok := result["host"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected host object")
				}
				if host["ip"] != "192.168.1.200" {
					t.Errorf("Expected IP '192.168.1.200', got %v", host["ip"])
				}
			},
		},
		{
			name: "List Issues",
			tool: "list_issues",
			params: map[string]interface{}{
				"project_id": "test-project",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				issues, ok := result["issues"].([]interface{})
				if !ok {
					t.Fatal("Expected issues array")
				}
				if len(issues) < 1 {
					t.Error("Expected at least one issue")
				}
			},
		},
		{
			name: "Create Issue",
			tool: "create_issue",
			params: map[string]interface{}{
				"project_id":  "test-project",
				"title":       "New Security Issue",
				"description": "Found during integration test",
				"severity":    "High",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				issue, ok := result["issue"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected issue object")
				}
				if issue["title"] != "New Security Issue" {
					t.Errorf("Expected title 'New Security Issue', got %v", issue["title"])
				}
			},
		},
		{
			name: "List Credentials",
			tool: "list_credentials",
			params: map[string]interface{}{
				"project_id": "test-project",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				creds, ok := result["credentials"].([]interface{})
				if !ok {
					t.Logf("Result: %+v", result)
					t.Fatal("Expected credentials array")
				}
				// Empty credentials is OK for this test
				t.Logf("Found %d credentials", len(creds))
				// Check that values are redacted
				for _, c := range creds {
					cred := c.(map[string]interface{})
					if cred["value"] != "***REDACTED***" {
						t.Error("Credential value should be redacted")
					}
				}
			},
		},
		{
			name: "Add Credential",
			tool: "add_credential",
			params: map[string]interface{}{
				"project_id": "test-project",
				"type":       "password",
				"username":   "testuser",
				"value":      "testpass123",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				cred, ok := result["credential"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected credential object")
				}
				if cred["value"] != "***REDACTED***" {
					t.Error("Credential value should be redacted")
				}
			},
		},
		{
			name: "Generate Report",
			tool: "generate_report",
			params: map[string]interface{}{
				"project_id":     "test-project",
				"format":         "pdf",
				"include_hosts":  true,
				"include_issues": true,
			},
			check: func(t *testing.T, result map[string]interface{}) {
				report, ok := result["report"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected report object")
				}
				if report["format"] != "pdf" {
					t.Errorf("Expected format 'pdf', got %v", report["format"])
				}
				if report["status"] != "completed" {
					t.Errorf("Expected status 'completed', got %v", report["status"])
				}
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare request
			body, err := json.Marshal(tc.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}
			
			// Make request
			resp, err := http.Post(
				fmt.Sprintf("%s/tools/%s", ts.URL, tc.tool),
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				t.Fatalf("Failed to execute tool: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				var errResp map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&errResp)
				t.Fatalf("Expected status 200, got %d: %v", resp.StatusCode, errResp)
			}
			
			// Decode response
			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			
			// Extract result
			result, ok := response["result"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected result in response")
			}
			
			// Run custom check
			tc.check(t, result)
		})
	}
}

// TestConcurrentRequests tests the server's ability to handle concurrent requests
func TestConcurrentRequests(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Integration tests not enabled. Set INTEGRATION_TESTS=true to run.")
	}
	
	// Start mock PCF server
	mockPCF := NewMockPCFServer()
	defer mockPCF.Close()
	
	// Create minimal configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Transport:          "http",
			Host:               "localhost",
			Port:               0,
			MaxConcurrentTools: 5,
			ToolTimeout:        10 * time.Second,
		},
		PCF: config.PCFConfig{
			URL:     mockPCF.URL,
			Timeout: 10 * time.Second,
		},
		Logging: config.LoggingConfig{
			Level:  "error", // Reduce logging noise
			Format: "json",
		},
	}
	
	// Initialize components
	logger, _ := observability.NewLogger(cfg.Logging)
	observability.SetGlobalLogger(logger)
	
	pcfClient, err := pcf.NewClient(cfg.PCF)
	if err != nil {
		t.Fatalf("Failed to create PCF client: %v", err)
	}
	
	mcpServer, err := mcp.NewServer(cfg.Server)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}
	
	if err := tools.RegisterAllTools(mcpServer, pcfClient); err != nil {
		t.Fatalf("Failed to register tools: %v", err)
	}
	
	// Start HTTP server
	handler := mcpServer.HTTPHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()
	
	// Number of concurrent requests
	concurrency := 20
	iterations := 100
	
	// Channel to collect errors
	errCh := make(chan error, concurrency*iterations)
	
	// Synchronization
	var wg sync.WaitGroup
	wg.Add(concurrency)
	
	// Run concurrent requests
	start := time.Now()
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Rotate through different tools
				toolIndex := (workerID + j) % 3
				var tool string
				var params map[string]interface{}
				
				switch toolIndex {
				case 0:
					tool = "list_projects"
					params = map[string]interface{}{}
				case 1:
					tool = "list_hosts"
					params = map[string]interface{}{
						"project_id": "test-project",
					}
				case 2:
					tool = "list_issues"
					params = map[string]interface{}{
						"project_id": "test-project",
					}
				}
				
				// Make request
				body, _ := json.Marshal(params)
				resp, err := http.Post(
					fmt.Sprintf("%s/tools/%s", ts.URL, tool),
					"application/json",
					bytes.NewReader(body),
				)
				
				if err != nil {
					select {
					case errCh <- fmt.Errorf("worker %d request %d failed: %w", workerID, j, err):
					default:
						// Channel full, skip error
					}
					continue
				}
				
				if resp.StatusCode != http.StatusOK {
					select {
					case errCh <- fmt.Errorf("worker %d request %d got status %d", workerID, j, resp.StatusCode):
					default:
						// Channel full, skip error
					}
				}
				
				resp.Body.Close()
			}
		}(i)
	}
	
	// Wait for completion with timeout
	timeout := time.After(30 * time.Second)
	done := make(chan bool)
	
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// Check for errors
		close(errCh)
		errorCount := 0
		for err := range errCh {
			t.Error(err)
			errorCount++
			if errorCount > 10 {
				t.Fatal("Too many errors, stopping test")
			}
		}
		
		duration := time.Since(start)
		totalRequests := concurrency * iterations
		rps := float64(totalRequests) / duration.Seconds()
		
		t.Logf("Completed %d requests in %v (%.2f req/s)", totalRequests, duration, rps)
		
	case <-timeout:
		t.Fatal("Test timed out")
	}
}

// TestAuthentication tests the HTTP authentication mechanism
func TestAuthentication(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Integration tests not enabled. Set INTEGRATION_TESTS=true to run.")
	}
	
	// Create configuration with auth enabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			Transport:    "http",
			Host:         "localhost",
			Port:         0,
			AuthRequired: true,
			AuthToken:    "test-secret-token",
		},
		PCF: config.PCFConfig{
			URL: "http://dummy",
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "json",
		},
	}
	
	// Initialize components
	logger, _ := observability.NewLogger(cfg.Logging)
	observability.SetGlobalLogger(logger)
	
	mcpServer, err := mcp.NewServer(cfg.Server)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}
	
	// Start HTTP server
	handler := mcpServer.HTTPHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()
	
	testCases := []struct {
		name       string
		path       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "Valid token",
			path:       "/info",
			authHeader: "Bearer test-secret-token",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid token",
			path:       "/info",
			authHeader: "Bearer wrong-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Missing token",
			path:       "/info",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Health check without auth",
			path:       "/health",
			authHeader: "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Metrics without auth",
			path:       "/metrics",
			authHeader: "",
			wantStatus: http.StatusOK,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", ts.URL+tc.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("Expected status %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}