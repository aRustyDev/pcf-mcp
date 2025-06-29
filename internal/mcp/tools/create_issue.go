package tools

import (
	"context"
	"fmt"

	"github.com/aRustyDev/pcf-mcp/internal/mcp"
	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// CreateIssueClient defines the interface for creating issues
type CreateIssueClient interface {
	CreateIssue(ctx context.Context, projectID string, req pcf.CreateIssueRequest) (*pcf.Issue, error)
}

// NewCreateIssueTool creates an MCP tool for creating security issues in a PCF project
func NewCreateIssueTool(client CreateIssueClient) mcp.Tool {
	return mcp.Tool{
		Name:        "create_issue",
		Description: "Create a new security issue/finding in a PCF project",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the project to create the issue in",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "The title of the security issue",
					"minLength":   1,
					"maxLength":   200,
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Detailed description of the security issue",
					"minLength":   1,
				},
				"severity": map[string]interface{}{
					"type":        "string",
					"description": "Severity level of the issue",
					"enum":        []string{"Critical", "High", "Medium", "Low", "Info"},
				},
				"host_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the affected host (optional)",
				},
				"cve": map[string]interface{}{
					"type":        "string",
					"description": "CVE identifier if applicable (optional)",
					"pattern":     "^CVE-\\d{4}-\\d{4,}$",
				},
				"cvss": map[string]interface{}{
					"type":        "number",
					"description": "CVSS score (0-10) if applicable (optional)",
					"minimum":     0,
					"maximum":     10,
				},
			},
			"required":             []string{"project_id", "title", "description", "severity"},
			"additionalProperties": false,
		},
		Handler: createCreateIssueHandler(client),
	}
}

// createCreateIssueHandler creates the handler function for creating issues
func createCreateIssueHandler(client CreateIssueClient) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Extract and validate project_id
		projectID, ok := params["project_id"].(string)
		if !ok {
			return nil, fmt.Errorf("project_id parameter must be a string")
		}

		if projectID == "" {
			return nil, fmt.Errorf("project_id cannot be empty")
		}

		// Extract and validate title
		title, ok := params["title"].(string)
		if !ok {
			return nil, fmt.Errorf("title parameter must be a string")
		}

		if title == "" {
			return nil, fmt.Errorf("title cannot be empty")
		}

		// Extract and validate description
		description, ok := params["description"].(string)
		if !ok {
			return nil, fmt.Errorf("description parameter must be a string")
		}

		if description == "" {
			return nil, fmt.Errorf("description cannot be empty")
		}

		// Extract and validate severity
		severity, ok := params["severity"].(string)
		if !ok {
			return nil, fmt.Errorf("severity parameter must be a string")
		}

		// Validate severity value
		validSeverities := map[string]bool{
			"Critical": true,
			"High":     true,
			"Medium":   true,
			"Low":      true,
			"Info":     true,
		}

		if !validSeverities[severity] {
			return nil, fmt.Errorf("invalid severity value: %s. Must be one of: Critical, High, Medium, Low, Info", severity)
		}

		// Create request
		req := pcf.CreateIssueRequest{
			Title:       title,
			Description: description,
			Severity:    severity,
		}

		// Extract optional host_id
		if hostID, ok := params["host_id"].(string); ok && hostID != "" {
			req.HostID = hostID
		}

		// Extract optional CVE
		if cve, ok := params["cve"].(string); ok && cve != "" {
			req.CVE = cve
		}

		// Extract optional CVSS score
		if cvssRaw, ok := params["cvss"]; ok {
			// Handle both float64 and int types
			var cvss float64
			switch v := cvssRaw.(type) {
			case float64:
				cvss = v
			case int:
				cvss = float64(v)
			default:
				return nil, fmt.Errorf("cvss parameter must be a number")
			}

			// Validate CVSS range
			if cvss < 0 || cvss > 10 {
				return nil, fmt.Errorf("cvss score must be between 0 and 10, got %f", cvss)
			}

			req.CVSS = cvss
		}

		// Call PCF client to create issue
		issue, err := client.CreateIssue(ctx, projectID, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create issue: %w", err)
		}

		// Build response
		issueMap := map[string]interface{}{
			"id":          issue.ID,
			"project_id":  issue.ProjectID,
			"title":       issue.Title,
			"description": issue.Description,
			"severity":    issue.Severity,
			"status":      issue.Status,
		}

		// Add optional fields if present
		if issue.HostID != "" {
			issueMap["host_id"] = issue.HostID
		}

		if issue.CVE != "" {
			issueMap["cve"] = issue.CVE
		}

		if issue.CVSS > 0 {
			issueMap["cvss"] = issue.CVSS
		}

		response := map[string]interface{}{
			"issue":   issueMap,
			"message": fmt.Sprintf("Issue '%s' created successfully in project %s", issue.Title, projectID),
		}

		return response, nil
	}
}
