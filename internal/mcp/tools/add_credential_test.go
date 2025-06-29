package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// MockAddCredentialClient extends MockPCFClient with AddCredential method
type MockAddCredentialClient struct {
	MockPCFClient
	AddCredentialFunc func(ctx context.Context, projectID string, req pcf.AddCredentialRequest) (*pcf.Credential, error)
}

func (m *MockAddCredentialClient) AddCredential(ctx context.Context, projectID string, req pcf.AddCredentialRequest) (*pcf.Credential, error) {
	if m.AddCredentialFunc != nil {
		return m.AddCredentialFunc(ctx, projectID, req)
	}
	return nil, errors.New("AddCredentialFunc not implemented")
}

// TestNewAddCredentialTool tests creating a new add credential tool
func TestNewAddCredentialTool(t *testing.T) {
	mockClient := &MockAddCredentialClient{}

	tool := NewAddCredentialTool(mockClient)

	if tool.Name != "add_credential" {
		t.Errorf("Expected tool name 'add_credential', got '%s'", tool.Name)
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

	requiredProps := []string{"project_id", "type", "username", "value"}
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

// TestAddCredentialHandler tests the add credential handler functionality
func TestAddCredentialHandler(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		expectedReq  pcf.AddCredentialRequest
		mockResponse *pcf.Credential
		mockError    error
		expectError  bool
	}{
		{
			name: "Add credential with minimal info",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"type":       "password",
				"username":   "admin",
				"value":      "P@ssw0rd123",
			},
			expectedReq: pcf.AddCredentialRequest{
				Type:     "password",
				Username: "admin",
				Value:    "P@ssw0rd123",
			},
			mockResponse: &pcf.Credential{
				ID:        "cred-new",
				ProjectID: "proj-123",
				Type:      "password",
				Username:  "admin",
				Value:     "***encrypted***",
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Add credential with full details",
			params: map[string]interface{}{
				"project_id": "proj-456",
				"type":       "hash",
				"username":   "root",
				"value":      "$2a$10$...",
				"host_id":    "host-123",
				"service":    "ssh",
				"notes":      "Found in /etc/shadow",
			},
			expectedReq: pcf.AddCredentialRequest{
				HostID:   "host-123",
				Type:     "hash",
				Username: "root",
				Value:    "$2a$10$...",
				Service:  "ssh",
				Notes:    "Found in /etc/shadow",
			},
			mockResponse: &pcf.Credential{
				ID:        "cred-full",
				ProjectID: "proj-456",
				HostID:    "host-123",
				Type:      "hash",
				Username:  "root",
				Value:     "***encrypted***",
				Service:   "ssh",
				Notes:     "Found in /etc/shadow",
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Missing project_id",
			params: map[string]interface{}{
				"type":     "password",
				"username": "admin",
				"value":    "password",
			},
			expectedReq:  pcf.AddCredentialRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Missing type",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"username":   "admin",
				"value":      "password",
			},
			expectedReq:  pcf.AddCredentialRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Missing username",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"type":       "password",
				"value":      "password",
			},
			expectedReq:  pcf.AddCredentialRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Missing value",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"type":       "password",
				"username":   "admin",
			},
			expectedReq:  pcf.AddCredentialRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Invalid type value",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"type":       "invalid_type",
				"username":   "admin",
				"value":      "password",
			},
			expectedReq:  pcf.AddCredentialRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Invalid project_id type",
			params: map[string]interface{}{
				"project_id": 123, // Should be string
				"type":       "password",
				"username":   "admin",
				"value":      "password",
			},
			expectedReq:  pcf.AddCredentialRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "PCF API error",
			params: map[string]interface{}{
				"project_id": "proj-error",
				"type":       "password",
				"username":   "admin",
				"value":      "password",
			},
			expectedReq: pcf.AddCredentialRequest{
				Type:     "password",
				Username: "admin",
				Value:    "password",
			},
			mockResponse: nil,
			mockError:    errors.New("project not found"),
			expectError:  true,
		},
		{
			name: "Empty username",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"type":       "password",
				"username":   "",
				"value":      "password",
			},
			expectedReq:  pcf.AddCredentialRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Empty value",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"type":       "password",
				"username":   "admin",
				"value":      "",
			},
			expectedReq:  pcf.AddCredentialRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "SSH key credential",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"type":       "key",
				"username":   "ubuntu",
				"value":      "-----BEGIN RSA PRIVATE KEY-----\n...",
				"service":    "ssh",
			},
			expectedReq: pcf.AddCredentialRequest{
				Type:     "key",
				Username: "ubuntu",
				Value:    "-----BEGIN RSA PRIVATE KEY-----\n...",
				Service:  "ssh",
			},
			mockResponse: &pcf.Credential{
				ID:        "cred-key",
				ProjectID: "proj-123",
				Type:      "key",
				Username:  "ubuntu",
				Value:     "***encrypted***",
				Service:   "ssh",
			},
			mockError:   nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockAddCredentialClient{
				AddCredentialFunc: func(ctx context.Context, projectID string, req pcf.AddCredentialRequest) (*pcf.Credential, error) {
					// Verify project ID if we expect the call to succeed
					if !tt.expectError || tt.mockError != nil {
						expectedProjectID, _ := tt.params["project_id"].(string)
						if projectID != expectedProjectID {
							t.Errorf("Expected project ID '%s', got '%s'", expectedProjectID, projectID)
						}

						// Verify request structure
						if req.Type != tt.expectedReq.Type {
							t.Errorf("Expected type '%s', got '%s'", tt.expectedReq.Type, req.Type)
						}
						if req.Username != tt.expectedReq.Username {
							t.Errorf("Expected username '%s', got '%s'", tt.expectedReq.Username, req.Username)
						}
						if req.Value != tt.expectedReq.Value {
							t.Errorf("Expected value '%s', got '%s'", tt.expectedReq.Value, req.Value)
						}
						if req.HostID != tt.expectedReq.HostID {
							t.Errorf("Expected host ID '%s', got '%s'", tt.expectedReq.HostID, req.HostID)
						}
						if req.Service != tt.expectedReq.Service {
							t.Errorf("Expected service '%s', got '%s'", tt.expectedReq.Service, req.Service)
						}
						if req.Notes != tt.expectedReq.Notes {
							t.Errorf("Expected notes '%s', got '%s'", tt.expectedReq.Notes, req.Notes)
						}
					}

					return tt.mockResponse, tt.mockError
				},
			}

			// Create tool
			tool := NewAddCredentialTool(mockClient)

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

			// Check for credential key
			credentialData, ok := resultMap["credential"]
			if !ok {
				t.Fatal("Result should contain 'credential' key")
			}

			// Verify credential structure
			credential, ok := credentialData.(map[string]interface{})
			if !ok {
				t.Fatal("Credential should be a map")
			}

			// Check required fields
			requiredFields := []string{"id", "project_id", "type", "username"}
			for _, field := range requiredFields {
				if _, ok := credential[field]; !ok {
					t.Errorf("Credential missing required field: %s", field)
				}
			}

			// Value should be redacted
			if value, ok := credential["value"].(string); ok {
				if value != "***REDACTED***" {
					t.Error("Credential value should be redacted in response")
				}
			}

			// Check message
			if message, ok := resultMap["message"].(string); !ok || message == "" {
				t.Error("Result should contain a non-empty message")
			}
		})
	}
}
