package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	coretesting "github.com/yourusername/infrastructure-resilience-engine/pkg/core/testing"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

func TestDefaultExecutionEngine_RegisterPlugin(t *testing.T) {
	tests := []struct {
		name    string
		plugin  types.Plugin
		wantErr bool
	}{
		{
			name:    "valid plugin",
			plugin:  coretesting.NewMockPlugin("test-plugin"),
			wantErr: false,
		},
		{
			name:    "nil plugin",
			plugin:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewDefaultExecutionEngine()
			err := engine.RegisterPlugin(tt.plugin)

			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterPlugin() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.plugin != nil {
				// Verify plugin was registered
				retrieved, err := engine.GetPlugin(tt.plugin.Metadata().Name)
				if err != nil {
					t.Errorf("GetPlugin() error = %v", err)
				}
				if retrieved != tt.plugin {
					t.Errorf("GetPlugin() returned different plugin")
				}
			}
		})
	}
}

func TestDefaultExecutionEngine_RegisterPlugin_Duplicate(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	plugin := coretesting.NewMockPlugin("test-plugin")

	// Register first time - should succeed
	err := engine.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("First RegisterPlugin() failed: %v", err)
	}

	// Register again - should fail
	err = engine.RegisterPlugin(plugin)
	if err == nil {
		t.Error("RegisterPlugin() should fail for duplicate plugin")
	}
}

func TestDefaultExecutionEngine_UnregisterPlugin(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	plugin := coretesting.NewMockPlugin("test-plugin")

	// Register plugin
	if err := engine.RegisterPlugin(plugin); err != nil {
		t.Fatalf("RegisterPlugin() failed: %v", err)
	}

	// Unregister plugin
	err := engine.UnregisterPlugin("test-plugin")
	if err != nil {
		t.Errorf("UnregisterPlugin() error = %v", err)
	}

	// Verify plugin was removed
	_, err = engine.GetPlugin("test-plugin")
	if err == nil {
		t.Error("GetPlugin() should fail after unregister")
	}
}

func TestDefaultExecutionEngine_UnregisterPlugin_NotFound(t *testing.T) {
	engine := NewDefaultExecutionEngine()

	err := engine.UnregisterPlugin("nonexistent")
	if err == nil {
		t.Error("UnregisterPlugin() should fail for nonexistent plugin")
	}
}

func TestDefaultExecutionEngine_ListPlugins(t *testing.T) {
	engine := NewDefaultExecutionEngine()

	// Register multiple plugins
	plugin1 := coretesting.NewMockPlugin("plugin1")
	plugin1.PluginMetadata.Version = "1.0.0"
	plugin1.PluginMetadata.SupportedKinds = []string{"container"}

	plugin2 := coretesting.NewMockPlugin("plugin2")
	plugin2.PluginMetadata.Version = "2.0.0"
	plugin2.PluginMetadata.SupportedKinds = []string{"pod"}

	engine.RegisterPlugin(plugin1)
	engine.RegisterPlugin(plugin2)

	tests := []struct {
		name      string
		filter    types.PluginFilter
		wantCount int
		wantNames []string
	}{
		{
			name:      "no filter",
			filter:    types.PluginFilter{},
			wantCount: 2,
			wantNames: []string{"plugin1", "plugin2"},
		},
		{
			name: "filter by name",
			filter: types.PluginFilter{
				Names: []string{"plugin1"},
			},
			wantCount: 1,
			wantNames: []string{"plugin1"},
		},
		{
			name: "filter by version",
			filter: types.PluginFilter{
				Versions: []string{"1.0.0"},
			},
			wantCount: 1,
			wantNames: []string{"plugin1"},
		},
		{
			name: "filter by supported kind",
			filter: types.PluginFilter{
				SupportedKinds: []string{"container"},
			},
			wantCount: 1,
			wantNames: []string{"plugin1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugins, err := engine.ListPlugins(tt.filter)
			if err != nil {
				t.Errorf("ListPlugins() error = %v", err)
			}

			if len(plugins) != tt.wantCount {
				t.Errorf("ListPlugins() returned %d plugins, want %d", len(plugins), tt.wantCount)
			}

			// Verify plugin names
			for _, wantName := range tt.wantNames {
				found := false
				for _, plugin := range plugins {
					if plugin.Name == wantName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ListPlugins() missing plugin %s", wantName)
				}
			}
		})
	}
}

func TestDefaultExecutionEngine_Execute_LifecycleOrder(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	plugin := coretesting.NewMockPlugin("test-plugin")
	resource := types.Resource{
		ID:   "resource-1",
		Name: "test-resource",
		Kind: "container",
	}

	// Register plugin
	if err := engine.RegisterPlugin(plugin); err != nil {
		t.Fatalf("RegisterPlugin() failed: %v", err)
	}

	// Execute plugin
	ctx := context.Background()
	request := types.ExecutionRequest{
		PluginName: "test-plugin",
		Resource:   resource,
	}

	result, err := engine.Execute(ctx, request)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if result.Status != types.StatusSuccess {
		t.Errorf("Execute() status = %v, want %v", result.Status, types.StatusSuccess)
	}

	// Verify lifecycle hooks were called in order
	if plugin.ValidateCalls != 1 {
		t.Errorf("Validate called %d times, want 1", plugin.ValidateCalls)
	}
	if plugin.PreExecuteCalls != 1 {
		t.Errorf("PreExecute called %d times, want 1", plugin.PreExecuteCalls)
	}
	if plugin.ExecuteCalls != 1 {
		t.Errorf("Execute called %d times, want 1", plugin.ExecuteCalls)
	}
	if plugin.PostExecuteCalls != 1 {
		t.Errorf("PostExecute called %d times, want 1", plugin.PostExecuteCalls)
	}
	if plugin.CleanupCalls != 1 {
		t.Errorf("Cleanup called %d times, want 1", plugin.CleanupCalls)
	}
}

func TestDefaultExecutionEngine_Execute_ValidationFailure(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	plugin := coretesting.NewMockPlugin("test-plugin")
	plugin.ValidateError = errors.New("validation failed")

	resource := types.Resource{
		ID:   "resource-1",
		Name: "test-resource",
		Kind: "container",
	}

	engine.RegisterPlugin(plugin)

	ctx := context.Background()
	request := types.ExecutionRequest{
		PluginName: "test-plugin",
		Resource:   resource,
	}

	result, err := engine.Execute(ctx, request)
	if err == nil {
		t.Error("Execute() should fail when validation fails")
	}

	if result.Status != types.StatusFailed {
		t.Errorf("Execute() status = %v, want %v", result.Status, types.StatusFailed)
	}

	// Verify only Validate was called
	if plugin.ValidateCalls != 1 {
		t.Errorf("Validate called %d times, want 1", plugin.ValidateCalls)
	}
	if plugin.PreExecuteCalls != 0 {
		t.Errorf("PreExecute should not be called after validation failure")
	}
	if plugin.ExecuteCalls != 0 {
		t.Errorf("Execute should not be called after validation failure")
	}
}

func TestDefaultExecutionEngine_Execute_Rollback(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	plugin := coretesting.NewMockPlugin("test-plugin")
	plugin.ExecuteError = errors.New("execution failed")

	resource := types.Resource{
		ID:   "resource-1",
		Name: "test-resource",
		Kind: "container",
	}

	engine.RegisterPlugin(plugin)

	ctx := context.Background()
	request := types.ExecutionRequest{
		PluginName: "test-plugin",
		Resource:   resource,
	}

	result, err := engine.Execute(ctx, request)
	if err == nil {
		t.Error("Execute() should fail when execution fails")
	}

	if result.Status != types.StatusFailed {
		t.Errorf("Execute() status = %v, want %v", result.Status, types.StatusFailed)
	}

	// Verify rollback was called
	if plugin.RollbackCalls != 1 {
		t.Errorf("Rollback called %d times, want 1", plugin.RollbackCalls)
	}

	// Verify cleanup was still called
	if plugin.CleanupCalls != 1 {
		t.Errorf("Cleanup called %d times, want 1", plugin.CleanupCalls)
	}
}

func TestDefaultExecutionEngine_Execute_Timeout(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	engine.SetDefaultTimeout(100 * time.Millisecond)

	plugin := coretesting.NewMockPlugin("test-plugin")
	// Make execute block longer than timeout
	plugin.ExecuteError = context.DeadlineExceeded

	resource := types.Resource{
		ID:   "resource-1",
		Name: "test-resource",
		Kind: "container",
	}

	engine.RegisterPlugin(plugin)

	ctx := context.Background()
	request := types.ExecutionRequest{
		PluginName: "test-plugin",
		Resource:   resource,
	}

	result, _ := engine.Execute(ctx, request)

	if result.Status != types.StatusTimeout {
		t.Errorf("Execute() status = %v, want %v", result.Status, types.StatusTimeout)
	}
}

func TestDefaultExecutionEngine_ExecuteAsync(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	plugin := coretesting.NewMockPlugin("test-plugin")

	resource := types.Resource{
		ID:   "resource-1",
		Name: "test-resource",
		Kind: "container",
	}

	engine.RegisterPlugin(plugin)

	ctx := context.Background()
	request := types.ExecutionRequest{
		PluginName: "test-plugin",
		Resource:   resource,
	}

	future, err := engine.ExecuteAsync(ctx, request)
	if err != nil {
		t.Fatalf("ExecuteAsync() error = %v", err)
	}

	// Wait for completion
	result, err := future.Wait()
	if err != nil {
		t.Errorf("Future.Wait() error = %v", err)
	}

	if result.Status != types.StatusSuccess {
		t.Errorf("Future.Wait() status = %v, want %v", result.Status, types.StatusSuccess)
	}

	// Verify plugin was executed
	if plugin.ExecuteCalls != 1 {
		t.Errorf("Execute called %d times, want 1", plugin.ExecuteCalls)
	}
}

func TestDefaultExecutionEngine_ExecuteAsync_Cancel(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	plugin := coretesting.NewMockPlugin("test-plugin")

	resource := types.Resource{
		ID:   "resource-1",
		Name: "test-resource",
		Kind: "container",
	}

	engine.RegisterPlugin(plugin)

	ctx := context.Background()
	request := types.ExecutionRequest{
		PluginName: "test-plugin",
		Resource:   resource,
	}

	future, err := engine.ExecuteAsync(ctx, request)
	if err != nil {
		t.Fatalf("ExecuteAsync() error = %v", err)
	}

	// Cancel immediately
	if err := future.Cancel(); err != nil {
		t.Errorf("Future.Cancel() error = %v", err)
	}

	// Wait for completion
	<-future.Done()
}

func TestDefaultExecutionEngine_SetConcurrencyLimit(t *testing.T) {
	engine := NewDefaultExecutionEngine()

	// Set concurrency limit
	engine.SetConcurrencyLimit(5)

	if cap(engine.semaphore) != 5 {
		t.Errorf("SetConcurrencyLimit() semaphore capacity = %d, want 5", cap(engine.semaphore))
	}

	// Test with zero (should default to 1)
	engine.SetConcurrencyLimit(0)
	if cap(engine.semaphore) != 1 {
		t.Errorf("SetConcurrencyLimit(0) semaphore capacity = %d, want 1", cap(engine.semaphore))
	}
}

func TestDefaultExecutionEngine_SetDefaultTimeout(t *testing.T) {
	engine := NewDefaultExecutionEngine()

	timeout := 10 * time.Second
	engine.SetDefaultTimeout(timeout)

	if engine.defaultTimeout != timeout {
		t.Errorf("SetDefaultTimeout() timeout = %v, want %v", engine.defaultTimeout, timeout)
	}
}

func TestDefaultExecutionEngine_ConcurrencyLimit(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	engine.SetConcurrencyLimit(2) // Allow only 2 concurrent executions

	plugin := coretesting.NewMockPlugin("test-plugin")
	engine.RegisterPlugin(plugin)

	resource1 := types.Resource{ID: "resource-1", Name: "test-1", Kind: "container"}
	resource2 := types.Resource{ID: "resource-2", Name: "test-2", Kind: "container"}
	resource3 := types.Resource{ID: "resource-3", Name: "test-3", Kind: "container"}

	ctx := context.Background()

	// Start 3 async executions
	future1, _ := engine.ExecuteAsync(ctx, types.ExecutionRequest{PluginName: "test-plugin", Resource: resource1})
	future2, _ := engine.ExecuteAsync(ctx, types.ExecutionRequest{PluginName: "test-plugin", Resource: resource2})
	future3, _ := engine.ExecuteAsync(ctx, types.ExecutionRequest{PluginName: "test-plugin", Resource: resource3})

	// Wait for all to complete
	future1.Wait()
	future2.Wait()
	future3.Wait()

	// All should complete successfully despite concurrency limit
	if plugin.ExecuteCalls != 3 {
		t.Errorf("Execute called %d times, want 3", plugin.ExecuteCalls)
	}
}

func TestDefaultExecutionEngine_ResourceLocking(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	plugin := coretesting.NewMockPlugin("test-plugin")
	engine.RegisterPlugin(plugin)

	// Same resource
	resource := types.Resource{ID: "resource-1", Name: "test", Kind: "container"}

	ctx := context.Background()

	// Start 2 async executions on the same resource
	future1, _ := engine.ExecuteAsync(ctx, types.ExecutionRequest{PluginName: "test-plugin", Resource: resource})
	future2, _ := engine.ExecuteAsync(ctx, types.ExecutionRequest{PluginName: "test-plugin", Resource: resource})

	// Wait for both to complete
	future1.Wait()
	future2.Wait()

	// Both should complete, but sequentially (not concurrently)
	if plugin.ExecuteCalls != 2 {
		t.Errorf("Execute called %d times, want 2", plugin.ExecuteCalls)
	}
}

func TestDefaultExecutionEngine_WithComponents(t *testing.T) {
	engine := NewDefaultExecutionEngine()

	// Create mock components
	monitor := coretesting.NewMockMonitor()
	reporter := coretesting.NewMockReporter()
	eventBus := coretesting.NewMockEventBus()

	// Set components
	engine.SetComponents(monitor, reporter, eventBus, nil, nil, nil, nil)

	// Verify components are set
	if engine.monitor != monitor {
		t.Error("Monitor not set correctly")
	}
	if engine.reporter != reporter {
		t.Error("Reporter not set correctly")
	}
	if engine.eventBus != eventBus {
		t.Error("EventBus not set correctly")
	}

	// Execute a plugin and verify reporter was called
	plugin := coretesting.NewMockPlugin("test-plugin")
	engine.RegisterPlugin(plugin)

	resource := types.Resource{ID: "resource-1", Name: "test", Kind: "container"}
	ctx := context.Background()

	engine.Execute(ctx, types.ExecutionRequest{PluginName: "test-plugin", Resource: resource})

	// Verify reporter recorded the execution
	if reporter.RecordExecutionCalls != 1 {
		t.Errorf("Reporter.RecordExecution called %d times, want 1", reporter.RecordExecutionCalls)
	}
}
