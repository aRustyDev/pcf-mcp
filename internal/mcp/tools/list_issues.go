package tools

import (
	"context"
	"fmt"

	"github.com/analyst/pcf-mcp/internal/mcp"
	"github.com/analyst/pcf-mcp/internal/pcf"
)

// ListIssuesClient defines the interface for listing issues
type ListIssuesClient interface {
	ListIssues(ctx context.Context, projectID string) ([]pcf.Issue, error)
}

// NewListIssuesTool creates an MCP tool for listing issues in a PCF project
func NewListIssuesTool(client ListIssuesClient) mcp.Tool {
	return mcp.Tool{
		Name:        "list_issues",
		Description: "List all security issues/findings in a specific PCF project",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the project to list issues from",
				},
				"severity": map[string]interface{}{
					"type":        "string",
					"description": "Filter issues by severity level",
					"enum":        []string{"Critical", "High", "Medium", "Low", "Info"},
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter issues by status",
					"enum":        []string{"Open", "In Progress", "Resolved", "Closed"},
				},
				"host_id": map[string]interface{}{
					"type":        "string",
					"description": "Filter issues by host ID",
				},
			},
			"required":             []string{"project_id"},
			"additionalProperties": false,
		},
		Handler: createListIssuesHandler(client),
	}
}

// createListIssuesHandler creates the handler function for listing issues
func createListIssuesHandler(client ListIssuesClient) mcp.ToolHandler {
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
		severityFilter := ""
		if severity, ok := params["severity"].(string); ok {
			severityFilter = severity
		}
		
		statusFilter := ""
		if status, ok := params["status"].(string); ok {
			statusFilter = status
		}
		
		hostIDFilter := ""
		if hostID, ok := params["host_id"].(string); ok {
			hostIDFilter = hostID
		}
		
		// Call PCF client to list issues
		issues, err := client.ListIssues(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to list issues: %w", err)
		}
		
		// Convert issues to response format and apply filters
		var issueList []map[string]interface{}
		severityCount := map[string]int{
			"Critical": 0,
			"High":     0,
			"Medium":   0,
			"Low":      0,
			"Info":     0,
		}
		
		for _, issue := range issues {
			// Count issues by severity (before filtering)
			if _, ok := severityCount[issue.Severity]; ok {
				severityCount[issue.Severity]++
			}
			
			// Apply severity filter if provided
			if severityFilter != "" && issue.Severity != severityFilter {
				continue
			}
			
			// Apply status filter if provided
			if statusFilter != "" && issue.Status != statusFilter {
				continue
			}
			
			// Apply host ID filter if provided
			if hostIDFilter != "" && issue.HostID != hostIDFilter {
				continue
			}
			
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
			
			issueList = append(issueList, issueMap)
		}
		
		// Build response
		response := map[string]interface{}{
			"issues":             issueList,
			"total_count":        len(issueList),
			"project_id":         projectID,
			"severity_breakdown": severityCount,
		}
		
		// Add filter information if filters were applied
		if severityFilter != "" || statusFilter != "" || hostIDFilter != "" {
			filters := make(map[string]interface{})
			if severityFilter != "" {
				filters["severity"] = severityFilter
			}
			if statusFilter != "" {
				filters["status"] = statusFilter
			}
			if hostIDFilter != "" {
				filters["host_id"] = hostIDFilter
			}
			response["filters"] = filters
		}
		
		return response, nil
	}
}