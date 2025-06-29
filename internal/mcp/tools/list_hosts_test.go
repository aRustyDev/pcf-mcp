package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// MockListHostsClient extends MockPCFClient with ListHosts method
type MockListHostsClient struct {
	MockPCFClient
	ListHostsFunc func(ctx context.Context, projectID string) ([]pcf.Host, error)
}

func (m *MockListHostsClient) ListHosts(ctx context.Context, projectID string) ([]pcf.Host, error) {
	if m.ListHostsFunc != nil {
		return m.ListHostsFunc(ctx, projectID)
	}
	return nil, errors.New("ListHostsFunc not implemented")
}

// TestNewListHostsTool tests creating a new list hosts tool
func TestNewListHostsTool(t *testing.T) {
	mockClient := &MockListHostsClient{}

	tool := NewListHostsTool(mockClient)

	if tool.Name != "list_hosts" {
		t.Errorf("Expected tool name 'list_hosts', got '%s'", tool.Name)
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

// TestListHostsHandler tests the list hosts handler functionality
func TestListHostsHandler(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]interface{}
		projectID     string
		mockResponse  []pcf.Host
		mockError     error
		expectError   bool
		expectedCount int
	}{
		{
			name: "Successful list with hosts",
			params: map[string]interface{}{
				"project_id": "proj-123",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Host{
				{
					ID:        "host-1",
					ProjectID: "proj-123",
					IP:        "192.168.1.100",
					Hostname:  "target1.example.com",
					OS:        "Linux",
					Services:  []string{"ssh", "http", "https"},
					Status:    "active",
				},
				{
					ID:        "host-2",
					ProjectID: "proj-123",
					IP:        "192.168.1.101",
					Hostname:  "target2.example.com",
					OS:        "Windows",
					Services:  []string{"rdp", "smb"},
					Status:    "active",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 2,
		},
		{
			name: "Empty host list",
			params: map[string]interface{}{
				"project_id": "proj-empty",
			},
			projectID:     "proj-empty",
			mockResponse:  []pcf.Host{},
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
			name: "Filter by status",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"status":     "active",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Host{
				{
					ID:     "host-1",
					IP:     "192.168.1.100",
					Status: "active",
				},
				{
					ID:     "host-2",
					IP:     "192.168.1.101",
					Status: "inactive",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 1,
		},
		{
			name: "Filter by OS",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"os":         "Linux",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Host{
				{
					ID: "host-1",
					IP: "192.168.1.100",
					OS: "Linux",
				},
				{
					ID: "host-2",
					IP: "192.168.1.101",
					OS: "Windows",
				},
				{
					ID: "host-3",
					IP: "192.168.1.102",
					OS: "Linux",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockListHostsClient{
				ListHostsFunc: func(ctx context.Context, projectID string) ([]pcf.Host, error) {
					if projectID != tt.projectID && tt.projectID != "" {
						t.Errorf("Expected project ID '%s', got '%s'", tt.projectID, projectID)
					}
					return tt.mockResponse, tt.mockError
				},
			}

			// Create tool
			tool := NewListHostsTool(mockClient)

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

			// Check for hosts key
			hostsData, ok := resultMap["hosts"]
			if !ok {
				t.Fatal("Result should contain 'hosts' key")
			}

			// Verify hosts array
			hosts, ok := hostsData.([]map[string]interface{})
			if !ok {
				t.Fatal("Hosts should be an array of maps")
			}

			// Check count
			if len(hosts) != tt.expectedCount {
				t.Errorf("Expected %d hosts, got %d", tt.expectedCount, len(hosts))
			}

			// Verify host structure if we have hosts
			if len(hosts) > 0 {
				firstHost := hosts[0]

				// Check required fields
				requiredFields := []string{"id", "ip", "project_id"}
				for _, field := range requiredFields {
					if _, ok := firstHost[field]; !ok {
						t.Errorf("Host missing required field: %s", field)
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

			// Check project_id in result
			if projectID, ok := resultMap["project_id"].(string); ok {
				if tt.projectID != "" && projectID != tt.projectID {
					t.Errorf("Expected project_id '%s', got '%s'", tt.projectID, projectID)
				}
			}
		})
	}
}
