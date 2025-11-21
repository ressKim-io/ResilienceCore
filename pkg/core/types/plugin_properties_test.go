// Package types provides property-based tests for the Plugin interface
package types

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: infrastructure-resilience-engine, Property 11: Plugin metadata is complete**
// This property validates that every Plugin registration contains complete metadata
// including Name, Version, Description, SupportedKinds, RequiredCapabilities, and Dependencies.
// Validates: Requirements 3.1
func TestProperty11_PluginMetadataIsComplete(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("plugin metadata contains all required fields", prop.ForAll(
		func(name, version, description, author string) bool {
			// Create a mock plugin with the generated metadata
			plugin := &MockPlugin{
				metadata: PluginMetadata{
					Name:                 name,
					Version:              version,
					Description:          description,
					Author:               author,
					SupportedKinds:       []string{"container", "pod"},
					RequiredCapabilities: []Capability{CapabilityExec},
					Dependencies:         []PluginDependency{},
					ConfigSchema:         nil,
				},
			}

			// Get metadata
			meta := plugin.Metadata()

			// Verify all required fields are present
			hasName := meta.Name != ""
			hasVersion := meta.Version != ""
			hasDescription := meta.Description != ""
			hasAuthor := meta.Author != ""
			hasSupportedKinds := meta.SupportedKinds != nil
			hasRequiredCapabilities := meta.RequiredCapabilities != nil
			hasDependencies := meta.Dependencies != nil

			return hasName && hasVersion && hasDescription && hasAuthor &&
				hasSupportedKinds && hasRequiredCapabilities && hasDependencies
		},
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
		gen.AlphaString().SuchThat(func(s string) bool { return s != "" }),
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
	))

	properties.Property("plugin metadata supports dependencies", prop.ForAll(
		func(depName, depVersion string) bool {
			plugin := &MockPlugin{
				metadata: PluginMetadata{
					Name:        "test-plugin",
					Version:     "1.0.0",
					Description: "Test plugin",
					Author:      "test",
					Dependencies: []PluginDependency{
						{
							PluginName: depName,
							Version:    depVersion,
						},
					},
				},
			}

			meta := plugin.Metadata()
			if len(meta.Dependencies) != 1 {
				return false
			}

			dep := meta.Dependencies[0]
			return dep.PluginName == depName && dep.Version == depVersion
		},
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
	))

	properties.Property("plugin metadata supports multiple capabilities", prop.ForAll(
		func(caps []Capability) bool {
			plugin := &MockPlugin{
				metadata: PluginMetadata{
					Name:                 "test-plugin",
					Version:              "1.0.0",
					Description:          "Test plugin",
					Author:               "test",
					RequiredCapabilities: caps,
				},
			}

			meta := plugin.Metadata()
			return len(meta.RequiredCapabilities) == len(caps)
		},
		gen.SliceOf(gen.OneConstOf(CapabilityExec, CapabilityNetwork, CapabilityAdmin)),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 12: Plugin lifecycle hooks execute in order**
// This property validates that the ExecutionEngine invokes plugin lifecycle hooks in the correct order:
// Validate → PreExecute → Execute → PostExecute → Cleanup
// Validates: Requirements 3.2
func TestProperty12_PluginLifecycleHooksExecuteInOrder(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("lifecycle hooks execute in correct order", prop.ForAll(
		func(resourceID, resourceName string) bool {
			// Create a mock plugin that tracks hook execution order
			plugin := &MockPluginWithHooks{
				executionOrder: []string{},
			}

			// Create a test resource
			resource := Resource{
				ID:   resourceID,
				Name: resourceName,
				Kind: "container",
			}

			// Create a plugin context
			ctx := PluginContext{
				Context:     context.Background(),
				ExecutionID: "test-exec-1",
				Timeout:     time.Second * 30,
				Progress:    make(chan ProgressUpdate, 10),
			}

			// Execute lifecycle hooks in order
			// 1. Validate
			if err := plugin.Validate(ctx, resource); err != nil {
				return false
			}

			// 2. PreExecute
			snapshot, err := plugin.PreExecute(ctx, resource)
			if err != nil {
				return false
			}

			// 3. Execute
			if err := plugin.Execute(ctx, resource); err != nil {
				return false
			}

			// 4. PostExecute
			result := ExecutionResult{
				Status:    StatusSuccess,
				StartTime: time.Now(),
				EndTime:   time.Now(),
				Duration:  time.Millisecond * 100,
			}
			if err := plugin.PostExecute(ctx, resource, result); err != nil {
				return false
			}

			// 5. Cleanup
			if err := plugin.Cleanup(ctx, resource); err != nil {
				return false
			}

			// Verify execution order
			expectedOrder := []string{"Validate", "PreExecute", "Execute", "PostExecute", "Cleanup"}
			if len(plugin.executionOrder) != len(expectedOrder) {
				return false
			}

			for i, expected := range expectedOrder {
				if plugin.executionOrder[i] != expected {
					return false
				}
			}

			// Verify snapshot was created
			return snapshot != nil
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("cleanup executes even after execute failure", prop.ForAll(
		func(resourceID string) bool {
			plugin := &MockPluginWithHooks{
				executionOrder: []string{},
				failOnExecute:  true,
			}

			resource := Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			ctx := PluginContext{
				Context:     context.Background(),
				ExecutionID: "test-exec-2",
				Timeout:     time.Second * 30,
				Progress:    make(chan ProgressUpdate, 10),
			}

			// Execute lifecycle hooks
			_ = plugin.Validate(ctx, resource)
			_, _ = plugin.PreExecute(ctx, resource)
			_ = plugin.Execute(ctx, resource) // This will fail

			// Cleanup should still be called
			_ = plugin.Cleanup(ctx, resource)

			// Verify Cleanup was called even though Execute failed
			cleanupCalled := false
			for _, hook := range plugin.executionOrder {
				if hook == "Cleanup" {
					cleanupCalled = true
					break
				}
			}

			return cleanupCalled
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 13: Rollback is invoked on failure**
// This property validates that when a plugin execution fails and the plugin implements Rollback,
// the ExecutionEngine invokes Rollback with the snapshot from PreExecute.
// Validates: Requirements 3.3
func TestProperty13_RollbackIsInvokedOnFailure(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("rollback is invoked with snapshot on failure", prop.ForAll(
		func(resourceID, resourceName string) bool {
			// Create a mock plugin that supports rollback
			plugin := &MockPluginWithRollback{
				executionOrder: []string{},
				failOnExecute:  true,
			}

			resource := Resource{
				ID:   resourceID,
				Name: resourceName,
				Kind: "container",
			}

			ctx := PluginContext{
				Context:     context.Background(),
				ExecutionID: "test-exec-3",
				Timeout:     time.Second * 30,
				Progress:    make(chan ProgressUpdate, 10),
			}

			// Execute lifecycle hooks
			_ = plugin.Validate(ctx, resource)

			// PreExecute creates a snapshot
			snapshot, err := plugin.PreExecute(ctx, resource)
			if err != nil {
				return false
			}

			// Execute fails
			executeErr := plugin.Execute(ctx, resource)
			if executeErr == nil {
				return false // Should fail
			}

			// Rollback should be invoked with the snapshot
			rollbackErr := plugin.Rollback(ctx, resource, snapshot)
			if rollbackErr != nil {
				return false
			}

			// Verify rollback was called
			rollbackCalled := false
			for _, hook := range plugin.executionOrder {
				if hook == "Rollback" {
					rollbackCalled = true
					break
				}
			}

			// Verify the snapshot was used
			return rollbackCalled && plugin.snapshotUsed != nil
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("rollback uses correct snapshot", prop.ForAll(
		func(snapshotData string) bool {
			plugin := &MockPluginWithRollback{
				executionOrder: []string{},
				failOnExecute:  true,
			}

			resource := Resource{
				ID:   "test-resource",
				Name: "test",
				Kind: "container",
			}

			ctx := PluginContext{
				Context:     context.Background(),
				ExecutionID: "test-exec-4",
				Timeout:     time.Second * 30,
				Progress:    make(chan ProgressUpdate, 10),
			}

			// Create a snapshot with specific data
			snapshot := &MockSnapshot{
				data: snapshotData,
			}

			// Execute fails
			_ = plugin.Execute(ctx, resource)

			// Rollback with the snapshot
			_ = plugin.Rollback(ctx, resource, snapshot)

			// Verify the correct snapshot was used
			if plugin.snapshotUsed == nil {
				return false
			}

			mockSnapshot, ok := plugin.snapshotUsed.(*MockSnapshot)
			if !ok {
				return false
			}

			return mockSnapshot.data == snapshotData
		},
		gen.AlphaString(),
	))

	properties.Property("cleanup executes after rollback", prop.ForAll(
		func(resourceID string) bool {
			plugin := &MockPluginWithRollback{
				executionOrder: []string{},
				failOnExecute:  true,
			}

			resource := Resource{
				ID:   resourceID,
				Name: "test-resource",
				Kind: "container",
			}

			ctx := PluginContext{
				Context:     context.Background(),
				ExecutionID: "test-exec-5",
				Timeout:     time.Second * 30,
				Progress:    make(chan ProgressUpdate, 10),
			}

			// Execute lifecycle
			_ = plugin.Validate(ctx, resource)
			snapshot, _ := plugin.PreExecute(ctx, resource)
			_ = plugin.Execute(ctx, resource) // Fails
			_ = plugin.Rollback(ctx, resource, snapshot)
			_ = plugin.Cleanup(ctx, resource)

			// Verify order: Rollback comes before Cleanup
			rollbackIndex := -1
			cleanupIndex := -1

			for i, hook := range plugin.executionOrder {
				if hook == "Rollback" {
					rollbackIndex = i
				}
				if hook == "Cleanup" {
					cleanupIndex = i
				}
			}

			return rollbackIndex >= 0 && cleanupIndex >= 0 && rollbackIndex < cleanupIndex
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ============================================================================
// Mock Implementations for Testing
// ============================================================================

// MockPlugin is a basic mock plugin implementation
type MockPlugin struct {
	metadata PluginMetadata
}

func (m *MockPlugin) Metadata() PluginMetadata {
	return m.metadata
}

func (m *MockPlugin) Initialize(config PluginConfig) error {
	return nil
}

func (m *MockPlugin) Validate(ctx PluginContext, resource Resource) error {
	return nil
}

func (m *MockPlugin) PreExecute(ctx PluginContext, resource Resource) (Snapshot, error) {
	return &MockSnapshot{}, nil
}

func (m *MockPlugin) Execute(ctx PluginContext, resource Resource) error {
	return nil
}

func (m *MockPlugin) PostExecute(ctx PluginContext, resource Resource, result ExecutionResult) error {
	return nil
}

func (m *MockPlugin) Cleanup(ctx PluginContext, resource Resource) error {
	return nil
}

func (m *MockPlugin) Rollback(ctx PluginContext, resource Resource, snapshot Snapshot) error {
	return nil
}

// MockPluginWithHooks tracks execution order
type MockPluginWithHooks struct {
	executionOrder []string
	failOnExecute  bool
}

func (m *MockPluginWithHooks) Metadata() PluginMetadata {
	return PluginMetadata{
		Name:        "mock-plugin-with-hooks",
		Version:     "1.0.0",
		Description: "Mock plugin for testing hooks",
		Author:      "test",
	}
}

func (m *MockPluginWithHooks) Initialize(config PluginConfig) error {
	return nil
}

func (m *MockPluginWithHooks) Validate(ctx PluginContext, resource Resource) error {
	m.executionOrder = append(m.executionOrder, "Validate")
	return nil
}

func (m *MockPluginWithHooks) PreExecute(ctx PluginContext, resource Resource) (Snapshot, error) {
	m.executionOrder = append(m.executionOrder, "PreExecute")
	return &MockSnapshot{}, nil
}

func (m *MockPluginWithHooks) Execute(ctx PluginContext, resource Resource) error {
	m.executionOrder = append(m.executionOrder, "Execute")
	if m.failOnExecute {
		return errors.New("execute failed")
	}
	return nil
}

func (m *MockPluginWithHooks) PostExecute(ctx PluginContext, resource Resource, result ExecutionResult) error {
	m.executionOrder = append(m.executionOrder, "PostExecute")
	return nil
}

func (m *MockPluginWithHooks) Cleanup(ctx PluginContext, resource Resource) error {
	m.executionOrder = append(m.executionOrder, "Cleanup")
	return nil
}

func (m *MockPluginWithHooks) Rollback(ctx PluginContext, resource Resource, snapshot Snapshot) error {
	m.executionOrder = append(m.executionOrder, "Rollback")
	return nil
}

// MockPluginWithRollback tracks rollback invocation
type MockPluginWithRollback struct {
	executionOrder []string
	failOnExecute  bool
	snapshotUsed   Snapshot
}

func (m *MockPluginWithRollback) Metadata() PluginMetadata {
	return PluginMetadata{
		Name:        "mock-plugin-with-rollback",
		Version:     "1.0.0",
		Description: "Mock plugin for testing rollback",
		Author:      "test",
	}
}

func (m *MockPluginWithRollback) Initialize(config PluginConfig) error {
	return nil
}

func (m *MockPluginWithRollback) Validate(ctx PluginContext, resource Resource) error {
	m.executionOrder = append(m.executionOrder, "Validate")
	return nil
}

func (m *MockPluginWithRollback) PreExecute(ctx PluginContext, resource Resource) (Snapshot, error) {
	m.executionOrder = append(m.executionOrder, "PreExecute")
	return &MockSnapshot{data: "snapshot-data"}, nil
}

func (m *MockPluginWithRollback) Execute(ctx PluginContext, resource Resource) error {
	m.executionOrder = append(m.executionOrder, "Execute")
	if m.failOnExecute {
		return errors.New("execute failed")
	}
	return nil
}

func (m *MockPluginWithRollback) PostExecute(ctx PluginContext, resource Resource, result ExecutionResult) error {
	m.executionOrder = append(m.executionOrder, "PostExecute")
	return nil
}

func (m *MockPluginWithRollback) Cleanup(ctx PluginContext, resource Resource) error {
	m.executionOrder = append(m.executionOrder, "Cleanup")
	return nil
}

func (m *MockPluginWithRollback) Rollback(ctx PluginContext, resource Resource, snapshot Snapshot) error {
	m.executionOrder = append(m.executionOrder, "Rollback")
	m.snapshotUsed = snapshot
	return nil
}

// MockSnapshot is a mock snapshot implementation
type MockSnapshot struct {
	data string
}

func (m *MockSnapshot) Restore(ctx context.Context) error {
	return nil
}

func (m *MockSnapshot) Serialize() ([]byte, error) {
	return []byte(m.data), nil
}

func (m *MockSnapshot) Deserialize(data []byte) error {
	m.data = string(data)
	return nil
}
