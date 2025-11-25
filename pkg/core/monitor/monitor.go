// Package monitor provides the default implementation of the Monitor interface
// for health checking and metrics collection
package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// DefaultMonitor is the default implementation of the Monitor interface
type DefaultMonitor struct {
	mu sync.RWMutex

	// Environment adapter for resource operations
	adapter types.EnvironmentAdapter

	// Health check strategies registry
	strategies map[string]types.HealthCheckStrategy

	// Default polling interval for condition waiting
	defaultPollInterval time.Duration
}

// NewDefaultMonitor creates a new DefaultMonitor instance
func NewDefaultMonitor(adapter types.EnvironmentAdapter) *DefaultMonitor {
	monitor := &DefaultMonitor{
		adapter:             adapter,
		strategies:          make(map[string]types.HealthCheckStrategy),
		defaultPollInterval: 1 * time.Second,
	}

	// Register built-in strategies
	monitor.registerBuiltInStrategies()

	return monitor
}

// registerBuiltInStrategies registers the built-in health check strategies
func (m *DefaultMonitor) registerBuiltInStrategies() {
	// HTTP health check strategy
	httpStrategy := &types.HTTPHealthCheckStrategy{
		ExpectedStatusCodes: []int{200},
	}
	if err := m.RegisterHealthCheckStrategy("http", httpStrategy); err != nil {
		// This should never fail for built-in strategies, but handle it gracefully
		panic(fmt.Sprintf("failed to register built-in http strategy: %v", err))
	}

	// TCP health check strategy
	tcpStrategy := &types.TCPHealthCheckStrategy{
		DialTimeout: 5 * time.Second,
	}
	if err := m.RegisterHealthCheckStrategy("tcp", tcpStrategy); err != nil {
		panic(fmt.Sprintf("failed to register built-in tcp strategy: %v", err))
	}

	// Exec health check strategy
	execStrategy := &types.ExecHealthCheckStrategy{
		ExpectedExitCode: 0,
	}
	if err := m.RegisterHealthCheckStrategy("exec", execStrategy); err != nil {
		panic(fmt.Sprintf("failed to register built-in exec strategy: %v", err))
	}
}

// CheckHealth checks the health of a resource using the appropriate strategy
func (m *DefaultMonitor) CheckHealth(ctx context.Context, resource types.Resource) (types.HealthStatus, error) {
	// Check if resource has health check configuration
	if resource.Spec.HealthCheck == nil {
		return types.HealthStatus{
			Status:  types.HealthStatusUnknown,
			Message: "no health check configuration",
		}, fmt.Errorf("resource %s has no health check configuration", resource.ID)
	}

	// Get the appropriate strategy
	strategyType := resource.Spec.HealthCheck.Type
	if strategyType == "" {
		return types.HealthStatus{
			Status:  types.HealthStatusUnknown,
			Message: "health check type not specified",
		}, fmt.Errorf("health check type not specified for resource %s", resource.ID)
	}

	strategy, err := m.GetHealthCheckStrategy(strategyType)
	if err != nil {
		return types.HealthStatus{
			Status:  types.HealthStatusUnknown,
			Message: fmt.Sprintf("strategy not found: %s", strategyType),
		}, fmt.Errorf("health check strategy %s not found for resource %s: %w", strategyType, resource.ID, err)
	}

	// Execute the health check
	status, err := strategy.Check(ctx, resource)
	if err != nil {
		return status, fmt.Errorf("health check failed for resource %s: %w", resource.ID, err)
	}

	return status, nil
}

// WaitForCondition waits for a condition to be met on a resource
func (m *DefaultMonitor) WaitForCondition(ctx context.Context, resource types.Resource, condition types.Condition, timeout time.Duration) error {
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Poll interval
	pollInterval := m.defaultPollInterval

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("timeout waiting for condition %s on resource %s", condition.Type, resource.ID)
			}
			return fmt.Errorf("context cancelled while waiting for condition %s on resource %s: %w", condition.Type, resource.ID, timeoutCtx.Err())

		case <-ticker.C:
			// Get the current resource state
			currentResource, err := m.adapter.GetResource(timeoutCtx, resource.ID)
			if err != nil {
				// Continue polling on transient errors
				continue
			}

			// Check if the condition is met
			if m.isConditionMet(currentResource, condition) {
				return nil
			}
		}
	}
}

// WaitForHealthy waits for a resource to become healthy
func (m *DefaultMonitor) WaitForHealthy(ctx context.Context, resource types.Resource, timeout time.Duration) error {
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Poll interval
	pollInterval := m.defaultPollInterval

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("timeout waiting for resource %s to become healthy", resource.ID)
			}
			return fmt.Errorf("context cancelled while waiting for resource %s to become healthy: %w", resource.ID, timeoutCtx.Err())

		case <-ticker.C:
			// Get the current resource state
			currentResource, err := m.adapter.GetResource(timeoutCtx, resource.ID)
			if err != nil {
				// Continue polling on transient errors
				continue
			}

			// Check health
			healthStatus, err := m.CheckHealth(timeoutCtx, currentResource)
			if err != nil {
				// Continue polling on health check errors
				continue
			}

			// Check if healthy
			if healthStatus.Status == types.HealthStatusHealthy {
				return nil
			}
		}
	}
}

// CollectMetrics collects metrics for a resource by delegating to the adapter
func (m *DefaultMonitor) CollectMetrics(ctx context.Context, resource types.Resource) (types.Metrics, error) {
	// Delegate to the environment adapter
	metrics, err := m.adapter.GetMetrics(ctx, resource.ID)
	if err != nil {
		return types.Metrics{}, fmt.Errorf("failed to collect metrics for resource %s: %w", resource.ID, err)
	}

	return metrics, nil
}

// WatchEvents watches for events on a resource
func (m *DefaultMonitor) WatchEvents(ctx context.Context, resource types.Resource) (<-chan types.Event, error) {
	// Create a channel for events
	eventChan := make(chan types.Event, 100)

	// Watch resource events from the adapter
	resourceEventChan, err := m.adapter.WatchResources(ctx, types.ResourceFilter{
		Names: []string{resource.Name},
	})
	if err != nil {
		close(eventChan)
		return nil, fmt.Errorf("failed to watch events for resource %s: %w", resource.ID, err)
	}

	// Convert ResourceEvent to Event
	go func() {
		defer close(eventChan)

		for {
			select {
			case <-ctx.Done():
				return

			case resourceEvent, ok := <-resourceEventChan:
				if !ok {
					return
				}

				// Convert ResourceEvent to Event
				event := types.Event{
					ID:        fmt.Sprintf("event-%d", time.Now().UnixNano()),
					Type:      string(resourceEvent.Type),
					Source:    "monitor",
					Timestamp: time.Now(),
					Resource:  resourceEvent.Resource,
					Data: map[string]interface{}{
						"event_type": resourceEvent.Type,
					},
					Metadata: make(map[string]string),
				}

				select {
				case eventChan <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return eventChan, nil
}

// RegisterHealthCheckStrategy registers a health check strategy
func (m *DefaultMonitor) RegisterHealthCheckStrategy(name string, strategy types.HealthCheckStrategy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "" {
		return fmt.Errorf("strategy name cannot be empty")
	}

	if strategy == nil {
		return fmt.Errorf("strategy cannot be nil")
	}

	m.strategies[name] = strategy
	return nil
}

// GetHealthCheckStrategy retrieves a health check strategy by name
func (m *DefaultMonitor) GetHealthCheckStrategy(name string) (types.HealthCheckStrategy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return nil, fmt.Errorf("health check strategy not found: %s", name)
	}

	return strategy, nil
}

// SetPollInterval sets the default polling interval for condition waiting
func (m *DefaultMonitor) SetPollInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defaultPollInterval = interval
}

// isConditionMet checks if a condition is met on a resource
func (m *DefaultMonitor) isConditionMet(resource types.Resource, targetCondition types.Condition) bool {
	// Check if the resource has the condition
	for _, condition := range resource.Status.Conditions {
		if condition.Type == targetCondition.Type {
			// Check if the status matches
			if targetCondition.Status != "" && condition.Status != targetCondition.Status {
				return false
			}

			// Check if the reason matches (if specified)
			if targetCondition.Reason != "" && condition.Reason != targetCondition.Reason {
				return false
			}

			// Condition is met
			return true
		}
	}

	// Condition not found
	return false
}
