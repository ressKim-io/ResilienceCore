# Plugin Registry

The Plugin Registry provides a centralized system for managing plugin discovery, registration, validation, and lifecycle management in the Infrastructure Resilience Engine.

## Overview

The `DefaultPluginRegistry` is the default implementation of the `PluginRegistry` interface. It provides:

- **Plugin Registration**: Register and unregister plugins with validation
- **Dependency Management**: Validate plugin dependencies and detect circular dependencies
- **Version Constraints**: Enforce semantic versioning constraints on dependencies
- **Plugin Discovery**: Discover plugins from directories
- **Filtering**: List plugins with flexible filtering criteria
- **Hot Reload**: Support for reloading plugins without restarting the application

## Usage

### Creating a Registry

```go
import "github.com/yourusername/infrastructure-resilience-engine/pkg/core/registry"

// Create a new registry
reg := registry.NewDefaultPluginRegistry()
```

### Registering Plugins

```go
// Create a plugin
plugin := &MyPlugin{
    metadata: types.PluginMetadata{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "My custom plugin",
        Author:      "Your Name",
    },
}

// Register the plugin
err := reg.Register(plugin)
if err != nil {
    log.Fatalf("Failed to register plugin: %v", err)
}
```

### Plugin Dependencies

Plugins can declare dependencies on other plugins:

```go
plugin := &MyPlugin{
    metadata: types.PluginMetadata{
        Name:        "dependent-plugin",
        Version:     "1.0.0",
        Description: "Plugin with dependencies",
        Author:      "Your Name",
        Dependencies: []types.PluginDependency{
            {
                PluginName: "base-plugin",
                Version:    ">=1.0.0",  // Semantic version constraint
            },
        },
    },
}
```

The registry will:
- Verify all dependencies are registered before allowing registration
- Validate version constraints using semantic versioning
- Detect circular dependencies and prevent registration

### Retrieving Plugins

```go
// Get a specific plugin
plugin, err := reg.Get("my-plugin")
if err != nil {
    log.Fatalf("Plugin not found: %v", err)
}

// List all plugins
plugins, err := reg.List(types.PluginFilter{})
if err != nil {
    log.Fatalf("Failed to list plugins: %v", err)
}
```

### Filtering Plugins

The registry supports flexible filtering:

```go
// Filter by name
plugins, _ := reg.List(types.PluginFilter{
    Names: []string{"plugin-1", "plugin-2"},
})

// Filter by version
plugins, _ := reg.List(types.PluginFilter{
    Versions: []string{"1.0.0"},
})

// Filter by supported resource kinds
plugins, _ := reg.List(types.PluginFilter{
    SupportedKinds: []string{"container", "pod"},
})

// Filter by required capabilities
plugins, _ := reg.List(types.PluginFilter{
    Capabilities: []types.Capability{types.CapabilityExec},
})
```

### Plugin Discovery

The registry can discover plugins from directories:

```go
// Set a plugin loader
loader := &MyPluginLoader{}
err := reg.SetLoader(loader)
if err != nil {
    log.Fatalf("Failed to set loader: %v", err)
}

// Discover plugins from a directory
err = reg.DiscoverFromDirectory("/path/to/plugins")
if err != nil {
    log.Fatalf("Failed to discover plugins: %v", err)
}

// Discover a single plugin
err = reg.Discover("/path/to/plugin.so")
if err != nil {
    log.Fatalf("Failed to discover plugin: %v", err)
}
```

### Unregistering Plugins

```go
// Unregister a plugin
err := reg.Unregister("my-plugin")
if err != nil {
    log.Fatalf("Failed to unregister plugin: %v", err)
}
```

Note: The registry will prevent unregistering a plugin if other plugins depend on it.

### Hot Reload

```go
// Reload a plugin (requires a plugin loader)
err := reg.Reload("my-plugin")
if err != nil {
    log.Fatalf("Failed to reload plugin: %v", err)
}
```

## Validation

The registry performs comprehensive validation on plugin registration:

### Metadata Validation

- **Name**: Required, must contain only alphanumeric characters, hyphens, and underscores
- **Version**: Required, must be a valid semantic version (e.g., "1.0.0", "2.1.3-beta")
- **Description**: Required, must not be empty

### Dependency Validation

- All dependencies must be registered before the dependent plugin
- Version constraints must be satisfied
- Circular dependencies are detected and prevented

### Version Constraints

The registry supports semantic versioning constraints:

- `>=1.0.0` - Greater than or equal to 1.0.0
- `^1.0.0` - Compatible with 1.0.0 (1.x.x)
- `~1.0.0` - Approximately 1.0.0 (1.0.x)
- `1.0.0` - Exactly 1.0.0

## Thread Safety

The `DefaultPluginRegistry` is thread-safe and can be safely accessed from multiple goroutines. All operations use appropriate locking mechanisms to prevent race conditions.

## Testing

The registry includes comprehensive test coverage:

- **Unit Tests**: Test individual methods and edge cases
- **Property-Based Tests**: Verify correctness properties using gopter
  - Property 60: Plugin validation occurs on registration
  - Property 61: Plugin retrieval returns plugin or error
  - Property 62: Plugin listing supports filtering
  - Property 84: Dependencies are declared in metadata
  - Property 85: Dependencies are verified on load
  - Property 86: Missing dependencies prevent registration
  - Property 87: Circular dependencies are detected
  - Property 88: Version constraints are validated

## Implementation Details

### Circular Dependency Detection

The registry uses a depth-first search algorithm to detect circular dependencies. When registering a plugin, it traverses the dependency graph and maintains a visited set to detect cycles.

### Version Constraint Validation

Version constraints are validated using the `github.com/Masterminds/semver/v3` library, which provides robust semantic versioning support.

### Plugin Filtering

Filtering is performed in-memory by iterating through registered plugins and applying filter criteria. Plugins with empty `SupportedKinds` lists are considered to support all kinds.

## Error Handling

The registry returns descriptive errors for common failure scenarios:

- `plugin cannot be nil`: Attempted to register a nil plugin
- `plugin name is required`: Plugin metadata is missing the name field
- `plugin version is required`: Plugin metadata is missing the version field
- `plugin <name> already registered`: Attempted to register a duplicate plugin
- `dependency <name> not found`: Plugin dependency is not registered
- `circular dependency detected`: Circular dependency in plugin graph
- `version <version> does not satisfy constraint <constraint>`: Version constraint not satisfied

## Best Practices

1. **Register dependencies first**: Always register base plugins before dependent plugins
2. **Use semantic versioning**: Follow semantic versioning for plugin versions
3. **Validate early**: Call `Validate()` before registration to catch errors early
4. **Handle errors**: Always check and handle errors from registry operations
5. **Use filters efficiently**: Use specific filters to reduce the number of plugins returned
6. **Document dependencies**: Clearly document plugin dependencies in metadata

## Example: Complete Plugin Registration Flow

```go
package main

import (
    "log"
    
    "github.com/yourusername/infrastructure-resilience-engine/pkg/core/registry"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

func main() {
    // Create registry
    reg := registry.NewDefaultPluginRegistry()
    
    // Register base plugin
    basePlugin := &MyBasePlugin{
        metadata: types.PluginMetadata{
            Name:        "base-plugin",
            Version:     "1.0.0",
            Description: "Base plugin",
            Author:      "Your Name",
        },
    }
    
    if err := reg.Register(basePlugin); err != nil {
        log.Fatalf("Failed to register base plugin: %v", err)
    }
    
    // Register dependent plugin
    dependentPlugin := &MyDependentPlugin{
        metadata: types.PluginMetadata{
            Name:        "dependent-plugin",
            Version:     "1.0.0",
            Description: "Dependent plugin",
            Author:      "Your Name",
            Dependencies: []types.PluginDependency{
                {
                    PluginName: "base-plugin",
                    Version:    ">=1.0.0",
                },
            },
        },
    }
    
    if err := reg.Register(dependentPlugin); err != nil {
        log.Fatalf("Failed to register dependent plugin: %v", err)
    }
    
    // List all plugins
    plugins, err := reg.List(types.PluginFilter{})
    if err != nil {
        log.Fatalf("Failed to list plugins: %v", err)
    }
    
    log.Printf("Registered %d plugins", len(plugins))
    for _, meta := range plugins {
        log.Printf("  - %s v%s", meta.Name, meta.Version)
    }
}
```

## See Also

- [Plugin Interface](../types/types.go) - Core plugin interface definition
- [Plugin Development Guide](../../../docs/plugin-development.md) - Guide for developing plugins
- [Testing Utilities](../testing/mocks.go) - Mock implementations for testing
