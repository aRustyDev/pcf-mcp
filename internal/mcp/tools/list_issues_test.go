package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// MockListIssuesClient extends MockPCFClient with ListIssues method
type MockListIssuesClient struct {
	MockPCFClient
	ListIssuesFunc func(ctx context.Context, projectID string) ([]pcf.Issue, error)
}

func (m *MockListIssuesClient) ListIssues(ctx context.Context, projectID string) ([]pcf.Issue, error) {
	if m.ListIssuesFunc != nil {
		return m.ListIssuesFunc(ctx, projectID)
	}
	return nil, errors.New("ListIssuesFunc not implemented")
}

// TestNewListIssuesTool tests creating a new list issues tool
func TestNewListIssuesTool(t *testing.T) {
	mockClient := &MockListIssuesClient{}

	tool := NewListIssuesTool(mockClient)

	if tool.Name != "list_issues" {
		t.Errorf("Expected tool name 'list_issues', got '%s'", tool.Name)
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

	if _, ok := props["project_id"]; !ok {
		t.Error("Input schema missing 'project_id' property")
	}

	// Check required fields
	required, ok := tool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Input schema should have required fields")
	}

	if len(required) == 0 || required[0] != "project_id" {
		t.Error("'project_id' should be a required field")
	}
}

// TestListIssuesHandler tests the list issues handler functionality
func TestListIssuesHandler(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]interface{}
		projectID     string
		mockResponse  []pcf.Issue
		mockError     error
		expectError   bool
		expectedCount int
	}{
		{
			name: "Successful list with issues",
			params: map[string]interface{}{
				"project_id": "proj-123",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Issue{
				{
					ID:          "issue-1",
					ProjectID:   "proj-123",
					HostID:      "host-1",
					Title:       "SQL Injection in Login Form",
					Description: "The login form is vulnerable to SQL injection attacks",
					Severity:    "Critical",
					Status:      "Open",
					CVE:         "CVE-2021-44228",
					CVSS:        9.8,
				},
				{
					ID:          "issue-2",
					ProjectID:   "proj-123",
					Title:       "Weak Password Policy",
					Description: "System allows weak passwords",
					Severity:    "Medium",
					Status:      "In Progress",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 2,
		},
		{
			name: "Empty issue list",
			params: map[string]interface{}{
				"project_id": "proj-empty",
			},
			projectID:     "proj-empty",
			mockResponse:  []pcf.Issue{},
			mockError:     nil,
			expectError:   false,
			expectedCount: 0,
		},
		{
			name:          "Missing project_id",
			params:        map[string]interface{}{},
			projectID:     "",
			mockResponse:  nil,
			mockError:     nil,
			expectError:   true,
			expectedCount: 0,
		},
		{
			name: "Invalid project_id type",
			params: map[string]interface{}{
				"project_id": 123, // Should be string
			},
			projectID:     "",
			mockResponse:  nil,
			mockError:     nil,
			expectError:   true,
			expectedCount: 0,
		},
		{
			name: "PCF API error",
			params: map[string]interface{}{
				"project_id": "proj-error",
			},
			projectID:     "proj-error",
			mockResponse:  nil,
			mockError:     errors.New("PCF connection failed"),
			expectError:   true,
			expectedCount: 0,
		},
		{
			name: "Filter by severity",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"severity":   "Critical",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Issue{
				{
					ID:       "issue-1",
					Title:    "Critical Issue",
					Severity: "Critical",
					Status:   "Open",
				},
				{
					ID:       "issue-2",
					Title:    "Medium Issue",
					Severity: "Medium",
					Status:   "Open",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 1, // Should filter out the Medium issue
		},
		{
			name: "Filter by status",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"status":     "Open",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Issue{
				{
					ID:       "issue-1",
					Title:    "Open Issue",
					Severity: "High",
					Status:   "Open",
				},
				{
					ID:       "issue-2",
					Title:    "Closed Issue",
					Severity: "Low",
					Status:   "Closed",
				},
				{
					ID:       "issue-3",
					Title:    "Another Open Issue",
					Severity: "Medium",
					Status:   "Open",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 2, // Should include only Open issues
		},
		{
			name: "Filter by host_id",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"host_id":    "host-1",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Issue{
				{
					ID:     "issue-1",
					HostID: "host-1",
					Title:  "Host 1 Issue",
				},
				{
					ID:     "issue-2",
					HostID: "host-2",
					Title:  "Host 2 Issue",
				},
				{
					ID:     "issue-3",
					HostID: "host-1",
					Title:  "Another Host 1 Issue",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 2, // Should include only host-1 issues
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockListIssuesClient{
				ListIssuesFunc: func(ctx context.Context, projectID string) ([]pcf.Issue, error) {
					if projectID != tt.projectID && tt.projectID != "" {
						t.Errorf("Expected project ID '%s', got '%s'", tt.projectID, projectID)
					}
					return tt.mockResponse, tt.mockError
				},
			}

			// Create tool
			tool := NewListIssuesTool(mockClient)

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

			// Check for issues key
			issuesData, ok := resultMap["issues"]
			if !ok {
				t.Fatal("Result should contain 'issues' key")
			}

			// Verify issues array
			issues, ok := issuesData.([]map[string]interface{})
			if !ok {
				t.Fatal("Issues should be an array of maps")
			}

			// Check count
			if len(issues) != tt.expectedCount {
				t.Errorf("Expected %d issues, got %d", tt.expectedCount, len(issues))
			}

			// Verify issue structure if we have issues
			if len(issues) > 0 {
				firstIssue := issues[0]

				// Check required fields
				requiredFields := []string{"id", "project_id", "title", "severity", "status"}
				for _, field := range requiredFields {
					if _, ok := firstIssue[field]; !ok {
						t.Errorf("Issue missing required field: %s", field)
					}
				}
			}

			// Check total count
			if totalCount, ok := resultMap["total_count"].(int); ok {
				if totalCount != tt.expectedCount {
					t.Errorf("Expected total_count %d, got %d", tt.expectedCount, totalCount)
				}
			} else {
				t.Error("Result should contain 'total_count' as int")
			}

			// Check severity breakdown if present
			if severityBreakdown, ok := resultMap["severity_breakdown"].(map[string]interface{}); ok {
				// Verify it's a map with counts
				if _, ok := severityBreakdown["Critical"]; !ok {
					t.Error("Severity breakdown should include all severity levels")
				}
			}
		})
	}
}
