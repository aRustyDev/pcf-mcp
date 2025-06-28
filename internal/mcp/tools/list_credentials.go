package tools

import (
	"context"
	"fmt"

	"github.com/analyst/pcf-mcp/internal/mcp"
	"github.com/analyst/pcf-mcp/internal/pcf"
)

// ListCredentialsClient defines the interface for listing credentials
type ListCredentialsClient interface {
	ListCredentials(ctx context.Context, projectID string) ([]pcf.Credential, error)
}

// NewListCredentialsTool creates an MCP tool for listing credentials in a PCF project
func NewListCredentialsTool(client ListCredentialsClient) mcp.Tool {
	return mcp.Tool{
		Name:        "list_credentials",
		Description: "List all stored credentials in a specific PCF project",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the project to list credentials from",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Filter credentials by type",
					"enum":        []string{"password", "hash", "key", "token", "certificate"},
				},
				"host_id": map[string]interface{}{
					"type":        "string",
					"description": "Filter credentials by host ID",
				},
				"service": map[string]interface{}{
					"type":        "string",
					"description": "Filter credentials by service",
				},
			},
			"required":             []string{"project_id"},
			"additionalProperties": false,
		},
		Handler: createListCredentialsHandler(client),
	}
}

// createListCredentialsHandler creates the handler function for listing credentials
func createListCredentialsHandler(client ListCredentialsClient) mcp.ToolHandler {
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
		typeFilter := ""
		if credType, ok := params["type"].(string); ok {
			typeFilter = credType
		}
		
		hostIDFilter := ""
		if hostID, ok := params["host_id"].(string); ok {
			hostIDFilter = hostID
		}
		
		serviceFilter := ""
		if service, ok := params["service"].(string); ok {
			serviceFilter = service
		}
		
		// Call PCF client to list credentials
		credentials, err := client.ListCredentials(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to list credentials: %w", err)
		}
		
		// Convert credentials to response format and apply filters
		credentialList := make([]map[string]interface{}, 0)
		typeCount := make(map[string]int)
		
		for _, cred := range credentials {
			// Count by type (before filtering)
			typeCount[cred.Type]++
			
			// Apply type filter if provided
			if typeFilter != "" && cred.Type != typeFilter {
				continue
			}
			
			// Apply host ID filter if provided
			if hostIDFilter != "" && cred.HostID != hostIDFilter {
				continue
			}
			
			// Apply service filter if provided
			if serviceFilter != "" && cred.Service != serviceFilter {
				continue
			}
			
			credMap := map[string]interface{}{
				"id":         cred.ID,
				"project_id": cred.ProjectID,
				"type":       cred.Type,
				"username":   cred.Username,
				"value":      "***REDACTED***", // Always redact credential values
			}
			
			// Add optional fields if present
			if cred.HostID != "" {
				credMap["host_id"] = cred.HostID
			}
			
			if cred.Service != "" {
				credMap["service"] = cred.Service
			}
			
			if cred.Notes != "" {
				credMap["notes"] = cred.Notes
			}
			
			credentialList = append(credentialList, credMap)
		}
		
		// Build response
		response := map[string]interface{}{
			"credentials":    credentialList,
			"total_count":    len(credentialList),
			"project_id":     projectID,
			"type_breakdown": typeCount,
		}
		
		// Add filter information if filters were applied
		if typeFilter != "" || hostIDFilter != "" || serviceFilter != "" {
			filters := make(map[string]interface{})
			if typeFilter != "" {
				filters["type"] = typeFilter
			}
			if hostIDFilter != "" {
				filters["host_id"] = hostIDFilter
			}
			if serviceFilter != "" {
				filters["service"] = serviceFilter
			}
			response["filters"] = filters
		}
		
		return response, nil
	}
}