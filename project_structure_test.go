package main

import (
	"os"
	"testing"
)

// TestProjectStructure validates that the required project structure exists
// and is properly organized according to Go best practices
func TestProjectStructure(t *testing.T) {
	// Define the expected project structure
	requiredDirs := []string{
		"cmd/pcf-mcp",
		"internal/config",
		"internal/pcf",
		"internal/mcp",
		"internal/observability",
		"internal/transport",
		"pkg",
		"tests",
		"docs",
	}

	requiredFiles := []string{
		"go.mod",
		"go.sum",
		".gitignore",
		"justfile",
		"Dockerfile",
		"README.md",
	}

	// Test that all required directories exist
	t.Run("RequiredDirectoriesExist", func(t *testing.T) {
		for _, dir := range requiredDirs {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				t.Errorf("Required directory missing: %s", dir)
			}
		}
	})

	// Test that all required files exist
	t.Run("RequiredFilesExist", func(t *testing.T) {
		for _, file := range requiredFiles {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				t.Errorf("Required file missing: %s", file)
			}
		}
	})

	// Test that go.mod has the correct module name
	t.Run("GoModuleNameCorrect", func(t *testing.T) {
		content, err := os.ReadFile("go.mod")
		if err != nil {
			t.Fatalf("Failed to read go.mod: %v", err)
		}
		
		expectedModule := "github.com/analyst/pcf-mcp"
		if !contains(string(content), "module "+expectedModule) {
			t.Errorf("go.mod does not contain expected module name: %s", expectedModule)
		}
	})

	// Test that .gitignore has Go-specific entries
	t.Run("GitignoreHasGoEntries", func(t *testing.T) {
		content, err := os.ReadFile(".gitignore")
		if err != nil {
			t.Fatalf("Failed to read .gitignore: %v", err)
		}

		requiredEntries := []string{
			"*.exe",
			"*.dll",
			"*.so",
			"*.dylib",
			"*.test",
			"*.out",
			"vendor/",
			"bin/",
			"coverage.out",
			"coverage.html",
		}

		for _, entry := range requiredEntries {
			if !contains(string(content), entry) {
				t.Errorf(".gitignore missing required entry: %s", entry)
			}
		}
	})

	// Test that justfile has basic tasks
	t.Run("JustfileHasBasicTasks", func(t *testing.T) {
		content, err := os.ReadFile("justfile")
		if err != nil {
			t.Fatalf("Failed to read justfile: %v", err)
		}

		requiredTasks := []string{
			"test:",
			"build:",
			"lint:",
			"fmt:",
			"run:",
			"docker:",
			"clean:",
		}

		for _, task := range requiredTasks {
			if !contains(string(content), task) {
				t.Errorf("justfile missing required task: %s", task)
			}
		}
	})
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}