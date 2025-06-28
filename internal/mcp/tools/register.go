package tools

import (
	"fmt"

	"github.com/analyst/pcf-mcp/internal/mcp"
)

// FullPCFClient defines the complete interface for all PCF operations
type FullPCFClient interface {
	PCFClient
	CreateProjectClient
	ListHostsClient
	AddHostClient
	ListIssuesClient
	CreateIssueClient
	ListCredentialsClient
	AddCredentialClient
	GenerateReportClient
}

// RegisterAllTools registers all available PCF tools with the MCP server
func RegisterAllTools(server *mcp.Server, pcfClient FullPCFClient) error {
	// List of all tools to register
	tools := []mcp.Tool{
		NewListProjectsTool(pcfClient),
		NewCreateProjectTool(pcfClient),
		NewListHostsTool(pcfClient),
		NewAddHostTool(pcfClient),
		NewListIssuesTool(pcfClient),
		NewCreateIssueTool(pcfClient),
		NewListCredentialsTool(pcfClient),
		NewAddCredentialTool(pcfClient),
		NewGenerateReportTool(pcfClient),
	}
	
	// Register each tool
	for _, tool := range tools {
		if err := server.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register tool '%s': %w", tool.Name, err)
		}
	}
	
	return nil
}