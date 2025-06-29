package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// MockListCredentialsClient extends MockPCFClient with ListCredentials method
type MockListCredentialsClient struct {
	MockPCFClient
	ListCredentialsFunc func(ctx context.Context, projectID string) ([]pcf.Credential, error)
}

func (m *MockListCredentialsClient) ListCredentials(ctx context.Context, projectID string) ([]pcf.Credential, error) {
	if m.ListCredentialsFunc != nil {
		return m.ListCredentialsFunc(ctx, projectID)
	}
	return nil, errors.New("ListCredentialsFunc not implemented")
}

// TestNewListCredentialsTool tests creating a new list credentials tool
func TestNewListCredentialsTool(t *testing.T) {
	mockClient := &MockListCredentialsClient{}

	tool := NewListCredentialsTool(mockClient)

	if tool.Name != "list_credentials" {
		t.Errorf("Expected tool name 'list_credentials', got '%s'", tool.Name)
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

// TestListCredentialsHandler tests the list credentials handler functionality
func TestListCredentialsHandler(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]interface{}
		projectID     string
		mockResponse  []pcf.Credential
		mockError     error
		expectError   bool
		expectedCount int
	}{
		{
			name: "Successful list with credentials",
			params: map[string]interface{}{
				"project_id": "proj-123",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Credential{
				{
					ID:        "cred-1",
					ProjectID: "proj-123",
					HostID:    "host-1",
					Type:      "password",
					Username:  "admin",
					Value:     "***encrypted***",
					Service:   "ssh",
					Notes:     "Found during initial scan",
				},
				{
					ID:        "cred-2",
					ProjectID: "proj-123",
					Type:      "hash",
					Username:  "user1",
					Value:     "***encrypted***",
					Service:   "smb",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 2,
		},
		{
			name: "Empty credential list",
			params: map[string]interface{}{
				"project_id": "proj-empty",
			},
			projectID:     "proj-empty",
			mockResponse:  []pcf.Credential{},
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
			name: "Filter by type",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"type":       "password",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Credential{
				{
					ID:       "cred-1",
					Type:     "password",
					Username: "admin",
				},
				{
					ID:       "cred-2",
					Type:     "hash",
					Username: "user1",
				},
				{
					ID:       "cred-3",
					Type:     "password",
					Username: "root",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 2, // Should filter out the hash type
		},
		{
			name: "Filter by host_id",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"host_id":    "host-1",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Credential{
				{
					ID:     "cred-1",
					HostID: "host-1",
				},
				{
					ID:     "cred-2",
					HostID: "host-2",
				},
				{
					ID:     "cred-3",
					HostID: "host-1",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 2, // Should include only host-1 credentials
		},
		{
			name: "Filter by service",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"service":    "ssh",
			},
			projectID: "proj-123",
			mockResponse: []pcf.Credential{
				{
					ID:      "cred-1",
					Service: "ssh",
				},
				{
					ID:      "cred-2",
					Service: "rdp",
				},
				{
					ID:      "cred-3",
					Service: "ssh",
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedCount: 2, // Should include only ssh credentials
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockListCredentialsClient{
				ListCredentialsFunc: func(ctx context.Context, projectID string) ([]pcf.Credential, error) {
					if projectID != tt.projectID && tt.projectID != "" {
						t.Errorf("Expected project ID '%s', got '%s'", tt.projectID, projectID)
					}
					return tt.mockResponse, tt.mockError
				},
			}

			// Create tool
			tool := NewListCredentialsTool(mockClient)

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

			// Check for credentials key
			credentialsData, ok := resultMap["credentials"]
			if !ok {
				t.Fatal("Result should contain 'credentials' key")
			}

			// Verify credentials array
			credentials, ok := credentialsData.([]map[string]interface{})
			if !ok {
				t.Fatal("Credentials should be an array of maps")
			}

			// Check count
			if len(credentials) != tt.expectedCount {
				t.Errorf("Expected %d credentials, got %d", tt.expectedCount, len(credentials))
			}

			// Verify credential structure if we have credentials
			if len(credentials) > 0 {
				firstCred := credentials[0]

				// Check required fields
				requiredFields := []string{"id", "project_id", "type", "username"}
				for _, field := range requiredFields {
					if _, ok := firstCred[field]; !ok {
						t.Errorf("Credential missing required field: %s", field)
					}
				}

				// Value should always be masked
				if value, ok := firstCred["value"].(string); ok {
					if value != "***REDACTED***" {
						t.Error("Credential value should be redacted")
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

			// Check type breakdown if present
			if typeBreakdown, ok := resultMap["type_breakdown"].(map[string]interface{}); ok {
				// Verify it's a map with counts
				if len(typeBreakdown) == 0 && len(tt.mockResponse) > 0 {
					t.Error("Type breakdown should not be empty when credentials exist")
				}
			}
		})
	}
}
