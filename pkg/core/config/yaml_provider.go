package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
	"gopkg.in/yaml.v3"
)

// YAMLProvider implements ConfigProvider for YAML files
type YAMLProvider struct {
	filePath string
	mu       sync.RWMutex
	values   map[string]interface{}
	watcher  *fsnotify.Watcher
	watchers []chan types.ConfigChange
}

// NewYAMLProvider creates a new YAML configuration provider
func NewYAMLProvider(filePath string) *YAMLProvider {
	return &YAMLProvider{
		filePath: filePath,
		values:   make(map[string]interface{}),
		watchers: make([]chan types.ConfigChange, 0),
	}
}

// Load loads configuration from the YAML file
func (p *YAMLProvider) Load(ctx context.Context) (map[string]interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Read file
	data, err := os.ReadFile(p.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty config
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to read YAML file %s: %w", p.filePath, err)
	}

	// Parse YAML
	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse YAML file %s: %w", p.filePath, err)
	}

	// Flatten nested maps
	flattened := flattenMap(values, "")

	// Store values
	p.values = flattened

	return flattened, nil
}

// Watch watches the YAML file for changes
func (p *YAMLProvider) Watch(ctx context.Context) (<-chan types.ConfigChange, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create watcher if not exists
	if p.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil, fmt.Errorf("failed to create file watcher: %w", err)
		}
		p.watcher = watcher

		// Watch the file
		if err := p.watcher.Add(p.filePath); err != nil {
			// If file doesn't exist, watch the directory
			dir := filepath.Dir(p.filePath)
			if err := p.watcher.Add(dir); err != nil {
				p.watcher.Close()
				p.watcher = nil
				return nil, fmt.Errorf("failed to watch file %s: %w", p.filePath, err)
			}
		}

		// Start watching goroutine
		go p.watchFile(ctx)
	}

	// Create channel for this watcher
	ch := make(chan types.ConfigChange, 10)
	p.watchers = append(p.watchers, ch)

	return ch, nil
}

// watchFile watches for file changes and notifies watchers
func (p *YAMLProvider) watchFile(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			p.mu.Lock()
			if p.watcher != nil {
				p.watcher.Close()
				p.watcher = nil
			}
			// Close all watcher channels
			for _, ch := range p.watchers {
				close(ch)
			}
			p.watchers = nil
			p.mu.Unlock()
			return

		case event, ok := <-p.watcher.Events:
			if !ok {
				return
			}

			// Check if it's our file
			if event.Name != p.filePath && filepath.Base(event.Name) != filepath.Base(p.filePath) {
				continue
			}

			// Only care about write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				p.handleFileChange(ctx)
			}

		case err, ok := <-p.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			_ = err
		}
	}
}

// handleFileChange handles file change events
func (p *YAMLProvider) handleFileChange(ctx context.Context) {
	// Load new values
	newValues, err := p.Load(ctx)
	if err != nil {
		// Failed to load, skip
		return
	}

	p.mu.RLock()
	oldValues := make(map[string]interface{})
	for k, v := range p.values {
		oldValues[k] = v
	}
	watchers := p.watchers
	p.mu.RUnlock()

	// Find changes
	changes := make([]types.ConfigChange, 0)

	// Check for modified and new keys
	for key, newValue := range newValues {
		oldValue, exists := oldValues[key]
		if !exists || oldValue != newValue {
			changes = append(changes, types.ConfigChange{
				Key:      key,
				OldValue: oldValue,
				NewValue: newValue,
			})
		}
	}

	// Check for deleted keys
	for key, oldValue := range oldValues {
		if _, exists := newValues[key]; !exists {
			changes = append(changes, types.ConfigChange{
				Key:      key,
				OldValue: oldValue,
				NewValue: nil,
			})
		}
	}

	// Notify watchers
	for _, change := range changes {
		for _, ch := range watchers {
			select {
			case ch <- change:
			default:
				// Don't block if watcher is not reading
			}
		}
	}
}

// Name returns the provider name
func (p *YAMLProvider) Name() string {
	return fmt.Sprintf("yaml:%s", p.filePath)
}

// flattenMap flattens a nested map into a flat map with dot-separated keys
func flattenMap(m map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Recursively flatten nested maps
			nested := flattenMap(v, fullKey)
			for k, val := range nested {
				result[k] = val
			}
		case map[interface{}]interface{}:
			// Convert map[interface{}]interface{} to map[string]interface{}
			converted := make(map[string]interface{})
			for k, val := range v {
				if strKey, ok := k.(string); ok {
					converted[strKey] = val
				}
			}
			nested := flattenMap(converted, fullKey)
			for k, val := range nested {
				result[k] = val
			}
		default:
			result[fullKey] = value
		}
	}

	return result
}
