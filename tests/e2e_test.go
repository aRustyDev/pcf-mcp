//go:build integration
// +build integration

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aRustyDev/pcf-mcp/internal/config"
	"github.com/aRustyDev/pcf-mcp/internal/mcp"
	"github.com/aRustyDev/pcf-mcp/internal/mcp/tools"
	"github.com/aRustyDev/pcf-mcp/internal/observability"
	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// TestEndToEndPentestWorkflow simulates a complete pentest workflow
func TestEndToEndPentestWorkflow(t *testing.T) {
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
			Port:         0,
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
			Port:    0,
		},
	}

	// Initialize components
	logger, err := observability.NewLogger(cfg.Logging)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	observability.SetGlobalLogger(logger)

	metrics, err := observability.InitMetrics(cfg.Metrics)
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	pcfClient, err := pcf.NewClient(cfg.PCF)
	if err != nil {
		t.Fatalf("Failed to create PCF client: %v", err)
	}

	mcpServer, err := mcp.NewServer(cfg.Server)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}

	mcpServer.SetMetrics(metrics)

	if err := tools.RegisterAllTools(mcpServer, pcfClient); err != nil {
		t.Fatalf("Failed to register tools: %v", err)
	}

	// Start HTTP server
	handler := mcpServer.HTTPHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Helper function to execute a tool
	executeTool := func(tool string, params map[string]interface{}) (map[string]interface{}, error) {
		body, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}

		resp, err := http.Post(
			fmt.Sprintf("%s/tools/%s", ts.URL, tool),
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errResp map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errResp)
			return nil, fmt.Errorf("tool execution failed: %v", errResp)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, err
		}

		result, ok := response["result"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("no result in response")
		}

		return result, nil
	}

	// Workflow steps
	var projectID string

	// Step 1: Create a new pentest project
	t.Run("Create Project", func(t *testing.T) {
		result, err := executeTool("create_project", map[string]interface{}{
			"name":        "ACME Corp Pentest",
			"description": "Q4 2024 Security Assessment",
			"team":        []string{"alice", "bob", "charlie"},
		})
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		project := result["project"].(map[string]interface{})
		projectID = project["id"].(string)

		t.Logf("Created project: %s", projectID)
	})

	// Step 2: Add discovered hosts
	var hostIDs []string
	t.Run("Add Hosts", func(t *testing.T) {
		hosts := []struct {
			ip       string
			hostname string
			os       string
			services []string
		}{
			{
				ip:       "10.0.1.10",
				hostname: "web-server.acme.local",
				os:       "Linux",
				services: []string{"ssh", "http", "https"},
			},
			{
				ip:       "10.0.1.20",
				hostname: "db-server.acme.local",
				os:       "Linux",
				services: []string{"ssh", "mysql"},
			},
			{
				ip:       "10.0.1.30",
				hostname: "dc01.acme.local",
				os:       "Windows",
				services: []string{"ldap", "kerberos", "smb", "rdp"},
			},
		}

		for _, host := range hosts {
			result, err := executeTool("add_host", map[string]interface{}{
				"project_id": projectID,
				"ip":         host.ip,
				"hostname":   host.hostname,
				"os":         host.os,
				"services":   host.services,
			})
			if err != nil {
				t.Fatalf("Failed to add host %s: %v", host.ip, err)
			}

			h := result["host"].(map[string]interface{})
			hostIDs = append(hostIDs, h["id"].(string))
			t.Logf("Added host: %s (%s)", host.hostname, host.ip)
		}
	})

	// Step 3: Add discovered vulnerabilities
	var issueIDs []string
	t.Run("Add Issues", func(t *testing.T) {
		issues := []struct {
			title       string
			description string
			severity    string
			hostIndex   int
			cve         string
			cvss        float64
		}{
			{
				title:       "SQL Injection in Login Form",
				description: "The login form at /admin/login is vulnerable to SQL injection via the username parameter",
				severity:    "Critical",
				hostIndex:   0, // web-server
				cve:         "",
				cvss:        0,
			},
			{
				title:       "Default MySQL Credentials",
				description: "MySQL root account is using default password 'root'",
				severity:    "High",
				hostIndex:   1, // db-server
				cve:         "",
				cvss:        0,
			},
			{
				title:       "EternalBlue (MS17-010)",
				description: "Windows SMB service is vulnerable to EternalBlue exploit",
				severity:    "Critical",
				hostIndex:   2, // dc01
				cve:         "CVE-2017-0144",
				cvss:        8.1,
			},
			{
				title:       "Weak TLS Configuration",
				description: "HTTPS service supports TLS 1.0 and weak cipher suites",
				severity:    "Medium",
				hostIndex:   0, // web-server
				cve:         "",
				cvss:        0,
			},
		}

		for _, issue := range issues {
			params := map[string]interface{}{
				"project_id":  projectID,
				"title":       issue.title,
				"description": issue.description,
				"severity":    issue.severity,
			}

			if issue.hostIndex < len(hostIDs) {
				params["host_id"] = hostIDs[issue.hostIndex]
			}

			if issue.cve != "" {
				params["cve"] = issue.cve
			}

			if issue.cvss > 0 {
				params["cvss"] = issue.cvss
			}

			result, err := executeTool("create_issue", params)
			if err != nil {
				t.Fatalf("Failed to create issue '%s': %v", issue.title, err)
			}

			i := result["issue"].(map[string]interface{})
			issueIDs = append(issueIDs, i["id"].(string))
			t.Logf("Created issue: %s (%s)", issue.title, issue.severity)
		}
	})

	// Step 4: Add discovered credentials
	t.Run("Add Credentials", func(t *testing.T) {
		creds := []struct {
			credType string
			username string
			value    string
			service  string
			hostIdx  int
		}{
			{
				credType: "password",
				username: "admin",
				value:    "admin123",
				service:  "http",
				hostIdx:  0,
			},
			{
				credType: "password",
				username: "root",
				value:    "root",
				service:  "mysql",
				hostIdx:  1,
			},
			{
				credType: "hash",
				username: "Administrator",
				value:    "aad3b435b51404eeaad3b435b51404ee:8846f7eaee8fb117ad06bdd830b7586c",
				service:  "smb",
				hostIdx:  2,
			},
		}

		for _, cred := range creds {
			params := map[string]interface{}{
				"project_id": projectID,
				"type":       cred.credType,
				"username":   cred.username,
				"value":      cred.value,
				"service":    cred.service,
			}

			if cred.hostIdx < len(hostIDs) {
				params["host_id"] = hostIDs[cred.hostIdx]
			}

			result, err := executeTool("add_credential", params)
			if err != nil {
				t.Fatalf("Failed to add credential: %v", err)
			}

			c := result["credential"].(map[string]interface{})
			if c["value"] != "***REDACTED***" {
				t.Error("Credential value should be redacted")
			}
			t.Logf("Added credential: %s@%s", cred.username, cred.service)
		}
	})

	// Step 5: Query the data
	t.Run("Query Data", func(t *testing.T) {
		// List all issues with Critical severity
		result, err := executeTool("list_issues", map[string]interface{}{
			"project_id": projectID,
			"severity":   "Critical",
		})
		if err != nil {
			t.Fatalf("Failed to list issues: %v", err)
		}

		issues := result["issues"].([]interface{})
		criticalCount := 0
		for _, i := range issues {
			issue := i.(map[string]interface{})
			if issue["severity"] == "Critical" {
				criticalCount++
			}
		}

		if criticalCount != 2 {
			t.Errorf("Expected 2 critical issues, found %d", criticalCount)
		}

		// Check severity breakdown
		breakdown := result["severity_breakdown"].(map[string]interface{})
		t.Logf("Severity breakdown: %+v", breakdown)

		// List hosts with Windows OS
		result, err = executeTool("list_hosts", map[string]interface{}{
			"project_id": projectID,
			"os":         "Windows",
		})
		if err != nil {
			t.Fatalf("Failed to list hosts: %v", err)
		}

		hosts := result["hosts"].([]interface{})
		windowsCount := 0
		for _, h := range hosts {
			host := h.(map[string]interface{})
			if host["os"] == "Windows" {
				windowsCount++
			}
		}

		if windowsCount < 1 {
			t.Error("Expected at least one Windows host")
		}

		// List all credentials
		result, err = executeTool("list_credentials", map[string]interface{}{
			"project_id": projectID,
		})
		if err != nil {
			t.Fatalf("Failed to list credentials: %v", err)
		}

		creds := result["credentials"].([]interface{})
		if len(creds) < 3 {
			t.Errorf("Expected at least 3 credentials, found %d", len(creds))
		}

		// Verify all credential values are redacted
		for _, c := range creds {
			cred := c.(map[string]interface{})
			if cred["value"] != "***REDACTED***" {
				t.Error("Credential value should be redacted")
			}
		}
	})

	// Step 6: Generate final report
	t.Run("Generate Report", func(t *testing.T) {
		formats := []string{"pdf", "html", "json"}

		for _, format := range formats {
			result, err := executeTool("generate_report", map[string]interface{}{
				"project_id":          projectID,
				"format":              format,
				"include_hosts":       true,
				"include_issues":      true,
				"include_credentials": true,
				"sections": []string{
					"executive_summary",
					"technical_findings",
					"risk_assessment",
					"remediation",
				},
			})
			if err != nil {
				t.Fatalf("Failed to generate %s report: %v", format, err)
			}

			report := result["report"].(map[string]interface{})
			if report["status"] != "completed" {
				t.Errorf("Expected report status 'completed', got %v", report["status"])
			}

			if report["format"] != format {
				t.Errorf("Expected format '%s', got %v", format, report["format"])
			}

			t.Logf("Generated %s report: %s", format, report["url"])
		}
	})

	// Step 7: Verify metrics
	t.Run("Check Metrics", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/metrics")
		if err != nil {
			t.Fatalf("Failed to get metrics: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Read metrics
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		metrics := buf.String()

		// Check for expected metrics
		expectedMetrics := []string{
			"http_requests_total",
			"http_request_duration_seconds",
		}

		for _, metric := range expectedMetrics {
			if !bytes.Contains([]byte(metrics), []byte(metric)) {
				t.Errorf("Expected metric '%s' not found", metric)
			}
		}

		t.Log("Metrics endpoint working correctly")
	})

	// Step 8: Final summary
	t.Run("Project Summary", func(t *testing.T) {
		// Get all projects to verify our test project
		result, err := executeTool("list_projects", map[string]interface{}{})
		if err != nil {
			t.Fatalf("Failed to list projects: %v", err)
		}

		projects := result["projects"].([]interface{})
		found := false
		for _, p := range projects {
			project := p.(map[string]interface{})
			if project["name"] == "ACME Corp Pentest" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Test project not found in project list")
		}

		t.Logf("\nPentest Workflow Summary:")
		t.Logf("- Project: ACME Corp Pentest")
		t.Logf("- Hosts discovered: %d", len(hostIDs))
		t.Logf("- Issues found: %d", len(issueIDs))
		t.Logf("- Credentials captured: 3")
		t.Logf("- Reports generated: 3 formats")
	})
}

// TestStressTest performs a stress test on the MCP server
func TestStressTest(t *testing.T) {
	if os.Getenv("STRESS_TEST") != "true" {
		t.Skip("Stress tests not enabled. Set STRESS_TEST=true to run.")
	}

	// Start mock PCF server with delay to simulate network latency
	mockPCF := NewMockPCFServer()
	defer mockPCF.Close()

	// Add artificial delay to mock server
	originalHandler := mockPCF.Server.Config.Handler
	mockPCF.Server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add 10-50ms delay
		delay := time.Duration(10+time.Now().UnixNano()%40) * time.Millisecond
		time.Sleep(delay)
		originalHandler.ServeHTTP(w, r)
	})

	// Create configuration with higher limits
	cfg := &config.Config{
		Server: config.ServerConfig{
			Transport:          "http",
			Host:               "localhost",
			Port:               0,
			MaxConcurrentTools: 50,
			ToolTimeout:        30 * time.Second,
		},
		PCF: config.PCFConfig{
			URL:        mockPCF.URL,
			Timeout:    30 * time.Second,
			MaxRetries: 5,
		},
		Logging: config.LoggingConfig{
			Level:  "error",
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

	// Stress test parameters
	workers := 100
	requestsPerWorker := 1000
	totalRequests := workers * requestsPerWorker

	t.Logf("Starting stress test: %d workers, %d requests each = %d total requests",
		workers, requestsPerWorker, totalRequests)

	// Metrics
	type metrics struct {
		requests  int
		errors    int
		durations []time.Duration
	}

	results := make(chan metrics, workers)

	// Start workers
	start := time.Now()
	for i := 0; i < workers; i++ {
		go func(workerID int) {
			m := metrics{
				durations: make([]time.Duration, 0, requestsPerWorker),
			}

			for j := 0; j < requestsPerWorker; j++ {
				reqStart := time.Now()

				// Make a simple request
				body := []byte(`{}`)
				resp, err := http.Post(
					ts.URL+"/tools/list_projects",
					"application/json",
					bytes.NewReader(body),
				)

				duration := time.Since(reqStart)
				m.durations = append(m.durations, duration)
				m.requests++

				if err != nil {
					m.errors++
					continue
				}

				if resp.StatusCode != http.StatusOK {
					m.errors++
				}
				resp.Body.Close()
			}

			results <- m
		}(i)
	}

	// Collect results
	var totalErrors int
	var allDurations []time.Duration

	for i := 0; i < workers; i++ {
		m := <-results
		totalErrors += m.errors
		allDurations = append(allDurations, m.durations...)
	}

	elapsed := time.Since(start)

	// Calculate statistics
	var sum time.Duration
	var min, max time.Duration = time.Hour, 0

	for _, d := range allDurations {
		sum += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	avg := sum / time.Duration(len(allDurations))
	rps := float64(totalRequests) / elapsed.Seconds()

	// Calculate percentiles (simplified)
	p50 := allDurations[len(allDurations)*50/100]
	p95 := allDurations[len(allDurations)*95/100]
	p99 := allDurations[len(allDurations)*99/100]

	// Report results
	t.Logf("\nStress Test Results:")
	t.Logf("Total time: %v", elapsed)
	t.Logf("Total requests: %d", totalRequests)
	t.Logf("Requests/second: %.2f", rps)
	t.Logf("Errors: %d (%.2f%%)", totalErrors, float64(totalErrors)/float64(totalRequests)*100)
	t.Logf("\nLatency statistics:")
	t.Logf("Min: %v", min)
	t.Logf("Avg: %v", avg)
	t.Logf("P50: %v", p50)
	t.Logf("P95: %v", p95)
	t.Logf("P99: %v", p99)
	t.Logf("Max: %v", max)

	// Assertions
	errorRate := float64(totalErrors) / float64(totalRequests)
	if errorRate > 0.01 { // 1% error rate threshold
		t.Errorf("Error rate too high: %.2f%%", errorRate*100)
	}

	if rps < 100 {
		t.Errorf("Throughput too low: %.2f req/s", rps)
	}

	if p99 > 5*time.Second {
		t.Errorf("P99 latency too high: %v", p99)
	}
}
