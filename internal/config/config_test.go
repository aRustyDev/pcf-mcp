package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewConfig tests the creation of a new configuration instance
func TestNewConfig(t *testing.T) {
	// Test that NewConfig returns a valid config with defaults
	cfg := New()
	
	if cfg == nil {
		t.Fatal("NewConfig returned nil")
	}
	
	// Test default values
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got '%s'", cfg.Server.Host)
	}
	
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}
	
	if cfg.Server.Transport != "stdio" {
		t.Errorf("Expected default transport 'stdio', got '%s'", cfg.Server.Transport)
	}
}

// TestLoadFromFile tests loading configuration from various file formats
func TestLoadFromFile(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		content  string
		expected Config
	}{
		{
			name:   "YAML configuration",
			format: "yaml",
			content: `
server:
  host: "127.0.0.1"
  port: 9090
  transport: "http"
pcf:
  url: "http://pcf.example.com"
  api_key: "test-key"
  timeout: "60s"
logging:
  level: "debug"
  format: "json"
`,
			expected: Config{
				Server: ServerConfig{
					Host:      "127.0.0.1",
					Port:      9090,
					Transport: "http",
				},
				PCF: PCFConfig{
					URL:     "http://pcf.example.com",
					APIKey:  "test-key",
					Timeout: 60 * time.Second,
				},
				Logging: LoggingConfig{
					Level:  "debug",
					Format: "json",
				},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config."+tt.format)
			
			err := os.WriteFile(configFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}
			
			// Load configuration
			cfg := New()
			err = cfg.LoadFromFile(configFile)
			if err != nil {
				t.Fatalf("Failed to load config from file: %v", err)
			}
			
			// Verify loaded values
			if cfg.Server.Host != tt.expected.Server.Host {
				t.Errorf("Expected host '%s', got '%s'", tt.expected.Server.Host, cfg.Server.Host)
			}
			
			if cfg.Server.Port != tt.expected.Server.Port {
				t.Errorf("Expected port %d, got %d", tt.expected.Server.Port, cfg.Server.Port)
			}
			
			if cfg.PCF.URL != tt.expected.PCF.URL {
				t.Errorf("Expected PCF URL '%s', got '%s'", tt.expected.PCF.URL, cfg.PCF.URL)
			}
		})
	}
}

// TestLoadFromEnvironment tests loading configuration from environment variables
func TestLoadFromEnvironment(t *testing.T) {
	// Save current environment and restore after test
	oldEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, e := range oldEnv {
			pair := splitEnv(e)
			os.Setenv(pair[0], pair[1])
		}
	}()
	
	// Set test environment variables
	testEnvVars := map[string]string{
		"PCF_MCP_SERVER_HOST":       "192.168.1.1",
		"PCF_MCP_SERVER_PORT":       "8888",
		"PCF_MCP_SERVER_TRANSPORT":  "http",
		"PCF_MCP_PCF_URL":          "http://test-pcf.local",
		"PCF_MCP_PCF_API_KEY":      "env-test-key",
		"PCF_MCP_LOGGING_LEVEL":    "warn",
		"PCF_MCP_LOGGING_FORMAT":   "text",
		"PCF_MCP_METRICS_ENABLED":  "true",
		"PCF_MCP_METRICS_PORT":     "9999",
		"PCF_MCP_TRACING_ENABLED":  "true",
		"PCF_MCP_TRACING_EXPORTER": "jaeger",
	}
	
	for k, v := range testEnvVars {
		os.Setenv(k, v)
	}
	
	// Load configuration
	cfg := New()
	err := cfg.LoadFromEnvironment()
	if err != nil {
		t.Fatalf("Failed to load config from environment: %v", err)
	}
	
	// Verify loaded values
	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("Expected host '192.168.1.1', got '%s'", cfg.Server.Host)
	}
	
	if cfg.Server.Port != 8888 {
		t.Errorf("Expected port 8888, got %d", cfg.Server.Port)
	}
	
	if cfg.PCF.URL != "http://test-pcf.local" {
		t.Errorf("Expected PCF URL 'http://test-pcf.local', got '%s'", cfg.PCF.URL)
	}
	
	if !cfg.Metrics.Enabled {
		t.Error("Expected metrics to be enabled")
	}
	
	if cfg.Metrics.Port != 9999 {
		t.Errorf("Expected metrics port 9999, got %d", cfg.Metrics.Port)
	}
}

// TestLoadFromCLI tests loading configuration from command-line arguments
func TestLoadFromCLI(t *testing.T) {
	// Test CLI arguments
	args := []string{
		"--server-host", "10.0.0.1",
		"--server-port", "7777",
		"--pcf-url", "http://cli-pcf.test",
		"--pcf-api-key", "cli-key",
		"--log-level", "error",
	}
	
	cfg := New()
	err := cfg.LoadFromCLI(args)
	if err != nil {
		t.Fatalf("Failed to load config from CLI: %v", err)
	}
	
	// Verify loaded values
	if cfg.Server.Host != "10.0.0.1" {
		t.Errorf("Expected host '10.0.0.1', got '%s'", cfg.Server.Host)
	}
	
	if cfg.Server.Port != 7777 {
		t.Errorf("Expected port 7777, got %d", cfg.Server.Port)
	}
	
	if cfg.PCF.URL != "http://cli-pcf.test" {
		t.Errorf("Expected PCF URL 'http://cli-pcf.test', got '%s'", cfg.PCF.URL)
	}
	
	if cfg.Logging.Level != "error" {
		t.Errorf("Expected log level 'error', got '%s'", cfg.Logging.Level)
	}
}

// TestConfigPrecedence tests that configuration sources follow correct precedence
// CLI > Environment > File > Defaults
func TestConfigPrecedence(t *testing.T) {
	// Create config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	
	fileContent := `
server:
  host: "file-host"
  port: 1111
logging:
  level: "info"
`
	
	err := os.WriteFile(configFile, []byte(fileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}
	
	// Set environment variables
	os.Setenv("PCF_MCP_SERVER_HOST", "env-host")
	os.Setenv("PCF_MCP_SERVER_PORT", "2222")
	defer func() {
		os.Unsetenv("PCF_MCP_SERVER_HOST")
		os.Unsetenv("PCF_MCP_SERVER_PORT")
	}()
	
	// Load configuration in order
	cfg := New()
	
	// Load from file first
	err = cfg.LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("Failed to load config from file: %v", err)
	}
	
	// Then environment (should override file)
	err = cfg.LoadFromEnvironment()
	if err != nil {
		t.Fatalf("Failed to load config from environment: %v", err)
	}
	
	// Finally CLI (should override everything)
	args := []string{"--server-host", "cli-host"}
	err = cfg.LoadFromCLI(args)
	if err != nil {
		t.Fatalf("Failed to load config from CLI: %v", err)
	}
	
	// Verify precedence
	if cfg.Server.Host != "cli-host" {
		t.Errorf("Expected CLI to override with 'cli-host', got '%s'", cfg.Server.Host)
	}
	
	if cfg.Server.Port != 2222 {
		t.Errorf("Expected env to override file with port 2222, got %d", cfg.Server.Port)
	}
	
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected file value 'info' to remain, got '%s'", cfg.Logging.Level)
	}
}

// TestValidate tests configuration validation
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: Config{
				Server: ServerConfig{
					Host:      "0.0.0.0",
					Port:      8080,
					Transport: "stdio",
				},
				PCF: PCFConfig{
					URL:     "http://localhost:5000",
					APIKey:  "test-key",
					Timeout: 30 * time.Second,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid transport",
			config: Config{
				Server: ServerConfig{
					Transport: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid log level",
			config: Config{
				Server: ServerConfig{
					Transport: "stdio",
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "Missing PCF URL",
			config: Config{
				Server: ServerConfig{
					Transport: "stdio",
				},
				PCF: PCFConfig{
					URL: "",
				},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to split environment variable strings
func splitEnv(env string) []string {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env, ""}
}