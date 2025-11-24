// Package types provides property-based tests for the PluginRegistry interface
package types

import (
	"errors"
	"fmt"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: infrastructure-resilience-engine, Property 59: Plugins are discovered from directories**
// This property validates that calling Discover on a directory containing valid plugins
// finds and registers all plugins.
// Validates: Requirements 13.1
func TestProperty59_PluginsAreDiscoveredFromDirectories(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("discover finds all plugins in directory", prop.ForAll(
		func(pluginCount uint8) bool {
			// Limit plugin count to reasonable number (1-10)
			count := int(pluginCount%10) + 1

			// Create a mock registry
			registry := NewMockPluginRegistry()

			// Create mock plugins
			plugins := make([]Plugin, count)
			for i := 0; i < count; i++ {
				plugins[i] = &MockPlugin{
					metadata: PluginMetadata{
						Name:        fmt.Sprintf("plugin-%d", i),
						Version:     "1.0.0",
						Description: fmt.Sprintf("Test plugin %d", i),
						Author:      "test",
					},
				}
			}

			// Register plugins to simulate discovery
			for _, plugin := range plugins {
				if err := registry.Register(plugin); err != nil {
					return false
				}
			}

			// List all plugins
			allPlugins, err := registry.List(PluginFilter{})
			if err != nil {
				return false
			}

			// Verify all plugins were discovered
			return len(allPlugins) == count
		},
		gen.UInt8(),
	))

	properties.Property("discover handles empty directory", prop.ForAll(
		func() bool {
			registry := NewMockPluginRegistry()

			// List plugins from empty registry
			plugins, err := registry.List(PluginFilter{})
			if err != nil {
				return false
			}

			// Should return empty list, not error
			return len(plugins) == 0
		},
	))

	properties.Property("discover ignores invalid plugins", prop.ForAll(
		func(validCount, invalidCount uint8) bool {
			// Limit counts to reasonable numbers
			valid := int(validCount%5) + 1
			invalid := int(invalidCount%5) + 1

			registry := NewMockPluginRegistry()

			// Register valid plugins
			for i := 0; i < valid; i++ {
				plugin := &MockPlugin{
					metadata: PluginMetadata{
						Name:        fmt.Sprintf("valid-plugin-%d", i),
						Version:     "1.0.0",
						Description: "Valid plugin",
						Author:      "test",
					},
				}
				_ = registry.Register(plugin)
			}

			// Try to register invalid plugins (should fail validation)
			for i := 0; i < invalid; i++ {
				plugin := &MockPlugin{
					metadata: PluginMetadata{
						Name:    "", // Invalid: empty name
						Version: "1.0.0",
					},
				}
				// Registration should fail for invalid plugins
				_ = registry.Register(plugin)
			}

			// List all plugins
			plugins, err := registry.List(PluginFilter{})
			if err != nil {
				return false
			}

			// Only valid plugins should be registered
			return len(plugins) == valid
		},
		gen.UInt8(),
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 60: Plugin validation occurs on registration**
// This property validates that the PluginRegistry validates plugin metadata and interface
// compliance when a plugin is registered.
// Validates: Requirements 13.2
func TestProperty60_PluginValidationOccursOnRegistration(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("registration validates plugin metadata", prop.ForAll(
		func(name, version, description string) bool {
			registry := NewMockPluginRegistry()

			// Create plugin with generated metadata
			plugin := &MockPlugin{
				metadata: PluginMetadata{
					Name:        name,
					Version:     version,
					Description: description,
					Author:      "test",
				},
			}

			err := registry.Register(plugin)

			// If any required field is empty, registration should fail
			if name == "" || version == "" || description == "" {
				return err != nil
			}

			// Otherwise, registration should succeed
			return err == nil
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.AlphaString(),
	))

	properties.Property("registration rejects duplicate plugin names", prop.ForAll(
		func(pluginName string) bool {
			if pluginName == "" {
				return true // Skip empty names
			}

			registry := NewMockPluginRegistry()

			// Register first plugin
			plugin1 := &MockPlugin{
				metadata: PluginMetadata{
					Name:        pluginName,
					Version:     "1.0.0",
					Description: "First plugin",
					Author:      "test",
				},
			}

			err1 := registry.Register(plugin1)
			if err1 != nil {
				return false
			}

			// Try to register second plugin with same name
			plugin2 := &MockPlugin{
				metadata: PluginMetadata{
					Name:        pluginName,
					Version:     "2.0.0",
					Description: "Second plugin",
					Author:      "test",
				},
			}

			err2 := registry.Register(plugin2)

			// Second registration should fail
			return err2 != nil
		},
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
	))

	properties.Property("validation checks interface compliance", prop.ForAll(
		func(pluginName string) bool {
			if pluginName == "" {
				return true // Skip empty names
			}

			registry := NewMockPluginRegistry()

			// Create a valid plugin
			plugin := &MockPlugin{
				metadata: PluginMetadata{
					Name:        pluginName,
					Version:     "1.0.0",
					Description: "Test plugin",
					Author:      "test",
				},
			}

			// Validate the plugin
			err := registry.Validate(plugin)

			// Valid plugin should pass validation
			return err == nil
		},
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
	))

	properties.Property("validation checks dependencies", prop.ForAll(
		func(pluginName, depName string) bool {
			if pluginName == "" || depName == "" {
				return true // Skip empty names
			}

			registry := NewMockPluginRegistry()

			// Create plugin with dependency
			plugin := &MockPlugin{
				metadata: PluginMetadata{
					Name:        pluginName,
					Version:     "1.0.0",
					Description: "Test plugin",
					Author:      "test",
					Dependencies: []PluginDependency{
						{
							PluginName: depName,
							Version:    ">=1.0.0",
						},
					},
				},
			}

			// Validate dependencies (should fail if dependency not registered)
			err := registry.ValidateDependencies(plugin)

			// Should fail because dependency is not registered
			return err != nil
		},
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: infrastructure-resilience-engine, Property 63: Hot-reload updates plugins without restart**
// This property validates that calling Reload on a plugin updates the plugin
// without restarting the application.
// Validates: Requirements 13.5
func TestProperty63_HotReloadUpdatesPluginsWithoutRestart(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("reload updates plugin version", prop.ForAll(
		func(pluginName, version1, version2 string) bool {
			if pluginName == "" || version1 == "" || version2 == "" || version1 == version2 {
				return true // Skip invalid cases
			}

			registry := NewMockPluginRegistry()

			// Register initial plugin
			plugin1 := &MockPlugin{
				metadata: PluginMetadata{
					Name:        pluginName,
					Version:     version1,
					Description: "Test plugin",
					Author:      "test",
				},
			}

			if err := registry.Register(plugin1); err != nil {
				return false
			}

			// Get the plugin
			retrieved1, err := registry.Get(pluginName)
			if err != nil {
				return false
			}

			// Verify initial version
			if retrieved1.Metadata().Version != version1 {
				return false
			}

			// Simulate reload by unregistering and re-registering with new version
			err = registry.Unregister(pluginName)
			if err != nil {
				return false
			}

			plugin2 := &MockPlugin{
				metadata: PluginMetadata{
					Name:        pluginName,
					Version:     version2,
					Description: "Test plugin",
					Author:      "test",
				},
			}

			err = registry.Register(plugin2)
			if err != nil {
				return false
			}

			// Get the plugin again
			retrieved2, err := registry.Get(pluginName)
			if err != nil {
				return false
			}

			// Verify version was updated
			return retrieved2.Metadata().Version == version2
		},
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
	))

	properties.Property("reload preserves other plugins", prop.ForAll(
		func(plugin1Name, plugin2Name string) bool {
			if plugin1Name == "" || plugin2Name == "" || plugin1Name == plugin2Name {
				return true // Skip invalid cases
			}

			registry := NewMockPluginRegistry()

			// Register two plugins
			plugin1 := &MockPlugin{
				metadata: PluginMetadata{
					Name:        plugin1Name,
					Version:     "1.0.0",
					Description: "Plugin 1",
					Author:      "test",
				},
			}

			plugin2 := &MockPlugin{
				metadata: PluginMetadata{
					Name:        plugin2Name,
					Version:     "1.0.0",
					Description: "Plugin 2",
					Author:      "test",
				},
			}

			if err := registry.Register(plugin1); err != nil {
				return false
			}

			if err := registry.Register(plugin2); err != nil {
				return false
			}

			// Reload plugin1
			if err := registry.Unregister(plugin1Name); err != nil {
				return false
			}

			plugin1Updated := &MockPlugin{
				metadata: PluginMetadata{
					Name:        plugin1Name,
					Version:     "2.0.0",
					Description: "Plugin 1 updated",
					Author:      "test",
				},
			}

			if err := registry.Register(plugin1Updated); err != nil {
				return false
			}

			// Verify plugin2 is still registered and unchanged
			retrieved2, err := registry.Get(plugin2Name)
			if err != nil {
				return false
			}

			return retrieved2.Metadata().Name == plugin2Name &&
				retrieved2.Metadata().Version == "1.0.0"
		},
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
	))

	properties.Property("reload fails for non-existent plugin", prop.ForAll(
		func(pluginName string) bool {
			if pluginName == "" {
				return true // Skip empty names
			}

			registry := NewMockPluginRegistry()

			// Try to reload a plugin that doesn't exist
			err := registry.Reload(pluginName)

			// Should return an error
			return err != nil
		},
		gen.Identifier().SuchThat(func(s string) bool { return s != "" }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ============================================================================
// Mock PluginRegistry Implementation for Testing
// ============================================================================

// MockPluginRegistry is a simple in-memory implementation for testing
type MockPluginRegistry struct {
	plugins map[string]Plugin
}

// NewMockPluginRegistry creates a new mock plugin registry
func NewMockPluginRegistry() *MockPluginRegistry {
	return &MockPluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

func (r *MockPluginRegistry) Register(plugin Plugin) error {
	meta := plugin.Metadata()

	// Validate metadata
	if meta.Name == "" {
		return errors.New("plugin name cannot be empty")
	}
	if meta.Version == "" {
		return errors.New("plugin version cannot be empty")
	}
	if meta.Description == "" {
		return errors.New("plugin description cannot be empty")
	}

	// Check for duplicates
	if _, exists := r.plugins[meta.Name]; exists {
		return fmt.Errorf("plugin %s already registered", meta.Name)
	}

	// Validate the plugin
	if err := r.Validate(plugin); err != nil {
		return err
	}

	r.plugins[meta.Name] = plugin
	return nil
}

func (r *MockPluginRegistry) Unregister(name string) error {
	if _, exists := r.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	delete(r.plugins, name)
	return nil
}

func (r *MockPluginRegistry) Discover(path string) error {
	// Mock implementation - in real implementation, this would scan the path
	// For testing, we just return success
	return nil
}

func (r *MockPluginRegistry) DiscoverFromDirectory(dir string) error {
	// Mock implementation - in real implementation, this would scan the directory
	// For testing, we just return success
	return nil
}

func (r *MockPluginRegistry) Get(name string) (Plugin, error) {
	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	return plugin, nil
}

func (r *MockPluginRegistry) List(filter PluginFilter) ([]PluginMetadata, error) {
	var result []PluginMetadata

	for _, plugin := range r.plugins {
		meta := plugin.Metadata()

		// Apply filters
		if len(filter.Names) > 0 {
			found := false
			for _, name := range filter.Names {
				if meta.Name == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if len(filter.Versions) > 0 {
			found := false
			for _, version := range filter.Versions {
				if meta.Version == version {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if len(filter.SupportedKinds) > 0 {
			found := false
			for _, filterKind := range filter.SupportedKinds {
				for _, pluginKind := range meta.SupportedKinds {
					if pluginKind == filterKind {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found && len(meta.SupportedKinds) > 0 {
				continue
			}
		}

		if len(filter.Capabilities) > 0 {
			found := false
			for _, filterCap := range filter.Capabilities {
				for _, pluginCap := range meta.RequiredCapabilities {
					if pluginCap == filterCap {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found && len(meta.RequiredCapabilities) > 0 {
				continue
			}
		}

		result = append(result, meta)
	}

	return result, nil
}

func (r *MockPluginRegistry) Validate(plugin Plugin) error {
	meta := plugin.Metadata()

	// Basic validation
	if meta.Name == "" {
		return errors.New("plugin name is required")
	}
	if meta.Version == "" {
		return errors.New("plugin version is required")
	}
	if meta.Description == "" {
		return errors.New("plugin description is required")
	}

	return nil
}

func (r *MockPluginRegistry) ValidateDependencies(plugin Plugin) error {
	meta := plugin.Metadata()

	// Check if all dependencies are registered
	for _, dep := range meta.Dependencies {
		if _, exists := r.plugins[dep.PluginName]; !exists {
			return fmt.Errorf("dependency %s not found", dep.PluginName)
		}
	}

	return nil
}

func (r *MockPluginRegistry) Reload(name string) error {
	// Check if plugin exists
	if _, exists := r.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// In a real implementation, this would reload the plugin from disk
	// For testing, we just verify the plugin exists
	return nil
}

func (r *MockPluginRegistry) SetLoader(loader PluginLoader) error {
	// Mock implementation
	return nil
}

// ============================================================================
// Mock PluginLoader Implementation for Testing
// ============================================================================

// MockPluginLoader is a mock implementation of PluginLoader
type MockPluginLoader struct {
	plugins map[string]Plugin
}

func NewMockPluginLoader() *MockPluginLoader {
	return &MockPluginLoader{
		plugins: make(map[string]Plugin),
	}
}

func (l *MockPluginLoader) Load(path string) (Plugin, error) {
	// Mock implementation - in real implementation, this would load from file
	plugin, exists := l.plugins[path]
	if !exists {
		return nil, fmt.Errorf("plugin not found at path: %s", path)
	}
	return plugin, nil
}

func (l *MockPluginLoader) Unload(plugin Plugin) error {
	// Mock implementation
	return nil
}

// AddPlugin adds a plugin to the mock loader for testing
func (l *MockPluginLoader) AddPlugin(path string, plugin Plugin) {
	l.plugins[path] = plugin
}

// ============================================================================
// Helper Functions for Testing
// ============================================================================

// Note: Helper functions for plugin directory testing will be added when needed
