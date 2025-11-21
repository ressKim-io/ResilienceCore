// Package types provides property-based tests for the Monitor interface
package types

import (
	"context"
	"fmt"
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
			if _, err := monitor.CheckHealth(ctx, resource); err != nil {
				// Error is acceptable for some strategies (e.g., exec not fully implemented)
				// but we still want to verify the strategy was attempted
			}

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
				name:           strategyName,
				alwaysHealthy:  true,
				customMessage:  "Custom health check passed",
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
