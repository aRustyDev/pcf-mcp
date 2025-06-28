package tools

import (
	"context"
	"testing"

	"github.com/analyst/pcf-mcp/internal/config"
	"github.com/analyst/pcf-mcp/internal/mcp"
	"github.com/analyst/pcf-mcp/internal/pcf"
)

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
	
	// Create mock PCF client
	mockClient := &MockPCFClient{
		ListProjectsFunc: func(ctx context.Context) ([]pcf.Project, error) {
			return []pcf.Project{
				{ID: "test", Name: "Test Project"},
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