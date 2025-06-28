package tools

import (
	"context"
	"testing"

	"github.com/analyst/pcf-mcp/internal/config"
	"github.com/analyst/pcf-mcp/internal/mcp"
	"github.com/analyst/pcf-mcp/internal/pcf"
)

// MockFullPCFClient implements all PCF client interfaces for testing
type MockFullPCFClient struct {
	ListProjectsFunc  func(ctx context.Context) ([]pcf.Project, error)
	CreateProjectFunc func(ctx context.Context, req pcf.CreateProjectRequest) (*pcf.Project, error)
	ListHostsFunc     func(ctx context.Context, projectID string) ([]pcf.Host, error)
	AddHostFunc       func(ctx context.Context, projectID string, req pcf.CreateHostRequest) (*pcf.Host, error)
	ListIssuesFunc    func(ctx context.Context, projectID string) ([]pcf.Issue, error)
	CreateIssueFunc   func(ctx context.Context, projectID string, req pcf.CreateIssueRequest) (*pcf.Issue, error)
}

func (m *MockFullPCFClient) ListProjects(ctx context.Context) ([]pcf.Project, error) {
	if m.ListProjectsFunc != nil {
		return m.ListProjectsFunc(ctx)
	}
	return nil, nil
}

func (m *MockFullPCFClient) CreateProject(ctx context.Context, req pcf.CreateProjectRequest) (*pcf.Project, error) {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockFullPCFClient) ListHosts(ctx context.Context, projectID string) ([]pcf.Host, error) {
	if m.ListHostsFunc != nil {
		return m.ListHostsFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *MockFullPCFClient) AddHost(ctx context.Context, projectID string, req pcf.CreateHostRequest) (*pcf.Host, error) {
	if m.AddHostFunc != nil {
		return m.AddHostFunc(ctx, projectID, req)
	}
	return nil, nil
}

func (m *MockFullPCFClient) ListIssues(ctx context.Context, projectID string) ([]pcf.Issue, error) {
	if m.ListIssuesFunc != nil {
		return m.ListIssuesFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *MockFullPCFClient) CreateIssue(ctx context.Context, projectID string, req pcf.CreateIssueRequest) (*pcf.Issue, error) {
	if m.CreateIssueFunc != nil {
		return m.CreateIssueFunc(ctx, projectID, req)
	}
	return nil, nil
}

// TestRegisterAllTools tests registering all PCF tools with the MCP server
func TestRegisterAllTools(t *testing.T) {
	// Create MCP server
	cfg := config.ServerConfig{
		Transport: "stdio",
	}
	
	server, err := mcp.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	
	// Create mock PCF client with all required methods
	mockClient := &MockFullPCFClient{
		ListProjectsFunc: func(ctx context.Context) ([]pcf.Project, error) {
			return []pcf.Project{
				{ID: "test", Name: "Test Project"},
			}, nil
		},
		CreateProjectFunc: func(ctx context.Context, req pcf.CreateProjectRequest) (*pcf.Project, error) {
			return &pcf.Project{
				ID:   "new-test",
				Name: req.Name,
			}, nil
		},
		ListHostsFunc: func(ctx context.Context, projectID string) ([]pcf.Host, error) {
			return []pcf.Host{
				{ID: "host-1", IP: "192.168.1.100"},
			}, nil
		},
	}
	
	// Register all tools
	err = RegisterAllTools(server, mockClient)
	if err != nil {
		t.Fatalf("Failed to register tools: %v", err)
	}
	
	// Verify tools are registered
	tools := server.ListTools()
	
	// Check that we have at least one tool
	if len(tools) == 0 {
		t.Error("No tools were registered")
	}
	
	// Find list_projects tool
	found := false
	for _, tool := range tools {
		if tool.Name == "list_projects" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("list_projects tool was not registered")
	}
	
	// Test executing the tool
	ctx := context.Background()
	result, err := server.ExecuteTool(ctx, "list_projects", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to execute list_projects tool: %v", err)
	}
	
	// Verify result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result should be a map")
	}
	
	if _, ok := resultMap["projects"]; !ok {
		t.Error("Result should contain 'projects' key")
	}
}