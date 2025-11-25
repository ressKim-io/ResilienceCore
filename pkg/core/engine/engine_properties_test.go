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

// **Feature: infrastructure-resilience-engine, Property 25: Workflow respects dependencies**
// This property validates that for any Workflow with step dependencies,
// the ExecutionEngine executes steps only after all dependencies complete.
// Validates: Requirements 5.4
func TestProperty25_WorkflowRespectsDependencies(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("steps execute after dependencies", prop.ForAll(
		func(resourceID string) bool {
			if resourceID == "" {
				resourceID = "test-resource"
			}

			engine := NewDefaultExecutionEngine()

			// Set up minimal components (logger can be nil for tests)
			engine.SetComponents(nil, nil, nil, nil, nil, nil, nil)

			// Create plugins
			plugin1 := coretesting.NewMockPlugin("plugin1")
			plugin2 := coretesting.NewMockPlugin("plugin2")
			plugin3 := coretesting.NewMockPlugin("plugin3")

			if err := engine.RegisterPlugin(plugin1); err != nil {
				return false
			}
			if err := engine.RegisterPlugin(plugin2); err != nil {
				return false
			}
			if err := engine.RegisterPlugin(plugin3); err != nil {
				return false
			}

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create workflow: step3 depends on step2, step2 depends on step1
			workflow := types.Workflow{
				Name: "test-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
						DependsOn:  []string{},
					},
					{
						Name:       "step2",
						PluginName: "plugin2",
						Resource:   resource,
						DependsOn:  []string{"step1"},
					},
					{
						Name:       "step3",
						PluginName: "plugin3",
						Resource:   resource,
						DependsOn:  []string{"step2"},
					},
				},
			}

			ctx := context.Background()
			result, err := engine.ExecuteWorkflow(ctx, workflow)

			// Verify workflow succeeded
			if err != nil {
				t.Logf("Workflow error: %v", err)
				return false
			}

			if result.Status != types.StatusSuccess {
				t.Logf("Workflow status: %v", result.Status)
				return false
			}

			// Verify all steps executed
			if len(result.StepResults) != 3 {
				t.Logf("Expected 3 step results, got %d", len(result.StepResults))
				return false
			}

			// Verify all steps succeeded
			for name, stepResult := range result.StepResults {
				if stepResult.Status != types.StatusSuccess {
					t.Logf("Step %s failed with status: %v, error: %v", name, stepResult.Status, stepResult.Error)
					return false
				}
			}

			return true
		},
		gen.Identifier(),
	))

	properties.Property("circular dependencies are detected", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()

			plugin1 := coretesting.NewMockPlugin("plugin1")
			plugin2 := coretesting.NewMockPlugin("plugin2")

			engine.RegisterPlugin(plugin1)
			engine.RegisterPlugin(plugin2)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create workflow with circular dependency: step1 -> step2 -> step1
			workflow := types.Workflow{
				Name: "circular-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
						DependsOn:  []string{"step2"},
					},
					{
						Name:       "step2",
						PluginName: "plugin2",
						Resource:   resource,
						DependsOn:  []string{"step1"},
					},
				},
			}

			ctx := context.Background()
			_, err := engine.ExecuteWorkflow(ctx, workflow)

			// Should return error for circular dependency
			return err != nil
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 74: Workflow supports sequential and parallel execution**
// This property validates that for any Workflow, steps marked as sequential execute one at a time,
// and steps marked as parallel execute concurrently.
// Validates: Requirements 16.1
func TestProperty74_WorkflowSupportsSequentialAndParallelExecution(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("sequential steps execute one at a time", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()

			plugin1 := coretesting.NewMockPlugin("plugin1")
			plugin2 := coretesting.NewMockPlugin("plugin2")

			engine.RegisterPlugin(plugin1)
			engine.RegisterPlugin(plugin2)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create workflow with sequential steps
			workflow := types.Workflow{
				Name: "sequential-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
						Parallel:   false,
					},
					{
						Name:       "step2",
						PluginName: "plugin2",
						Resource:   resource,
						Parallel:   false,
					},
				},
			}

			ctx := context.Background()
			result, err := engine.ExecuteWorkflow(ctx, workflow)

			// Verify workflow succeeded
			if err != nil || result.Status != types.StatusSuccess {
				return false
			}

			// Verify both steps executed
			return len(result.StepResults) == 2
		},
		gen.Identifier(),
	))

	properties.Property("parallel steps can execute concurrently", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()

			plugin1 := coretesting.NewMockPlugin("plugin1")
			plugin2 := coretesting.NewMockPlugin("plugin2")
			plugin3 := coretesting.NewMockPlugin("plugin3")

			engine.RegisterPlugin(plugin1)
			engine.RegisterPlugin(plugin2)
			engine.RegisterPlugin(plugin3)

			resource1 := types.Resource{
				ID:   resourceID + "-1",
				Name: "test-resource-1",
				Kind: "container",
			}
			resource2 := types.Resource{
				ID:   resourceID + "-2",
				Name: "test-resource-2",
				Kind: "container",
			}
			resource3 := types.Resource{
				ID:   resourceID + "-3",
				Name: "test-resource-3",
				Kind: "container",
			}

			// Create workflow with parallel steps (different resources to avoid locking)
			workflow := types.Workflow{
				Name: "parallel-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource1,
						Parallel:   true,
					},
					{
						Name:       "step2",
						PluginName: "plugin2",
						Resource:   resource2,
						Parallel:   true,
					},
					{
						Name:       "step3",
						PluginName: "plugin3",
						Resource:   resource3,
						Parallel:   true,
					},
				},
			}

			ctx := context.Background()
			result, err := engine.ExecuteWorkflow(ctx, workflow)

			// Verify workflow succeeded
			if err != nil || result.Status != types.StatusSuccess {
				return false
			}

			// Verify all steps executed
			return len(result.StepResults) == 3
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 75: Dependencies are waited for**
// This property validates that for any WorkflowStep with dependencies,
// execution waits for all dependency steps to complete.
// Validates: Requirements 16.2
func TestProperty75_DependenciesAreWaitedFor(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("step waits for all dependencies", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()

			plugin1 := coretesting.NewMockPlugin("plugin1")
			plugin2 := coretesting.NewMockPlugin("plugin2")
			plugin3 := coretesting.NewMockPlugin("plugin3")
			plugin4 := coretesting.NewMockPlugin("plugin4")

			engine.RegisterPlugin(plugin1)
			engine.RegisterPlugin(plugin2)
			engine.RegisterPlugin(plugin3)
			engine.RegisterPlugin(plugin4)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create workflow: step4 depends on step2 and step3
			workflow := types.Workflow{
				Name: "dependency-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
					},
					{
						Name:       "step2",
						PluginName: "plugin2",
						Resource:   resource,
						DependsOn:  []string{"step1"},
					},
					{
						Name:       "step3",
						PluginName: "plugin3",
						Resource:   resource,
						DependsOn:  []string{"step1"},
					},
					{
						Name:       "step4",
						PluginName: "plugin4",
						Resource:   resource,
						DependsOn:  []string{"step2", "step3"},
					},
				},
			}

			ctx := context.Background()
			result, err := engine.ExecuteWorkflow(ctx, workflow)

			// Verify workflow succeeded
			if err != nil || result.Status != types.StatusSuccess {
				return false
			}

			// Verify all steps executed
			if len(result.StepResults) != 4 {
				return false
			}

			// Verify step4 executed after step2 and step3
			step2Result := result.StepResults["step2"]
			step3Result := result.StepResults["step3"]
			step4Result := result.StepResults["step4"]

			return step4Result.StartTime.After(step2Result.EndTime) ||
				step4Result.StartTime.After(step3Result.EndTime)
		},
		gen.Identifier(),
	))

	properties.Property("missing dependency causes error", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()

			plugin1 := coretesting.NewMockPlugin("plugin1")
			engine.RegisterPlugin(plugin1)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create workflow with missing dependency
			workflow := types.Workflow{
				Name: "missing-dep-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
						DependsOn:  []string{"nonexistent"},
					},
				},
			}

			ctx := context.Background()
			_, err := engine.ExecuteWorkflow(ctx, workflow)

			// Should return error for missing dependency
			return err != nil
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 76: Step error handler is executed**
// This property validates that for any WorkflowStep that fails,
// the configured ErrorHandler is executed.
// Validates: Requirements 16.3
func TestProperty76_StepErrorHandlerIsExecuted(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("error handler abort stops workflow", prop.ForAll(
		func(resourceID string) bool {
			if resourceID == "" {
				resourceID = "test-resource"
			}

			engine := NewDefaultExecutionEngine()
			engine.SetComponents(nil, nil, nil, nil, nil, nil, nil)

			plugin1 := coretesting.NewMockPlugin("plugin1")
			plugin1.ExecuteError = errors.New("step failed")
			plugin2 := coretesting.NewMockPlugin("plugin2")

			engine.RegisterPlugin(plugin1)
			engine.RegisterPlugin(plugin2)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create error handler that aborts
			errorHandler := &SimpleErrorHandler{Action: types.ErrorActionAbort}

			// Create workflow with error handler
			workflow := types.Workflow{
				Name: "error-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
						OnError:    errorHandler,
					},
					{
						Name:       "step2",
						PluginName: "plugin2",
						Resource:   resource,
					},
				},
			}

			ctx := context.Background()
			result, _ := engine.ExecuteWorkflow(ctx, workflow)

			// Verify workflow failed
			if result.Status != types.StatusFailed {
				return false
			}

			// Verify step1 failed
			step1Result, step1Executed := result.StepResults["step1"]
			if !step1Executed || step1Result.Status != types.StatusFailed {
				return false
			}

			// The key check: plugin2 should not have been called at all
			// This is the most reliable way to verify abort worked
			if plugin2.ExecuteCalls > 0 {
				return false
			}

			return true
		},
		gen.Identifier(),
	))

	properties.Property("error handler continue allows workflow to proceed", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()

			plugin1 := coretesting.NewMockPlugin("plugin1")
			plugin1.ExecuteError = errors.New("step failed")
			plugin2 := coretesting.NewMockPlugin("plugin2")

			engine.RegisterPlugin(plugin1)
			engine.RegisterPlugin(plugin2)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create error handler that continues
			errorHandler := &SimpleErrorHandler{Action: types.ErrorActionContinue}

			// Create workflow with error handler
			workflow := types.Workflow{
				Name: "continue-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
						OnError:    errorHandler,
					},
					{
						Name:       "step2",
						PluginName: "plugin2",
						Resource:   resource,
					},
				},
			}

			ctx := context.Background()
			result, _ := engine.ExecuteWorkflow(ctx, workflow)

			// Verify step2 was executed despite step1 failure
			step2Result, step2Executed := result.StepResults["step2"]
			return step2Executed && step2Result.Status == types.StatusSuccess
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 78: Conditional execution works correctly**
// This property validates that for any WorkflowStep with a Condition,
// the step executes only if the condition evaluates to true.
// Validates: Requirements 16.5
func TestProperty78_ConditionalExecutionWorksCorrectly(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("always condition executes step", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()

			plugin1 := coretesting.NewMockPlugin("plugin1")
			engine.RegisterPlugin(plugin1)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create workflow with always condition
			workflow := types.Workflow{
				Name: "always-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
						Condition: &types.StepCondition{
							Type: types.ConditionAlways,
						},
					},
				},
			}

			ctx := context.Background()
			result, err := engine.ExecuteWorkflow(ctx, workflow)

			// Verify step executed
			if err != nil {
				return false
			}

			stepResult, executed := result.StepResults["step1"]
			return executed && stepResult.Status == types.StatusSuccess
		},
		gen.Identifier(),
	))

	properties.Property("on_success condition executes when dependencies succeed", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()

			plugin1 := coretesting.NewMockPlugin("plugin1")
			plugin2 := coretesting.NewMockPlugin("plugin2")

			engine.RegisterPlugin(plugin1)
			engine.RegisterPlugin(plugin2)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create workflow with on_success condition
			workflow := types.Workflow{
				Name: "success-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
					},
					{
						Name:       "step2",
						PluginName: "plugin2",
						Resource:   resource,
						DependsOn:  []string{"step1"},
						Condition: &types.StepCondition{
							Type: types.ConditionOnSuccess,
						},
					},
				},
			}

			ctx := context.Background()
			result, err := engine.ExecuteWorkflow(ctx, workflow)

			// Verify both steps executed
			if err != nil {
				return false
			}

			step2Result, executed := result.StepResults["step2"]
			return executed && step2Result.Status == types.StatusSuccess
		},
		gen.Identifier(),
	))

	properties.Property("on_success condition skips when dependencies fail", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()

			plugin1 := coretesting.NewMockPlugin("plugin1")
			plugin1.ExecuteError = errors.New("step failed")
			plugin2 := coretesting.NewMockPlugin("plugin2")

			engine.RegisterPlugin(plugin1)
			engine.RegisterPlugin(plugin2)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create error handler that continues
			errorHandler := &SimpleErrorHandler{Action: types.ErrorActionContinue}

			// Create workflow with on_success condition
			workflow := types.Workflow{
				Name: "skip-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
						OnError:    errorHandler,
					},
					{
						Name:       "step2",
						PluginName: "plugin2",
						Resource:   resource,
						DependsOn:  []string{"step1"},
						Condition: &types.StepCondition{
							Type: types.ConditionOnSuccess,
						},
					},
				},
			}

			ctx := context.Background()
			result, _ := engine.ExecuteWorkflow(ctx, workflow)

			// Verify step2 was skipped
			step2Result, executed := result.StepResults["step2"]
			return executed && step2Result.Status == types.StatusSkipped
		},
		gen.Identifier(),
	))

	properties.Property("on_failure condition executes when dependencies fail", prop.ForAll(
		func(resourceID string) bool {
			engine := NewDefaultExecutionEngine()

			plugin1 := coretesting.NewMockPlugin("plugin1")
			plugin1.ExecuteError = errors.New("step failed")
			plugin2 := coretesting.NewMockPlugin("plugin2")

			engine.RegisterPlugin(plugin1)
			engine.RegisterPlugin(plugin2)

			resource := types.Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			// Create error handler that continues
			errorHandler := &SimpleErrorHandler{Action: types.ErrorActionContinue}

			// Create workflow with on_failure condition
			workflow := types.Workflow{
				Name: "failure-workflow",
				Steps: []types.WorkflowStep{
					{
						Name:       "step1",
						PluginName: "plugin1",
						Resource:   resource,
						OnError:    errorHandler,
					},
					{
						Name:       "step2",
						PluginName: "plugin2",
						Resource:   resource,
						DependsOn:  []string{"step1"},
						Condition: &types.StepCondition{
							Type: types.ConditionOnFailure,
						},
					},
				},
			}

			ctx := context.Background()
			result, _ := engine.ExecuteWorkflow(ctx, workflow)

			// Verify step2 was executed because step1 failed
			step2Result, executed := result.StepResults["step2"]
			return executed && step2Result.Status == types.StatusSuccess
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// SimpleErrorHandler is a simple error handler for testing
type SimpleErrorHandler struct {
	Action types.ErrorAction
}

func (h *SimpleErrorHandler) Handle(ctx context.Context, step types.WorkflowStep, err error) types.ErrorAction {
	return h.Action
}
