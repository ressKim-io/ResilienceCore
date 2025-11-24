// Package testing provides test harness and builder utilities
package testing

import (
	"context"
	"time"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// ============================================================================
// TestHarness
// ============================================================================

// TestHarness provides a fully configured test environment with all Core components
type TestHarness struct {
	Adapter  *MockEnvironmentAdapter
	Monitor  *MockMonitor
	Reporter *MockReporter
	EventBus *MockEventBus
	Logger   *types.MockLogger
	Metrics  *types.MockMetricsCollector
	Tracer   *types.MockTracer
}

// NewTestHarness creates a new TestHarness with all components initialized
func NewTestHarness() *TestHarness {
	return &TestHarness{
		Adapter:  NewMockEnvironmentAdapter(),
		Monitor:  NewMockMonitor(),
		Reporter: NewMockReporter(),
		EventBus: NewMockEventBus(),
		Logger:   types.NewMockLogger(),
		Metrics:  types.NewMockMetricsCollector(),
		Tracer:   types.NewMockTracer(),
	}
}

// CreatePluginContext creates a PluginContext for testing
func (h *TestHarness) CreatePluginContext(ctx context.Context, executionID string) types.PluginContext {
	return types.PluginContext{
		Context:     ctx,
		ExecutionID: executionID,
		Env:         h.Adapter,
		Monitor:     h.Monitor,
		Reporter:    h.Reporter,
		EventBus:    h.EventBus,
		Logger:      h.Logger,
		Tracer:      h.Tracer,
		Progress:    make(chan types.ProgressUpdate, 10),
		Principal:   "test-user",
	}
}

// Cleanup cleans up all resources
func (h *TestHarness) Cleanup() error {
	if err := h.Adapter.Close(); err != nil {
		return err
	}
	if err := h.EventBus.Close(); err != nil {
		return err
	}
	return nil
}

// ============================================================================
// ResourceBuilder
// ============================================================================

// ResourceBuilder provides a fluent interface for building test resources
type ResourceBuilder struct {
	resource types.Resource
}

// NewResourceBuilder creates a new ResourceBuilder
func NewResourceBuilder() *ResourceBuilder {
	return &ResourceBuilder{
		resource: types.Resource{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Metadata:    make(map[string]interface{}),
			Status: types.ResourceStatus{
				Phase:      "running",
				Conditions: make([]types.Condition, 0),
			},
			Spec: types.ResourceSpec{
				Environment: make(map[string]string),
				Ports:       make([]types.Port, 0),
				Volumes:     make([]types.Volume, 0),
			},
		},
	}
}

// WithID sets the resource ID
func (b *ResourceBuilder) WithID(id string) *ResourceBuilder {
	b.resource.ID = id
	return b
}

// WithName sets the resource name
func (b *ResourceBuilder) WithName(name string) *ResourceBuilder {
	b.resource.Name = name
	return b
}

// WithKind sets the resource kind
func (b *ResourceBuilder) WithKind(kind string) *ResourceBuilder {
	b.resource.Kind = kind
	return b
}

// WithLabel adds a label
func (b *ResourceBuilder) WithLabel(key, value string) *ResourceBuilder {
	b.resource.Labels[key] = value
	return b
}

// WithLabels sets multiple labels
func (b *ResourceBuilder) WithLabels(labels map[string]string) *ResourceBuilder {
	for k, v := range labels {
		b.resource.Labels[k] = v
	}
	return b
}

// WithAnnotation adds an annotation
func (b *ResourceBuilder) WithAnnotation(key, value string) *ResourceBuilder {
	b.resource.Annotations[key] = value
	return b
}

// WithAnnotations sets multiple annotations
func (b *ResourceBuilder) WithAnnotations(annotations map[string]string) *ResourceBuilder {
	for k, v := range annotations {
		b.resource.Annotations[k] = v
	}
	return b
}

// WithStatus sets the resource status
func (b *ResourceBuilder) WithStatus(phase string) *ResourceBuilder {
	b.resource.Status.Phase = phase
	return b
}

// WithStatusMessage sets the status message
func (b *ResourceBuilder) WithStatusMessage(message string) *ResourceBuilder {
	b.resource.Status.Message = message
	return b
}

// WithCondition adds a condition
func (b *ResourceBuilder) WithCondition(condition types.Condition) *ResourceBuilder {
	b.resource.Status.Conditions = append(b.resource.Status.Conditions, condition)
	return b
}

// WithImage sets the image
func (b *ResourceBuilder) WithImage(image string) *ResourceBuilder {
	b.resource.Spec.Image = image
	return b
}

// WithReplicas sets the replica count
func (b *ResourceBuilder) WithReplicas(replicas int32) *ResourceBuilder {
	b.resource.Spec.Replicas = &replicas
	return b
}

// WithPort adds a port
func (b *ResourceBuilder) WithPort(name string, containerPort, hostPort int32, protocol string) *ResourceBuilder {
	b.resource.Spec.Ports = append(b.resource.Spec.Ports, types.Port{
		Name:          name,
		ContainerPort: containerPort,
		HostPort:      hostPort,
		Protocol:      protocol,
	})
	return b
}

// WithEnvironmentVariable adds an environment variable
func (b *ResourceBuilder) WithEnvironmentVariable(key, value string) *ResourceBuilder {
	b.resource.Spec.Environment[key] = value
	return b
}

// WithVolume adds a volume
func (b *ResourceBuilder) WithVolume(name, mountPath string, readOnly bool, source types.VolumeSource) *ResourceBuilder {
	b.resource.Spec.Volumes = append(b.resource.Spec.Volumes, types.Volume{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  readOnly,
		Source:    source,
	})
	return b
}

// WithCommand sets the command
func (b *ResourceBuilder) WithCommand(command []string) *ResourceBuilder {
	b.resource.Spec.Command = command
	return b
}

// WithArgs sets the arguments
func (b *ResourceBuilder) WithArgs(args []string) *ResourceBuilder {
	b.resource.Spec.Args = args
	return b
}

// WithHealthCheck sets the health check configuration
func (b *ResourceBuilder) WithHealthCheck(config *types.HealthCheckConfig) *ResourceBuilder {
	b.resource.Spec.HealthCheck = config
	return b
}

// WithMetadata adds metadata
func (b *ResourceBuilder) WithMetadata(key string, value interface{}) *ResourceBuilder {
	b.resource.Metadata[key] = value
	return b
}

// Build returns the built resource
func (b *ResourceBuilder) Build() types.Resource {
	return b.resource
}

// ============================================================================
// WorkflowTestRunner
// ============================================================================

// WorkflowTestRunner provides utilities for testing workflows
type WorkflowTestRunner struct {
	harness         *TestHarness
	executedSteps   map[string]bool
	skippedSteps    map[string]bool
	stepResults     map[string]types.ExecutionResult
}

// NewWorkflowTestRunner creates a new WorkflowTestRunner
func NewWorkflowTestRunner(harness *TestHarness) *WorkflowTestRunner {
	return &WorkflowTestRunner{
		harness:       harness,
		executedSteps: make(map[string]bool),
		skippedSteps:  make(map[string]bool),
		stepResults:   make(map[string]types.ExecutionResult),
	}
}

// RecordStepExecution records that a step was executed
func (r *WorkflowTestRunner) RecordStepExecution(stepName string, result types.ExecutionResult) {
	r.executedSteps[stepName] = true
	r.stepResults[stepName] = result
}

// RecordStepSkipped records that a step was skipped
func (r *WorkflowTestRunner) RecordStepSkipped(stepName string) {
	r.skippedSteps[stepName] = true
}

// AssertStepExecuted asserts that a step was executed
func (r *WorkflowTestRunner) AssertStepExecuted(stepName string) bool {
	return r.executedSteps[stepName]
}

// AssertStepSkipped asserts that a step was skipped
func (r *WorkflowTestRunner) AssertStepSkipped(stepName string) bool {
	return r.skippedSteps[stepName]
}

// GetStepResult returns the result of a step
func (r *WorkflowTestRunner) GetStepResult(stepName string) (types.ExecutionResult, bool) {
	result, exists := r.stepResults[stepName]
	return result, exists
}

// Reset resets the runner state
func (r *WorkflowTestRunner) Reset() {
	r.executedSteps = make(map[string]bool)
	r.skippedSteps = make(map[string]bool)
	r.stepResults = make(map[string]types.ExecutionResult)
}

// ============================================================================
// Helper Functions
// ============================================================================

// CreateTestResource creates a simple test resource
func CreateTestResource(id, name, kind string) types.Resource {
	return NewResourceBuilder().
		WithID(id).
		WithName(name).
		WithKind(kind).
		Build()
}

// CreateTestPlugin creates a simple test plugin
func CreateTestPlugin(name string) *MockPlugin {
	return NewMockPlugin(name)
}

// CreateTestEvent creates a simple test event
func CreateTestEvent(eventType, source string, resource types.Resource) types.Event {
	return types.Event{
		ID:        "event-1",
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now(),
		Resource:  resource,
		Data:      make(map[string]interface{}),
		Metadata:  make(map[string]string),
	}
}
