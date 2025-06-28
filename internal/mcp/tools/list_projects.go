// Package tools provides MCP tool implementations for PCF operations
package tools

import (
	"context"
	"fmt"

	"github.com/analyst/pcf-mcp/internal/mcp"
	"github.com/analyst/pcf-mcp/internal/pcf"
)

// PCFClient defines the interface for PCF operations needed by tools
type PCFClient interface {
	// ListProjects retrieves all projects from PCF
	ListProjects(ctx context.Context) ([]pcf.Project, error)
}

// NewListProjectsTool creates an MCP tool for listing PCF projects
func NewListProjectsTool(client PCFClient) mcp.Tool {
	return mcp.Tool{
		Name:        "list_projects",
		Description: "List all projects in the Pentest Collaboration Framework",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter projects by status (active, completed, on-hold)",
					"enum":        []string{"active", "completed", "on-hold"},
				},
			},
			"additionalProperties": false,
		},
		Handler: createListProjectsHandler(client),
	}
}

// createListProjectsHandler creates the handler function for listing projects
func createListProjectsHandler(client PCFClient) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Validate parameters
		statusFilter := ""
		if status, ok := params["status"]; ok {
			statusStr, ok := status.(string)
			if !ok {
				return nil, fmt.Errorf("status parameter must be a string")
			}
			statusFilter = statusStr
		}
		
		// Call PCF client to list projects
		projects, err := client.ListProjects(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}
		
		// Convert projects to response format
		var projectList []map[string]interface{}
		
		for _, project := range projects {
			// Apply status filter if provided
			if statusFilter != "" && project.Status != statusFilter {
				continue
			}
			
			projectMap := map[string]interface{}{
				"id":          project.ID,
				"name":        project.Name,
				"description": project.Description,
				"status":      project.Status,
				"created_at":  project.CreatedAt.Format("2006-01-02T15:04:05Z"),
				"updated_at":  project.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			}
			
			// Add team members if present
			if len(project.Team) > 0 {
				projectMap["team"] = project.Team
			}
			
			projectList = append(projectList, projectMap)
		}
		
		// Build response
		response := map[string]interface{}{
			"projects":    projectList,
			"total_count": len(projectList),
		}
		
		return response, nil
	}
}