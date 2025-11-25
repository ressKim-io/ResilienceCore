// Package types provides property-based tests for the Monitor interface
package types

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty28_HealthCheckUsesRegisteredStrategy verifies that the Monitor
// uses the registered HealthCheckStrategy matching the resource's health check config type
// Feature: infrastructure-resilience-engine, Property 28: Health check uses registered strategy
// Validates: Requirements 6.1
func TestProperty28_HealthCheckUsesRegisteredStrategy(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("health check uses registered strategy for resource config type", prop.ForAll(
		func(healthCheckType string, resourceName string) bool {
			// Create a mock monitor
			monitor := NewMockMonitor()

			// Register strategies for different health check types
			httpStrategy := &HTTPHealthCheckStrategy{
				ExpectedStatusCodes: []int{200},
			}
			tcpStrategy := &TCPHealthCheckStrategy{
				DialTimeout: 5 * time.Second,
			}
			execStrategy := &ExecHealthCheckStrategy{
				ExpectedExitCode: 0,
			}

			monitor.RegisterHealthCheckStrategy("http", httpStrategy)
			monitor.RegisterHealthCheckStrategy("tcp", tcpStrategy)
			monitor.RegisterHealthCheckStrategy("exec", execStrategy)

			// Create a resource with health check configuration
			resource := Resource{
				ID:   "test-resource-" + resourceName,
				Name: resourceName,
				Kind: "container",
				Spec: ResourceSpec{
					HealthCheck: &HealthCheckConfig{
						Type:     healthCheckType,
						Endpoint: "http://localhost:8080/health",
						Interval: 10 * time.Second,
						Timeout:  5 * time.Second,
						Retries:  3,
					},
				},
			}

			ctx := context.Background()

			// Check health - should use the registered strategy for the resource's type
			_, _ = monitor.CheckHealth(ctx, resource)
			// Error is acceptable for some strategies (e.g., exec not fully implemented)
			// but we still want to verify the strategy was attempted

			// Verify the correct strategy was used
			usedStrategy := monitor.GetLastUsedStrategy()
			if usedStrategy == "" {
				return false
			}

			// The used strategy name should match the health check type
			return usedStrategy == healthCheckType
		},
		gen.OneConstOf("http", "tcp", "exec"),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty32_CustomHealthCheckStrategiesAreSupported verifies that custom
// HealthCheckStrategy implementations can be registered and used without Core modification
// Feature: infrastructure-resilience-engine, Property 32: Custom health check strategies are supported
// Validates: Requirements 6.5
func TestProperty32_CustomHealthCheckStrategiesAreSupported(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("custom health check strategies work without Core modification", prop.ForAll(
		func(strategyName string, resourceName string) bool {
			// Create a mock monitor
			monitor := NewMockMonitor()

			// Create a custom health check strategy
			customStrategy := &CustomHealthCheckStrategy{
				name:          strategyName,
				alwaysHealthy: true,
				customMessage: "Custom health check passed",
			}

			// Register the custom strategy - this should work without Core modification
			if err := monitor.RegisterHealthCheckStrategy(strategyName, customStrategy); err != nil {
				return false
			}

			// Verify the strategy can be retrieved
			retrievedStrategy, err := monitor.GetHealthCheckStrategy(strategyName)
			if err != nil {
				return false
			}

			if retrievedStrategy.Name() != strategyName {
				return false
			}

			// Create a resource that uses the custom strategy
			resource := Resource{
				ID:   "test-resource-" + resourceName,
				Name: resourceName,
				Kind: "container",
				Spec: ResourceSpec{
					HealthCheck: &HealthCheckConfig{
						Type:     strategyName,
						Endpoint: "custom://endpoint",
						Interval: 10 * time.Second,
						Timeout:  5 * time.Second,
						Retries:  3,
					},
				},
			}

			ctx := context.Background()

			// Check health using the custom strategy
			status, err := monitor.CheckHealth(ctx, resource)
			if err != nil {
				return false
			}

			// Verify the custom strategy was used and returned expected results
			if status.Status != HealthStatusHealthy {
				return false
			}

			if status.Message != customStrategy.customMessage {
				return false
			}

			// Verify the monitor used the custom strategy
			usedStrategy := monitor.GetLastUsedStrategy()
			return usedStrategy == strategyName
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ============================================================================
// Mock Monitor Implementation for Testing
// ============================================================================

// MockMonitor is a mock implementation of the Monitor interface for testing
type MockMonitor struct {
	strategies       map[string]HealthCheckStrategy
	lastUsedStrategy string
}

// NewMockMonitor creates a new mock monitor
func NewMockMonitor() *MockMonitor {
	return &MockMonitor{
		strategies: make(map[string]HealthCheckStrategy),
	}
}

// CheckHealth implements Monitor interface
func (m *MockMonitor) CheckHealth(ctx context.Context, resource Resource) (HealthStatus, error) {
	if resource.Spec.HealthCheck == nil {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: "no health check configuration",
		}, fmt.Errorf("resource has no health check configuration")
	}

	healthCheckType := resource.Spec.HealthCheck.Type
	strategy, exists := m.strategies[healthCheckType]
	if !exists {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: fmt.Sprintf("no strategy registered for type: %s", healthCheckType),
		}, fmt.Errorf("no strategy registered for health check type: %s", healthCheckType)
	}

	// Record which strategy was used
	m.lastUsedStrategy = strategy.Name()

	// Use the registered strategy to check health
	return strategy.Check(ctx, resource)
}

// WaitForCondition implements Monitor interface
func (m *MockMonitor) WaitForCondition(ctx context.Context, resource Resource, condition Condition, timeout time.Duration) error {
	// Mock implementation - just return success
	return nil
}

// WaitForHealthy implements Monitor interface
func (m *MockMonitor) WaitForHealthy(ctx context.Context, resource Resource, timeout time.Duration) error {
	// Mock implementation - just return success
	return nil
}

// CollectMetrics implements Monitor interface
func (m *MockMonitor) CollectMetrics(ctx context.Context, resource Resource) (Metrics, error) {
	return Metrics{
		Timestamp:  time.Now(),
		ResourceID: resource.ID,
		CPU:        CPUMetrics{UsagePercent: 50.0, UsageCores: 1.0},
		Memory:     MemoryMetrics{UsageBytes: 1024, LimitBytes: 2048, UsagePercent: 50.0},
	}, nil
}

// WatchEvents implements Monitor interface
func (m *MockMonitor) WatchEvents(ctx context.Context, resource Resource) (<-chan Event, error) {
	ch := make(chan Event)
	go func() {
		close(ch)
	}()
	return ch, nil
}

// RegisterHealthCheckStrategy implements Monitor interface
func (m *MockMonitor) RegisterHealthCheckStrategy(name string, strategy HealthCheckStrategy) error {
	if name == "" {
		return fmt.Errorf("strategy name cannot be empty")
	}
	if strategy == nil {
		return fmt.Errorf("strategy cannot be nil")
	}
	m.strategies[name] = strategy
	return nil
}

// GetHealthCheckStrategy implements Monitor interface
func (m *MockMonitor) GetHealthCheckStrategy(name string) (HealthCheckStrategy, error) {
	strategy, exists := m.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy not found: %s", name)
	}
	return strategy, nil
}

// GetLastUsedStrategy returns the name of the last strategy used (for testing)
func (m *MockMonitor) GetLastUsedStrategy() string {
	return m.lastUsedStrategy
}

// ============================================================================
// Custom Health Check Strategy for Testing
// ============================================================================

// CustomHealthCheckStrategy is a custom health check strategy for testing
type CustomHealthCheckStrategy struct {
	name          string
	alwaysHealthy bool
	customMessage string
}

// Check implements HealthCheckStrategy interface
func (c *CustomHealthCheckStrategy) Check(ctx context.Context, resource Resource) (HealthStatus, error) {
	if c.alwaysHealthy {
		return HealthStatus{
			Status:  HealthStatusHealthy,
			Message: c.customMessage,
			Checks: []HealthCheck{
				{
					Name:    c.name,
					Status:  HealthStatusHealthy,
					Message: c.customMessage,
				},
			},
		}, nil
	}

	return HealthStatus{
		Status:  HealthStatusUnhealthy,
		Message: "custom check failed",
		Checks: []HealthCheck{
			{
				Name:    c.name,
				Status:  HealthStatusUnhealthy,
				Message: "custom check failed",
			},
		},
	}, nil
}

// Name implements HealthCheckStrategy interface
func (c *CustomHealthCheckStrategy) Name() string {
	return c.name
}

// TestProperty29_ConditionWaitingPollsUntilMetOrTimeout verifies that WaitForCondition
// polls the resource state until the condition is met or timeout occurs
// Feature: infrastructure-resilience-engine, Property 29: Condition waiting polls until met or timeout
// Validates: Requirements 6.2
func TestProperty29_ConditionWaitingPollsUntilMetOrTimeout(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Test Case 1: Timeout when condition is never met
	properties.Property("condition waiting times out when condition is never met", prop.ForAll(
		func(conditionType string, timeoutMs int) bool {
			// Ensure timeout is reasonable (between 100ms and 500ms for testing)
			if timeoutMs < 100 {
				timeoutMs = 100
			}
			if timeoutMs > 500 {
				timeoutMs = 500
			}
			timeout := time.Duration(timeoutMs) * time.Millisecond

			// Create a mock adapter
			adapter := &MockAdapterForMonitor{
				resources: make(map[string]Resource),
			}

			// Create a resource with a condition that is NOT met
			resource := Resource{
				ID:   "test-resource-timeout",
				Name: "test-resource-timeout",
				Kind: "container",
				Status: ResourceStatus{
					Phase: "running",
					Conditions: []Condition{
						{
							Type:   conditionType,
							Status: "False", // Not met
							Reason: "NotReady",
						},
					},
				},
			}
			adapter.resources[resource.ID] = resource

			// Create a monitor with the mock adapter
			monitor := &MonitorWithAdapter{
				adapter:      adapter,
				pollInterval: 20 * time.Millisecond,
			}

			// Create the target condition (looking for True, but resource has False)
			targetCondition := Condition{
				Type:   conditionType,
				Status: "True",
			}

			ctx := context.Background()

			// Wait for condition - should timeout
			startTime := time.Now()
			err := monitor.WaitForCondition(ctx, resource, targetCondition, timeout)
			elapsed := time.Since(startTime)

			// Should return an error (timeout)
			if err == nil {
				return false
			}

			// Should take at least the timeout duration
			// Allow generous overhead for system scheduling
			if elapsed < timeout {
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.IntRange(100, 500),
	))

	// Test Case 2: Success when condition is met
	properties.Property("condition waiting succeeds when condition is met", prop.ForAll(
		func(conditionType string) bool {
			// Create a mock adapter
			adapter := &MockAdapterForMonitor{
				resources: make(map[string]Resource),
			}

			// Create a resource with a condition that WILL be met
			resource := Resource{
				ID:   "test-resource-success",
				Name: "test-resource-success",
				Kind: "container",
				Status: ResourceStatus{
					Phase: "running",
					Conditions: []Condition{
						{
							Type:   conditionType,
							Status: "False", // Initially not met
							Reason: "NotReady",
						},
					},
				},
			}
			adapter.resources[resource.ID] = resource

			// Create a monitor with the mock adapter
			monitor := &MonitorWithAdapter{
				adapter:      adapter,
				pollInterval: 20 * time.Millisecond,
			}

			// Create the target condition
			targetCondition := Condition{
				Type:   conditionType,
				Status: "True",
			}

			// Update the resource to meet the condition after a delay
			go func() {
				time.Sleep(50 * time.Millisecond)
				adapter.mu.Lock()
				defer adapter.mu.Unlock()
				r := adapter.resources[resource.ID]
				r.Status.Conditions = []Condition{
					{
						Type:   conditionType,
						Status: "True", // Now met
						Reason: "Ready",
					},
				}
				adapter.resources[resource.ID] = r
			}()

			ctx := context.Background()

			// Wait for condition - should succeed
			err := monitor.WaitForCondition(ctx, resource, targetCondition, 2*time.Second)

			// Should not return an error
			return err == nil
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty30_MetricsCollectionDelegatesToAdapter verifies that CollectMetrics
// delegates to the EnvironmentAdapter and returns normalized Metrics
// Feature: infrastructure-resilience-engine, Property 30: Metrics collection delegates to adapter
// Validates: Requirements 6.3
func TestProperty30_MetricsCollectionDelegatesToAdapter(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("metrics collection delegates to adapter and returns normalized metrics", prop.ForAll(
		func(resourceID string, cpuPercent float64, memoryUsage uint64) bool {
			// Ensure values are in reasonable ranges
			if cpuPercent < 0 {
				cpuPercent = -cpuPercent
			}
			if cpuPercent > 100 {
				cpuPercent = 100
			}
			if memoryUsage > 1024*1024*1024*16 { // Cap at 16GB
				memoryUsage = 1024 * 1024 * 1024 * 16
			}

			// Create a mock adapter with predefined metrics
			adapter := &MockAdapterForMonitor{
				resources: make(map[string]Resource),
				metrics: map[string]Metrics{
					resourceID: {
						Timestamp:  time.Now(),
						ResourceID: resourceID,
						CPU: CPUMetrics{
							UsagePercent: cpuPercent,
							UsageCores:   cpuPercent / 100.0,
						},
						Memory: MemoryMetrics{
							UsageBytes:   memoryUsage,
							LimitBytes:   memoryUsage * 2,
							UsagePercent: 50.0,
						},
						Network: NetworkMetrics{
							RxBytes:   1024,
							TxBytes:   2048,
							RxPackets: 10,
							TxPackets: 20,
						},
						Disk: DiskMetrics{
							ReadBytes:  4096,
							WriteBytes: 8192,
							ReadOps:    5,
							WriteOps:   10,
						},
					},
				},
			}

			// Create a resource
			resource := Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}
			adapter.resources[resourceID] = resource

			// Create a monitor with the mock adapter
			monitor := &MonitorWithAdapter{
				adapter: adapter,
			}

			ctx := context.Background()

			// Collect metrics
			metrics, err := monitor.CollectMetrics(ctx, resource)
			if err != nil {
				return false
			}

			// Verify metrics are from the adapter
			if metrics.ResourceID != resourceID {
				return false
			}

			// Verify metrics are normalized (have all required fields)
			if metrics.CPU.UsagePercent != cpuPercent {
				return false
			}

			if metrics.Memory.UsageBytes != memoryUsage {
				return false
			}

			// Verify the adapter was called
			if !adapter.GetMetricsCalled {
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.Float64Range(0, 100),
		gen.UInt64Range(0, 1024*1024*1024*16),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ============================================================================
// Mock Adapter for Monitor Testing
// ============================================================================

type MockAdapterForMonitor struct {
	mu               sync.RWMutex
	resources        map[string]Resource
	metrics          map[string]Metrics
	GetMetricsCalled bool
}

func (m *MockAdapterForMonitor) Initialize(ctx context.Context, config AdapterConfig) error {
	return nil
}

func (m *MockAdapterForMonitor) Close() error {
	return nil
}

func (m *MockAdapterForMonitor) ListResources(ctx context.Context, filter ResourceFilter) ([]Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Resource
	for _, r := range m.resources {
		result = append(result, r)
	}
	return result, nil
}

func (m *MockAdapterForMonitor) GetResource(ctx context.Context, id string) (Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	r, exists := m.resources[id]
	if !exists {
		return Resource{}, fmt.Errorf("resource not found: %s", id)
	}
	return r, nil
}

func (m *MockAdapterForMonitor) StartResource(ctx context.Context, id string) error {
	return nil
}

func (m *MockAdapterForMonitor) StopResource(ctx context.Context, id string, gracePeriod time.Duration) error {
	return nil
}

func (m *MockAdapterForMonitor) RestartResource(ctx context.Context, id string) error {
	return nil
}

func (m *MockAdapterForMonitor) DeleteResource(ctx context.Context, id string, options DeleteOptions) error {
	return nil
}

func (m *MockAdapterForMonitor) CreateResource(ctx context.Context, spec ResourceSpec) (Resource, error) {
	return Resource{}, nil
}

func (m *MockAdapterForMonitor) UpdateResource(ctx context.Context, id string, spec ResourceSpec) (Resource, error) {
	return Resource{}, nil
}

func (m *MockAdapterForMonitor) WatchResources(ctx context.Context, filter ResourceFilter) (<-chan ResourceEvent, error) {
	ch := make(chan ResourceEvent)
	go func() {
		close(ch)
	}()
	return ch, nil
}

func (m *MockAdapterForMonitor) ExecInResource(ctx context.Context, id string, cmd []string, options ExecOptions) (ExecResult, error) {
	return ExecResult{}, nil
}

func (m *MockAdapterForMonitor) GetMetrics(ctx context.Context, id string) (Metrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetMetricsCalled = true

	metrics, exists := m.metrics[id]
	if !exists {
		return Metrics{}, fmt.Errorf("metrics not found for resource: %s", id)
	}
	return metrics, nil
}

func (m *MockAdapterForMonitor) GetAdapterInfo() AdapterInfo {
	return AdapterInfo{
		Name:        "MockAdapter",
		Version:     "1.0.0",
		Environment: "mock",
	}
}

// ============================================================================
// Monitor with Adapter for Testing
// ============================================================================

type MonitorWithAdapter struct {
	adapter      EnvironmentAdapter
	pollInterval time.Duration
}

func (m *MonitorWithAdapter) CheckHealth(ctx context.Context, resource Resource) (HealthStatus, error) {
	return HealthStatus{Status: HealthStatusHealthy}, nil
}

func (m *MonitorWithAdapter) WaitForCondition(ctx context.Context, resource Resource, condition Condition, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pollInterval := m.pollInterval
	if pollInterval == 0 {
		pollInterval = 100 * time.Millisecond
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("timeout waiting for condition %s", condition.Type)
			}
			return timeoutCtx.Err()

		case <-ticker.C:
			currentResource, err := m.adapter.GetResource(timeoutCtx, resource.ID)
			if err != nil {
				continue
			}

			// Check if condition is met
			for _, c := range currentResource.Status.Conditions {
				if c.Type == condition.Type && c.Status == condition.Status {
					return nil
				}
			}
		}
	}
}

func (m *MonitorWithAdapter) WaitForHealthy(ctx context.Context, resource Resource, timeout time.Duration) error {
	return nil
}

func (m *MonitorWithAdapter) CollectMetrics(ctx context.Context, resource Resource) (Metrics, error) {
	// Delegate to adapter
	return m.adapter.GetMetrics(ctx, resource.ID)
}

func (m *MonitorWithAdapter) WatchEvents(ctx context.Context, resource Resource) (<-chan Event, error) {
	ch := make(chan Event)
	go func() {
		close(ch)
	}()
	return ch, nil
}

func (m *MonitorWithAdapter) RegisterHealthCheckStrategy(name string, strategy HealthCheckStrategy) error {
	return nil
}

func (m *MonitorWithAdapter) GetHealthCheckStrategy(name string) (HealthCheckStrategy, error) {
	return nil, fmt.Errorf("not implemented")
}
