# Docker Compose Adapter

The ComposeAdapter implements the `EnvironmentAdapter` interface for Docker Compose environments. It allows the Infrastructure Resilience Engine to interact with containers managed by Docker Compose.

## Features

- **Resource Management**: List, get, start, stop, restart, delete, create, and update containers
- **Real-time Monitoring**: Watch for container events using Docker's event stream
- **Command Execution**: Execute commands inside running containers
- **Metrics Collection**: Collect CPU, memory, network, and disk metrics
- **Compose File Parsing**: Parse docker-compose.yml files and convert services to Resources

## Configuration

The adapter accepts the following configuration options:

```go
config := types.AdapterConfig{
    Config: map[string]interface{}{
        "project_name": "myproject",      // Docker Compose project name (default: "default")
        "compose_file": "docker-compose.yml", // Path to compose file (default: "docker-compose.yml")
    },
}
```

## Usage

### Basic Initialization

```go
import (
    "context"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/adapters/compose"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// Create adapter
adapter := compose.NewComposeAdapter()

// Initialize with configuration
config := types.AdapterConfig{
    Config: map[string]interface{}{
        "project_name": "myapp",
    },
}

ctx := context.Background()
err := adapter.Initialize(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer adapter.Close()
```

### Listing Resources

```go
// List all containers in the project
resources, err := adapter.ListResources(ctx, types.ResourceFilter{})
if err != nil {
    log.Fatal(err)
}

for _, resource := range resources {
    fmt.Printf("Container: %s (Status: %s)\n", resource.Name, resource.Status.Phase)
}
```

### Filtering Resources

```go
// Filter by status
filter := types.ResourceFilter{
    Statuses: []string{"running"},
}
resources, err := adapter.ListResources(ctx, filter)

// Filter by labels
filter = types.ResourceFilter{
    LabelSelector: types.LabelSelector{
        MatchLabels: map[string]string{
            "app": "web",
        },
    },
}
resources, err = adapter.ListResources(ctx, filter)
```

### Resource Operations

```go
// Get a specific resource
resource, err := adapter.GetResource(ctx, containerID)

// Start a container
err = adapter.StartResource(ctx, containerID)

// Stop a container with grace period
err = adapter.StopResource(ctx, containerID, 10*time.Second)

// Restart a container
err = adapter.RestartResource(ctx, containerID)

// Delete a container
err = adapter.DeleteResource(ctx, containerID, types.DeleteOptions{
    Force: true,
})
```

### Watching Events

```go
// Watch for container events
events, err := adapter.WatchResources(ctx, types.ResourceFilter{})
if err != nil {
    log.Fatal(err)
}

for event := range events {
    fmt.Printf("Event: %s - Container: %s\n", event.Type, event.Resource.Name)
}
```

### Executing Commands

```go
// Execute a command in a container
result, err := adapter.ExecInResource(ctx, containerID, []string{"echo", "hello"}, types.ExecOptions{})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Exit Code: %d\n", result.ExitCode)
fmt.Printf("Output: %s\n", result.Stdout)
```

### Collecting Metrics

```go
// Get metrics for a container
metrics, err := adapter.GetMetrics(ctx, containerID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("CPU Usage: %.2f%%\n", metrics.CPU.UsagePercent)
fmt.Printf("Memory Usage: %d bytes\n", metrics.Memory.UsageBytes)
```

## Docker Compose File Parsing

The adapter includes utilities for parsing docker-compose.yml files:

```go
import "github.com/yourusername/infrastructure-resilience-engine/pkg/adapters/compose"

// Parse a compose file
composeFile, err := compose.ParseComposeFile("docker-compose.yml")
if err != nil {
    log.Fatal(err)
}

// Convert services to ResourceSpecs
for name, service := range composeFile.Services {
    spec := compose.ServiceToResourceSpec(name, service)
    fmt.Printf("Service: %s, Image: %s\n", name, spec.Image)
}
```

## Resource Mapping

The adapter maps Docker containers to the unified Resource model:

| Docker Field | Resource Field | Notes |
|--------------|----------------|-------|
| Container ID | ID | Unique identifier |
| Service Name | Name | From `com.docker.compose.service` label |
| "container" | Kind | Fixed value |
| Labels | Labels | All container labels |
| State | Status.Phase | Normalized: running, stopped, pending, failed, unknown |
| Image | Spec.Image | Container image |
| Env | Spec.Environment | Environment variables |
| Ports | Spec.Ports | Port mappings |

## Status Normalization

Docker container states are normalized to standard phases:

| Docker State | Resource Phase |
|--------------|----------------|
| running | running |
| exited | stopped |
| created | pending |
| paused | stopped |
| restarting | pending |
| removing | pending |
| dead | failed |

## Requirements

- Docker Engine API v1.41+
- Docker daemon running and accessible
- Appropriate permissions to interact with Docker

## Testing

The adapter includes comprehensive tests:

```bash
# Run all tests
go test ./pkg/adapters/compose/...

# Run property-based tests only
go test -v -run TestProperty ./pkg/adapters/compose/

# Run integration tests (requires Docker)
go test -v -run TestComposeAdapter ./pkg/adapters/compose/
```

## Limitations

- The `UpdateResource` method is not fully supported as Docker doesn't allow updating running containers directly. Use delete and create instead.
- Metrics collection returns basic structure but requires additional implementation for full stats parsing.
- The adapter requires Docker daemon to be running and accessible.

## Implementation Notes

This adapter demonstrates the Core's immutability principle - it was implemented without any modifications to the Core interfaces or types. All environment-specific logic is contained within the adapter implementation.
