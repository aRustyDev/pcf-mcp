package tools

import (
	"context"
	"fmt"
	"net"

	"github.com/aRustyDev/pcf-mcp/internal/mcp"
	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// AddHostClient defines the interface for adding hosts
type AddHostClient interface {
	AddHost(ctx context.Context, projectID string, req pcf.CreateHostRequest) (*pcf.Host, error)
}

// NewAddHostTool creates an MCP tool for adding hosts to a PCF project
func NewAddHostTool(client AddHostClient) mcp.Tool {
	return mcp.Tool{
		Name:        "add_host",
		Description: "Add a new host to a PCF project",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the project to add the host to",
				},
				"ip": map[string]interface{}{
					"type":        "string",
					"description": "The IP address of the host",
				},
				"hostname": map[string]interface{}{
					"type":        "string",
					"description": "The hostname (optional)",
				},
				"os": map[string]interface{}{
					"type":        "string",
					"description": "The operating system of the host (optional)",
				},
				"services": map[string]interface{}{
					"type":        "array",
					"description": "List of services running on the host (optional)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required":             []string{"project_id", "ip"},
			"additionalProperties": false,
		},
		Handler: createAddHostHandler(client),
	}
}

// createAddHostHandler creates the handler function for adding hosts
func createAddHostHandler(client AddHostClient) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Extract and validate project_id
		projectID, ok := params["project_id"].(string)
		if !ok {
			return nil, fmt.Errorf("project_id parameter must be a string")
		}

		if projectID == "" {
			return nil, fmt.Errorf("project_id cannot be empty")
		}

		// Extract and validate IP address
		ip, ok := params["ip"].(string)
		if !ok {
			return nil, fmt.Errorf("ip parameter must be a string")
		}

		if ip == "" {
			return nil, fmt.Errorf("ip address cannot be empty")
		}

		// Validate IP address format
		if net.ParseIP(ip) == nil {
			return nil, fmt.Errorf("invalid IP address format: %s", ip)
		}

		// Create request
		req := pcf.CreateHostRequest{
			IP: ip,
		}

		// Extract optional hostname
		if hostname, ok := params["hostname"].(string); ok && hostname != "" {
			req.Hostname = hostname
		}

		// Extract optional OS
		if os, ok := params["os"].(string); ok && os != "" {
			req.OS = os
		}

		// Extract optional notes
		// Note: CreateHostRequest doesn't have a Notes field, so we'll ignore it

		// Extract optional services
		if servicesRaw, ok := params["services"]; ok {
			// Handle different types that might come from JSON
			switch services := servicesRaw.(type) {
			case []string:
				req.Services = services
			case []interface{}:
				// Convert []interface{} to []string
				serviceList := make([]string, 0, len(services))
				for _, service := range services {
					if serviceStr, ok := service.(string); ok {
						serviceList = append(serviceList, serviceStr)
					} else {
						return nil, fmt.Errorf("services must be strings")
					}
				}
				req.Services = serviceList
			default:
				return nil, fmt.Errorf("services parameter must be an array of strings")
			}
		}

		// Call PCF client to add host
		host, err := client.AddHost(ctx, projectID, req)
		if err != nil {
			return nil, fmt.Errorf("failed to add host: %w", err)
		}

		// Build response
		hostMap := map[string]interface{}{
			"id":         host.ID,
			"project_id": host.ProjectID,
			"ip":         host.IP,
			"status":     host.Status,
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

		response := map[string]interface{}{
			"host":    hostMap,
			"message": fmt.Sprintf("Host %s added successfully to project %s", host.IP, projectID),
		}

		return response, nil
	}
}
