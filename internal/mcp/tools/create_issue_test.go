package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/analyst/pcf-mcp/internal/pcf"
)

// MockCreateIssueClient extends MockPCFClient with CreateIssue method
type MockCreateIssueClient struct {
	MockPCFClient
	CreateIssueFunc func(ctx context.Context, projectID string, req pcf.CreateIssueRequest) (*pcf.Issue, error)
}

func (m *MockCreateIssueClient) CreateIssue(ctx context.Context, projectID string, req pcf.CreateIssueRequest) (*pcf.Issue, error) {
	if m.CreateIssueFunc != nil {
		return m.CreateIssueFunc(ctx, projectID, req)
	}
	return nil, errors.New("CreateIssueFunc not implemented")
}

// TestNewCreateIssueTool tests creating a new create issue tool
func TestNewCreateIssueTool(t *testing.T) {
	mockClient := &MockCreateIssueClient{}
	
	tool := NewCreateIssueTool(mockClient)
	
	if tool.Name != "create_issue" {
		t.Errorf("Expected tool name 'create_issue', got '%s'", tool.Name)
	}
	
	if tool.Description == "" {
		t.Error("Tool description should not be empty")
	}
	
	if tool.Handler == nil {
		t.Error("Tool handler should not be nil")
	}
	
	// Check input schema
	if tool.InputSchema == nil {
		t.Error("Tool should have input schema")
	}
	
	// Verify required properties
	props, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Input schema should have properties")
	}
	
	requiredProps := []string{"project_id", "title", "description", "severity"}
	for _, prop := range requiredProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("Input schema missing '%s' property", prop)
		}
	}
	
	// Check required fields
	required, ok := tool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Input schema should have required fields")
	}
	
	if len(required) != 4 {
		t.Errorf("Expected 4 required fields, got %d", len(required))
	}
}

// TestCreateIssueHandler tests the create issue handler functionality
func TestCreateIssueHandler(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]interface{}
		expectedReq   pcf.CreateIssueRequest
		mockResponse  *pcf.Issue
		mockError     error
		expectError   bool
	}{
		{
			name: "Create issue with minimal info",
			params: map[string]interface{}{
				"project_id":  "proj-123",
				"title":       "SQL Injection Vulnerability",
				"description": "Found SQL injection in login form",
				"severity":    "Critical",
			},
			expectedReq: pcf.CreateIssueRequest{
				Title:       "SQL Injection Vulnerability",
				Description: "Found SQL injection in login form",
				Severity:    "Critical",
			},
			mockResponse: &pcf.Issue{
				ID:          "issue-new",
				ProjectID:   "proj-123",
				Title:       "SQL Injection Vulnerability",
				Description: "Found SQL injection in login form",
				Severity:    "Critical",
				Status:      "Open",
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Create issue with full details",
			params: map[string]interface{}{
				"project_id":  "proj-456",
				"title":       "Remote Code Execution",
				"description": "RCE vulnerability in file upload",
				"severity":    "Critical",
				"host_id":     "host-123",
				"cve":         "CVE-2024-12345",
				"cvss":        9.8,
			},
			expectedReq: pcf.CreateIssueRequest{
				HostID:      "host-123",
				Title:       "Remote Code Execution",
				Description: "RCE vulnerability in file upload",
				Severity:    "Critical",
				CVE:         "CVE-2024-12345",
				CVSS:        9.8,
			},
			mockResponse: &pcf.Issue{
				ID:          "issue-full",
				ProjectID:   "proj-456",
				HostID:      "host-123",
				Title:       "Remote Code Execution",
				Description: "RCE vulnerability in file upload",
				Severity:    "Critical",
				Status:      "Open",
				CVE:         "CVE-2024-12345",
				CVSS:        9.8,
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Missing project_id",
			params: map[string]interface{}{
				"title":       "Test Issue",
				"description": "Test description",
				"severity":    "Low",
			},
			expectedReq:  pcf.CreateIssueRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Missing title",
			params: map[string]interface{}{
				"project_id":  "proj-123",
				"description": "Test description",
				"severity":    "Low",
			},
			expectedReq:  pcf.CreateIssueRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Missing description",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"title":      "Test Issue",
				"severity":   "Low",
			},
			expectedReq:  pcf.CreateIssueRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Missing severity",
			params: map[string]interface{}{
				"project_id":  "proj-123",
				"title":       "Test Issue",
				"description": "Test description",
			},
			expectedReq:  pcf.CreateIssueRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Invalid severity value",
			params: map[string]interface{}{
				"project_id":  "proj-123",
				"title":       "Test Issue",
				"description": "Test description",
				"severity":    "SuperCritical", // Invalid severity
			},
			expectedReq:  pcf.CreateIssueRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Invalid project_id type",
			params: map[string]interface{}{
				"project_id":  123, // Should be string
				"title":       "Test Issue",
				"description": "Test description",
				"severity":    "Low",
			},
			expectedReq:  pcf.CreateIssueRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "PCF API error",
			params: map[string]interface{}{
				"project_id":  "proj-error",
				"title":       "Test Issue",
				"description": "Test description",
				"severity":    "Low",
			},
			expectedReq: pcf.CreateIssueRequest{
				Title:       "Test Issue",
				Description: "Test description",
				Severity:    "Low",
			},
			mockResponse: nil,
			mockError:    errors.New("project not found"),
			expectError:  true,
		},
		{
			name: "Invalid CVSS score",
			params: map[string]interface{}{
				"project_id":  "proj-123",
				"title":       "Test Issue",
				"description": "Test description",
				"severity":    "Low",
				"cvss":        "high", // Should be a number
			},
			expectedReq:  pcf.CreateIssueRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "CVSS score out of range",
			params: map[string]interface{}{
				"project_id":  "proj-123",
				"title":       "Test Issue",
				"description": "Test description",
				"severity":    "Low",
				"cvss":        11.0, // Should be 0-10
			},
			expectedReq:  pcf.CreateIssueRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockCreateIssueClient{
				CreateIssueFunc: func(ctx context.Context, projectID string, req pcf.CreateIssueRequest) (*pcf.Issue, error) {
					// Verify project ID if we expect the call to succeed
					if !tt.expectError || tt.mockError != nil {
						expectedProjectID, _ := tt.params["project_id"].(string)
						if projectID != expectedProjectID {
							t.Errorf("Expected project ID '%s', got '%s'", expectedProjectID, projectID)
						}
						
						// Verify request structure
						if req.Title != tt.expectedReq.Title {
							t.Errorf("Expected title '%s', got '%s'", tt.expectedReq.Title, req.Title)
						}
						if req.Description != tt.expectedReq.Description {
							t.Errorf("Expected description '%s', got '%s'", tt.expectedReq.Description, req.Description)
						}
						if req.Severity != tt.expectedReq.Severity {
							t.Errorf("Expected severity '%s', got '%s'", tt.expectedReq.Severity, req.Severity)
						}
						if req.HostID != tt.expectedReq.HostID {
							t.Errorf("Expected host ID '%s', got '%s'", tt.expectedReq.HostID, req.HostID)
						}
						if req.CVE != tt.expectedReq.CVE {
							t.Errorf("Expected CVE '%s', got '%s'", tt.expectedReq.CVE, req.CVE)
						}
						if req.CVSS != tt.expectedReq.CVSS {
							t.Errorf("Expected CVSS %f, got %f", tt.expectedReq.CVSS, req.CVSS)
						}
					}
					
					return tt.mockResponse, tt.mockError
				},
			}
			
			// Create tool
			tool := NewCreateIssueTool(mockClient)
			
			// Execute handler
			ctx := context.Background()
			result, err := tool.Handler(ctx, tt.params)
			
			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			// Verify result structure
			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Fatal("Result should be a map")
			}
			
			// Check for issue key
			issueData, ok := resultMap["issue"]
			if !ok {
				t.Fatal("Result should contain 'issue' key")
			}
			
			// Verify issue structure
			issue, ok := issueData.(map[string]interface{})
			if !ok {
				t.Fatal("Issue should be a map")
			}
			
			// Check required fields
			requiredFields := []string{"id", "project_id", "title", "description", "severity", "status"}
			for _, field := range requiredFields {
				if _, ok := issue[field]; !ok {
					t.Errorf("Issue missing required field: %s", field)
				}
			}
			
			// Check message
			if message, ok := resultMap["message"].(string); !ok || message == "" {
				t.Error("Result should contain a non-empty message")
			}
		})
	}
}