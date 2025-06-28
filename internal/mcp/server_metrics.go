package mcp

import (
	"context"
	"time"
)

// MetricsRecorder interface defines the metrics recording methods we need
type MetricsRecorder interface {
	RecordToolExecution(toolName string, success bool, duration time.Duration)
}

// SetMetrics sets the metrics instance for the server
func (s *Server) SetMetrics(metrics MetricsRecorder) {
	s.metrics = metrics
}

// ExecuteToolWithMetrics wraps ExecuteTool to record metrics
func (s *Server) ExecuteToolWithMetrics(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	start := time.Now()
	
	// Execute the tool
	result, err := s.ExecuteTool(ctx, name, params)
	
	// Record metrics
	if s.metrics != nil {
		duration := time.Since(start)
		success := err == nil
		if recorder, ok := s.metrics.(MetricsRecorder); ok {
			recorder.RecordToolExecution(name, success, duration)
		}
	}
	
	return result, err
}