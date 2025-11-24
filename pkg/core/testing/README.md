# Testing Package

This package provides comprehensive mock implementations and test utilities for testing Infrastructure Resilience Engine components.

## Overview

The testing package includes:

- **Mock Implementations**: Full mock implementations of all Core interfaces
- **Test Harness**: Pre-configured test environment with all components
- **Resource Builder**: Fluent interface for building test resources
- **Workflow Test Runner**: Utilities for testing workflow execution

## Mock Implementations

### MockEnvironmentAdapter

A complete mock implementation of the `EnvironmentAdapter` interface.

```go
adapter := testing.NewMockEnvironmentAdapter()

// Configure behavior
adapter.InitializeError = errors.New("initialization failed")

// Add test resources
resource := types.Resource{
    ID:   "test-1",
    Name: "test-resource",
    Kind: "container",
}
adapter.AddResource(resource)

// Use in tests
resources, err := adapter.ListResources(ctx, types.ResourceFilter{})

// Verify calls
if adapter.ListResourcesCalls != 1 {
    t.Error("Expected 1 call to ListResources")
}
```

**Features:**
- Configurable error responses for all methods
- Call tracking for verification
- Resource state management
- Event emission for watchers
- Metrics and exec result configuration

### MockPlugin

A complete mock implementation of the `Plugin` interface.

```go
plugin := testing.NewMockPlugin("test-plugin")

// Configure behavior
plugin.ExecuteError = errors.New("execution failed")
plugin.SnapshotToReturn = customSnapshot

// Use in tests
err := plugin.Execute(pluginCtx, resource)

// Verify calls and arguments
if plugin.ExecuteCalls != 1 {
    t.Error("Expected 1 call to Execute")
}
if plugin.LastExecuteResource.ID != "expected-id" {
    t.Error("Unexpected resource ID")
}
```

**Features:**
- Configurable error responses for all lifecycle hooks
- Call tracking for all methods
- Argument capture for verification
- Custom snapshot support

### MockMonitor

A complete mock implementation of the `Monitor` interface.

```go
monitor := testing.NewMockMonitor()

// Configure behavior
monitor.HealthStatusToReturn = types.HealthStatus{
    Status:  types.HealthStatusUnhealthy,
    Message: "Resource is unhealthy",
}

// Use in tests
status, err := monitor.CheckHealth(ctx, resource)

// Verify calls
if monitor.CheckHealthCalls != 1 {
    t.Error("Expected 1 call to CheckHealth")
}
```

**Features:**
- Configurable health status responses
- Configurable metrics responses
- Call tracking
- Strategy registration support

### MockReporter

A complete mock implementation of the `Reporter` interface.

```go
reporter := testing.NewMockReporter()

// Record events and executions
reporter.RecordEvent(ctx, event)
reporter.RecordExecution(ctx, executionRecord)

// Verify recorded data
if len(reporter.Events) != 1 {
    t.Error("Expected 1 recorded event")
}
if len(reporter.Executions) != 1 {
    t.Error("Expected 1 recorded execution")
}
```

**Features:**
- In-memory storage of events and executions
- Configurable statistics responses
- Call tracking
- Formatter registration support

### MockEventBus

A complete mock implementation of the `EventBus` interface.

```go
eventBus := testing.NewMockEventBus()

// Publish events
eventBus.Publish(ctx, event)

// Subscribe to events
sub, err := eventBus.Subscribe(ctx, types.EventFilter{
    Types: []string{"test"},
})

// Verify published events
if len(eventBus.PublishedEvents) != 1 {
    t.Error("Expected 1 published event")
}

// Receive events
select {
case evt := <-sub.Events():
    // Process event
case <-time.After(time.Second):
    t.Error("Timeout waiting for event")
}
```

**Features:**
- Event filtering and delivery
- Subscription management
- Call tracking
- Graceful shutdown

## Test Harness

The `TestHarness` provides a fully configured test environment with all Core components.

```go
harness := testing.NewTestHarness()
defer harness.Cleanup()

// Access components
adapter := harness.Adapter
monitor := harness.Monitor
reporter := harness.Reporter
eventBus := harness.EventBus
logger := harness.Logger
metrics := harness.Metrics
tracer := harness.Tracer

// Create plugin context
pluginCtx := harness.CreatePluginContext(ctx, "exec-123")

// Use in plugin tests
plugin := NewMyPlugin()
err := plugin.Execute(pluginCtx, resource)
```

**Benefits:**
- All components pre-initialized
- Consistent test setup
- Easy cleanup
- Plugin context creation

## Resource Builder

The `ResourceBuilder` provides a fluent interface for building test resources.

```go
resource := testing.NewResourceBuilder().
    WithID("test-1").
    WithName("test-resource").
    WithKind("container").
    WithLabel("env", "test").
    WithLabel("app", "myapp").
    WithStatus("running").
    WithImage("nginx:latest").
    WithPort("http", 80, 8080, "TCP").
    WithEnvironmentVariable("DEBUG", "true").
    WithHealthCheck(&types.HealthCheckConfig{
        Type:     "http",
        Endpoint: "http://localhost:8080/health",
        Interval: 10 * time.Second,
        Timeout:  5 * time.Second,
        Retries:  3,
    }).
    Build()
```

**Features:**
- Fluent interface
- All resource fields supported
- Sensible defaults
- Type-safe construction

## Workflow Test Runner

The `WorkflowTestRunner` provides utilities for testing workflow execution.

```go
harness := testing.NewTestHarness()
runner := testing.NewWorkflowTestRunner(harness)

// Record step execution
runner.RecordStepExecution("step1", types.ExecutionResult{
    Status: types.StatusSuccess,
})

// Record step skipped
runner.RecordStepSkipped("step2")

// Verify execution
if !runner.AssertStepExecuted("step1") {
    t.Error("Expected step1 to be executed")
}

if !runner.AssertStepSkipped("step2") {
    t.Error("Expected step2 to be skipped")
}

// Get step result
result, exists := runner.GetStepResult("step1")
if !exists {
    t.Error("Expected result for step1")
}
```

**Features:**
- Step execution tracking
- Step skip tracking
- Result retrieval
- State reset

## Helper Functions

### CreateTestResource

Quickly create a simple test resource:

```go
resource := testing.CreateTestResource("test-1", "test-resource", "container")
```

### CreateTestPlugin

Quickly create a simple test plugin:

```go
plugin := testing.CreateTestPlugin("test-plugin")
```

### CreateTestEvent

Quickly create a simple test event:

```go
event := testing.CreateTestEvent("test-type", "test-source", resource)
```

## Example: Testing a Plugin

```go
func TestMyPlugin_Execute(t *testing.T) {
    // Setup
    harness := testing.NewTestHarness()
    defer harness.Cleanup()
    
    plugin := NewMyPlugin()
    plugin.Initialize(nil)
    
    resource := testing.NewResourceBuilder().
        WithID("test-1").
        WithName("test-resource").
        WithKind("container").
        Build()
    
    harness.Adapter.AddResource(resource)
    
    // Execute
    pluginCtx := harness.CreatePluginContext(context.Background(), "exec-123")
    err := plugin.Execute(pluginCtx, resource)
    
    // Verify
    if err != nil {
        t.Fatalf("Execute failed: %v", err)
    }
    
    // Verify adapter was called
    if harness.Adapter.StopResourceCalls != 1 {
        t.Error("Expected 1 call to StopResource")
    }
    
    // Verify event was published
    if len(harness.EventBus.PublishedEvents) != 1 {
        t.Error("Expected 1 published event")
    }
}
```

## Example: Testing an Adapter

```go
func TestMyAdapter_ListResources(t *testing.T) {
    adapter := NewMyAdapter()
    
    // Initialize
    err := adapter.Initialize(context.Background(), types.AdapterConfig{})
    if err != nil {
        t.Fatalf("Initialize failed: %v", err)
    }
    defer adapter.Close()
    
    // List resources
    resources, err := adapter.ListResources(context.Background(), types.ResourceFilter{})
    if err != nil {
        t.Fatalf("ListResources failed: %v", err)
    }
    
    // Verify
    if len(resources) == 0 {
        t.Error("Expected at least one resource")
    }
    
    for _, resource := range resources {
        if resource.ID == "" {
            t.Error("Resource ID is empty")
        }
        if resource.Kind == "" {
            t.Error("Resource Kind is empty")
        }
    }
}
```

## Property-Based Testing

The testing package includes comprehensive property-based tests using gopter:

```go
func TestProperty_MyFeature(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("my feature works correctly", prop.ForAll(
        func(input string) bool {
            // Test logic
            result := MyFunction(input)
            return result != ""
        },
        gen.AnyString(),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

## Best Practices

1. **Use TestHarness for integration tests**: It provides all components pre-configured
2. **Use ResourceBuilder for complex resources**: It's more readable than manual construction
3. **Configure mock errors for negative tests**: Test error handling paths
4. **Verify call counts**: Ensure methods are called the expected number of times
5. **Check captured arguments**: Verify the correct data was passed to mocks
6. **Clean up resources**: Always call `Cleanup()` or `Close()` in defer statements
7. **Use property-based tests**: Verify properties hold across many inputs

## Thread Safety

All mock implementations are thread-safe and can be used in concurrent tests. They use `sync.RWMutex` for synchronization.

## Performance

Mock implementations are designed for testing and prioritize correctness over performance. They maintain in-memory state and are suitable for unit and integration tests but not for performance benchmarking.
