package config

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// DefaultConfig implements the Config interface with support for multiple providers
type DefaultConfig struct {
	mu        sync.RWMutex
	values    map[string]interface{}
	providers []providerWithPriority
	watchers  map[string][]chan interface{}
	ctx       context.Context
	cancel    context.CancelFunc
}

type providerWithPriority struct {
	provider types.ConfigProvider
	priority int
}

// NewDefaultConfig creates a new DefaultConfig instance
func NewDefaultConfig() *DefaultConfig {
	ctx, cancel := context.WithCancel(context.Background())
	return &DefaultConfig{
		values:    make(map[string]interface{}),
		providers: make([]providerWithPriority, 0),
		watchers:  make(map[string][]chan interface{}),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Get retrieves a configuration value by key
func (c *DefaultConfig) Get(key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.values[key]
	if !exists {
		return nil, fmt.Errorf("config key not found: %s", key)
	}
	return value, nil
}

// GetString retrieves a string configuration value
func (c *DefaultConfig) GetString(key string) (string, error) {
	value, err := c.Get(key)
	if err != nil {
		return "", err
	}

	switch v := value.(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// GetInt retrieves an integer configuration value
func (c *DefaultConfig) GetInt(key string) (int, error) {
	value, err := c.Get(key)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case int32:
		return int(v), nil
	case float64:
		return int(v), nil
	case float32:
		return int(v), nil
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %s to int: %w", key, err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("cannot convert %s to int: unsupported type %T", key, v)
	}
}

// GetBool retrieves a boolean configuration value
func (c *DefaultConfig) GetBool(key string) (bool, error) {
	value, err := c.Get(key)
	if err != nil {
		return false, err
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, fmt.Errorf("cannot convert %s to bool: %w", key, err)
		}
		return b, nil
	case int:
		return v != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %s to bool: unsupported type %T", key, v)
	}
}

// GetDuration retrieves a duration configuration value
func (c *DefaultConfig) GetDuration(key string) (time.Duration, error) {
	value, err := c.Get(key)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case time.Duration:
		return v, nil
	case string:
		d, err := time.ParseDuration(v)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %s to duration: %w", key, err)
		}
		return d, nil
	case int64:
		return time.Duration(v), nil
	case int:
		return time.Duration(v), nil
	case float64:
		return time.Duration(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %s to duration: unsupported type %T", key, v)
	}
}

// Set sets a configuration value
func (c *DefaultConfig) Set(key string, value interface{}) error {
	c.mu.Lock()
	oldValue := c.values[key]
	c.values[key] = value
	watchers := c.watchers[key]
	c.mu.Unlock()

	// Notify watchers
	for _, ch := range watchers {
		select {
		case ch <- value:
		default:
			// Don't block if watcher is not reading
		}
	}

	// If value changed, notify watchers
	if oldValue != value {
		c.notifyProviderWatchers(key, oldValue, value)
	}

	return nil
}

// Watch returns a channel that receives updates for the specified key
func (c *DefaultConfig) Watch(key string) (<-chan interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan interface{}, 10)
	c.watchers[key] = append(c.watchers[key], ch)

	return ch, nil
}

// AddProvider adds a configuration provider with the specified priority
// Higher priority values take precedence (e.g., priority 100 overrides priority 50)
func (c *DefaultConfig) AddProvider(provider types.ConfigProvider, priority int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add provider to list
	c.providers = append(c.providers, providerWithPriority{
		provider: provider,
		priority: priority,
	})

	// Sort providers by priority (highest first)
	c.sortProviders()

	// Load initial values from provider
	values, err := provider.Load(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to load config from provider %s: %w", provider.Name(), err)
	}

	// Merge values with precedence
	c.mergeValues(values, priority)

	// Start watching for changes
	go c.watchProvider(provider)

	return nil
}

// sortProviders sorts providers by priority (highest first)
func (c *DefaultConfig) sortProviders() {
	// Simple bubble sort since we expect few providers
	for i := 0; i < len(c.providers); i++ {
		for j := i + 1; j < len(c.providers); j++ {
			if c.providers[j].priority > c.providers[i].priority {
				c.providers[i], c.providers[j] = c.providers[j], c.providers[i]
			}
		}
	}
}

// mergeValues merges new values into the config with precedence
func (c *DefaultConfig) mergeValues(newValues map[string]interface{}, priority int) {
	for key, newValue := range newValues {
		// Check if key exists and what priority it has
		shouldUpdate := true
		
		// Find the highest priority provider that has this key
		for _, p := range c.providers {
			if p.priority > priority {
				// Check if this higher priority provider has the key
				values, err := p.provider.Load(c.ctx)
				if err == nil {
					if _, exists := values[key]; exists {
						shouldUpdate = false
						break
					}
				}
			}
		}

		if shouldUpdate {
			oldValue := c.values[key]
			c.values[key] = newValue

			// Notify watchers if value changed
			if oldValue != newValue {
				watchers := c.watchers[key]
				for _, ch := range watchers {
					select {
					case ch <- newValue:
					default:
					}
				}
			}
		}
	}
}

// watchProvider watches a provider for configuration changes
func (c *DefaultConfig) watchProvider(provider types.ConfigProvider) {
	changeCh, err := provider.Watch(c.ctx)
	if err != nil {
		// Provider doesn't support watching
		return
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		case change, ok := <-changeCh:
			if !ok {
				return
			}

			c.mu.Lock()
			// Find provider priority
			priority := 0
			for _, p := range c.providers {
				if p.provider.Name() == provider.Name() {
					priority = p.priority
					break
				}
			}

			// Update value if this provider has highest priority for this key
			shouldUpdate := true
			for _, p := range c.providers {
				if p.priority > priority && p.provider.Name() != provider.Name() {
					values, err := p.provider.Load(c.ctx)
					if err == nil {
						if _, exists := values[change.Key]; exists {
							shouldUpdate = false
							break
						}
					}
				}
			}

			if shouldUpdate {
				c.values[change.Key] = change.NewValue

				// Notify watchers
				watchers := c.watchers[change.Key]
				c.mu.Unlock()

				for _, ch := range watchers {
					select {
					case ch <- change.NewValue:
					default:
					}
				}
			} else {
				c.mu.Unlock()
			}
		}
	}
}

// notifyProviderWatchers is a helper to notify watchers of changes
func (c *DefaultConfig) notifyProviderWatchers(key string, oldValue, newValue interface{}) {
	// This is already handled in Set and watchProvider
}

// Close closes the config and stops all watchers
func (c *DefaultConfig) Close() error {
	c.cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Close all watcher channels
	for _, watchers := range c.watchers {
		for _, ch := range watchers {
			close(ch)
		}
	}
	c.watchers = make(map[string][]chan interface{})

	return nil
}
