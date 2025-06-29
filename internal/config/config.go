// Package config provides multi-source configuration management
// using Viper. Configuration precedence: CLI > ENV > File > Defaults
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	PCF     PCFConfig     `mapstructure:"pcf"`
	Logging LoggingConfig `mapstructure:"logging"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Tracing TracingConfig `mapstructure:"tracing"`
}

// ServerConfig contains MCP server configuration
type ServerConfig struct {
	// Host is the server bind address
	Host string `mapstructure:"host"`
	// Port is the server listen port
	Port int `mapstructure:"port"`
	// Transport specifies the MCP transport type (stdio or http)
	Transport string `mapstructure:"transport"`
	// ReadTimeout is the maximum duration for reading the entire request
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
	// WriteTimeout is the maximum duration before timing out writes of the response
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	// MaxConcurrentTools limits concurrent tool executions
	MaxConcurrentTools int `mapstructure:"max_concurrent_tools"`
	// ToolTimeout is the maximum duration for tool execution
	ToolTimeout time.Duration `mapstructure:"tool_timeout"`
	// AuthRequired enables authentication for HTTP transport
	AuthRequired bool `mapstructure:"auth_required"`
	// AuthToken is the bearer token for authentication
	AuthToken string `mapstructure:"auth_token"`
}

// PCFConfig contains Pentest Collaboration Framework client configuration
type PCFConfig struct {
	// URL is the base URL of the PCF instance
	URL string `mapstructure:"url"`
	// APIKey is the authentication key for PCF API
	APIKey string `mapstructure:"api_key"`
	// Timeout is the HTTP client timeout for PCF requests
	Timeout time.Duration `mapstructure:"timeout"`
	// MaxRetries is the maximum number of retry attempts for failed requests
	MaxRetries int `mapstructure:"max_retries"`
	// InsecureSkipVerify skips TLS certificate verification (not recommended for production)
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	// Level sets the minimum log level (debug, info, warn, error)
	Level string `mapstructure:"level"`
	// Format specifies the log output format (json or text)
	Format string `mapstructure:"format"`
	// AddSource includes source code location in logs
	AddSource bool `mapstructure:"add_source"`
}

// MetricsConfig contains Prometheus metrics configuration
type MetricsConfig struct {
	// Enabled determines if metrics collection is active
	Enabled bool `mapstructure:"enabled"`
	// Port is the metrics endpoint listen port
	Port int `mapstructure:"port"`
	// Path is the metrics endpoint path
	Path string `mapstructure:"path"`
}

// TracingConfig contains OpenTelemetry tracing configuration
type TracingConfig struct {
	// Enabled determines if distributed tracing is active
	Enabled bool `mapstructure:"enabled"`
	// Exporter specifies the trace exporter type (jaeger, zipkin, otlp)
	Exporter string `mapstructure:"exporter"`
	// Endpoint is the trace collector endpoint
	Endpoint string `mapstructure:"endpoint"`
	// SamplingRate is the trace sampling rate (0.0 to 1.0)
	SamplingRate float64 `mapstructure:"sampling_rate"`
	// ServiceName overrides the default service name in traces
	ServiceName string `mapstructure:"service_name"`
}

// viperInstance holds the global viper instance
var viperInstance *viper.Viper

// init initializes the viper instance with default values
func init() {
	viperInstance = viper.New()
	setDefaults()
}

// setDefaults configures all default values
func setDefaults() {
	// Server defaults
	viperInstance.SetDefault("server.host", "0.0.0.0")
	viperInstance.SetDefault("server.port", 8080)
	viperInstance.SetDefault("server.transport", "stdio")
	viperInstance.SetDefault("server.read_timeout", 30*time.Second)
	viperInstance.SetDefault("server.write_timeout", 30*time.Second)
	viperInstance.SetDefault("server.max_concurrent_tools", 10)
	viperInstance.SetDefault("server.tool_timeout", 60*time.Second)
	viperInstance.SetDefault("server.auth_required", false)
	viperInstance.SetDefault("server.auth_token", "")

	// PCF defaults
	viperInstance.SetDefault("pcf.url", "http://localhost:5000")
	viperInstance.SetDefault("pcf.api_key", "")
	viperInstance.SetDefault("pcf.timeout", 30*time.Second)
	viperInstance.SetDefault("pcf.max_retries", 3)
	viperInstance.SetDefault("pcf.insecure_skip_verify", false)

	// Logging defaults
	viperInstance.SetDefault("logging.level", "info")
	viperInstance.SetDefault("logging.format", "json")
	viperInstance.SetDefault("logging.add_source", false)

	// Metrics defaults
	viperInstance.SetDefault("metrics.enabled", true)
	viperInstance.SetDefault("metrics.port", 9090)
	viperInstance.SetDefault("metrics.path", "/metrics")

	// Tracing defaults
	viperInstance.SetDefault("tracing.enabled", false)
	viperInstance.SetDefault("tracing.exporter", "otlp")
	viperInstance.SetDefault("tracing.endpoint", "http://localhost:4317")
	viperInstance.SetDefault("tracing.sampling_rate", 1.0)
	viperInstance.SetDefault("tracing.service_name", "pcf-mcp")
}

// New creates a new configuration instance with default values
func New() *Config {
	cfg := &Config{}
	// Load defaults into config struct
	if err := viperInstance.Unmarshal(cfg); err != nil {
		// This should never fail with defaults, but handle it gracefully
		return &Config{
			Server: ServerConfig{
				Host:      "0.0.0.0",
				Port:      8080,
				Transport: "stdio",
			},
		}
	}
	return cfg
}

// LoadFromFile loads configuration from a file
func (c *Config) LoadFromFile(path string) error {
	viperInstance.SetConfigFile(path)

	if err := viperInstance.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := viperInstance.Unmarshal(c); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// LoadFromEnvironment loads configuration from environment variables
// Environment variables should be prefixed with PCF_MCP_ and use underscores
// Example: PCF_MCP_SERVER_HOST maps to server.host
func (c *Config) LoadFromEnvironment() error {
	viperInstance.SetEnvPrefix("PCF_MCP")
	viperInstance.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viperInstance.AutomaticEnv()

	if err := viperInstance.Unmarshal(c); err != nil {
		return fmt.Errorf("failed to unmarshal config from environment: %w", err)
	}

	return nil
}

// LoadFromCLI loads configuration from command-line arguments
func (c *Config) LoadFromCLI(args []string) error {
	cmd := &cobra.Command{
		Use:           "pcf-mcp",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// Define CLI flags
	flags := cmd.PersistentFlags()

	// Server flags
	flags.String("server-host", "", "Server bind address")
	flags.Int("server-port", 0, "Server listen port")
	flags.String("server-transport", "", "MCP transport type (stdio or http)")
	flags.Bool("server-auth-required", false, "Enable authentication for HTTP transport")
	flags.String("server-auth-token", "", "Bearer token for authentication")

	// PCF flags
	flags.String("pcf-url", "", "PCF base URL")
	flags.String("pcf-api-key", "", "PCF API key")

	// Logging flags
	flags.String("log-level", "", "Log level (debug, info, warn, error)")
	flags.String("log-format", "", "Log format (json or text)")

	// Bind flags to viper
	_ = viperInstance.BindPFlag("server.host", flags.Lookup("server-host"))
	_ = viperInstance.BindPFlag("server.port", flags.Lookup("server-port"))
	_ = viperInstance.BindPFlag("server.transport", flags.Lookup("server-transport"))
	_ = viperInstance.BindPFlag("server.auth_required", flags.Lookup("server-auth-required"))
	_ = viperInstance.BindPFlag("server.auth_token", flags.Lookup("server-auth-token"))
	_ = viperInstance.BindPFlag("pcf.url", flags.Lookup("pcf-url"))
	_ = viperInstance.BindPFlag("pcf.api_key", flags.Lookup("pcf-api-key"))
	_ = viperInstance.BindPFlag("logging.level", flags.Lookup("log-level"))
	_ = viperInstance.BindPFlag("logging.format", flags.Lookup("log-format"))

	// Parse arguments
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		return fmt.Errorf("failed to parse CLI arguments: %w", err)
	}

	// Unmarshal updated config
	if err := viperInstance.Unmarshal(c); err != nil {
		return fmt.Errorf("failed to unmarshal config from CLI: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate transport type
	if c.Server.Transport != "stdio" && c.Server.Transport != "http" {
		return fmt.Errorf("invalid transport type: %s (must be 'stdio' or 'http')", c.Server.Transport)
	}

	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	// Validate log format
	if c.Logging.Format != "json" && c.Logging.Format != "text" {
		return fmt.Errorf("invalid log format: %s (must be 'json' or 'text')", c.Logging.Format)
	}

	// Validate PCF configuration
	if c.PCF.URL == "" {
		return fmt.Errorf("PCF URL is required")
	}

	// Validate port numbers
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Metrics.Enabled && (c.Metrics.Port < 1 || c.Metrics.Port > 65535) {
		return fmt.Errorf("invalid metrics port: %d", c.Metrics.Port)
	}

	// Validate tracing configuration
	if c.Tracing.Enabled {
		validExporters := map[string]bool{
			"jaeger": true,
			"zipkin": true,
			"otlp":   true,
		}
		if !validExporters[c.Tracing.Exporter] {
			return fmt.Errorf("invalid tracing exporter: %s", c.Tracing.Exporter)
		}

		if c.Tracing.SamplingRate < 0.0 || c.Tracing.SamplingRate > 1.0 {
			return fmt.Errorf("invalid sampling rate: %f (must be between 0.0 and 1.0)", c.Tracing.SamplingRate)
		}
	}

	return nil
}

// String returns a string representation of the configuration (with sensitive data masked)
func (c *Config) String() string {
	maskedAPIKey := "***"
	if c.PCF.APIKey != "" {
		if len(c.PCF.APIKey) > 4 {
			maskedAPIKey = c.PCF.APIKey[:2] + "***" + c.PCF.APIKey[len(c.PCF.APIKey)-2:]
		}
	}

	return fmt.Sprintf(
		"Config{Server:%+v, PCF:{URL:%s, APIKey:%s, Timeout:%s}, Logging:%+v, Metrics:%+v, Tracing:%+v}",
		c.Server, c.PCF.URL, maskedAPIKey, c.PCF.Timeout, c.Logging, c.Metrics, c.Tracing,
	)
}
