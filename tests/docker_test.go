package tests

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestDockerBuild tests that the Docker image can be built successfully
func TestDockerBuild(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available, skipping test")
	}

	// Build the Docker image
	cmd := exec.Command("docker", "build", "-t", "pcf-mcp:test", "..")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Docker build failed: %v\nOutput:\n%s", err, string(output))
	}

	// Verify image was created
	cmd = exec.Command("docker", "images", "pcf-mcp:test", "--format", "{{.Repository}}:{{.Tag}}")
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("Failed to list images: %v", err)
	}

	if !strings.Contains(string(output), "pcf-mcp:test") {
		t.Error("Docker image was not created")
	}
}

// TestDockerImageSize tests that the Docker image is reasonably small
func TestDockerImageSize(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available, skipping test")
	}

	// Get image size
	cmd := exec.Command("docker", "images", "pcf-mcp:test", "--format", "{{.Size}}")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get image size: %v", err)
	}

	sizeStr := strings.TrimSpace(string(output))
	if sizeStr == "" {
		t.Skip("Image not found, skipping size test")
	}

	// Parse size (Docker outputs in human-readable format like "15.2MB")
	t.Logf("Docker image size: %s", sizeStr)

	// Check if size contains "GB" which would indicate it's too large
	if strings.Contains(sizeStr, "GB") {
		t.Errorf("Docker image is too large: %s", sizeStr)
	}
}

// TestDockerImageSecurity tests security aspects of the Docker image
func TestDockerImageSecurity(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available, skipping test")
	}

	// Create a container to inspect
	cmd := exec.Command("docker", "create", "--name", "pcf-mcp-test", "pcf-mcp:test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer exec.Command("docker", "rm", "pcf-mcp-test").Run()

	// Check that it runs as non-root
	cmd = exec.Command("docker", "inspect", "pcf-mcp-test", "--format", "{{.Config.User}}")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to inspect container: %v", err)
	}

	user := strings.TrimSpace(string(output))
	if user == "" || user == "root" || user == "0" {
		t.Errorf("Container should not run as root, but user is: %s", user)
	}
}

// TestDockerRun tests that the container can start and respond to health checks
func TestDockerRun(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available, skipping test")
	}

	// Run the container with test configuration
	cmd := exec.Command("docker", "run", "-d",
		"--name", "pcf-mcp-run-test",
		"-e", "PCF_MCP_SERVER_TRANSPORT=http",
		"-e", "PCF_MCP_PCF_URL=http://test-pcf:5000",
		"-e", "PCF_MCP_PCF_API_KEY=test-key",
		"-p", "8080:8080",
		"pcf-mcp:test",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput:\n%s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))
	defer exec.Command("docker", "rm", "-f", containerID).Run()

	// Give container time to start
	time.Sleep(2 * time.Second)

	// Check if container is still running
	cmd = exec.Command("docker", "ps", "-q", "-f", "id="+containerID)
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("Failed to check container status: %v", err)
	}

	if strings.TrimSpace(string(output)) == "" {
		// Container stopped, get logs
		cmd = exec.Command("docker", "logs", containerID)
		logs, _ := cmd.Output()
		t.Fatalf("Container is not running. Logs:\n%s", string(logs))
	}
}

// TestDockerCompose tests Docker Compose configuration
func TestDockerCompose(t *testing.T) {
	// Check if docker-compose is available
	if _, err := exec.LookPath("docker-compose"); err != nil {
		t.Skip("Docker Compose not available, skipping test")
	}

	// We'll create docker-compose.yml in the next step
	t.Skip("Docker Compose configuration not yet implemented")
}
