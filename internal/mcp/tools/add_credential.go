package tools

import (
	"context"
	"fmt"

	"github.com/aRustyDev/pcf-mcp/internal/mcp"
	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// AddCredentialClient defines the interface for adding credentials
type AddCredentialClient interface {
	AddCredential(ctx context.Context, projectID string, req pcf.AddCredentialRequest) (*pcf.Credential, error)
}

// NewAddCredentialTool creates an MCP tool for adding credentials to a PCF project
func NewAddCredentialTool(client AddCredentialClient) mcp.Tool {
	return mcp.Tool{
		Name:        "add_credential",
		Description: "Add a new credential to a PCF project",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the project to add the credential to",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "The type of credential",
					"enum":        []string{"password", "hash", "key", "token", "certificate"},
				},
				"username": map[string]interface{}{
					"type":        "string",
					"description": "The username associated with the credential",
					"minLength":   1,
				},
				"value": map[string]interface{}{
					"type":        "string",
					"description": "The credential value (will be encrypted)",
					"minLength":   1,
				},
				"host_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the associated host (optional)",
				},
				"service": map[string]interface{}{
					"type":        "string",
					"description": "The service this credential is for (optional)",
				},
				"notes": map[string]interface{}{
					"type":        "string",
					"description": "Additional notes about the credential (optional)",
				},
			},
			"required":             []string{"project_id", "type", "username", "value"},
			"additionalProperties": false,
		},
		Handler: createAddCredentialHandler(client),
	}
}

// createAddCredentialHandler creates the handler function for adding credentials
func createAddCredentialHandler(client AddCredentialClient) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Extract and validate project_id
		projectID, ok := params["project_id"].(string)
		if !ok {
			return nil, fmt.Errorf("project_id parameter must be a string")
		}

		if projectID == "" {
			return nil, fmt.Errorf("project_id cannot be empty")
		}

		// Extract and validate type
		credType, ok := params["type"].(string)
		if !ok {
			return nil, fmt.Errorf("type parameter must be a string")
		}

		// Validate credential type
		validTypes := map[string]bool{
			"password":    true,
			"hash":        true,
			"key":         true,
			"token":       true,
			"certificate": true,
		}

		if !validTypes[credType] {
			return nil, fmt.Errorf("invalid credential type: %s. Must be one of: password, hash, key, token, certificate", credType)
		}

		// Extract and validate username
		username, ok := params["username"].(string)
		if !ok {
			return nil, fmt.Errorf("username parameter must be a string")
		}

		if username == "" {
			return nil, fmt.Errorf("username cannot be empty")
		}

		// Extract and validate value
		value, ok := params["value"].(string)
		if !ok {
			return nil, fmt.Errorf("value parameter must be a string")
		}

		if value == "" {
			return nil, fmt.Errorf("credential value cannot be empty")
		}

		// Create request
		req := pcf.AddCredentialRequest{
			Type:     credType,
			Username: username,
			Value:    value,
		}

		// Extract optional host_id
		if hostID, ok := params["host_id"].(string); ok && hostID != "" {
			req.HostID = hostID
		}

		// Extract optional service
		if service, ok := params["service"].(string); ok && service != "" {
			req.Service = service
		}

		// Extract optional notes
		if notes, ok := params["notes"].(string); ok && notes != "" {
			req.Notes = notes
		}

		// Call PCF client to add credential
		credential, err := client.AddCredential(ctx, projectID, req)
		if err != nil {
			return nil, fmt.Errorf("failed to add credential: %w", err)
		}

		// Build response - always redact the credential value
		credMap := map[string]interface{}{
			"id":         credential.ID,
			"project_id": credential.ProjectID,
			"type":       credential.Type,
			"username":   credential.Username,
			"value":      "***REDACTED***", // Never expose the actual value
		}

		// Add optional fields if present
		if credential.HostID != "" {
			credMap["host_id"] = credential.HostID
		}

		if credential.Service != "" {
			credMap["service"] = credential.Service
		}

		if credential.Notes != "" {
			credMap["notes"] = credential.Notes
		}

		response := map[string]interface{}{
			"credential": credMap,
			"message":    fmt.Sprintf("Credential for user '%s' added successfully to project %s", credential.Username, projectID),
		}

		return response, nil
	}
}
