// Package registry provides the default implementation of PluginRegistry
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// DefaultPluginRegistry is the default implementation of PluginRegistry
type DefaultPluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string]types.Plugin
	loader  types.PluginLoader
}

// NewDefaultPluginRegistry creates a new DefaultPluginRegistry
func NewDefaultPluginRegistry() *DefaultPluginRegistry {
	return &DefaultPluginRegistry{
		plugins: make(map[string]types.Plugin),
	}
}

// Register registers a plugin with the registry
func (r *DefaultPluginRegistry) Register(plugin types.Plugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}

	// Validate the plugin
	if err := r.Validate(plugin); err != nil {
		return fmt.Errorf("plugin validation failed: %w", err)
	}

	meta := plugin.Metadata()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicates
	if _, exists := r.plugins[meta.Name]; exists {
		return fmt.Errorf("plugin %s already registered", meta.Name)
	}

	// Validate dependencies
	if err := r.validateDependenciesLocked(plugin); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	r.plugins[meta.Name] = plugin
	return nil
}

// Unregister removes a plugin from the registry
func (r *DefaultPluginRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Check if any other plugins depend on this one
	for _, plugin := range r.plugins {
		meta := plugin.Metadata()
		for _, dep := range meta.Dependencies {
			if dep.PluginName == name {
				return fmt.Errorf("cannot unregister plugin %s: plugin %s depends on it", name, meta.Name)
			}
		}
	}

	delete(r.plugins, name)
	return nil
}

// Discover discovers and registers a plugin from a path
func (r *DefaultPluginRegistry) Discover(path string) error {
	if r.loader == nil {
		return fmt.Errorf("no plugin loader configured")
	}

	// Load the plugin
	plugin, err := r.loader.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load plugin from %s: %w", path, err)
	}

	// Register the plugin
	if err := r.Register(plugin); err != nil {
		return fmt.Errorf("failed to register plugin from %s: %w", path, err)
	}

	return nil
}

// DiscoverFromDirectory discovers and registers all plugins in a directory
func (r *DefaultPluginRegistry) DiscoverFromDirectory(dir string) error {
	if r.loader == nil {
		return fmt.Errorf("no plugin loader configured")
	}

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, return success (no plugins to discover)
			return nil
		}
		return fmt.Errorf("failed to stat directory %s: %w", dir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	// Walk the directory
	var discoveryErrors []error
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Try to discover plugin from this file
		// We attempt to load each file, but don't fail if some files aren't plugins
		if err := r.Discover(path); err != nil {
			// Collect errors but don't stop discovery
			discoveryErrors = append(discoveryErrors, fmt.Errorf("failed to discover plugin from %s: %w", path, err))
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}

	// If all files failed to load, return an error
	if len(discoveryErrors) > 0 && len(r.plugins) == 0 {
		return fmt.Errorf("no valid plugins found in directory %s: %d errors occurred", dir, len(discoveryErrors))
	}

	return nil
}

// Get retrieves a plugin by name
func (r *DefaultPluginRegistry) Get(name string) (types.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// List returns a list of plugin metadata matching the filter
func (r *DefaultPluginRegistry) List(filter types.PluginFilter) ([]types.PluginMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []types.PluginMetadata

	for _, plugin := range r.plugins {
		meta := plugin.Metadata()

		// Apply name filter
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

		// Apply version filter
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

		// Apply supported kinds filter
		if len(filter.SupportedKinds) > 0 {
			// If plugin supports all kinds (empty list), it matches
			if len(meta.SupportedKinds) > 0 {
				// Plugin has specific kinds, check if any match
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
				if !found {
					continue
				}
			}
			// If meta.SupportedKinds is empty, plugin supports all kinds, so it matches
		}

		// Apply capabilities filter
		if len(filter.Capabilities) > 0 {
			// Check if plugin has all required capabilities
			found := true
			for _, filterCap := range filter.Capabilities {
				hasCapability := false
				for _, pluginCap := range meta.RequiredCapabilities {
					if pluginCap == filterCap {
						hasCapability = true
						break
					}
				}
				if !hasCapability {
					found = false
					break
				}
			}
			if !found {
				continue
			}
		}

		result = append(result, meta)
	}

	return result, nil
}

// Validate validates a plugin's metadata and interface compliance
func (r *DefaultPluginRegistry) Validate(plugin types.Plugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}

	meta := plugin.Metadata()

	// Validate required fields
	if meta.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if meta.Version == "" {
		return fmt.Errorf("plugin version is required")
	}

	if meta.Description == "" {
		return fmt.Errorf("plugin description is required")
	}

	// Validate version format (semantic versioning)
	if _, err := semver.NewVersion(strings.TrimPrefix(meta.Version, "v")); err != nil {
		return fmt.Errorf("invalid plugin version %s: %w", meta.Version, err)
	}

	// Validate plugin name format (alphanumeric, hyphens, underscores)
	if !isValidPluginName(meta.Name) {
		return fmt.Errorf("invalid plugin name %s: must contain only alphanumeric characters, hyphens, and underscores", meta.Name)
	}

	return nil
}

// ValidateDependencies validates that all plugin dependencies are satisfied
func (r *DefaultPluginRegistry) ValidateDependencies(plugin types.Plugin) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.validateDependenciesLocked(plugin)
}

// validateDependenciesLocked validates dependencies while holding the lock
func (r *DefaultPluginRegistry) validateDependenciesLocked(plugin types.Plugin) error {
	meta := plugin.Metadata()

	// Check for circular dependencies
	if err := r.detectCircularDependencies(meta.Name, meta.Dependencies, make(map[string]bool)); err != nil {
		return err
	}

	// Validate each dependency
	for _, dep := range meta.Dependencies {
		// Check if dependency is registered
		depPlugin, exists := r.plugins[dep.PluginName]
		if !exists {
			return fmt.Errorf("dependency %s not found", dep.PluginName)
		}

		// Validate version constraint
		if dep.Version != "" {
			depMeta := depPlugin.Metadata()
			if err := validateVersionConstraint(depMeta.Version, dep.Version); err != nil {
				return fmt.Errorf("dependency %s version constraint not satisfied: %w", dep.PluginName, err)
			}
		}
	}

	return nil
}

// detectCircularDependencies detects circular dependencies in the plugin graph
func (r *DefaultPluginRegistry) detectCircularDependencies(pluginName string, deps []types.PluginDependency, visited map[string]bool) error {
	// Mark current plugin as visited
	visited[pluginName] = true

	// Check each dependency
	for _, dep := range deps {
		// If we've already visited this dependency in the current path, we have a cycle
		if visited[dep.PluginName] {
			return fmt.Errorf("circular dependency detected: %s -> %s", pluginName, dep.PluginName)
		}

		// Get the dependency plugin
		depPlugin, exists := r.plugins[dep.PluginName]
		if !exists {
			// Dependency not registered yet, skip circular check
			continue
		}

		// Recursively check the dependency's dependencies
		depMeta := depPlugin.Metadata()
		if err := r.detectCircularDependencies(dep.PluginName, depMeta.Dependencies, copyMap(visited)); err != nil {
			return err
		}
	}

	return nil
}

// Reload reloads a plugin from disk
func (r *DefaultPluginRegistry) Reload(name string) error {
	if r.loader == nil {
		return fmt.Errorf("no plugin loader configured")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if plugin exists
	oldPlugin, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Unload the old plugin
	if err := r.loader.Unload(oldPlugin); err != nil {
		return fmt.Errorf("failed to unload plugin %s: %w", name, err)
	}

	// Remove from registry temporarily
	delete(r.plugins, name)

	// Try to load the new version
	// Note: In a real implementation, we would need to know the path to reload from
	// For now, we just return an error indicating reload is not fully implemented
	return fmt.Errorf("reload not fully implemented: plugin path tracking required")
}

// SetLoader sets the plugin loader
func (r *DefaultPluginRegistry) SetLoader(loader types.PluginLoader) error {
	if loader == nil {
		return fmt.Errorf("loader cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.loader = loader
	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// isValidPluginName checks if a plugin name is valid
func isValidPluginName(name string) bool {
	if name == "" {
		return false
	}

	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_') {
			return false
		}
	}

	return true
}

// validateVersionConstraint validates that a version satisfies a constraint
func validateVersionConstraint(version, constraint string) error {
	// Parse the actual version
	v, err := semver.NewVersion(strings.TrimPrefix(version, "v"))
	if err != nil {
		return fmt.Errorf("invalid version %s: %w", version, err)
	}

	// Parse the constraint
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return fmt.Errorf("invalid version constraint %s: %w", constraint, err)
	}

	// Check if version satisfies constraint
	if !c.Check(v) {
		return fmt.Errorf("version %s does not satisfy constraint %s", version, constraint)
	}

	return nil
}

// copyMap creates a copy of a map
func copyMap(m map[string]bool) map[string]bool {
	result := make(map[string]bool, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
