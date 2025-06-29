package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// TestNewCreateProjectTool tests creating a new create project tool
func TestNewCreateProjectTool(t *testing.T) {
	mockClient := &MockCreateProjectClient{}

	tool := NewCreateProjectTool(mockClient)

	if tool.Name != "create_project" {
		t.Errorf("Expected tool name 'create_project', got '%s'", tool.Name)
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

	if _, ok := props["name"]; !ok {
		t.Error("Input schema missing 'name' property")
	}

	if _, ok := props["description"]; !ok {
		t.Error("Input schema missing 'description' property")
	}

	// Check required fields
	required, ok := tool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Input schema should have required fields")
	}

	if len(required) == 0 || required[0] != "name" {
		t.Error("'name' should be a required field")
	}
}

// MockCreateProjectClient extends MockPCFClient with CreateProject method
type MockCreateProjectClient struct {
	MockPCFClient
	CreateProjectFunc func(ctx context.Context, req pcf.CreateProjectRequest) (*pcf.Project, error)
}

func (m *MockCreateProjectClient) CreateProject(ctx context.Context, req pcf.CreateProjectRequest) (*pcf.Project, error) {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, req)
	}
	return nil, errors.New("CreateProjectFunc not implemented")
}

// TestCreateProjectHandler tests the create project handler functionality
func TestCreateProjectHandler(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		mockResponse   *pcf.Project
		mockError      error
		expectError    bool
		validateResult func(t *testing.T, result interface{})
	}{
		{
			name: "Successful project creation",
			params: map[string]interface{}{
				"name":        "Test Project",
				"description": "A test project for PCF",
				"team":        []string{"alice", "bob"},
			},
			mockResponse: &pcf.Project{
				ID:          "proj-123",
				Name:        "Test Project",
				Description: "A test project for PCF",
				Team:        []string{"alice", "bob"},
				Status:      "active",
			},
			mockError:   nil,
			expectError: false,
			validateResult: func(t *testing.T, result interface{}) {
				res, ok := result.(map[string]interface{})
				if !ok {
					t.Fatal("Result should be a map")
				}

				project, ok := res["project"].(map[string]interface{})
				if !ok {
					t.Fatal("Result should contain 'project' key")
				}

				if project["id"] != "proj-123" {
					t.Errorf("Expected project ID 'proj-123', got '%v'", project["id"])
				}

				if project["name"] != "Test Project" {
					t.Errorf("Expected project name 'Test Project', got '%v'", project["name"])
				}
			},
		},
		{
			name: "Missing required name",
			params: map[string]interface{}{
				"description": "A project without name",
			},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Invalid name type",
			params: map[string]interface{}{
				"name":        123, // Should be string
				"description": "Invalid name type",
			},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "PCF API error",
			params: map[string]interface{}{
				"name":        "Failed Project",
				"description": "This will fail",
			},
			mockResponse: nil,
			mockError:    errors.New("PCF API error"),
			expectError:  true,
		},
		{
			name: "Empty name",
			params: map[string]interface{}{
				"name":        "",
				"description": "Empty name",
			},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Invalid team type",
			params: map[string]interface{}{
				"name":        "Test Project",
				"description": "Test",
				"team":        "not-an-array", // Should be array
			},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockCreateProjectClient{
				CreateProjectFunc: func(ctx context.Context, req pcf.CreateProjectRequest) (*pcf.Project, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			// Create tool
			tool := NewCreateProjectTool(mockClient)

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

			// Validate result
			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}
