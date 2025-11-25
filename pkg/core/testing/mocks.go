// Package testing provides comprehensive mock implementations and test utilities
// for testing Infrastructure Resilience Engine components
package testing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// ============================================================================
// MockEnvironmentAdapter
// ============================================================================

// MockEnvironmentAdapter is a mock implementation of EnvironmentAdapter for testing
type MockEnvironmentAdapter struct {
	mu sync.RWMutex

	// State
	Resources     map[string]types.Resource
	Initialized   bool
	Closed        bool
	WatchChannels []chan types.ResourceEvent
	MetricsData   map[string]types.Metrics
	ExecResults   map[string]types.ExecResult

	// Behavior configuration
	InitializeError      error
	ListResourcesError   error
	GetResourceError     error
	StartResourceError   error
	StopResourceError    error
	RestartResourceError error
	DeleteResourceError  error
	CreateResourceError  error
	UpdateResourceError  error
	ExecError            error
	GetMetricsError      error

	// Call tracking
	InitializeCalls      int
	ListResourcesCalls   int
	GetResourceCalls     int
	StartResourceCalls   int
	StopResourceCalls    int
	RestartResourceCalls int
	DeleteResourceCalls  int
	CreateResourceCalls  int
	UpdateResourceCalls  int
	ExecCalls            int
	GetMetricsCalls      int
	CloseCalls           int
}

// NewMockEnvironmentAdapter creates a new MockEnvironmentAdapter
func NewMockEnvironmentAdapter() *MockEnvironmentAdapter {
	return &MockEnvironmentAdapter{
		Resources:     make(map[string]types.Resource),
		WatchChannels: make([]chan types.ResourceEvent, 0),
		MetricsData:   make(map[string]types.Metrics),
		ExecResults:   make(map[string]types.ExecResult),
	}
}

// Initialize initializes the adapter
func (m *MockEnvironmentAdapter) Initialize(ctx context.Context, config types.AdapterConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.InitializeCalls++
	if m.InitializeError != nil {
		return m.InitializeError
	}

	m.Initialized = true
	return nil
}

// Close closes the adapter
func (m *MockEnvironmentAdapter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CloseCalls++
	m.Closed = true

	// Close all watch channels
	for _, ch := range m.WatchChannels {
		close(ch)
	}
	m.WatchChannels = nil

	return nil
}

// ListResources lists resources matching the filter
func (m *MockEnvironmentAdapter) ListResources(ctx context.Context, filter types.ResourceFilter) ([]types.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.ListResourcesCalls++
	if m.ListResourcesError != nil {
		return nil, m.ListResourcesError
	}

	var result []types.Resource
	for _, resource := range m.Resources {
		if m.matchesFilter(resource, filter) {
			result = append(result, resource)
		}
	}

	return result, nil
}

// GetResource gets a resource by ID
func (m *MockEnvironmentAdapter) GetResource(ctx context.Context, id string) (types.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetResourceCalls++
	if m.GetResourceError != nil {
		return types.Resource{}, m.GetResourceError
	}

	resource, exists := m.Resources[id]
	if !exists {
		return types.Resource{}, fmt.Errorf("resource not found: %s", id)
	}

	return resource, nil
}

// StartResource starts a resource
func (m *MockEnvironmentAdapter) StartResource(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StartResourceCalls++
	if m.StartResourceError != nil {
		return m.StartResourceError
	}

	resource, exists := m.Resources[id]
	if !exists {
		return fmt.Errorf("resource not found: %s", id)
	}

	resource.Status.Phase = "running"
	m.Resources[id] = resource
	m.emitEvent(types.EventModified, resource)

	return nil
}

// StopResource stops a resource
func (m *MockEnvironmentAdapter) StopResource(ctx context.Context, id string, gracePeriod time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StopResourceCalls++
	if m.StopResourceError != nil {
		return m.StopResourceError
	}

	resource, exists := m.Resources[id]
	if !exists {
		return fmt.Errorf("resource not found: %s", id)
	}

	resource.Status.Phase = "stopped"
	m.Resources[id] = resource
	m.emitEvent(types.EventModified, resource)

	return nil
}

// RestartResource restarts a resource
func (m *MockEnvironmentAdapter) RestartResource(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RestartResourceCalls++
	if m.RestartResourceError != nil {
		return m.RestartResourceError
	}

	resource, exists := m.Resources[id]
	if !exists {
		return fmt.Errorf("resource not found: %s", id)
	}

	resource.Status.Phase = "running"
	m.Resources[id] = resource
	m.emitEvent(types.EventModified, resource)

	return nil
}

// DeleteResource deletes a resource
func (m *MockEnvironmentAdapter) DeleteResource(ctx context.Context, id string, options types.DeleteOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DeleteResourceCalls++
	if m.DeleteResourceError != nil {
		return m.DeleteResourceError
	}

	resource, exists := m.Resources[id]
	if !exists {
		return fmt.Errorf("resource not found: %s", id)
	}

	delete(m.Resources, id)
	m.emitEvent(types.EventDeleted, resource)

	return nil
}

// CreateResource creates a new resource
func (m *MockEnvironmentAdapter) CreateResource(ctx context.Context, spec types.ResourceSpec) (types.Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateResourceCalls++
	if m.CreateResourceError != nil {
		return types.Resource{}, m.CreateResourceError
	}

	resource := types.Resource{
		ID:   fmt.Sprintf("resource-%d", len(m.Resources)+1),
		Name: fmt.Sprintf("resource-%d", len(m.Resources)+1),
		Kind: "container",
		Status: types.ResourceStatus{
			Phase: "running",
		},
		Spec:        spec,
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}

	m.Resources[resource.ID] = resource
	m.emitEvent(types.EventAdded, resource)

	return resource, nil
}

// UpdateResource updates a resource
func (m *MockEnvironmentAdapter) UpdateResource(ctx context.Context, id string, spec types.ResourceSpec) (types.Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UpdateResourceCalls++
	if m.UpdateResourceError != nil {
		return types.Resource{}, m.UpdateResourceError
	}

	resource, exists := m.Resources[id]
	if !exists {
		return types.Resource{}, fmt.Errorf("resource not found: %s", id)
	}

	resource.Spec = spec
	m.Resources[id] = resource
	m.emitEvent(types.EventModified, resource)

	return resource, nil
}

// WatchResources watches for resource changes
func (m *MockEnvironmentAdapter) WatchResources(ctx context.Context, filter types.ResourceFilter) (<-chan types.ResourceEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan types.ResourceEvent, 100)
	m.WatchChannels = append(m.WatchChannels, ch)

	// Send initial events for existing resources
	go func() {
		m.mu.RLock()
		defer m.mu.RUnlock()

		for _, resource := range m.Resources {
			if m.matchesFilter(resource, filter) {
				select {
				case ch <- types.ResourceEvent{Type: types.EventAdded, Resource: resource}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// ExecInResource executes a command in a resource
func (m *MockEnvironmentAdapter) ExecInResource(ctx context.Context, id string, cmd []string, options types.ExecOptions) (types.ExecResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExecCalls++
	if m.ExecError != nil {
		return types.ExecResult{}, m.ExecError
	}

	if _, exists := m.Resources[id]; !exists {
		return types.ExecResult{}, fmt.Errorf("resource not found: %s", id)
	}

	// Return pre-configured result if available
	if result, exists := m.ExecResults[id]; exists {
		return result, nil
	}

	// Default success result
	return types.ExecResult{
		Stdout:   "command output",
		Stderr:   "",
		ExitCode: 0,
	}, nil
}

// GetMetrics gets metrics for a resource
func (m *MockEnvironmentAdapter) GetMetrics(ctx context.Context, id string) (types.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetMetricsCalls++
	if m.GetMetricsError != nil {
		return types.Metrics{}, m.GetMetricsError
	}

	if _, exists := m.Resources[id]; !exists {
		return types.Metrics{}, fmt.Errorf("resource not found: %s", id)
	}

	// Return pre-configured metrics if available
	if metrics, exists := m.MetricsData[id]; exists {
		return metrics, nil
	}

	// Default metrics
	return types.Metrics{
		Timestamp:  time.Now(),
		ResourceID: id,
		CPU: types.CPUMetrics{
			UsagePercent: 50.0,
			UsageCores:   1.0,
		},
		Memory: types.MemoryMetrics{
			UsageBytes:   1024 * 1024 * 512,  // 512 MB
			LimitBytes:   1024 * 1024 * 1024, // 1 GB
			UsagePercent: 50.0,
		},
	}, nil
}

// GetAdapterInfo returns adapter metadata
func (m *MockEnvironmentAdapter) GetAdapterInfo() types.AdapterInfo {
	return types.AdapterInfo{
		Name:         "MockAdapter",
		Version:      "1.0.0",
		Environment:  "mock",
		Capabilities: []string{"exec", "metrics", "watch"},
	}
}

// Helper methods

// AddResource adds a resource to the mock adapter
func (m *MockEnvironmentAdapter) AddResource(resource types.Resource) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Resources[resource.ID] = resource
}

// SetExecResult sets the result for ExecInResource calls
func (m *MockEnvironmentAdapter) SetExecResult(resourceID string, result types.ExecResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExecResults[resourceID] = result
}

// SetMetrics sets the metrics for a resource
func (m *MockEnvironmentAdapter) SetMetrics(resourceID string, metrics types.Metrics) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.MetricsData[resourceID] = metrics
}

func (m *MockEnvironmentAdapter) matchesFilter(resource types.Resource, filter types.ResourceFilter) bool {
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

	// Check label selector
	if len(filter.LabelSelector.MatchLabels) > 0 {
		for key, value := range filter.LabelSelector.MatchLabels {
			if resource.Labels[key] != value {
				return false
			}
		}
	}

	return true
}

func (m *MockEnvironmentAdapter) emitEvent(eventType types.EventType, resource types.Resource) {
	event := types.ResourceEvent{
		Type:     eventType,
		Resource: resource,
	}

	for _, ch := range m.WatchChannels {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
}

// ============================================================================
// MockPlugin
// ============================================================================

// MockPlugin is a mock implementation of Plugin for testing
type MockPlugin struct {
	mu sync.RWMutex

	// Configuration
	PluginMetadata types.PluginMetadata

	// State
	Initialized bool

	// Behavior configuration
	InitializeError  error
	ValidateError    error
	PreExecuteError  error
	ExecuteError     error
	PostExecuteError error
	CleanupError     error
	RollbackError    error

	// Call tracking
	InitializeCalls  int
	ValidateCalls    int
	PreExecuteCalls  int
	ExecuteCalls     int
	PostExecuteCalls int
	CleanupCalls     int
	RollbackCalls    int

	// Captured arguments
	LastValidateResource    *types.Resource
	LastPreExecuteResource  *types.Resource
	LastExecuteResource     *types.Resource
	LastPostExecuteResource *types.Resource
	LastCleanupResource     *types.Resource
	LastRollbackResource    *types.Resource
	LastRollbackSnapshot    types.Snapshot

	// Snapshot to return from PreExecute
	SnapshotToReturn types.Snapshot
}

// NewMockPlugin creates a new MockPlugin
func NewMockPlugin(name string) *MockPlugin {
	return &MockPlugin{
		PluginMetadata: types.PluginMetadata{
			Name:        name,
			Version:     "1.0.0",
			Description: "Mock plugin for testing",
			Author:      "Test",
		},
	}
}

// Metadata returns plugin metadata
func (m *MockPlugin) Metadata() types.PluginMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.PluginMetadata
}

// Initialize initializes the plugin
func (m *MockPlugin) Initialize(config types.PluginConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.InitializeCalls++
	if m.InitializeError != nil {
		return m.InitializeError
	}

	m.Initialized = true
	return nil
}

// Validate validates the plugin can execute on the resource
func (m *MockPlugin) Validate(ctx types.PluginContext, resource types.Resource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ValidateCalls++
	m.LastValidateResource = &resource

	if m.ValidateError != nil {
		return m.ValidateError
	}

	return nil
}

// PreExecute prepares for execution and creates a snapshot
func (m *MockPlugin) PreExecute(ctx types.PluginContext, resource types.Resource) (types.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PreExecuteCalls++
	m.LastPreExecuteResource = &resource

	if m.PreExecuteError != nil {
		return nil, m.PreExecuteError
	}

	if m.SnapshotToReturn != nil {
		return m.SnapshotToReturn, nil
	}

	return NewMockSnapshot(resource), nil
}

// Execute executes the plugin
func (m *MockPlugin) Execute(ctx types.PluginContext, resource types.Resource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExecuteCalls++
	m.LastExecuteResource = &resource

	if m.ExecuteError != nil {
		return m.ExecuteError
	}

	return nil
}

// PostExecute performs post-execution tasks
func (m *MockPlugin) PostExecute(ctx types.PluginContext, resource types.Resource, result types.ExecutionResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PostExecuteCalls++
	m.LastPostExecuteResource = &resource

	if m.PostExecuteError != nil {
		return m.PostExecuteError
	}

	return nil
}

// Cleanup performs cleanup tasks
func (m *MockPlugin) Cleanup(ctx types.PluginContext, resource types.Resource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CleanupCalls++
	m.LastCleanupResource = &resource

	if m.CleanupError != nil {
		return m.CleanupError
	}

	return nil
}

// Rollback rolls back changes using the snapshot
func (m *MockPlugin) Rollback(ctx types.PluginContext, resource types.Resource, snapshot types.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RollbackCalls++
	m.LastRollbackResource = &resource
	m.LastRollbackSnapshot = snapshot

	if m.RollbackError != nil {
		return m.RollbackError
	}

	return nil
}

// ============================================================================
// MockSnapshot
// ============================================================================

// MockSnapshot is a mock implementation of Snapshot for testing
type MockSnapshot struct {
	Resource types.Resource
	Data     []byte
}

// NewMockSnapshot creates a new MockSnapshot
func NewMockSnapshot(resource types.Resource) *MockSnapshot {
	return &MockSnapshot{
		Resource: resource,
	}
}

// Restore restores the resource to the snapshot state
func (m *MockSnapshot) Restore(ctx context.Context) error {
	// In a real implementation, this would restore the resource
	return nil
}

// Serialize converts the snapshot to bytes
func (m *MockSnapshot) Serialize() ([]byte, error) {
	if m.Data != nil {
		return m.Data, nil
	}
	return []byte(fmt.Sprintf("snapshot-%s", m.Resource.ID)), nil
}

// Deserialize reconstructs the snapshot from bytes
func (m *MockSnapshot) Deserialize(data []byte) error {
	m.Data = data
	return nil
}

// ============================================================================
// MockMonitor
// ============================================================================

// MockMonitor is a mock implementation of Monitor for testing
type MockMonitor struct {
	mu sync.RWMutex

	// State
	HealthCheckStrategies map[string]types.HealthCheckStrategy

	// Behavior configuration
	CheckHealthError      error
	WaitForConditionError error
	WaitForHealthyError   error
	CollectMetricsError   error
	WatchEventsError      error
	RegisterStrategyError error
	GetStrategyError      error

	// Return values
	HealthStatusToReturn types.HealthStatus
	MetricsToReturn      types.Metrics

	// Call tracking
	CheckHealthCalls      int
	WaitForConditionCalls int
	WaitForHealthyCalls   int
	CollectMetricsCalls   int
	WatchEventsCalls      int
	RegisterStrategyCalls int
	GetStrategyCalls      int
}

// NewMockMonitor creates a new MockMonitor
func NewMockMonitor() *MockMonitor {
	return &MockMonitor{
		HealthCheckStrategies: make(map[string]types.HealthCheckStrategy),
		HealthStatusToReturn: types.HealthStatus{
			Status:  types.HealthStatusHealthy,
			Message: "Resource is healthy",
		},
	}
}

// CheckHealth checks the health of a resource
func (m *MockMonitor) CheckHealth(ctx context.Context, resource types.Resource) (types.HealthStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CheckHealthCalls++
	if m.CheckHealthError != nil {
		return types.HealthStatus{}, m.CheckHealthError
	}

	return m.HealthStatusToReturn, nil
}

// WaitForCondition waits for a condition to be met
func (m *MockMonitor) WaitForCondition(ctx context.Context, resource types.Resource, condition types.Condition, timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.WaitForConditionCalls++
	if m.WaitForConditionError != nil {
		return m.WaitForConditionError
	}

	return nil
}

// WaitForHealthy waits for a resource to become healthy
func (m *MockMonitor) WaitForHealthy(ctx context.Context, resource types.Resource, timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.WaitForHealthyCalls++
	if m.WaitForHealthyError != nil {
		return m.WaitForHealthyError
	}

	return nil
}

// CollectMetrics collects metrics for a resource
func (m *MockMonitor) CollectMetrics(ctx context.Context, resource types.Resource) (types.Metrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CollectMetricsCalls++
	if m.CollectMetricsError != nil {
		return types.Metrics{}, m.CollectMetricsError
	}

	if m.MetricsToReturn.ResourceID != "" {
		return m.MetricsToReturn, nil
	}

	return types.Metrics{
		Timestamp:  time.Now(),
		ResourceID: resource.ID,
		CPU: types.CPUMetrics{
			UsagePercent: 50.0,
		},
	}, nil
}

// WatchEvents watches for events on a resource
func (m *MockMonitor) WatchEvents(ctx context.Context, resource types.Resource) (<-chan types.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.WatchEventsCalls++
	if m.WatchEventsError != nil {
		return nil, m.WatchEventsError
	}

	ch := make(chan types.Event, 10)
	go func() {
		<-ctx.Done()
		close(ch)
	}()

	return ch, nil
}

// RegisterHealthCheckStrategy registers a health check strategy
func (m *MockMonitor) RegisterHealthCheckStrategy(name string, strategy types.HealthCheckStrategy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RegisterStrategyCalls++
	if m.RegisterStrategyError != nil {
		return m.RegisterStrategyError
	}

	m.HealthCheckStrategies[name] = strategy
	return nil
}

// GetHealthCheckStrategy gets a health check strategy by name
func (m *MockMonitor) GetHealthCheckStrategy(name string) (types.HealthCheckStrategy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetStrategyCalls++
	if m.GetStrategyError != nil {
		return nil, m.GetStrategyError
	}

	strategy, exists := m.HealthCheckStrategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy not found: %s", name)
	}

	return strategy, nil
}

// ============================================================================
// MockReporter
// ============================================================================

// MockReporter is a mock implementation of Reporter for testing
type MockReporter struct {
	mu sync.RWMutex

	// State
	Events     []types.Event
	Executions []types.ExecutionRecord
	Storage    types.StorageBackend
	Formatters map[string]types.ReportFormatter

	// Behavior configuration
	RecordEventError       error
	RecordExecutionError   error
	QueryEventsError       error
	QueryExecutionsError   error
	ComputeStatisticsError error
	GenerateReportError    error
	SetStorageError        error
	RegisterFormatterError error

	// Return values
	StatisticsToReturn types.Statistics

	// Call tracking
	RecordEventCalls       int
	RecordExecutionCalls   int
	QueryEventsCalls       int
	QueryExecutionsCalls   int
	ComputeStatisticsCalls int
	GenerateReportCalls    int
	SetStorageCalls        int
	RegisterFormatterCalls int
}

// NewMockReporter creates a new MockReporter
func NewMockReporter() *MockReporter {
	return &MockReporter{
		Events:     make([]types.Event, 0),
		Executions: make([]types.ExecutionRecord, 0),
		Formatters: make(map[string]types.ReportFormatter),
	}
}

// RecordEvent records an event
func (m *MockReporter) RecordEvent(ctx context.Context, event types.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecordEventCalls++
	if m.RecordEventError != nil {
		return m.RecordEventError
	}

	m.Events = append(m.Events, event)
	return nil
}

// RecordExecution records an execution
func (m *MockReporter) RecordExecution(ctx context.Context, exec types.ExecutionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecordExecutionCalls++
	if m.RecordExecutionError != nil {
		return m.RecordExecutionError
	}

	m.Executions = append(m.Executions, exec)
	return nil
}

// QueryEvents queries events
func (m *MockReporter) QueryEvents(ctx context.Context, query types.EventQuery) ([]types.Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.QueryEventsCalls++
	if m.QueryEventsError != nil {
		return nil, m.QueryEventsError
	}

	return m.Events, nil
}

// QueryExecutions queries executions
func (m *MockReporter) QueryExecutions(ctx context.Context, query types.ExecutionQuery) ([]types.ExecutionRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.QueryExecutionsCalls++
	if m.QueryExecutionsError != nil {
		return nil, m.QueryExecutionsError
	}

	return m.Executions, nil
}

// ComputeStatistics computes statistics
func (m *MockReporter) ComputeStatistics(ctx context.Context, filter types.StatisticsFilter) (types.Statistics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ComputeStatisticsCalls++
	if m.ComputeStatisticsError != nil {
		return types.Statistics{}, m.ComputeStatisticsError
	}

	if m.StatisticsToReturn.TotalExecutions > 0 {
		return m.StatisticsToReturn, nil
	}

	return types.Statistics{
		TotalExecutions: len(m.Executions),
		SuccessCount:    len(m.Executions),
		SuccessRate:     1.0,
	}, nil
}

// GenerateReport generates a report
func (m *MockReporter) GenerateReport(ctx context.Context, format string, filter types.ReportFilter) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GenerateReportCalls++
	if m.GenerateReportError != nil {
		return nil, m.GenerateReportError
	}

	return []byte(fmt.Sprintf("Report in %s format", format)), nil
}

// SetStorage sets the storage backend
func (m *MockReporter) SetStorage(storage types.StorageBackend) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SetStorageCalls++
	if m.SetStorageError != nil {
		return m.SetStorageError
	}

	m.Storage = storage
	return nil
}

// RegisterFormatter registers a report formatter
func (m *MockReporter) RegisterFormatter(name string, formatter types.ReportFormatter) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RegisterFormatterCalls++
	if m.RegisterFormatterError != nil {
		return m.RegisterFormatterError
	}

	m.Formatters[name] = formatter
	return nil
}

// ============================================================================
// MockEventBus
// ============================================================================

// MockEventBus is a mock implementation of EventBus for testing
type MockEventBus struct {
	mu sync.RWMutex

	// State
	Subscriptions   map[string]*MockSubscription
	PublishedEvents []types.Event
	Closed          bool

	// Behavior configuration
	PublishError      error
	PublishAsyncError error
	SubscribeError    error

	// Call tracking
	PublishCalls      int
	PublishAsyncCalls int
	SubscribeCalls    int
	CloseCalls        int
}

// NewMockEventBus creates a new MockEventBus
func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		Subscriptions:   make(map[string]*MockSubscription),
		PublishedEvents: make([]types.Event, 0),
	}
}

// Publish publishes an event synchronously
func (m *MockEventBus) Publish(ctx context.Context, event types.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PublishCalls++
	if m.PublishError != nil {
		return m.PublishError
	}

	m.PublishedEvents = append(m.PublishedEvents, event)

	// Deliver to matching subscriptions
	for _, sub := range m.Subscriptions {
		if sub.matchesFilter(event) {
			select {
			case sub.eventChan <- event:
			default:
				// Channel full, skip
			}
		}
	}

	return nil
}

// PublishAsync publishes an event asynchronously
func (m *MockEventBus) PublishAsync(ctx context.Context, event types.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PublishAsyncCalls++
	if m.PublishAsyncError != nil {
		return m.PublishAsyncError
	}

	go func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.PublishedEvents = append(m.PublishedEvents, event)

		for _, sub := range m.Subscriptions {
			if sub.matchesFilter(event) {
				select {
				case sub.eventChan <- event:
				default:
				}
			}
		}
	}()

	return nil
}

// Subscribe subscribes to events
func (m *MockEventBus) Subscribe(ctx context.Context, filter types.EventFilter) (types.Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SubscribeCalls++
	if m.SubscribeError != nil {
		return nil, m.SubscribeError
	}

	sub := &MockSubscription{
		id:        fmt.Sprintf("sub-%d", len(m.Subscriptions)+1),
		filter:    filter,
		eventChan: make(chan types.Event, 100),
		bus:       m,
	}

	m.Subscriptions[sub.id] = sub
	return sub, nil
}

// Close closes the event bus
func (m *MockEventBus) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CloseCalls++
	m.Closed = true

	for _, sub := range m.Subscriptions {
		close(sub.eventChan)
	}
	m.Subscriptions = make(map[string]*MockSubscription)

	return nil
}

// ============================================================================
// MockSubscription
// ============================================================================

// MockSubscription is a mock implementation of Subscription
type MockSubscription struct {
	id        string
	filter    types.EventFilter
	eventChan chan types.Event
	bus       *MockEventBus
}

// ID returns the subscription ID
func (m *MockSubscription) ID() string {
	return m.id
}

// Events returns the event channel
func (m *MockSubscription) Events() <-chan types.Event {
	return m.eventChan
}

// Unsubscribe unsubscribes from events
func (m *MockSubscription) Unsubscribe() error {
	m.bus.mu.Lock()
	defer m.bus.mu.Unlock()

	delete(m.bus.Subscriptions, m.id)
	close(m.eventChan)

	return nil
}

func (m *MockSubscription) matchesFilter(event types.Event) bool {
	// Check types
	if len(m.filter.Types) > 0 {
		found := false
		for _, t := range m.filter.Types {
			if event.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check sources
	if len(m.filter.Sources) > 0 {
		found := false
		for _, s := range m.filter.Sources {
			if event.Source == s {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check metadata
	for key, value := range m.filter.Metadata {
		if event.Metadata[key] != value {
			return false
		}
	}

	return true
}

// ============================================================================
// Mock PluginLoader Implementation
// ============================================================================

// MockPluginLoader is a mock implementation of PluginLoader for testing
type MockPluginLoader struct {
	mu      sync.RWMutex
	plugins map[string]types.Plugin

	// Behavior configuration
	LoadError   error
	UnloadError error

	// Call tracking
	LoadCalls   int
	UnloadCalls int
}

// NewMockPluginLoader creates a new MockPluginLoader
func NewMockPluginLoader() *MockPluginLoader {
	return &MockPluginLoader{
		plugins: make(map[string]types.Plugin),
	}
}

// Load loads a plugin from the specified path
func (l *MockPluginLoader) Load(path string) (types.Plugin, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.LoadCalls++

	if l.LoadError != nil {
		return nil, l.LoadError
	}

	plugin, exists := l.plugins[path]
	if !exists {
		return nil, fmt.Errorf("plugin not found at path: %s", path)
	}

	return plugin, nil
}

// Unload unloads a plugin
func (l *MockPluginLoader) Unload(plugin types.Plugin) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.UnloadCalls++

	if l.UnloadError != nil {
		return l.UnloadError
	}

	return nil
}

// AddPlugin adds a plugin to the mock loader for testing
func (l *MockPluginLoader) AddPlugin(path string, plugin types.Plugin) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.plugins[path] = plugin
}
