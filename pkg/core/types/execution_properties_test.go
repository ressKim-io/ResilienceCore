// Package types provides property-based tests for the ExecutionEngine interface
package types

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty22_ExecutionStrategyIsApplied verifies that the ExecutionEngine
// applies the specified ExecutionStrategy during execution
// Feature: infrastructure-resilience-engine, Property 22: Execution strategy is applied
// Validates: Requirements 5.1
func TestProperty22_ExecutionStrategyIsApplied(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("execution strategy is applied", prop.ForAll(
		func(strategyName string, pluginName string) bool {
			// Create a mock execution engine
			engine := NewMockExecutionEngine()

			// Create a mock plugin
			plugin := &MockPluginWithFuncs{
				metadata: PluginMetadata{
					Name:    pluginName,
					Version: "1.0.0",
				},
			}

			// Register the plugin
			if err := engine.RegisterPlugin(plugin); err != nil {
				return false
			}

			// Create a mock strategy that tracks if it was called
			strategy := &MockStrategy{name: strategyName}

			// Create execution request with the strategy
			request := ExecutionRequest{
				PluginName: pluginName,
				Resource: Resource{
					ID:   "test-resource",
					Name: "test",
					Kind: "container",
				},
				Config:   &MockPluginConfig{},
				Strategy: strategy,
			}

			// Execute
			ctx := context.Background()
			_, err := engine.Execute(ctx, request)
			if err != nil {
				return false
			}

			// Verify the strategy was called
			return strategy.called
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty23_ConcurrencyLimitsAreRespected verifies that the ExecutionEngine
// respects concurrency limits
// Feature: infrastructure-resilience-engine, Property 23: Concurrency limits are respected
// Validates: Requirements 5.2
func TestProperty23_ConcurrencyLimitsAreRespected(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("concurrency limits are respected", prop.ForAll(
		func(concurrencyLimit int, numExecutions int) bool {
			// Ensure valid test parameters
			if concurrencyLimit < 1 || concurrencyLimit > 100 {
				concurrencyLimit = 10
			}
			if numExecutions < 1 || numExecutions > 100 {
				numExecutions = 20
			}

			// Create a mock execution engine
			engine := NewMockExecutionEngine()
			engine.SetConcurrencyLimit(concurrencyLimit)

			// Track concurrent executions
			var currentConcurrent int32
			var maxConcurrent int32
			var mu sync.Mutex

			// Create a plugin that tracks concurrent executions
			plugin := &MockPluginWithFuncs{
				metadata: PluginMetadata{
					Name:    "test-plugin",
					Version: "1.0.0",
				},
				executeFunc: func(ctx PluginContext, resource Resource) error {
					// Increment concurrent counter
					current := atomic.AddInt32(&currentConcurrent, 1)

					// Update max if needed
					mu.Lock()
					if current > maxConcurrent {
						maxConcurrent = current
					}
					mu.Unlock()

					// Simulate work
					time.Sleep(10 * time.Millisecond)

					// Decrement concurrent counter
					atomic.AddInt32(&currentConcurrent, -1)
					return nil
				},
			}

			// Register the plugin
			if err := engine.RegisterPlugin(plugin); err != nil {
				return false
			}

			// Execute multiple times concurrently
			var wg sync.WaitGroup
			for i := 0; i < numExecutions; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					request := ExecutionRequest{
						PluginName: "test-plugin",
						Resource: Resource{
							ID:   "test-resource",
							Name: "test",
							Kind: "container",
						},
						Config:   &MockPluginConfig{},
						Strategy: &SimpleStrategy{},
					}
					ctx := context.Background()
					_, _ = engine.Execute(ctx, request)
				}()
			}

			wg.Wait()

			// Verify max concurrent executions did not exceed limit
			return int(maxConcurrent) <= concurrencyLimit
		},
		gen.IntRange(1, 20),
		gen.IntRange(10, 50),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty24_TimeoutCancelsExecution verifies that the ExecutionEngine
// cancels execution when timeout is exceeded
// Feature: infrastructure-resilience-engine, Property 24: Timeout cancels execution
// Validates: Requirements 5.3
func TestProperty24_TimeoutCancelsExecution(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("timeout cancels execution", prop.ForAll(
		func(timeoutMs int, executionMs int) bool {
			// Ensure valid test parameters
			if timeoutMs < 10 || timeoutMs > 1000 {
				timeoutMs = 100
			}
			if executionMs < 10 || executionMs > 2000 {
				executionMs = 500
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond
			executionTime := time.Duration(executionMs) * time.Millisecond

			// Add a buffer to account for timing variations
			// Only test cases where there's a clear difference
			// We need a larger buffer to account for goroutine scheduling and other timing variations
			if executionTime > timeout && executionTime < timeout+100*time.Millisecond {
				// Too close to timeout boundary, skip this test case
				return true
			}

			// Create a mock execution engine
			engine := NewMockExecutionEngine()
			engine.SetDefaultTimeout(timeout)

			// Track if cleanup was called
			cleanupCalled := false

			// Create a plugin that takes longer than timeout
			plugin := &MockPluginWithFuncs{
				metadata: PluginMetadata{
					Name:    "slow-plugin",
					Version: "1.0.0",
				},
				executeFunc: func(ctx PluginContext, resource Resource) error {
					// Simulate long-running work
					select {
					case <-time.After(executionTime):
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				},
				cleanupFunc: func(ctx PluginContext, resource Resource) error {
					cleanupCalled = true
					return nil
				},
			}

			// Register the plugin
			if err := engine.RegisterPlugin(plugin); err != nil {
				return false
			}

			// Execute with timeout
			request := ExecutionRequest{
				PluginName: "slow-plugin",
				Resource: Resource{
					ID:   "test-resource",
					Name: "test",
					Kind: "container",
				},
				Config:   &MockPluginConfig{},
				Strategy: &SimpleStrategy{},
			}

			ctx := context.Background()
			result, _ := engine.Execute(ctx, request)

			// If execution time significantly exceeds timeout, verify:
			// 1. Execution was cancelled (status is timeout or canceled)
			// 2. Cleanup was called
			if executionTime > timeout+100*time.Millisecond {
				return (result.Status == StatusTimeout || result.Status == StatusCanceled) && cleanupCalled
			}

			// If execution time is within timeout, it should succeed
			return result.Status == StatusSuccess
		},
		gen.IntRange(50, 200),
		gen.IntRange(50, 400),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ============================================================================
// Mock Implementations for Testing
// ============================================================================

// MockExecutionEngine is a mock implementation of ExecutionEngine for testing
type MockExecutionEngine struct {
	plugins          map[string]Plugin
	concurrencyLimit int
	defaultTimeout   time.Duration
	mu               sync.RWMutex
	semaphore        chan struct{}
}

func NewMockExecutionEngine() *MockExecutionEngine {
	return &MockExecutionEngine{
		plugins:          make(map[string]Plugin),
		concurrencyLimit: 10,
		defaultTimeout:   30 * time.Second,
		semaphore:        make(chan struct{}, 10),
	}
}

func (m *MockExecutionEngine) RegisterPlugin(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[plugin.Metadata().Name] = plugin
	return nil
}

func (m *MockExecutionEngine) UnregisterPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.plugins, name)
	return nil
}

func (m *MockExecutionEngine) GetPlugin(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	plugin, ok := m.plugins[name]
	if !ok {
		return nil, nil
	}
	return plugin, nil
}

func (m *MockExecutionEngine) ListPlugins(filter PluginFilter) ([]PluginMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []PluginMetadata
	for _, plugin := range m.plugins {
		result = append(result, plugin.Metadata())
	}
	return result, nil
}

func (m *MockExecutionEngine) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	// Acquire semaphore for concurrency control
	select {
	case m.semaphore <- struct{}{}:
		defer func() { <-m.semaphore }()
	case <-ctx.Done():
		return ExecutionResult{Status: StatusCanceled}, ctx.Err()
	}

	// Get plugin
	plugin, err := m.GetPlugin(request.PluginName)
	if err != nil || plugin == nil {
		return ExecutionResult{Status: StatusFailed}, err
	}

	// Apply strategy if provided
	if request.Strategy != nil {
		// For strategies, we still need to ensure cleanup is called
		startTime := time.Now()

		// Create context with timeout
		execCtx, cancel := context.WithTimeout(ctx, m.defaultTimeout)
		defer cancel()

		pluginCtx := PluginContext{
			Context:     execCtx,
			ExecutionID: "test-exec-id",
			Timeout:     m.defaultTimeout,
			Env:         &MockAdapter{},
		}

		// Execute via strategy
		result, stratErr := request.Strategy.Execute(execCtx, plugin, request.Resource)

		// Always call cleanup with the same context
		_ = plugin.Cleanup(pluginCtx, request.Resource)

		// Update result timing if needed
		if result.StartTime.IsZero() {
			result.StartTime = startTime
		}
		if result.EndTime.IsZero() {
			result.EndTime = time.Now()
		}
		if result.Duration == 0 {
			result.Duration = result.EndTime.Sub(result.StartTime)
		}

		return result, stratErr
	}

	// Default execution
	startTime := time.Now()

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, m.defaultTimeout)
	defer cancel()

	pluginCtx := PluginContext{
		Context:     execCtx,
		ExecutionID: "test-exec-id",
		Timeout:     m.defaultTimeout,
		Env:         &MockAdapter{},
	}

	// Execute plugin
	err = plugin.Execute(pluginCtx, request.Resource)

	// Always call cleanup
	_ = plugin.Cleanup(pluginCtx, request.Resource)

	endTime := time.Now()
	status := StatusSuccess
	if err != nil {
		if err == context.DeadlineExceeded {
			status = StatusTimeout
		} else if err == context.Canceled {
			status = StatusCanceled
		} else {
			status = StatusFailed
		}
	}

	return ExecutionResult{
		Status:    status,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Error:     err,
	}, nil
}

func (m *MockExecutionEngine) ExecuteAsync(ctx context.Context, request ExecutionRequest) (Future, error) {
	// Not implemented for basic tests
	return nil, nil
}

func (m *MockExecutionEngine) ExecuteWorkflow(ctx context.Context, workflow Workflow) (WorkflowResult, error) {
	// Not implemented for basic tests
	return WorkflowResult{}, nil
}

func (m *MockExecutionEngine) Cancel(executionID string) error {
	return nil
}

func (m *MockExecutionEngine) GetStatus(executionID string) (ExecutionStatus, error) {
	return StatusSuccess, nil
}

func (m *MockExecutionEngine) SetConcurrencyLimit(limit int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.concurrencyLimit = limit
	m.semaphore = make(chan struct{}, limit)
}

func (m *MockExecutionEngine) SetDefaultTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultTimeout = timeout
}

// MockStrategy is a mock implementation of ExecutionStrategy for testing
type MockStrategy struct {
	name   string
	called bool
}

func (m *MockStrategy) Execute(ctx context.Context, plugin Plugin, resource Resource) (ExecutionResult, error) {
	m.called = true

	startTime := time.Now()

	pluginCtx := PluginContext{
		Context:     ctx,
		ExecutionID: "test-exec-id",
		Timeout:     30 * time.Second,
		Env:         &MockAdapter{},
	}

	err := plugin.Execute(pluginCtx, resource)

	endTime := time.Now()
	status := StatusSuccess
	if err != nil {
		status = StatusFailed
	}

	return ExecutionResult{
		Status:    status,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Error:     err,
	}, nil
}

func (m *MockStrategy) Name() string {
	return m.name
}

// MockPluginConfig is a mock implementation of PluginConfig for testing
type MockPluginConfig struct{}

func (m *MockPluginConfig) Validate() error {
	return nil
}

// MockPluginWithFuncs extends MockPlugin with custom execute and cleanup functions
type MockPluginWithFuncs struct {
	metadata    PluginMetadata
	executeFunc func(ctx PluginContext, resource Resource) error
	cleanupFunc func(ctx PluginContext, resource Resource) error
}

func (m *MockPluginWithFuncs) Metadata() PluginMetadata {
	return m.metadata
}

func (m *MockPluginWithFuncs) Initialize(config PluginConfig) error {
	return nil
}

func (m *MockPluginWithFuncs) Validate(ctx PluginContext, resource Resource) error {
	return nil
}

func (m *MockPluginWithFuncs) PreExecute(ctx PluginContext, resource Resource) (Snapshot, error) {
	return &MockSnapshot{}, nil
}

func (m *MockPluginWithFuncs) Execute(ctx PluginContext, resource Resource) error {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, resource)
	}
	return nil
}

func (m *MockPluginWithFuncs) PostExecute(ctx PluginContext, resource Resource, result ExecutionResult) error {
	return nil
}

func (m *MockPluginWithFuncs) Cleanup(ctx PluginContext, resource Resource) error {
	if m.cleanupFunc != nil {
		return m.cleanupFunc(ctx, resource)
	}
	return nil
}

func (m *MockPluginWithFuncs) Rollback(ctx PluginContext, resource Resource, snapshot Snapshot) error {
	return nil
}
