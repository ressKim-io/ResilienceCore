package registry

import (
	"testing"

	coretesting "github.com/yourusername/infrastructure-resilience-engine/pkg/core/testing"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// TestRegister tests plugin registration
func TestRegister(t *testing.T) {
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
		{
			name: "plugin with empty name",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "",
					Version:     "1.0.0",
					Description: "Test",
				},
			},
			wantErr: true,
		},
		{
			name: "plugin with empty version",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "test",
					Version:     "",
					Description: "Test",
				},
			},
			wantErr: true,
		},
		{
			name: "plugin with empty description",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "test",
					Version:     "1.0.0",
					Description: "",
				},
			},
			wantErr: true,
		},
		{
			name: "plugin with invalid version",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "test",
					Version:     "invalid",
					Description: "Test",
				},
			},
			wantErr: true,
		},
		{
			name: "plugin with invalid name characters",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "test@plugin",
					Version:     "1.0.0",
					Description: "Test",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewDefaultPluginRegistry()
			err := registry.Register(tt.plugin)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRegisterDuplicate tests that duplicate plugin names are rejected
func TestRegisterDuplicate(t *testing.T) {
	registry := NewDefaultPluginRegistry()

	plugin1 := coretesting.NewMockPlugin("test-plugin")
	err := registry.Register(plugin1)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	plugin2 := coretesting.NewMockPlugin("test-plugin")
	err = registry.Register(plugin2)
	if err == nil {
		t.Error("Expected error when registering duplicate plugin, got nil")
	}
}

// TestUnregister tests plugin unregistration
func TestUnregister(t *testing.T) {
	registry := NewDefaultPluginRegistry()

	plugin := coretesting.NewMockPlugin("test-plugin")
	err := registry.Register(plugin)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	// Unregister should succeed
	err = registry.Unregister("test-plugin")
	if err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	// Plugin should no longer be retrievable
	_, err = registry.Get("test-plugin")
	if err == nil {
		t.Error("Expected error when getting unregistered plugin, got nil")
	}
}

// TestUnregisterNonExistent tests unregistering a non-existent plugin
func TestUnregisterNonExistent(t *testing.T) {
	registry := NewDefaultPluginRegistry()

	err := registry.Unregister("non-existent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent plugin, got nil")
	}
}

// TestUnregisterWithDependents tests that plugins with dependents cannot be unregistered
func TestUnregisterWithDependents(t *testing.T) {
	registry := NewDefaultPluginRegistry()

	// Register base plugin
	basePlugin := coretesting.NewMockPlugin("base-plugin")
	err := registry.Register(basePlugin)
	if err != nil {
		t.Fatalf("Failed to register base plugin: %v", err)
	}

	// Register dependent plugin
	dependentPlugin := &coretesting.MockPlugin{
		PluginMetadata: types.PluginMetadata{
			Name:        "dependent-plugin",
			Version:     "1.0.0",
			Description: "Dependent plugin",
			Author:      "Test",
			Dependencies: []types.PluginDependency{
				{
					PluginName: "base-plugin",
					Version:    ">=1.0.0",
				},
			},
		},
	}
	err = registry.Register(dependentPlugin)
	if err != nil {
		t.Fatalf("Failed to register dependent plugin: %v", err)
	}

	// Try to unregister base plugin (should fail)
	err = registry.Unregister("base-plugin")
	if err == nil {
		t.Error("Expected error when unregistering plugin with dependents, got nil")
	}
}

// TestGet tests plugin retrieval
func TestGet(t *testing.T) {
	registry := NewDefaultPluginRegistry()

	plugin := coretesting.NewMockPlugin("test-plugin")
	err := registry.Register(plugin)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	// Get should succeed
	retrieved, err := registry.Get("test-plugin")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}

	if retrieved == nil {
		t.Error("Get() returned nil plugin")
	}

	if retrieved.Metadata().Name != "test-plugin" {
		t.Errorf("Get() returned wrong plugin: got %s, want test-plugin", retrieved.Metadata().Name)
	}
}

// TestGetNonExistent tests getting a non-existent plugin
func TestGetNonExistent(t *testing.T) {
	registry := NewDefaultPluginRegistry()

	_, err := registry.Get("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent plugin, got nil")
	}
}

// TestList tests plugin listing
func TestList(t *testing.T) {
	registry := NewDefaultPluginRegistry()

	// Register multiple plugins
	plugin1 := coretesting.NewMockPlugin("plugin-1")
	plugin2 := coretesting.NewMockPlugin("plugin-2")
	plugin3 := &coretesting.MockPlugin{
		PluginMetadata: types.PluginMetadata{
			Name:                 "plugin-3",
			Version:              "2.0.0",
			Description:          "Plugin 3",
			Author:               "Test",
			SupportedKinds:       []string{"container"},
			RequiredCapabilities: []types.Capability{types.CapabilityExec},
		},
	}

	registry.Register(plugin1)
	registry.Register(plugin2)
	registry.Register(plugin3)

	tests := []struct {
		name      string
		filter    types.PluginFilter
		wantCount int
		wantNames []string
	}{
		{
			name:      "no filter",
			filter:    types.PluginFilter{},
			wantCount: 3,
		},
		{
			name: "filter by name",
			filter: types.PluginFilter{
				Names: []string{"plugin-1"},
			},
			wantCount: 1,
			wantNames: []string{"plugin-1"},
		},
		{
			name: "filter by version",
			filter: types.PluginFilter{
				Versions: []string{"2.0.0"},
			},
			wantCount: 1,
			wantNames: []string{"plugin-3"},
		},
		{
			name: "filter by supported kind",
			filter: types.PluginFilter{
				SupportedKinds: []string{"container"},
			},
			wantCount: 3, // All plugins match: plugin-1 and plugin-2 support all kinds, plugin-3 explicitly supports container
			wantNames: []string{"plugin-1", "plugin-2", "plugin-3"},
		},
		{
			name: "filter by capability",
			filter: types.PluginFilter{
				Capabilities: []types.Capability{types.CapabilityExec},
			},
			wantCount: 1,
			wantNames: []string{"plugin-3"},
		},
		{
			name: "filter with no matches",
			filter: types.PluginFilter{
				Names: []string{"non-existent"},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugins, err := registry.List(tt.filter)
			if err != nil {
				t.Errorf("List() error = %v", err)
			}

			if len(plugins) != tt.wantCount {
				t.Errorf("List() returned %d plugins, want %d", len(plugins), tt.wantCount)
			}

			if tt.wantNames != nil {
				for _, wantName := range tt.wantNames {
					found := false
					for _, plugin := range plugins {
						if plugin.Name == wantName {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("List() did not return plugin %s", wantName)
					}
				}
			}
		})
	}
}

// TestValidate tests plugin validation
func TestValidate(t *testing.T) {
	registry := NewDefaultPluginRegistry()

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
		{
			name: "empty name",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "",
					Version:     "1.0.0",
					Description: "Test",
				},
			},
			wantErr: true,
		},
		{
			name: "empty version",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "test",
					Version:     "",
					Description: "Test",
				},
			},
			wantErr: true,
		},
		{
			name: "empty description",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "test",
					Version:     "1.0.0",
					Description: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Validate(tt.plugin)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateDependencies tests dependency validation
func TestValidateDependencies(t *testing.T) {
	registry := NewDefaultPluginRegistry()

	// Register base plugin
	basePlugin := coretesting.NewMockPlugin("base-plugin")
	registry.Register(basePlugin)

	tests := []struct {
		name    string
		plugin  types.Plugin
		wantErr bool
	}{
		{
			name:    "no dependencies",
			plugin:  coretesting.NewMockPlugin("test-plugin"),
			wantErr: false,
		},
		{
			name: "satisfied dependency",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "dependent-plugin",
					Version:     "1.0.0",
					Description: "Dependent plugin",
					Author:      "Test",
					Dependencies: []types.PluginDependency{
						{
							PluginName: "base-plugin",
							Version:    ">=1.0.0",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing dependency",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "dependent-plugin",
					Version:     "1.0.0",
					Description: "Dependent plugin",
					Author:      "Test",
					Dependencies: []types.PluginDependency{
						{
							PluginName: "non-existent",
							Version:    ">=1.0.0",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "unsatisfied version constraint",
			plugin: &coretesting.MockPlugin{
				PluginMetadata: types.PluginMetadata{
					Name:        "dependent-plugin",
					Version:     "1.0.0",
					Description: "Dependent plugin",
					Author:      "Test",
					Dependencies: []types.PluginDependency{
						{
							PluginName: "base-plugin",
							Version:    ">=2.0.0",
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateDependencies(tt.plugin)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCircularDependencyDetection tests circular dependency detection
func TestCircularDependencyDetection(t *testing.T) {
	registry := NewDefaultPluginRegistry()

	// Register plugin A
	pluginA := coretesting.NewMockPlugin("plugin-a")
	registry.Register(pluginA)

	// Register plugin B that depends on A
	pluginB := &coretesting.MockPlugin{
		PluginMetadata: types.PluginMetadata{
			Name:        "plugin-b",
			Version:     "1.0.0",
			Description: "Plugin B",
			Author:      "Test",
			Dependencies: []types.PluginDependency{
				{
					PluginName: "plugin-a",
					Version:    ">=1.0.0",
				},
			},
		},
	}
	registry.Register(pluginB)

	// Try to register plugin C that depends on B and creates a cycle back to A
	pluginC := &coretesting.MockPlugin{
		PluginMetadata: types.PluginMetadata{
			Name:        "plugin-c",
			Version:     "1.0.0",
			Description: "Plugin C",
			Author:      "Test",
			Dependencies: []types.PluginDependency{
				{
					PluginName: "plugin-b",
					Version:    ">=1.0.0",
				},
			},
		},
	}
	err := registry.Register(pluginC)
	if err != nil {
		t.Fatalf("Failed to register plugin C: %v", err)
	}

	// Now try to update plugin A to depend on C (creating a cycle)
	registry.Unregister("plugin-a")
	pluginAWithCycle := &coretesting.MockPlugin{
		PluginMetadata: types.PluginMetadata{
			Name:        "plugin-a",
			Version:     "1.0.0",
			Description: "Plugin A",
			Author:      "Test",
			Dependencies: []types.PluginDependency{
				{
					PluginName: "plugin-c",
					Version:    ">=1.0.0",
				},
			},
		},
	}

	err = registry.Register(pluginAWithCycle)
	if err == nil {
		t.Error("Expected error when registering plugin with circular dependency, got nil")
	}
}

// TestSetLoader tests setting the plugin loader
func TestSetLoader(t *testing.T) {
	registry := NewDefaultPluginRegistry()

	loader := &coretesting.MockPluginLoader{}
	err := registry.SetLoader(loader)
	if err != nil {
		t.Errorf("SetLoader() error = %v", err)
	}

	// Try to set nil loader
	err = registry.SetLoader(nil)
	if err == nil {
		t.Error("Expected error when setting nil loader, got nil")
	}
}
