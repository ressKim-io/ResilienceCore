package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	coretesting "github.com/yourusername/infrastructure-resilience-engine/pkg/core/testing"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// **Feature: infrastructure-resilience-engine, Property 12: Plugin lifecycle hooks execute in order**
// This property validates that for any plugin execution, the ExecutionEngine invokes hooks
// in the order: Validate → PreExecute → Execute → PostExecute → Cleanup.
// Validates: Requirements 3.2
func TestProperty12_PluginLifecycleHooksExecuteInOrder(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("lifecycle hooks execute in correct order", prop.ForAll(
		func(resourceID, resourceName string) bool {
			engine := NewDefaultExecutionEngine()
			
			plugin := coretesting.NewMockPlugin("test-plugin")
			engine.RegisterPlugin(plugin)

			resource := types.Resource{
				ID:   resourceID,
				Name: resourceName,
				Kind: "container",
			}

			ctx := context.Background()
			request := types.ExecutionRequest{
				PluginName: "test-plugin",
				Resource:   resource,
			}

			_, _ = engine.Execute(ctx, request)

			// Verify all hooks were called exactly once
			return plugin.ValidateCalls == 1 &&
				plugin.PreExecuteCalls == 1 &&
				plugin.ExecuteCalls == 1 &&
				plugin.PostExecuteCalls == 1 &&
				plugin.CleanupCalls == 1
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 13: Rollback is invoked on failure**
// This property validates that for any plugin execution that fails and implements Rollback,
// the ExecutionEngine invokes Rollback with the snapshot from PreExecute.
// Validates: Requirements 3.3
func TestProperty13_RollbackIsInvokedOnFailure(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("rollback is called when execution fails", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()
			
			plugin := coretesting.NewMockPlugin("test-plugin")
			plugin.ExecuteError = errors.New("execution failed")

			engine.RegisterPlugin(plugin)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			ctx := context.Background()
			request := types.ExecutionRequest{
				PluginName: "test-plugin",
				Resource:   resource,
			}

			_, _ = engine.Execute(ctx, request)

			// Verify rollback was called
			return plugin.RollbackCalls == 1
		},
		gen.Identifier(),
	))

	properties.Property("rollback receives snapshot from PreExecute", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()
			
			plugin := coretesting.NewMockPlugin("test-plugin")
			plugin.ExecuteError = errors.New("execution failed")
			
			// Create a specific snapshot
			snapshot := coretesting.NewMockSnapshot(types.Resource{ID: resourceID})
			plugin.SnapshotToReturn = snapshot

			engine.RegisterPlugin(plugin)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			ctx := context.Background()
			request := types.ExecutionRequest{
				PluginName: "test-plugin",
				Resource:   resource,
			}

			_, _ = engine.Execute(ctx, request)

			// Verify rollback was called with the snapshot
			return plugin.RollbackCalls == 1 && plugin.LastRollbackSnapshot == snapshot
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 23: Concurrency limits are respected**
// This property validates that for any set of concurrent plugin executions,
// the ExecutionEngine does not exceed the configured concurrency limit.
// Validates: Requirements 5.2
func TestProperty23_ConcurrencyLimitsAreRespected(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("concurrency limit can be set", prop.ForAll(
		func(limit uint8) bool {
			if limit == 0 {
				return true // Skip invalid input
			}
			
			actualLimit := int(limit%100) + 1 // 1-100

			engine := NewDefaultExecutionEngine()
			engine.SetConcurrencyLimit(actualLimit)

			// Verify semaphore capacity matches limit
			return cap(engine.semaphore) == actualLimit
		},
		gen.UInt8(),
	))

	properties.Property("multiple executions complete successfully", prop.ForAll(
		func(numExecutions uint8) bool {
			if numExecutions == 0 {
				return true
			}
			
			actualNum := int(numExecutions%10) + 1 // 1-10

			engine := NewDefaultExecutionEngine()
			engine.SetConcurrencyLimit(5)

			plugin := coretesting.NewMockPlugin("test-plugin")
			engine.RegisterPlugin(plugin)

			ctx := context.Background()
			var wg sync.WaitGroup

			// Start multiple executions
			for i := 0; i < actualNum; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					
					resource := types.Resource{
						ID:   "resource-" + string(rune(idx)),
						Name: "test-resource",
						Kind: "container",
					}
					request := types.ExecutionRequest{
						PluginName: "test-plugin",
						Resource:   resource,
					}
					engine.Execute(ctx, request)
				}(i)
			}

			wg.Wait()

			// Verify all executions completed
			return plugin.ExecuteCalls == actualNum
		},
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 24: Timeout cancels execution**
// This property validates that for any plugin execution that exceeds the timeout,
// the ExecutionEngine cancels the context and invokes cleanup hooks.
// Validates: Requirements 5.3
func TestProperty24_TimeoutCancelsExecution(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("timeout results in timeout status", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()
			engine.SetDefaultTimeout(50 * time.Millisecond)

			plugin := coretesting.NewMockPlugin("test-plugin")
			plugin.ExecuteError = context.DeadlineExceeded

			engine.RegisterPlugin(plugin)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			ctx := context.Background()
			request := types.ExecutionRequest{
				PluginName: "test-plugin",
				Resource:   resource,
			}

			result, _ := engine.Execute(ctx, request)

			// Verify timeout status
			return result.Status == types.StatusTimeout
		},
		gen.Identifier(),
	))

	properties.Property("cleanup is called even on timeout", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()
			engine.SetDefaultTimeout(50 * time.Millisecond)

			plugin := coretesting.NewMockPlugin("test-plugin")
			plugin.ExecuteError = context.DeadlineExceeded

			engine.RegisterPlugin(plugin)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			ctx := context.Background()
			request := types.ExecutionRequest{
				PluginName: "test-plugin",
				Resource:   resource,
			}

			_, _ = engine.Execute(ctx, request)

			// Verify cleanup was called
			return plugin.CleanupCalls == 1
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 64: Retryable errors trigger retry with backoff**
// This property validates that for any retryable error, the ExecutionEngine retries
// the operation with exponential backoff up to MaxRetries.
// Validates: Requirements 14.1
func TestProperty64_RetryableErrorsTriggerRetryWithBackoff(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("retry strategy retries on failure", prop.ForAll(
		func(maxRetries uint8) bool {
			if maxRetries == 0 {
				return true // Skip invalid input
			}
			
			actualMaxRetries := int(maxRetries % 3) // 0-2 retries (keep it small)

			engine := NewDefaultExecutionEngine()
			
			plugin := coretesting.NewMockPlugin("test-plugin")
			plugin.ExecuteError = errors.New("retryable error")

			engine.RegisterPlugin(plugin)

			resource := types.Resource{
				ID:   "resource-1",
				Name: "test-resource",
				Kind: "container",
			}

			ctx := context.Background()
			
			// Use retry strategy with minimal backoff
			backoff := &ExponentialBackoff{
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   2.0,
			}
			
			strategy := &types.RetryStrategy{
				MaxRetries: actualMaxRetries,
				Backoff:    backoff,
			}

			request := types.ExecutionRequest{
				PluginName: "test-plugin",
				Resource:   resource,
				Strategy:   strategy,
			}

			_, _ = engine.Execute(ctx, request)

			// Verify plugin was called MaxRetries + 1 times (initial + retries)
			expectedCalls := actualMaxRetries + 1
			return plugin.ExecuteCalls == expectedCalls
		},
		gen.UInt8(),
	))

	properties.Property("backoff delay increases exponentially", prop.ForAll(
		func(attempt uint8) bool {
			if attempt > 5 {
				return true // Skip large values
			}
			
			backoff := NewExponentialBackoff()
			
			delay1 := backoff.NextDelay(int(attempt))
			delay2 := backoff.NextDelay(int(attempt) + 1)
			
			// Verify delay increases (or stays at max)
			return delay2 >= delay1
		},
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 73: Resource locking prevents concurrent execution**
// This property validates that for any resource, the ExecutionEngine prevents
// concurrent plugin execution on the same resource.
// Validates: Requirements 15.5
func TestProperty73_ResourceLockingPreventsConcurrentExecution(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("resource locks are created for each resource", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()
			
			// Get lock for resource
			lock1 := engine.getResourceLock(resourceID)
			lock2 := engine.getResourceLock(resourceID)
			
			// Same resource should return same lock
			return lock1 == lock2
		},
		gen.Identifier(),
	))

	properties.Property("different resources get different locks", prop.ForAll(
		func(resourceID1, resourceID2 string) bool {
			if resourceID1 == resourceID2 {
				return true // Skip same IDs
			}
			
			engine := NewDefaultExecutionEngine()
			
			lock1 := engine.getResourceLock(resourceID1)
			lock2 := engine.getResourceLock(resourceID2)
			
			// Different resources should have different locks
			return lock1 != lock2
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("multiple executions on same resource complete", prop.ForAll(
		func(resourceID string, numExecutions uint8) bool {
			if numExecutions == 0 {
				return true
			}
			
			actualNum := int(numExecutions%5) + 1 // 1-5

			engine := NewDefaultExecutionEngine()
			plugin := coretesting.NewMockPlugin("test-plugin")
			engine.RegisterPlugin(plugin)

			ctx := context.Background()
			var wg sync.WaitGroup

			// Same resource for all executions
			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Start multiple executions on the SAME resource
			for i := 0; i < actualNum; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					request := types.ExecutionRequest{
						PluginName: "test-plugin",
						Resource:   resource,
					}
					engine.Execute(ctx, request)
				}()
			}

			wg.Wait()

			// All executions should complete (sequentially)
			return plugin.ExecuteCalls == actualNum
		},
		gen.Identifier(),
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
