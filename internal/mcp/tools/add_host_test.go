package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/analyst/pcf-mcp/internal/pcf"
)

// MockAddHostClient extends MockPCFClient with AddHost method
type MockAddHostClient struct {
	MockPCFClient
	AddHostFunc func(ctx context.Context, projectID string, req pcf.CreateHostRequest) (*pcf.Host, error)
}

func (m *MockAddHostClient) AddHost(ctx context.Context, projectID string, req pcf.CreateHostRequest) (*pcf.Host, error) {
	if m.AddHostFunc != nil {
		return m.AddHostFunc(ctx, projectID, req)
	}
	return nil, errors.New("AddHostFunc not implemented")
}

// TestNewAddHostTool tests creating a new add host tool
func TestNewAddHostTool(t *testing.T) {
	mockClient := &MockAddHostClient{}
	
	tool := NewAddHostTool(mockClient)
	
	if tool.Name != "add_host" {
		t.Errorf("Expected tool name 'add_host', got '%s'", tool.Name)
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
	
	requiredProps := []string{"project_id", "ip"}
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
	
	if len(required) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(required))
	}
}

// TestAddHostHandler tests the add host handler functionality
func TestAddHostHandler(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		expectedReq  pcf.CreateHostRequest
		mockResponse *pcf.Host
		mockError    error
		expectError  bool
		checkResult  func(t *testing.T, result interface{})
	}{
		{
			name: "Add host with minimal info",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"ip":         "192.168.1.100",
			},
			expectedReq: pcf.CreateHostRequest{
				IP: "192.168.1.100",
			},
			mockResponse: &pcf.Host{
				ID:        "host-new",
				ProjectID: "proj-123",
				IP:        "192.168.1.100",
				Status:    "active",
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Add host with full details",
			params: map[string]interface{}{
				"project_id": "proj-456",
				"ip":         "10.0.0.50",
				"hostname":   "target.example.com",
				"os":         "Linux",
				"services":   []string{"ssh", "http", "https"},
			},
			expectedReq: pcf.CreateHostRequest{
				IP:       "10.0.0.50",
				Hostname: "target.example.com",
				OS:       "Linux",
				Services: []string{"ssh", "http", "https"},
			},
			mockResponse: &pcf.Host{
				ID:        "host-full",
				ProjectID: "proj-456",
				IP:        "10.0.0.50",
				Hostname:  "target.example.com",
				OS:        "Linux",
				Services:  []string{"ssh", "http", "https"},
				Status:    "active",
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "Missing project_id",
			params:      map[string]interface{}{
				"ip": "192.168.1.100",
			},
			expectedReq:  pcf.CreateHostRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name:        "Missing IP address",
			params:      map[string]interface{}{
				"project_id": "proj-123",
			},
			expectedReq:  pcf.CreateHostRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Invalid IP address format",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"ip":         "not-an-ip",
			},
			expectedReq:  pcf.CreateHostRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Invalid project_id type",
			params: map[string]interface{}{
				"project_id": 123, // Should be string
				"ip":         "192.168.1.100",
			},
			expectedReq:  pcf.CreateHostRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "PCF API error",
			params: map[string]interface{}{
				"project_id": "proj-error",
				"ip":         "192.168.1.100",
			},
			expectedReq: pcf.CreateHostRequest{
				IP: "192.168.1.100",
			},
			mockResponse: nil,
			mockError:    errors.New("project not found"),
			expectError:  true,
		},
		{
			name: "Services as interface array",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"ip":         "192.168.1.100",
				"services":   []interface{}{"ssh", "http"},
			},
			expectedReq: pcf.CreateHostRequest{
				IP:       "192.168.1.100",
				Services: []string{"ssh", "http"},
			},
			mockResponse: &pcf.Host{
				ID:        "host-services",
				ProjectID: "proj-123",
				IP:        "192.168.1.100",
				Services:  []string{"ssh", "http"},
				Status:    "active",
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Duplicate host",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"ip":         "192.168.1.100",
			},
			expectedReq: pcf.CreateHostRequest{
				IP: "192.168.1.100",
			},
			mockResponse: nil,
			mockError:    errors.New("host with IP 192.168.1.100 already exists"),
			expectError:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockAddHostClient{
				AddHostFunc: func(ctx context.Context, projectID string, req pcf.CreateHostRequest) (*pcf.Host, error) {
					// Verify project ID if we expect the call to succeed
					if !tt.expectError || tt.mockError != nil {
						expectedProjectID, _ := tt.params["project_id"].(string)
						if projectID != expectedProjectID {
							t.Errorf("Expected project ID '%s', got '%s'", expectedProjectID, projectID)
						}
						
						// Verify request structure
						if req.IP != tt.expectedReq.IP {
							t.Errorf("Expected IP '%s', got '%s'", tt.expectedReq.IP, req.IP)
						}
						if req.Hostname != tt.expectedReq.Hostname {
							t.Errorf("Expected hostname '%s', got '%s'", tt.expectedReq.Hostname, req.Hostname)
						}
						if req.OS != tt.expectedReq.OS {
							t.Errorf("Expected OS '%s', got '%s'", tt.expectedReq.OS, req.OS)
						}
						
						// Check services array
						if len(req.Services) != len(tt.expectedReq.Services) {
							t.Errorf("Expected %d services, got %d", len(tt.expectedReq.Services), len(req.Services))
						}
					}
					
					return tt.mockResponse, tt.mockError
				},
			}
			
			// Create tool
			tool := NewAddHostTool(mockClient)
			
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
			
			// Check for host key
			hostData, ok := resultMap["host"]
			if !ok {
				t.Fatal("Result should contain 'host' key")
			}
			
			// Verify host structure
			host, ok := hostData.(map[string]interface{})
			if !ok {
				t.Fatal("Host should be a map")
			}
			
			// Check required fields
			requiredFields := []string{"id", "project_id", "ip", "status"}
			for _, field := range requiredFields {
				if _, ok := host[field]; !ok {
					t.Errorf("Host missing required field: %s", field)
				}
			}
			
			// Check message
			if message, ok := resultMap["message"].(string); !ok || message == "" {
				t.Error("Result should contain a non-empty message")
			}
			
			// Run custom result check if provided
			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}