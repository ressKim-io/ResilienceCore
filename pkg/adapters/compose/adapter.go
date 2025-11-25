// Package compose provides a Docker Compose adapter for the Infrastructure Resilience Engine.
// It implements the EnvironmentAdapter interface to interact with Docker Compose environments.
package compose

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	coretypes "github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// Adapter implements EnvironmentAdapter for docker-compose environments
type Adapter struct {
	client      *client.Client
	composeFile string
	projectName string
	mu          sync.RWMutex
	watchers    map[string]context.CancelFunc
}

// NewAdapter creates a new Adapter
func NewAdapter() *Adapter {
	return &Adapter{
		watchers: make(map[string]context.CancelFunc),
	}
}

// NewComposeAdapter creates a new Adapter (deprecated: use NewAdapter)
func NewComposeAdapter() *Adapter {
	return NewAdapter()
}

// Initialize initializes the adapter with configuration
func (a *Adapter) Initialize(ctx context.Context, config coretypes.AdapterConfig) error {
	// Extract configuration
	composeFile, ok := config.Config["compose_file"].(string)
	if !ok || composeFile == "" {
		composeFile = "docker-compose.yml"
	}
	a.composeFile = composeFile

	projectName, ok := config.Config["project_name"].(string)
	if !ok || projectName == "" {
		projectName = "default"
	}
	a.projectName = projectName

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	a.client = cli

	// Verify connection
	if _, err := a.client.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping docker daemon: %w", err)
	}

	return nil
}

// Close closes the adapter and releases resources
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Cancel all watchers
	for _, cancel := range a.watchers {
		cancel()
	}
	a.watchers = make(map[string]context.CancelFunc)

	// Close Docker client
	if a.client != nil {
		return a.client.Close()
	}
	return nil
}

// ListResources lists all resources matching the filter
func (a *Adapter) ListResources(ctx context.Context, filter coretypes.ResourceFilter) ([]coretypes.Resource, error) {
	// Build Docker filters
	dockerFilters := filters.NewArgs()
	dockerFilters.Add("label", fmt.Sprintf("com.docker.compose.project=%s", a.projectName))

	// List containers
	containers, err := a.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: dockerFilters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Convert to Resources
	var resources []coretypes.Resource
	for _, c := range containers {
		resource := a.containerToResource(c)

		// Apply filters
		if a.matchesFilter(resource, filter) {
			resources = append(resources, resource)
		}
	}

	// Apply pagination
	if filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if start > len(resources) {
			return []coretypes.Resource{}, nil
		}
		if end > len(resources) {
			end = len(resources)
		}
		resources = resources[start:end]
	}

	return resources, nil
}

// GetResource gets a specific resource by ID
func (a *Adapter) GetResource(ctx context.Context, id string) (coretypes.Resource, error) {
	containerJSON, err := a.client.ContainerInspect(ctx, id)
	if err != nil {
		return coretypes.Resource{}, fmt.Errorf("failed to inspect container: %w", err)
	}

	return a.containerJSONToResource(containerJSON), nil
}

// StartResource starts a resource
func (a *Adapter) StartResource(ctx context.Context, id string) error {
	if err := a.client.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// StopResource stops a resource
func (a *Adapter) StopResource(ctx context.Context, id string, gracePeriod time.Duration) error {
	timeout := int(gracePeriod.Seconds())
	if err := a.client.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

// RestartResource restarts a resource
func (a *Adapter) RestartResource(ctx context.Context, id string) error {
	timeout := int(10) // 10 seconds default
	if err := a.client.ContainerRestart(ctx, id, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}
	return nil
}

// DeleteResource deletes a resource
func (a *Adapter) DeleteResource(ctx context.Context, id string, options coretypes.DeleteOptions) error {
	removeOptions := container.RemoveOptions{
		Force:         options.Force,
		RemoveVolumes: true,
	}

	if err := a.client.ContainerRemove(ctx, id, removeOptions); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	return nil
}

// CreateResource creates a new resource
func (a *Adapter) CreateResource(ctx context.Context, spec coretypes.ResourceSpec) (coretypes.Resource, error) {
	// Convert ResourceSpec to container config
	config := &container.Config{
		Image: spec.Image,
		Env:   envMapToSlice(spec.Environment),
		Cmd:   spec.Command,
	}

	// Convert ports
	exposedPorts := make(nat.PortSet)
	portBindings := make(nat.PortMap)
	for _, p := range spec.Ports {
		port, err := nat.NewPort(strings.ToLower(p.Protocol), fmt.Sprintf("%d", p.ContainerPort))
		if err != nil {
			continue
		}
		exposedPorts[port] = struct{}{}
		if p.HostPort > 0 {
			portBindings[port] = []nat.PortBinding{
				{
					HostPort: fmt.Sprintf("%d", p.HostPort),
				},
			}
		}
	}
	config.ExposedPorts = exposedPorts

	// Host config
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}

	// Create container
	resp, err := a.client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return coretypes.Resource{}, fmt.Errorf("failed to create container: %w", err)
	}

	// Get the created container
	return a.GetResource(ctx, resp.ID)
}

// UpdateResource updates a resource (limited support in Docker)
func (a *Adapter) UpdateResource(ctx context.Context, id string, spec coretypes.ResourceSpec) (coretypes.Resource, error) {
	// Docker doesn't support updating running containers directly
	// We would need to recreate the container
	return coretypes.Resource{}, fmt.Errorf("update not supported for docker containers, use delete and create instead")
}

// WatchResources watches for resource changes
func (a *Adapter) WatchResources(ctx context.Context, filter coretypes.ResourceFilter) (<-chan coretypes.ResourceEvent, error) {
	eventChan := make(chan coretypes.ResourceEvent, 100)

	// Build Docker event filters
	dockerFilters := filters.NewArgs()
	dockerFilters.Add("type", "container")
	dockerFilters.Add("label", fmt.Sprintf("com.docker.compose.project=%s", a.projectName))

	// Start watching Docker events
	eventsChan, errsChan := a.client.Events(ctx, events.ListOptions{
		Filters: dockerFilters,
	})

	go func() {
		defer close(eventChan)

		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errsChan:
				if err != nil && err != io.EOF {
					// Log error but continue watching
					continue
				}
				return
			case event := <-eventsChan:
				// Convert Docker event to ResourceEvent
				resourceEvent := a.dockerEventToResourceEvent(ctx, event)
				if resourceEvent != nil {
					// Apply filter
					if a.matchesFilter(resourceEvent.Resource, filter) {
						select {
						case eventChan <- *resourceEvent:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return eventChan, nil
}

// ExecInResource executes a command in a resource
func (a *Adapter) ExecInResource(ctx context.Context, id string, cmd []string, options coretypes.ExecOptions) (coretypes.ExecResult, error) {
	// Create exec instance
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          options.TTY,
	}

	execID, err := a.client.ContainerExecCreate(ctx, id, execConfig)
	if err != nil {
		return coretypes.ExecResult{}, fmt.Errorf("failed to create exec: %w", err)
	}

	// Attach to exec
	resp, err := a.client.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return coretypes.ExecResult{}, fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer resp.Close()

	// Read output
	var stdout, stderr strings.Builder
	if options.Stdout != nil {
		if _, copyErr := io.Copy(options.Stdout, resp.Reader); copyErr != nil {
			return coretypes.ExecResult{}, fmt.Errorf("failed to copy output: %w", copyErr)
		}
	} else {
		if _, copyErr := io.Copy(&stdout, resp.Reader); copyErr != nil {
			return coretypes.ExecResult{}, fmt.Errorf("failed to read output: %w", copyErr)
		}
	}

	// Get exit code
	inspectResp, err := a.client.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return coretypes.ExecResult{}, fmt.Errorf("failed to inspect exec: %w", err)
	}

	return coretypes.ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: inspectResp.ExitCode,
	}, nil
}

// GetMetrics gets metrics for a resource
func (a *Adapter) GetMetrics(ctx context.Context, id string) (coretypes.Metrics, error) {
	stats, err := a.client.ContainerStats(ctx, id, false)
	if err != nil {
		return coretypes.Metrics{}, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer func() {
		// nolint:errcheck,gosec // Ignore close error as we're already returning metrics
		stats.Body.Close()
	}()

	// Parse stats (simplified - real implementation would parse JSON)
	// For now, return empty metrics
	return coretypes.Metrics{
		Timestamp:  time.Now(),
		ResourceID: id,
		CPU: coretypes.CPUMetrics{
			UsagePercent: 0,
			UsageCores:   0,
		},
		Memory: coretypes.MemoryMetrics{
			UsageBytes:   0,
			LimitBytes:   0,
			UsagePercent: 0,
		},
		Network: coretypes.NetworkMetrics{},
		Disk:    coretypes.DiskMetrics{},
	}, nil
}

// GetAdapterInfo returns adapter metadata
func (a *Adapter) GetAdapterInfo() coretypes.AdapterInfo {
	return coretypes.AdapterInfo{
		Name:        "ComposeAdapter",
		Version:     "1.0.0",
		Environment: "compose",
		Capabilities: []string{
			"list", "get", "start", "stop", "restart",
			"delete", "create", "watch", "exec", "metrics",
		},
	}
}

// Helper methods

func (a *Adapter) containerToResource(c container.Summary) coretypes.Resource {
	// Extract service name from labels
	serviceName := c.Labels["com.docker.compose.service"]
	if serviceName == "" {
		serviceName = c.Names[0]
	}

	// Determine status
	status := a.containerStateToStatus(c.State, c.Status)

	// Build resource
	resource := coretypes.Resource{
		ID:          c.ID,
		Name:        serviceName,
		Kind:        "container",
		Labels:      c.Labels,
		Annotations: make(map[string]string),
		Status:      status,
		Spec: coretypes.ResourceSpec{
			Image: c.Image,
		},
		Metadata: map[string]interface{}{
			"docker_id":    c.ID,
			"docker_names": c.Names,
			"created":      c.Created,
		},
	}

	return resource
}

func (a *Adapter) containerJSONToResource(c container.InspectResponse) coretypes.Resource {
	// Extract service name
	serviceName := c.Config.Labels["com.docker.compose.service"]
	if serviceName == "" {
		serviceName = c.Name
	}

	// Determine status
	status := a.containerJSONStateToStatus(c.State)

	// Parse environment
	env := make(map[string]string)
	for _, e := range c.Config.Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	// Build resource
	resource := coretypes.Resource{
		ID:          c.ID,
		Name:        serviceName,
		Kind:        "container",
		Labels:      c.Config.Labels,
		Annotations: make(map[string]string),
		Status:      status,
		Spec: coretypes.ResourceSpec{
			Image:       c.Config.Image,
			Command:     c.Config.Cmd,
			Environment: env,
		},
		Metadata: map[string]interface{}{
			"docker_id": c.ID,
			"created":   c.Created,
		},
	}

	return resource
}

func (a *Adapter) containerStateToStatus(state, status string) coretypes.ResourceStatus {
	phase := "unknown"
	message := status

	switch state {
	case "running":
		phase = "running"
	case "exited":
		phase = "stopped"
	case "created":
		phase = "pending"
	case "paused":
		phase = "stopped"
	case "restarting":
		phase = "pending"
	case "removing":
		phase = "pending"
	case "dead":
		phase = "failed"
	}

	return coretypes.ResourceStatus{
		Phase:   phase,
		Message: message,
		Reason:  state,
		Conditions: []coretypes.Condition{
			{
				Type:               "Ready",
				Status:             boolToStatus(phase == "running"),
				Reason:             state,
				Message:            message,
				LastTransitionTime: time.Now(),
			},
		},
	}
}

func (a *Adapter) containerJSONStateToStatus(state *container.State) coretypes.ResourceStatus {
	message := state.Status

	var phase string
	switch {
	case state.Running:
		phase = "running"
	case state.Paused:
		phase = "stopped"
	case state.Restarting:
		phase = "pending"
	case state.Dead:
		phase = "failed"
	default:
		phase = "stopped"
	}

	return coretypes.ResourceStatus{
		Phase:   phase,
		Message: message,
		Reason:  state.Status,
		Conditions: []coretypes.Condition{
			{
				Type:               "Ready",
				Status:             boolToStatus(state.Running),
				Reason:             state.Status,
				Message:            message,
				LastTransitionTime: time.Now(),
			},
		},
	}
}

func (a *Adapter) dockerEventToResourceEvent(ctx context.Context, event events.Message) *coretypes.ResourceEvent {
	var eventType coretypes.EventType

	switch event.Action {
	case "create", "start":
		eventType = coretypes.EventAdded
	case "die", "stop", "kill", "pause", "unpause":
		eventType = coretypes.EventModified
	case "destroy":
		eventType = coretypes.EventDeleted
	default:
		eventType = coretypes.EventModified
	}

	// Get container details
	resource, err := a.GetResource(ctx, event.Actor.ID)
	if err != nil {
		// If we can't get the resource, create a minimal one
		resource = coretypes.Resource{
			ID:   event.Actor.ID,
			Name: event.Actor.Attributes["name"],
			Kind: "container",
		}
	}

	return &coretypes.ResourceEvent{
		Type:     eventType,
		Resource: resource,
	}
}

func (a *Adapter) matchesFilter(resource coretypes.Resource, filter coretypes.ResourceFilter) bool {
	// Check kinds
	if len(filter.Kinds) > 0 {
		found := false
		for _, kind := range filter.Kinds {
			if resource.Kind == kind {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check statuses
	if len(filter.Statuses) > 0 {
		found := false
		for _, status := range filter.Statuses {
			if resource.Status.Phase == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check names
	if len(filter.Names) > 0 {
		found := false
		for _, name := range filter.Names {
			if resource.Name == name || strings.Contains(resource.Name, name) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check label selector
	if len(filter.LabelSelector.MatchLabels) > 0 {
		for k, v := range filter.LabelSelector.MatchLabels {
			if resource.Labels[k] != v {
				return false
			}
		}
	}

	return true
}

func envMapToSlice(env map[string]string) []string {
	var result []string
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func boolToStatus(b bool) string {
	if b {
		return "True"
	}
	return "False"
}
