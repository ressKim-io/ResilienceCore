package compose

import (
	"context"
	"testing"
	"time"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// TestComposeAdapter_Initialize tests adapter initialization
func TestComposeAdapter_Initialize(t *testing.T) {
	adapter := NewAdapter()

	config := types.AdapterConfig{
		Config: map[string]interface{}{
			"project_name": "test",
			"compose_file": "docker-compose.yml",
		},
	}

	ctx := context.Background()
	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Skipf("Skipping test - Docker not available: %v", err)
	}
	defer adapter.Close()

	// Verify adapter info
	info := adapter.GetAdapterInfo()
	if info.Name != "ComposeAdapter" {
		t.Errorf("Expected adapter name 'ComposeAdapter', got '%s'", info.Name)
	}
	if info.Environment != "compose" {
		t.Errorf("Expected environment 'compose', got '%s'", info.Environment)
	}
}

// TestComposeAdapter_ListResources tests listing resources
func TestComposeAdapter_ListResources(t *testing.T) {
	adapter := NewAdapter()

	config := types.AdapterConfig{
		Config: map[string]interface{}{
			"project_name": "test",
		},
	}

	ctx := context.Background()
	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Skipf("Skipping test - Docker not available: %v", err)
	}
	defer adapter.Close()

	// List all resources
	resources, err := adapter.ListResources(ctx, types.ResourceFilter{})
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}

	// Should return empty list or actual containers
	t.Logf("Found %d resources", len(resources))
}

// TestComposeAdapter_GetResource tests getting a specific resource
func TestComposeAdapter_GetResource(t *testing.T) {
	adapter := NewAdapter()

	config := types.AdapterConfig{
		Config: map[string]interface{}{
			"project_name": "test",
		},
	}

	ctx := context.Background()
	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Skipf("Skipping test - Docker not available: %v", err)
	}
	defer adapter.Close()

	// List resources first
	resources, err := adapter.ListResources(ctx, types.ResourceFilter{})
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}

	if len(resources) == 0 {
		t.Skip("No resources available for testing")
	}

	// Get first resource
	resource, err := adapter.GetResource(ctx, resources[0].ID)
	if err != nil {
		t.Fatalf("Failed to get resource: %v", err)
	}

	if resource.ID != resources[0].ID {
		t.Errorf("Expected resource ID '%s', got '%s'", resources[0].ID, resource.ID)
	}
}

// TestComposeAdapter_ResourceOperations tests start/stop/restart operations
func TestComposeAdapter_ResourceOperations(t *testing.T) {
	adapter := NewAdapter()

	config := types.AdapterConfig{
		Config: map[string]interface{}{
			"project_name": "test",
		},
	}

	ctx := context.Background()
	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Skipf("Skipping test - Docker not available: %v", err)
	}
	defer adapter.Close()

	// Create a test container
	spec := types.ResourceSpec{
		Image:   "alpine:latest",
		Command: []string{"sleep", "3600"},
	}

	resource, err := adapter.CreateResource(ctx, spec)
	if err != nil {
		t.Skipf("Skipping test - cannot create container: %v", err)
	}
	defer adapter.DeleteResource(ctx, resource.ID, types.DeleteOptions{Force: true})

	// Start the container
	err = adapter.StartResource(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to start resource: %v", err)
	}

	// Give it a moment to start
	time.Sleep(1 * time.Second)

	// Check status
	resource, err = adapter.GetResource(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to get resource: %v", err)
	}
	if resource.Status.Phase != "running" {
		t.Errorf("Expected status 'running', got '%s'", resource.Status.Phase)
	}

	// Stop the container
	err = adapter.StopResource(ctx, resource.ID, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to stop resource: %v", err)
	}

	// Give it a moment to stop
	time.Sleep(1 * time.Second)

	// Check status
	resource, err = adapter.GetResource(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to get resource: %v", err)
	}
	if resource.Status.Phase != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", resource.Status.Phase)
	}

	// Restart the container
	err = adapter.RestartResource(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to restart resource: %v", err)
	}

	// Give it a moment to restart
	time.Sleep(1 * time.Second)

	// Check status
	resource, err = adapter.GetResource(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to get resource: %v", err)
	}
	if resource.Status.Phase != "running" {
		t.Errorf("Expected status 'running' after restart, got '%s'", resource.Status.Phase)
	}
}

// TestComposeAdapter_ExecInResource tests command execution
func TestComposeAdapter_ExecInResource(t *testing.T) {
	adapter := NewAdapter()

	config := types.AdapterConfig{
		Config: map[string]interface{}{
			"project_name": "test",
		},
	}

	ctx := context.Background()
	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Skipf("Skipping test - Docker not available: %v", err)
	}
	defer adapter.Close()

	// Create and start a test container
	spec := types.ResourceSpec{
		Image:   "alpine:latest",
		Command: []string{"sleep", "3600"},
	}

	resource, err := adapter.CreateResource(ctx, spec)
	if err != nil {
		t.Skipf("Skipping test - cannot create container: %v", err)
	}
	defer adapter.DeleteResource(ctx, resource.ID, types.DeleteOptions{Force: true})

	err = adapter.StartResource(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to start resource: %v", err)
	}

	// Give it a moment to start
	time.Sleep(1 * time.Second)

	// Execute a command
	result, err := adapter.ExecInResource(ctx, resource.ID, []string{"echo", "hello"}, types.ExecOptions{})
	if err != nil {
		t.Fatalf("Failed to exec in resource: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}

// TestComposeAdapter_WatchResources tests resource watching
func TestComposeAdapter_WatchResources(t *testing.T) {
	adapter := NewAdapter()

	config := types.AdapterConfig{
		Config: map[string]interface{}{
			"project_name": "test",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Skipf("Skipping test - Docker not available: %v", err)
	}
	defer adapter.Close()

	// Start watching
	events, err := adapter.WatchResources(ctx, types.ResourceFilter{})
	if err != nil {
		t.Fatalf("Failed to watch resources: %v", err)
	}

	// Create a test container in another goroutine
	go func() {
		time.Sleep(1 * time.Second)
		spec := types.ResourceSpec{
			Image:   "alpine:latest",
			Command: []string{"sleep", "10"},
		}
		resource, err := adapter.CreateResource(context.Background(), spec)
		if err == nil {
			time.Sleep(2 * time.Second)
			adapter.DeleteResource(context.Background(), resource.ID, types.DeleteOptions{Force: true})
		}
	}()

	// Wait for events
	eventCount := 0
	timeout := time.After(8 * time.Second)

	for {
		select {
		case event, ok := <-events:
			if !ok {
				t.Log("Event channel closed")
				return
			}
			t.Logf("Received event: %s for resource %s", event.Type, event.Resource.Name)
			eventCount++
			if eventCount >= 2 {
				// Received enough events
				return
			}
		case <-timeout:
			t.Log("Timeout waiting for events")
			return
		case <-ctx.Done():
			return
		}
	}
}

// TestComposeAdapter_GetMetrics tests metrics collection
func TestComposeAdapter_GetMetrics(t *testing.T) {
	adapter := NewAdapter()

	config := types.AdapterConfig{
		Config: map[string]interface{}{
			"project_name": "test",
		},
	}

	ctx := context.Background()
	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Skipf("Skipping test - Docker not available: %v", err)
	}
	defer adapter.Close()

	// Create and start a test container
	spec := types.ResourceSpec{
		Image:   "alpine:latest",
		Command: []string{"sleep", "3600"},
	}

	resource, err := adapter.CreateResource(ctx, spec)
	if err != nil {
		t.Skipf("Skipping test - cannot create container: %v", err)
	}
	defer adapter.DeleteResource(ctx, resource.ID, types.DeleteOptions{Force: true})

	err = adapter.StartResource(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to start resource: %v", err)
	}

	// Give it a moment to start
	time.Sleep(1 * time.Second)

	// Get metrics
	metrics, err := adapter.GetMetrics(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}

	if metrics.ResourceID != resource.ID {
		t.Errorf("Expected resource ID '%s', got '%s'", resource.ID, metrics.ResourceID)
	}
}
