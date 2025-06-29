package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/aRustyDev/pcf-mcp/internal/config"
)

// TestNewServer tests the creation of a new MCP server
func TestNewServer(t *testing.T) {
	cfg := config.ServerConfig{
		Host:      "localhost",
		Port:      8080,
		Transport: "stdio",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	// Verify server properties
	if server.Name() != "pcf-mcp" {
		t.Errorf("Expected server name 'pcf-mcp', got '%s'", server.Name())
	}

	if server.Version() == "" {
		t.Error("Server version should not be empty")
	}
}

// TestServerCapabilities tests that the server declares correct capabilities
func TestServerCapabilities(t *testing.T) {
	cfg := config.ServerConfig{
		Transport: "stdio",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	capabilities := server.Capabilities()

	// Check that tools capability is enabled
	if !capabilities.Tools {
		t.Error("Server should have tools capability enabled")
	}

	// Check that resources capability is enabled
	if !capabilities.Resources {
		t.Error("Server should have resources capability enabled")
	}

	// Check that prompts capability is enabled
	if !capabilities.Prompts {
		t.Error("Server should have prompts capability enabled")
	}
}

// TestRegisterTool tests tool registration
func TestRegisterTool(t *testing.T) {
	cfg := config.ServerConfig{
		Transport: "stdio",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Define a test tool
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "Test parameter",
				},
			},
			"required": []string{"param1"},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{"result": "success"}, nil
		},
	}

	// Register the tool
	err = server.RegisterTool(tool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Verify tool is registered
	tools := server.ListTools()
	found := false
	for _, registeredTool := range tools {
		if registeredTool.Name == "test_tool" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Tool was not properly registered")
	}
}

// TestRegisterDuplicateTool tests that duplicate tool registration fails
func TestRegisterDuplicateTool(t *testing.T) {
	cfg := config.ServerConfig{
		Transport: "stdio",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tool := Tool{
		Name:        "duplicate_tool",
		Description: "A tool to test duplicate registration",
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	}

	// Register once
	err = server.RegisterTool(tool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Try to register again
	err = server.RegisterTool(tool)
	if err == nil {
		t.Error("Expected error when registering duplicate tool, got nil")
	}
}

// TestExecuteTool tests tool execution
func TestExecuteTool(t *testing.T) {
	cfg := config.ServerConfig{
		Transport: "stdio",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Register a test tool
	executed := false
	tool := Tool{
		Name:        "execute_test",
		Description: "Tool to test execution",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			executed = true
			message, _ := params["message"].(string)
			return map[string]interface{}{
				"echo": message,
			}, nil
		},
	}

	err = server.RegisterTool(tool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Execute the tool
	ctx := context.Background()
	params := map[string]interface{}{
		"message": "hello",
	}

	result, err := server.ExecuteTool(ctx, "execute_test", params)
	if err != nil {
		t.Fatalf("Failed to execute tool: %v", err)
	}

	if !executed {
		t.Error("Tool handler was not executed")
	}

	// Check result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if resultMap["echo"] != "hello" {
		t.Errorf("Expected echo 'hello', got '%v'", resultMap["echo"])
	}
}

// TestExecuteNonExistentTool tests executing a tool that doesn't exist
func TestExecuteNonExistentTool(t *testing.T) {
	cfg := config.ServerConfig{
		Transport: "stdio",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()
	_, err = server.ExecuteTool(ctx, "non_existent", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error when executing non-existent tool")
	}
}

// TestServerStart tests starting the server
func TestServerStart(t *testing.T) {
	// Test with stdio transport
	t.Run("stdio transport", func(t *testing.T) {
		cfg := config.ServerConfig{
			Transport: "stdio",
		}

		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		// Start server in background
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			errCh <- server.Start(ctx)
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Cancel context to stop server
		cancel()

		// Check if server stopped without error
		select {
		case err := <-errCh:
			if err != nil && err != context.Canceled {
				t.Errorf("Server stopped with error: %v", err)
			}
		case <-time.After(1 * time.Second):
			t.Error("Server did not stop in time")
		}
	})
}

// TestTransportValidation tests that invalid transports are rejected
func TestTransportValidation(t *testing.T) {
	cfg := config.ServerConfig{
		Transport: "invalid",
	}

	_, err := NewServer(cfg)
	if err == nil {
		t.Error("Expected error for invalid transport, got nil")
	}
}

// TestServerWithHTTPTransport tests creating server with HTTP transport
func TestServerWithHTTPTransport(t *testing.T) {
	cfg := config.ServerConfig{
		Host:      "localhost",
		Port:      8080,
		Transport: "http",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server with HTTP transport: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}
}

// TestToolValidation tests that tool validation works correctly
func TestToolValidation(t *testing.T) {
	cfg := config.ServerConfig{
		Transport: "stdio",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tests := []struct {
		name    string
		tool    Tool
		wantErr bool
	}{
		{
			name: "Valid tool",
			tool: Tool{
				Name:        "valid_tool",
				Description: "A valid tool",
				Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
					return nil, nil
				},
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			tool: Tool{
				Description: "Tool without name",
				Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
					return nil, nil
				},
			},
			wantErr: true,
		},
		{
			name: "Missing handler",
			tool: Tool{
				Name:        "no_handler",
				Description: "Tool without handler",
			},
			wantErr: true,
		},
		{
			name: "Invalid name format",
			tool: Tool{
				Name:        "invalid name with spaces",
				Description: "Tool with invalid name",
				Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
					return nil, nil
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := server.RegisterTool(tt.tool)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterTool() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
