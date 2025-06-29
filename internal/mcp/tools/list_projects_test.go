package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// MockPCFClient is a mock implementation of the PCF client for testing
type MockPCFClient struct {
	// ListProjectsFunc allows customizing the ListProjects behavior
	ListProjectsFunc func(ctx context.Context) ([]pcf.Project, error)
}

// ListProjects implements the PCF client interface
func (m *MockPCFClient) ListProjects(ctx context.Context) ([]pcf.Project, error) {
	if m.ListProjectsFunc != nil {
		return m.ListProjectsFunc(ctx)
	}
	return nil, errors.New("ListProjectsFunc not implemented")
}

// TestNewListProjectsTool tests creating a new list projects tool
func TestNewListProjectsTool(t *testing.T) {
	mockClient := &MockPCFClient{}

	tool := NewListProjectsTool(mockClient)

	if tool.Name != "list_projects" {
		t.Errorf("Expected tool name 'list_projects', got '%s'", tool.Name)
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

	// Verify schema structure
	schemaType, ok := tool.InputSchema["type"].(string)
	if !ok || schemaType != "object" {
		t.Error("Input schema should be of type 'object'")
	}
}

// TestListProjectsHandler tests the list projects handler functionality
func TestListProjectsHandler(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  []pcf.Project
		mockError     error
		params        map[string]interface{}
		expectError   bool
		expectedCount int
	}{
		{
			name: "Successful list with projects",
			mockResponse: []pcf.Project{
				{
					ID:          "proj1",
					Name:        "Test Project 1",
					Description: "First test project",
					Status:      "active",
				},
				{
					ID:          "proj2",
					Name:        "Test Project 2",
					Description: "Second test project",
					Status:      "completed",
				},
			},
			mockError:     nil,
			params:        map[string]interface{}{},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:          "Empty project list",
			mockResponse:  []pcf.Project{},
			mockError:     nil,
			params:        map[string]interface{}{},
			expectError:   false,
			expectedCount: 0,
		},
		{
			name:         "PCF client error",
			mockResponse: nil,
			mockError:    errors.New("PCF connection failed"),
			params:       map[string]interface{}{},
			expectError:  true,
		},
		{
			name: "Filter by status",
			mockResponse: []pcf.Project{
				{
					ID:     "proj1",
					Name:   "Active Project",
					Status: "active",
				},
				{
					ID:     "proj2",
					Name:   "Completed Project",
					Status: "completed",
				},
			},
			mockError: nil,
			params: map[string]interface{}{
				"status": "active",
			},
			expectError:   false,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockPCFClient{
				ListProjectsFunc: func(ctx context.Context) ([]pcf.Project, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			// Create tool
			tool := NewListProjectsTool(mockClient)

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

			// Check for projects key
			projectsData, ok := resultMap["projects"]
			if !ok {
				t.Fatal("Result should contain 'projects' key")
			}

			// Verify projects array
			projects, ok := projectsData.([]map[string]interface{})
			if !ok {
				t.Fatal("Projects should be an array of maps")
			}

			// Check count
			if len(projects) != tt.expectedCount {
				t.Errorf("Expected %d projects, got %d", tt.expectedCount, len(projects))
			}

			// Verify project structure if we have projects
			if len(projects) > 0 {
				firstProject := projects[0]

				// Check required fields
				requiredFields := []string{"id", "name", "description", "status"}
				for _, field := range requiredFields {
					if _, ok := firstProject[field]; !ok {
						t.Errorf("Project missing required field: %s", field)
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
		})
	}
}

// TestListProjectsInputValidation tests input parameter validation
func TestListProjectsInputValidation(t *testing.T) {
	mockClient := &MockPCFClient{
		ListProjectsFunc: func(ctx context.Context) ([]pcf.Project, error) {
			return []pcf.Project{}, nil
		},
	}

	tool := NewListProjectsTool(mockClient)
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
	}{
		{
			name:        "Valid empty params",
			params:      map[string]interface{}{},
			expectError: false,
		},
		{
			name: "Valid status filter",
			params: map[string]interface{}{
				"status": "active",
			},
			expectError: false,
		},
		{
			name: "Invalid status type",
			params: map[string]interface{}{
				"status": 123, // Should be string
			},
			expectError: true,
		},
		{
			name: "Unknown parameter",
			params: map[string]interface{}{
				"unknown": "value",
			},
			expectError: false, // Should ignore unknown params
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Handler(ctx, tt.params)

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestListProjectsContextCancellation tests that the handler respects context cancellation
func TestListProjectsContextCancellation(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockClient := &MockPCFClient{
		ListProjectsFunc: func(ctx context.Context) ([]pcf.Project, error) {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return []pcf.Project{}, nil
			}
		},
	}

	tool := NewListProjectsTool(mockClient)

	_, err := tool.Handler(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}
