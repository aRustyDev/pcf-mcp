package tools

import (
	"context"
	"fmt"

	"github.com/aRustyDev/pcf-mcp/internal/mcp"
	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// CreateProjectClient defines the interface for creating projects
type CreateProjectClient interface {
	CreateProject(ctx context.Context, req pcf.CreateProjectRequest) (*pcf.Project, error)
}

// NewCreateProjectTool creates an MCP tool for creating PCF projects
func NewCreateProjectTool(client CreateProjectClient) mcp.Tool {
	return mcp.Tool{
		Name:        "create_project",
		Description: "Create a new project in the Pentest Collaboration Framework",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the project",
					"minLength":   1,
					"maxLength":   100,
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "A description of the project",
					"maxLength":   500,
				},
				"team": map[string]interface{}{
					"type":        "array",
					"description": "List of team member usernames",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required":             []string{"name"},
			"additionalProperties": false,
		},
		Handler: createCreateProjectHandler(client),
	}
}

// createCreateProjectHandler creates the handler function for creating projects
func createCreateProjectHandler(client CreateProjectClient) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Extract and validate name
		name, ok := params["name"].(string)
		if !ok {
			return nil, fmt.Errorf("name parameter must be a string")
		}

		if name == "" {
			return nil, fmt.Errorf("project name cannot be empty")
		}

		// Create request
		req := pcf.CreateProjectRequest{
			Name: name,
		}

		// Extract optional description
		if desc, ok := params["description"].(string); ok {
			req.Description = desc
		}

		// Extract optional team members
		if teamRaw, ok := params["team"]; ok {
			// Handle different types that might come from JSON
			switch team := teamRaw.(type) {
			case []string:
				req.Team = team
			case []interface{}:
				// Convert []interface{} to []string
				teamMembers := make([]string, 0, len(team))
				for _, member := range team {
					if memberStr, ok := member.(string); ok {
						teamMembers = append(teamMembers, memberStr)
					} else {
						return nil, fmt.Errorf("team members must be strings")
					}
				}
				req.Team = teamMembers
			default:
				return nil, fmt.Errorf("team parameter must be an array of strings")
			}
		}

		// Call PCF client to create project
		project, err := client.CreateProject(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create project: %w", err)
		}

		// Build response
		response := map[string]interface{}{
			"project": map[string]interface{}{
				"id":          project.ID,
				"name":        project.Name,
				"description": project.Description,
				"status":      project.Status,
				"created_at":  project.CreatedAt.Format("2006-01-02T15:04:05Z"),
				"updated_at":  project.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			},
			"message": fmt.Sprintf("Project '%s' created successfully", project.Name),
		}

		// Add team if present
		if len(project.Team) > 0 {
			response["project"].(map[string]interface{})["team"] = project.Team
		}

		return response, nil
	}
}
