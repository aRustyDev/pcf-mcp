package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/analyst/pcf-mcp/internal/pcf"
)

// MockGenerateReportClient extends MockPCFClient with GenerateReport method
type MockGenerateReportClient struct {
	MockPCFClient
	GenerateReportFunc func(ctx context.Context, projectID string, req pcf.GenerateReportRequest) (*pcf.Report, error)
}

func (m *MockGenerateReportClient) GenerateReport(ctx context.Context, projectID string, req pcf.GenerateReportRequest) (*pcf.Report, error) {
	if m.GenerateReportFunc != nil {
		return m.GenerateReportFunc(ctx, projectID, req)
	}
	return nil, errors.New("GenerateReportFunc not implemented")
}

// TestNewGenerateReportTool tests creating a new generate report tool
func TestNewGenerateReportTool(t *testing.T) {
	mockClient := &MockGenerateReportClient{}
	
	tool := NewGenerateReportTool(mockClient)
	
	if tool.Name != "generate_report" {
		t.Errorf("Expected tool name 'generate_report', got '%s'", tool.Name)
	}
	
	if tool.Description == "" {
		t.Error("Tool description should not be empty")
	}
	
	if tool.Handler == nil {
		t.Error("Tool handler should not be nil")
	}
	
	// Check input schema
	if tool.InputSchema == nil {
		t.Error("Tool should have input schema")
	}
	
	// Verify required properties
	props, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Input schema should have properties")
	}
	
	requiredProps := []string{"project_id", "format"}
	for _, prop := range requiredProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("Input schema missing '%s' property", prop)
		}
	}
	
	// Check required fields
	required, ok := tool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Input schema should have required fields")
	}
	
	if len(required) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(required))
	}
}

// TestGenerateReportHandler tests the generate report handler functionality
func TestGenerateReportHandler(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		expectedReq  pcf.GenerateReportRequest
		mockResponse *pcf.Report
		mockError    error
		expectError  bool
	}{
		{
			name: "Generate PDF report with all sections",
			params: map[string]interface{}{
				"project_id":          "proj-123",
				"format":              "pdf",
				"include_hosts":       true,
				"include_issues":      true,
				"include_credentials": true,
			},
			expectedReq: pcf.GenerateReportRequest{
				Format:             "pdf",
				IncludeHosts:       true,
				IncludeIssues:      true,
				IncludeCredentials: true,
			},
			mockResponse: &pcf.Report{
				ID:        "report-123",
				ProjectID: "proj-123",
				Format:    "pdf",
				Status:    "completed",
				URL:       "https://pcf.example.com/reports/report-123.pdf",
				CreatedAt: time.Now(),
				Size:      1048576,
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Generate JSON report with minimal sections",
			params: map[string]interface{}{
				"project_id": "proj-456",
				"format":     "json",
			},
			expectedReq: pcf.GenerateReportRequest{
				Format:             "json",
				IncludeHosts:       false,
				IncludeIssues:      false,
				IncludeCredentials: false,
			},
			mockResponse: &pcf.Report{
				ID:        "report-456",
				ProjectID: "proj-456",
				Format:    "json",
				Status:    "completed",
				URL:       "https://pcf.example.com/reports/report-456.json",
				CreatedAt: time.Now(),
				Size:      524288,
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Generate HTML report with custom sections",
			params: map[string]interface{}{
				"project_id":     "proj-789",
				"format":         "html",
				"include_hosts":  true,
				"include_issues": true,
				"sections":       []string{"executive_summary", "technical_details", "remediation"},
			},
			expectedReq: pcf.GenerateReportRequest{
				Format:             "html",
				IncludeHosts:       true,
				IncludeIssues:      true,
				IncludeCredentials: false,
				Sections:           []string{"executive_summary", "technical_details", "remediation"},
			},
			mockResponse: &pcf.Report{
				ID:        "report-789",
				ProjectID: "proj-789",
				Format:    "html",
				Status:    "completed",
				URL:       "https://pcf.example.com/reports/report-789.html",
				CreatedAt: time.Now(),
				Size:      2097152,
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Missing project_id",
			params: map[string]interface{}{
				"format": "pdf",
			},
			expectedReq:  pcf.GenerateReportRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Missing format",
			params: map[string]interface{}{
				"project_id": "proj-123",
			},
			expectedReq:  pcf.GenerateReportRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Invalid format",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"format":     "docx", // Not supported
			},
			expectedReq:  pcf.GenerateReportRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "Invalid project_id type",
			params: map[string]interface{}{
				"project_id": 123, // Should be string
				"format":     "pdf",
			},
			expectedReq:  pcf.GenerateReportRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
		{
			name: "PCF API error",
			params: map[string]interface{}{
				"project_id": "proj-error",
				"format":     "pdf",
			},
			expectedReq: pcf.GenerateReportRequest{
				Format: "pdf",
			},
			mockResponse: nil,
			mockError:    errors.New("project not found"),
			expectError:  true,
		},
		{
			name: "Report generation in progress",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"format":     "pdf",
			},
			expectedReq: pcf.GenerateReportRequest{
				Format: "pdf",
			},
			mockResponse: &pcf.Report{
				ID:        "report-progress",
				ProjectID: "proj-123",
				Format:    "pdf",
				Status:    "in_progress",
				CreatedAt: time.Now(),
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Sections as interface array",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"format":     "pdf",
				"sections":   []interface{}{"summary", "findings"},
			},
			expectedReq: pcf.GenerateReportRequest{
				Format:   "pdf",
				Sections: []string{"summary", "findings"},
			},
			mockResponse: &pcf.Report{
				ID:        "report-sections",
				ProjectID: "proj-123",
				Format:    "pdf",
				Status:    "completed",
				URL:       "https://pcf.example.com/reports/report-sections.pdf",
				CreatedAt: time.Now(),
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Invalid sections type",
			params: map[string]interface{}{
				"project_id": "proj-123",
				"format":     "pdf",
				"sections":   "invalid", // Should be array
			},
			expectedReq:  pcf.GenerateReportRequest{},
			mockResponse: nil,
			mockError:    nil,
			expectError:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockGenerateReportClient{
				GenerateReportFunc: func(ctx context.Context, projectID string, req pcf.GenerateReportRequest) (*pcf.Report, error) {
					// Verify project ID if we expect the call to succeed
					if !tt.expectError || tt.mockError != nil {
						expectedProjectID, _ := tt.params["project_id"].(string)
						if projectID != expectedProjectID {
							t.Errorf("Expected project ID '%s', got '%s'", expectedProjectID, projectID)
						}
						
						// Verify request structure
						if req.Format != tt.expectedReq.Format {
							t.Errorf("Expected format '%s', got '%s'", tt.expectedReq.Format, req.Format)
						}
						if req.IncludeHosts != tt.expectedReq.IncludeHosts {
							t.Errorf("Expected include_hosts %v, got %v", tt.expectedReq.IncludeHosts, req.IncludeHosts)
						}
						if req.IncludeIssues != tt.expectedReq.IncludeIssues {
							t.Errorf("Expected include_issues %v, got %v", tt.expectedReq.IncludeIssues, req.IncludeIssues)
						}
						if req.IncludeCredentials != tt.expectedReq.IncludeCredentials {
							t.Errorf("Expected include_credentials %v, got %v", tt.expectedReq.IncludeCredentials, req.IncludeCredentials)
						}
						
						// Check sections array
						if len(req.Sections) != len(tt.expectedReq.Sections) {
							t.Errorf("Expected %d sections, got %d", len(tt.expectedReq.Sections), len(req.Sections))
						}
					}
					
					return tt.mockResponse, tt.mockError
				},
			}
			
			// Create tool
			tool := NewGenerateReportTool(mockClient)
			
			// Execute handler
			ctx := context.Background()
			result, err := tool.Handler(ctx, tt.params)
			
			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			// Verify result structure
			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Fatal("Result should be a map")
			}
			
			// Check for report key
			reportData, ok := resultMap["report"]
			if !ok {
				t.Fatal("Result should contain 'report' key")
			}
			
			// Verify report structure
			report, ok := reportData.(map[string]interface{})
			if !ok {
				t.Fatal("Report should be a map")
			}
			
			// Check required fields
			requiredFields := []string{"id", "project_id", "format", "status", "created_at"}
			for _, field := range requiredFields {
				if _, ok := report[field]; !ok {
					t.Errorf("Report missing required field: %s", field)
				}
			}
			
			// Check message
			if message, ok := resultMap["message"].(string); !ok || message == "" {
				t.Error("Result should contain a non-empty message")
			}
			
			// If status is completed, should have URL
			if status, ok := report["status"].(string); ok && status == "completed" {
				if _, ok := report["url"]; !ok {
					t.Error("Completed report should have URL")
				}
			}
		})
	}
}