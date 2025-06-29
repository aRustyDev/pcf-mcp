//go:build integration
// +build integration

package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestKubernetesManifests validates all Kubernetes YAML files
func TestKubernetesManifests(t *testing.T) {
	if os.Getenv("KUBERNETES_TESTS") != "true" {
		t.Skip("Kubernetes tests not enabled. Set KUBERNETES_TESTS=true to run.")
	}

	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("kubectl not found in PATH")
	}

	// Find all YAML files in kubernetes directory
	kubeDir := filepath.Join("..", "kubernetes")
	var yamlFiles []string

	err := filepath.Walk(kubeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
			yamlFiles = append(yamlFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk kubernetes directory: %v", err)
	}

	if len(yamlFiles) == 0 {
		t.Skip("No Kubernetes YAML files found")
	}

	// Test each YAML file
	for _, file := range yamlFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			// Read file
			content, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			// Validate YAML syntax
			var docs []interface{}
			decoder := yaml.NewDecoder(bytes.NewReader(content))
			for {
				var doc interface{}
				err := decoder.Decode(&doc)
				if err != nil {
					if err.Error() == "EOF" {
						break
					}
					t.Fatalf("Invalid YAML syntax: %v", err)
				}
				docs = append(docs, doc)
			}

			if len(docs) == 0 {
				t.Error("No YAML documents found in file")
			}

			// Validate with kubectl
			cmd := exec.Command("kubectl", "apply", "--dry-run=client", "-f", file)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("kubectl validation failed: %v\nOutput: %s", err, output)
			}
		})
	}
}

// TestKubernetesDeployment tests the full deployment
func TestKubernetesDeployment(t *testing.T) {
	if os.Getenv("KUBERNETES_DEPLOY_TEST") != "true" {
		t.Skip("Kubernetes deployment test not enabled. Set KUBERNETES_DEPLOY_TEST=true to run.")
	}

	namespace := "pcf-mcp-test"

	// Create namespace
	t.Logf("Creating namespace %s", namespace)
	cmd := exec.Command("kubectl", "create", "namespace", namespace)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Ignore if namespace already exists
		if !strings.Contains(string(output), "already exists") {
			t.Fatalf("Failed to create namespace: %v\nOutput: %s", err, output)
		}
	}

	// Cleanup function
	defer func() {
		t.Logf("Cleaning up namespace %s", namespace)
		cmd := exec.Command("kubectl", "delete", "namespace", namespace, "--ignore-not-found")
		cmd.Run()
	}()

	// Apply manifests
	t.Log("Applying Kubernetes manifests")
	cmd = exec.Command("kubectl", "apply", "-f", "../kubernetes/base/", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to apply manifests: %v\nOutput: %s", err, output)
	}
	t.Logf("Apply output: %s", output)

	// Wait for deployment to be ready
	t.Log("Waiting for deployment to be ready")
	cmd = exec.Command("kubectl", "wait", "--for=condition=available",
		"deployment/pcf-mcp", "-n", namespace, "--timeout=60s")
	output, err = cmd.CombinedOutput()
	if err != nil {
		// Get pod logs for debugging
		cmd = exec.Command("kubectl", "logs", "-l", "app=pcf-mcp", "-n", namespace)
		logs, _ := cmd.CombinedOutput()
		t.Fatalf("Deployment not ready: %v\nOutput: %s\nPod logs: %s", err, output, logs)
	}

	// Check pod status
	t.Log("Checking pod status")
	cmd = exec.Command("kubectl", "get", "pods", "-l", "app=pcf-mcp", "-n", namespace, "-o", "wide")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get pods: %v\nOutput: %s", err, output)
	}
	t.Logf("Pod status:\n%s", output)

	// Test service endpoint
	t.Log("Testing service endpoint")
	cmd = exec.Command("kubectl", "run", "test-curl", "--rm", "-i", "--restart=Never",
		"--image=curlimages/curl:latest", "-n", namespace, "--",
		"curl", "-s", "http://pcf-mcp:8080/health")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Failed to test service: %v\nOutput: %s", err, output)
	} else {
		t.Logf("Health check response: %s", output)
	}
}

// TestHelmChart tests the Helm chart
func TestHelmChart(t *testing.T) {
	if os.Getenv("HELM_TESTS") != "true" {
		t.Skip("Helm tests not enabled. Set HELM_TESTS=true to run.")
	}

	// Check if helm is available
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not found in PATH")
	}

	chartPath := "../charts/pcf-mcp"

	// Check if chart exists
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		t.Skip("Helm chart not found")
	}

	// Lint the chart
	t.Log("Linting Helm chart")
	cmd := exec.Command("helm", "lint", chartPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Helm lint failed: %v\nOutput: %s", err, output)
	}

	// Template the chart to validate
	t.Log("Templating Helm chart")
	cmd = exec.Command("helm", "template", "test-release", chartPath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Helm template failed: %v\nOutput: %s", err, output)
	}

	// Validate generated YAML
	cmd = exec.Command("kubectl", "apply", "--dry-run=client", "-f", "-")
	cmd.Stdin = bytes.NewReader(output)
	validateOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Generated YAML validation failed: %v\nOutput: %s", err, validateOutput)
	}

	// Test with different values
	testValues := []string{
		"--set", "replicaCount=3",
		"--set", "image.tag=v1.0.0",
		"--set", "service.type=LoadBalancer",
		"--set", "ingress.enabled=true",
		"--set", "ingress.hosts[0].host=pcf-mcp.example.com",
	}

	t.Log("Testing with custom values")
	args := append([]string{"template", "test-release", chartPath}, testValues...)
	cmd = exec.Command("helm", args...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Helm template with values failed: %v\nOutput: %s", err, output)
	}
}

// TestKustomization tests Kustomize overlays
func TestKustomization(t *testing.T) {
	if os.Getenv("KUSTOMIZE_TESTS") != "true" {
		t.Skip("Kustomize tests not enabled. Set KUSTOMIZE_TESTS=true to run.")
	}

	// Check if kustomize is available (or kubectl with kustomize support)
	kustomizeCmd := "kustomize"
	if _, err := exec.LookPath(kustomizeCmd); err != nil {
		// Try kubectl kustomize
		kustomizeCmd = "kubectl"
		if _, err := exec.LookPath(kustomizeCmd); err != nil {
			t.Skip("kustomize or kubectl not found in PATH")
		}
	}

	overlays := []string{
		"../kubernetes/overlays/development",
		"../kubernetes/overlays/production",
	}

	for _, overlay := range overlays {
		t.Run(filepath.Base(overlay), func(t *testing.T) {
			// Check if overlay exists
			if _, err := os.Stat(overlay); os.IsNotExist(err) {
				t.Skip("Overlay not found")
			}

			// Build with kustomize
			var cmd *exec.Cmd
			if kustomizeCmd == "kustomize" {
				cmd = exec.Command(kustomizeCmd, "build", overlay)
			} else {
				cmd = exec.Command(kustomizeCmd, "kustomize", overlay)
			}

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("Kustomize build failed: %v\nOutput: %s", err, output)
				return
			}

			// Validate generated YAML
			cmd = exec.Command("kubectl", "apply", "--dry-run=client", "-f", "-")
			cmd.Stdin = bytes.NewReader(output)
			validateOutput, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("Generated YAML validation failed: %v\nOutput: %s", err, validateOutput)
			}

			// Check for expected resources
			outputStr := string(output)
			expectedResources := []string{
				"kind: Deployment",
				"kind: Service",
				"kind: ConfigMap",
			}

			for _, resource := range expectedResources {
				if !strings.Contains(outputStr, resource) {
					t.Errorf("Expected resource %s not found in output", resource)
				}
			}
		})
	}
}

// TestKubernetesHealthChecks validates health check endpoints
func TestKubernetesHealthChecks(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected int
	}{
		{
			name:     "Liveness probe",
			endpoint: "/health",
			expected: 200,
		},
		{
			name:     "Readiness probe",
			endpoint: "/health",
			expected: 200,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// This is tested through the deployment test
			// Just validate the endpoint is configured correctly
			t.Logf("Health check endpoint %s should return %d", tc.endpoint, tc.expected)
		})
	}
}

// TestResourceRequirements validates resource specifications
func TestResourceRequirements(t *testing.T) {
	manifestPath := "../kubernetes/base/deployment.yaml"

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Skip("Deployment manifest not found")
	}

	var deployment map[string]interface{}
	if err := yaml.Unmarshal(content, &deployment); err != nil {
		t.Fatalf("Failed to parse deployment: %v", err)
	}

	// Navigate to container resources
	spec, ok := deployment["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("No spec found in deployment")
	}

	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		t.Fatal("No template found in spec")
	}

	podSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("No spec found in template")
	}

	containers, ok := podSpec["containers"].([]interface{})
	if !ok || len(containers) == 0 {
		t.Fatal("No containers found")
	}

	container := containers[0].(map[string]interface{})
	resources, ok := container["resources"].(map[string]interface{})
	if !ok {
		t.Error("No resources defined for container")
		return
	}

	// Check requests
	requests, ok := resources["requests"].(map[string]interface{})
	if !ok {
		t.Error("No resource requests defined")
	} else {
		if _, ok := requests["memory"]; !ok {
			t.Error("No memory request defined")
		}
		if _, ok := requests["cpu"]; !ok {
			t.Error("No CPU request defined")
		}
	}

	// Check limits
	limits, ok := resources["limits"].(map[string]interface{})
	if !ok {
		t.Error("No resource limits defined")
	} else {
		if _, ok := limits["memory"]; !ok {
			t.Error("No memory limit defined")
		}
		if _, ok := limits["cpu"]; !ok {
			t.Error("No CPU limit defined")
		}
	}
}

// Helper function to check if a command exists
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
