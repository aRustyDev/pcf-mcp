// Package mcp provides the Model Context Protocol server implementation
// for PCF integration. It handles MCP protocol communication, tool registration,
// and execution with support for both stdio and HTTP transports.
package mcp

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/aRustyDev/pcf-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server represents the MCP server instance
type Server struct {
	// config holds the server configuration
	config config.ServerConfig

	// mcpServer is the underlying MCP server implementation
	mcpServer *server.MCPServer

	// tools stores registered MCP tools
	tools map[string]Tool

	// toolsMutex protects concurrent access to tools map
	toolsMutex sync.RWMutex

	// metrics for observability
	metrics interface{} // Will be *observability.Metrics but avoiding import cycle

	// logger for server operations
	// Will be added when we integrate logging
}

// Tool represents an MCP tool definition
type Tool struct {
	// Name is the unique identifier for the tool
	Name string

	// Description explains what the tool does
	Description string

	// InputSchema defines the expected parameters using JSON Schema
	InputSchema map[string]interface{}

	// Handler is the function that executes the tool logic
	Handler ToolHandler
}

// ToolHandler is the function signature for tool execution
type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// Capabilities represents the server's MCP capabilities
type Capabilities struct {
	// Tools indicates if the server supports tool execution
	Tools bool

	// Resources indicates if the server supports resource access
	Resources bool

	// Prompts indicates if the server supports prompt templates
	Prompts bool
}

// toolNameRegex validates tool names (alphanumeric, underscore, hyphen)
var toolNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Version of the MCP server
const Version = "0.1.0"

// NewServer creates a new MCP server instance with the given configuration
func NewServer(cfg config.ServerConfig) (*Server, error) {
	// Validate transport type
	if cfg.Transport != "stdio" && cfg.Transport != "http" {
		return nil, fmt.Errorf("invalid transport type: %s (must be 'stdio' or 'http')", cfg.Transport)
	}

	// Create MCP server
	mcpServer := server.NewMCPServer("pcf-mcp", Version)

	s := &Server{
		config:    cfg,
		tools:     make(map[string]Tool),
		mcpServer: mcpServer,
	}

	return s, nil
}

// Name returns the server name
func (s *Server) Name() string {
	return "pcf-mcp"
}

// Version returns the server version
func (s *Server) Version() string {
	return Version
}

// Capabilities returns the server's MCP capabilities
func (s *Server) Capabilities() Capabilities {
	return Capabilities{
		Tools:     true,
		Resources: true,
		Prompts:   true,
	}
}

// RegisterTool registers a new tool with the server
func (s *Server) RegisterTool(tool Tool) error {
	// Validate tool
	if err := s.validateTool(tool); err != nil {
		return fmt.Errorf("tool validation failed: %w", err)
	}

	s.toolsMutex.Lock()
	defer s.toolsMutex.Unlock()

	// Check for duplicate
	if _, exists := s.tools[tool.Name]; exists {
		return fmt.Errorf("tool '%s' is already registered", tool.Name)
	}

	// Register the tool internally
	s.tools[tool.Name] = tool

	// Create MCP tool definition
	mcpTool := mcp.Tool{
		Name:        tool.Name,
		Description: tool.Description,
	}

	// Add input schema if provided
	if tool.InputSchema != nil {
		// Convert our InputSchema to MCP's ToolInputSchema
		mcpTool.InputSchema = mcp.ToolInputSchema{
			Type:       "object",
			Properties: tool.InputSchema,
		}
	}

	// Add tool to MCP server with handler
	s.mcpServer.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Use ExecuteToolWithMetrics to track metrics
		result, err := s.ExecuteToolWithMetrics(ctx, tool.Name, request.Params.Arguments.(map[string]interface{}))
		if err != nil {
			return nil, err
		}

		// Convert result to CallToolResult
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("%v", result),
				},
			},
		}, nil
	})

	return nil
}

// validateTool checks if a tool definition is valid
func (s *Server) validateTool(tool Tool) error {
	// Check name
	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}

	// Validate name format
	if !toolNameRegex.MatchString(tool.Name) {
		return fmt.Errorf("tool name must contain only alphanumeric characters, underscores, and hyphens")
	}

	// Check handler
	if tool.Handler == nil {
		return fmt.Errorf("tool handler is required")
	}

	return nil
}

// ListTools returns all registered tools
func (s *Server) ListTools() []Tool {
	s.toolsMutex.RLock()
	defer s.toolsMutex.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}

	return tools
}

// ExecuteTool executes a tool by name with the given parameters
func (s *Server) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	s.toolsMutex.RLock()
	tool, exists := s.tools[name]
	s.toolsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	// Execute the tool handler
	return tool.Handler(ctx, params)
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	switch s.config.Transport {
	case "stdio":
		// Start stdio server
		return server.ServeStdio(s.mcpServer)
	case "http":
		// Start HTTP server
		return s.StartHTTP(ctx)
	default:
		return fmt.Errorf("unsupported transport: %s", s.config.Transport)
	}
}
