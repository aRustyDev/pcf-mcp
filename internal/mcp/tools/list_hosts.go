package tools

import (
	"context"
	"fmt"

	"github.com/analyst/pcf-mcp/internal/mcp"
	"github.com/analyst/pcf-mcp/internal/pcf"
)

// ListHostsClient defines the interface for listing hosts
type ListHostsClient interface {
	ListHosts(ctx context.Context, projectID string) ([]pcf.Host, error)
}

// NewListHostsTool creates an MCP tool for listing hosts in a PCF project
func NewListHostsTool(client ListHostsClient) mcp.Tool {
	return mcp.Tool{
		Name:        "list_hosts",
		Description: "List all hosts in a specific PCF project",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the project to list hosts from",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter hosts by status (active, inactive)",
					"enum":        []string{"active", "inactive"},
				},
				"os": map[string]interface{}{
					"type":        "string",
					"description": "Filter hosts by operating system",
				},
			},
			"required":             []string{"project_id"},
			"additionalProperties": false,
		},
		Handler: createListHostsHandler(client),
	}
}

// createListHostsHandler creates the handler function for listing hosts
func createListHostsHandler(client ListHostsClient) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Extract and validate project_id
		projectID, ok := params["project_id"].(string)
		if !ok {
			return nil, fmt.Errorf("project_id parameter must be a string")
		}
		
		if projectID == "" {
			return nil, fmt.Errorf("project_id cannot be empty")
		}
		
		// Extract optional filters
		statusFilter := ""
		if status, ok := params["status"].(string); ok {
			statusFilter = status
		}
		
		osFilter := ""
		if osParam, ok := params["os"].(string); ok {
			osFilter = osParam
		}
		
		// Call PCF client to list hosts
		hosts, err := client.ListHosts(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to list hosts: %w", err)
		}
		
		// Convert hosts to response format and apply filters
		var hostList []map[string]interface{}
		
		for _, host := range hosts {
			// Apply status filter if provided
			if statusFilter != "" && host.Status != statusFilter {
				continue
			}
			
			// Apply OS filter if provided
			if osFilter != "" && host.OS != osFilter {
				continue
			}
			
			hostMap := map[string]interface{}{
				"id":         host.ID,
				"project_id": host.ProjectID,
				"ip":         host.IP,
			}
			
			// Add optional fields if present
			if host.Hostname != "" {
				hostMap["hostname"] = host.Hostname
			}
			
			if host.OS != "" {
				hostMap["os"] = host.OS
			}
			
			if len(host.Services) > 0 {
				hostMap["services"] = host.Services
			}
			
			if host.Status != "" {
				hostMap["status"] = host.Status
			}
			
			hostList = append(hostList, hostMap)
		}
		
		// Build response
		response := map[string]interface{}{
			"hosts":       hostList,
			"total_count": len(hostList),
			"project_id":  projectID,
		}
		
		// Add filter information if filters were applied
		if statusFilter != "" || osFilter != "" {
			filters := make(map[string]interface{})
			if statusFilter != "" {
				filters["status"] = statusFilter
			}
			if osFilter != "" {
				filters["os"] = osFilter
			}
			response["filters"] = filters
		}
		
		return response, nil
	}
}