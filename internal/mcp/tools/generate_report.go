package tools

import (
	"context"
	"fmt"

	"github.com/aRustyDev/pcf-mcp/internal/mcp"
	"github.com/aRustyDev/pcf-mcp/internal/pcf"
)

// GenerateReportClient defines the interface for generating reports
type GenerateReportClient interface {
	GenerateReport(ctx context.Context, projectID string, req pcf.GenerateReportRequest) (*pcf.Report, error)
}

// NewGenerateReportTool creates an MCP tool for generating reports from a PCF project
func NewGenerateReportTool(client GenerateReportClient) mcp.Tool {
	return mcp.Tool{
		Name:        "generate_report",
		Description: "Generate a security assessment report for a PCF project",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the project to generate a report for",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "The output format for the report",
					"enum":        []string{"pdf", "html", "json", "markdown", "csv"},
				},
				"include_hosts": map[string]interface{}{
					"type":        "boolean",
					"description": "Include host information in the report",
					"default":     false,
				},
				"include_issues": map[string]interface{}{
					"type":        "boolean",
					"description": "Include security issues in the report",
					"default":     false,
				},
				"include_credentials": map[string]interface{}{
					"type":        "boolean",
					"description": "Include credential information in the report (redacted)",
					"default":     false,
				},
				"sections": map[string]interface{}{
					"type":        "array",
					"description": "Custom sections to include in the report",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required":             []string{"project_id", "format"},
			"additionalProperties": false,
		},
		Handler: createGenerateReportHandler(client),
	}
}

// createGenerateReportHandler creates the handler function for generating reports
func createGenerateReportHandler(client GenerateReportClient) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Extract and validate project_id
		projectID, ok := params["project_id"].(string)
		if !ok {
			return nil, fmt.Errorf("project_id parameter must be a string")
		}

		if projectID == "" {
			return nil, fmt.Errorf("project_id cannot be empty")
		}

		// Extract and validate format
		format, ok := params["format"].(string)
		if !ok {
			return nil, fmt.Errorf("format parameter must be a string")
		}

		// Validate format value
		validFormats := map[string]bool{
			"pdf":      true,
			"html":     true,
			"json":     true,
			"markdown": true,
			"csv":      true,
		}

		if !validFormats[format] {
			return nil, fmt.Errorf("invalid format: %s. Must be one of: pdf, html, json, markdown, csv", format)
		}

		// Create request
		req := pcf.GenerateReportRequest{
			Format: format,
		}

		// Extract optional boolean flags
		if includeHosts, ok := params["include_hosts"].(bool); ok {
			req.IncludeHosts = includeHosts
		}

		if includeIssues, ok := params["include_issues"].(bool); ok {
			req.IncludeIssues = includeIssues
		}

		if includeCredentials, ok := params["include_credentials"].(bool); ok {
			req.IncludeCredentials = includeCredentials
		}

		// Extract optional sections
		if sectionsRaw, ok := params["sections"]; ok {
			// Handle different types that might come from JSON
			switch sections := sectionsRaw.(type) {
			case []string:
				req.Sections = sections
			case []interface{}:
				// Convert []interface{} to []string
				sectionList := make([]string, 0, len(sections))
				for _, section := range sections {
					if sectionStr, ok := section.(string); ok {
						sectionList = append(sectionList, sectionStr)
					} else {
						return nil, fmt.Errorf("sections must be strings")
					}
				}
				req.Sections = sectionList
			default:
				return nil, fmt.Errorf("sections parameter must be an array of strings")
			}
		}

		// Call PCF client to generate report
		report, err := client.GenerateReport(ctx, projectID, req)
		if err != nil {
			return nil, fmt.Errorf("failed to generate report: %w", err)
		}

		// Build response
		reportMap := map[string]interface{}{
			"id":         report.ID,
			"project_id": report.ProjectID,
			"format":     report.Format,
			"status":     report.Status,
			"created_at": report.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		// Add optional fields if present
		if report.URL != "" {
			reportMap["url"] = report.URL
		}

		if report.Size > 0 {
			reportMap["size"] = report.Size
			reportMap["size_human"] = formatBytes(report.Size)
		}

		// Create appropriate message based on status
		var message string
		switch report.Status {
		case "completed":
			message = fmt.Sprintf("Report generated successfully in %s format", report.Format)
			if report.URL != "" {
				message += fmt.Sprintf(". Download from: %s", report.URL)
			}
		case "in_progress":
			message = fmt.Sprintf("Report generation in progress. Check back later with report ID: %s", report.ID)
		case "failed":
			message = fmt.Sprintf("Report generation failed. Please try again or contact support with report ID: %s", report.ID)
		default:
			message = fmt.Sprintf("Report %s created with status: %s", report.ID, report.Status)
		}

		response := map[string]interface{}{
			"report":  reportMap,
			"message": message,
		}

		return response, nil
	}
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
