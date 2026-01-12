package knowledge

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	agentforge "github.com/thinktwice/agentForge/src"
)

const (
	qdrantContainerName = "thinktwice-qdrant"
	qdrantImage         = "qdrant/qdrant:latest"
	qdrantHTTPPort      = "6333"
	qdrantGRPCPort      = "6334"
)

// qdrantDocker manages Qdrant Docker container lifecycle
type qdrantDocker struct {
	containerStarted bool
	containerName    string
}

// ensureQdrantRunning checks if Qdrant is running and starts it if not
func ensureQdrantRunning(ctx context.Context) (*qdrantDocker, error) {
	docker := &qdrantDocker{
		containerName: qdrantContainerName,
	}

	// First, check if Qdrant is already accessible
	if docker.isQdrantAccessible() {
		agentforge.Info("Qdrant is already running and accessible")
		return docker, nil
	}

	// Check if Docker is available
	if !docker.isDockerAvailable() {
		return nil, fmt.Errorf("Docker is not available. Please install Docker or start Qdrant manually")
	}

	// Check if container already exists
	exists, err := docker.containerExists()
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing container: %w", err)
	}

	if exists {
		// Container exists, try to start it
		agentforge.Info("Starting existing Qdrant container: %s", docker.containerName)
		if err := docker.startContainer(); err != nil {
			return nil, fmt.Errorf("failed to start existing container: %w", err)
		}
	} else {
		// Container doesn't exist, create and start it
		agentforge.Info("Creating and starting Qdrant container: %s", docker.containerName)
		if err := docker.createAndStartContainer(); err != nil {
			return nil, fmt.Errorf("failed to create container: %w", err)
		}
	}

	// Wait for Qdrant to be ready
	agentforge.Info("Waiting for Qdrant to be ready...")
	if err := docker.waitForQdrant(ctx, 30*time.Second); err != nil {
		return nil, fmt.Errorf("Qdrant failed to become ready: %w", err)
	}

	docker.containerStarted = true
	agentforge.Info("Qdrant is ready and accessible")
	return docker, nil
}

// isQdrantAccessible checks if Qdrant HTTP API is accessible
func (d *qdrantDocker) isQdrantAccessible() bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get("http://localhost:" + qdrantHTTPPort + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// isDockerAvailable checks if Docker command is available
func (d *qdrantDocker) isDockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

// containerExists checks if the container exists
func (d *qdrantDocker) containerExists() (bool, error) {
	cmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", d.containerName), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) == d.containerName, nil
}

// startContainer starts an existing container
func (d *qdrantDocker) startContainer() error {
	cmd := exec.Command("docker", "start", d.containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// createAndStartContainer creates and starts a new Qdrant container
func (d *qdrantDocker) createAndStartContainer() error {
	// Pull the image if needed (this might take a while, but ensures we have the latest)
	agentforge.Debug("Ensuring Qdrant Docker image is available...")
	pullCmd := exec.Command("docker", "pull", qdrantImage)
	if err := pullCmd.Run(); err != nil {
		agentforge.Warn("Failed to pull Qdrant image (will try to use existing): %v", err)
	}

	// Create and start the container
	cmd := exec.Command("docker", "run", "-d",
		"--name", d.containerName,
		"-p", qdrantHTTPPort+":"+qdrantHTTPPort,
		"-p", qdrantGRPCPort+":"+qdrantGRPCPort,
		qdrantImage)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	return nil
}

// waitForQdrant waits for Qdrant to become ready
func (d *qdrantDocker) waitForQdrant(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if d.isQdrantAccessible() {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for Qdrant to become ready")
			}
		}
	}
}

// stopContainer stops the container (optional cleanup)
func (d *qdrantDocker) stopContainer() error {
	if !d.containerStarted {
		return nil
	}

	agentforge.Info("Stopping Qdrant container: %s", d.containerName)
	cmd := exec.Command("docker", "stop", d.containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

// removeContainer removes the container (optional cleanup)
func (d *qdrantDocker) removeContainer() error {
	if !d.containerStarted {
		return nil
	}

	agentforge.Info("Removing Qdrant container: %s", d.containerName)
	cmd := exec.Command("docker", "rm", d.containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	return nil
}
