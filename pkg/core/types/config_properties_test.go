// Package types provides property-based tests for the Config interface
package types

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty43_MultipleConfigProvidersAreSupported verifies that
// any set of ConfigProvider implementations can be used simultaneously
// Feature: infrastructure-resilience-engine, Property 43: Multiple config providers are supported
// Validates: Requirements 9.1
func TestProperty43_MultipleConfigProvidersAreSupported(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("multiple config providers are supported", prop.ForAll(
		func(numProviders int, key string, values []string) bool {
			// Limit to reasonable range
			if numProviders < 1 || numProviders > 5 {
				return true // Skip invalid inputs
			}
			if len(values) < numProviders {
				return true // Skip if not enough values
			}

			ctx := context.Background()
			config := NewMockConfig()

			// Add multiple providers with different priorities
			for i := 0; i < numProviders; i++ {
				provider := NewMockConfigProvider(
					fmt.Sprintf("provider-%d", i),
					map[string]interface{}{
						key: values[i],
					},
				)
				if err := config.AddProvider(provider, i); err != nil {
					return false
				}
			}

			// Load configuration from all providers
			mockConfig := config.(*MockConfig)
			if err := mockConfig.loadAll(ctx); err != nil {
				return false
			}

			// Verify we can get the value (should be from highest priority provider)
			_, err := config.Get(key)
			return err == nil
		},
		gen.IntRange(1, 5),
		gen.Identifier(),
		gen.SliceOf(gen.Identifier()),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty44_ConfigMergeFollowsPrecedence verifies that
// for any set of config sources with defined priorities, Get returns the value from the highest priority source
// Feature: infrastructure-resilience-engine, Property 44: Config merge follows precedence
// Validates: Requirements 9.2
func TestProperty44_ConfigMergeFollowsPrecedence(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("config merge follows precedence", prop.ForAll(
		func(key string, lowPriorityValue string, highPriorityValue string) bool {
			// Skip if values are the same (can't test precedence)
			if lowPriorityValue == highPriorityValue {
				return true
			}

			ctx := context.Background()
			config := NewMockConfig()

			// Add low priority provider
			lowProvider := NewMockConfigProvider(
				"low-priority",
				map[string]interface{}{
					key: lowPriorityValue,
				},
			)
			if err := config.AddProvider(lowProvider, 1); err != nil {
				return false
			}

			// Add high priority provider
			highProvider := NewMockConfigProvider(
				"high-priority",
				map[string]interface{}{
					key: highPriorityValue,
				},
			)
			if err := config.AddProvider(highProvider, 10); err != nil {
				return false
			}

			// Load configuration
			mockConfig := config.(*MockConfig)
			if err := mockConfig.loadAll(ctx); err != nil {
				return false
			}

			// Get value - should be from high priority provider
			value, err := config.Get(key)
			if err != nil {
				return false
			}

			// Verify it's the high priority value
			return value == highPriorityValue
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty47_CustomConfigProvidersAreSupported verifies that
// any custom ConfigProvider implementation can be used without Core modification
// Feature: infrastructure-resilience-engine, Property 47: Custom config providers are supported
// Validates: Requirements 9.5
func TestProperty47_CustomConfigProvidersAreSupported(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("custom config providers are supported", prop.ForAll(
		func(providerName string, configData map[string]string) bool {
			// Skip empty provider names
			if providerName == "" {
				return true
			}

			ctx := context.Background()
			config := NewMockConfig()

			// Create a custom config provider
			customData := make(map[string]interface{})
			for k, v := range configData {
				customData[k] = v
			}

			customProvider := NewMockConfigProvider(providerName, customData)

			// Add the custom provider
			if err := config.AddProvider(customProvider, 1); err != nil {
				return false
			}

			// Load configuration
			mockConfig := config.(*MockConfig)
			if err := mockConfig.loadAll(ctx); err != nil {
				return false
			}

			// Verify all keys from the custom provider are accessible
			for key, expectedValue := range configData {
				value, err := config.GetString(key)
				if err != nil {
					return false
				}
				if value != expectedValue {
					return false
				}
			}

			return true
		},
		gen.Identifier(),
		gen.MapOf(gen.Identifier(), gen.Identifier()),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ============================================================================
// Mock Implementations for Testing
// ============================================================================

// MockConfig is a simple in-memory implementation of Config for testing
type MockConfig struct {
	data      map[string]interface{}
	providers []providerWithPriority
	watchers  map[string][]chan interface{}
	mu        sync.RWMutex
}

type providerWithPriority struct {
	provider ConfigProvider
	priority int
}

// NewMockConfig creates a new MockConfig
func NewMockConfig() Config {
	return &MockConfig{
		data:      make(map[string]interface{}),
		providers: make([]providerWithPriority, 0),
		watchers:  make(map[string][]chan interface{}),
	}
}

func (m *MockConfig) Get(key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}

func (m *MockConfig) GetString(key string) (string, error) {
	value, err := m.Get(key)
	if err != nil {
		return "", err
	}

	switch v := value.(type) {
	case string:
		return v, nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func (m *MockConfig) GetInt(key string) (int, error) {
	value, err := m.Get(key)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert to int: %v", value)
	}
}

func (m *MockConfig) GetBool(key string) (bool, error) {
	value, err := m.Get(key)
	if err != nil {
		return false, err
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("cannot convert to bool: %v", value)
	}
}

func (m *MockConfig) GetDuration(key string) (time.Duration, error) {
	value, err := m.Get(key)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case time.Duration:
		return v, nil
	case string:
		return time.ParseDuration(v)
	default:
		return 0, fmt.Errorf("cannot convert to duration: %v", value)
	}
}

func (m *MockConfig) Set(key string, value interface{}) error {
	m.mu.Lock()
	oldValue := m.data[key]
	m.data[key] = value
	watchers := m.watchers[key]
	m.mu.Unlock()

	// Notify watchers if value changed
	if oldValue != value {
		for _, ch := range watchers {
			select {
			case ch <- value:
			case <-time.After(10 * time.Millisecond):
				// Skip slow watchers
			}
		}
	}

	return nil
}

func (m *MockConfig) Watch(key string) (<-chan interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan interface{}, 10)
	m.watchers[key] = append(m.watchers[key], ch)
	return ch, nil
}

func (m *MockConfig) AddProvider(provider ConfigProvider, priority int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.providers = append(m.providers, providerWithPriority{
		provider: provider,
		priority: priority,
	})

	return nil
}

// loadAll loads configuration from all providers, respecting priority
func (m *MockConfig) loadAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Sort providers by priority (lower priority first, so higher priority overwrites)
	// Simple bubble sort for small arrays
	for i := 0; i < len(m.providers); i++ {
		for j := i + 1; j < len(m.providers); j++ {
			if m.providers[i].priority > m.providers[j].priority {
				m.providers[i], m.providers[j] = m.providers[j], m.providers[i]
			}
		}
	}

	// Load from each provider in order (lower priority first)
	for _, pwp := range m.providers {
		data, err := pwp.provider.Load(ctx)
		if err != nil {
			return err
		}

		// Merge data (higher priority overwrites)
		for k, v := range data {
			m.data[k] = v
		}
	}

	return nil
}

// MockConfigProvider is a mock implementation of ConfigProvider
type MockConfigProvider struct {
	name string
	data map[string]interface{}
}

// NewMockConfigProvider creates a new MockConfigProvider
func NewMockConfigProvider(name string, data map[string]interface{}) *MockConfigProvider {
	return &MockConfigProvider{
		name: name,
		data: data,
	}
}

func (m *MockConfigProvider) Load(ctx context.Context) (map[string]interface{}, error) {
	return m.data, nil
}

func (m *MockConfigProvider) Watch(ctx context.Context) (<-chan ConfigChange, error) {
	ch := make(chan ConfigChange)
	// For testing, we don't need to implement watching
	return ch, nil
}

func (m *MockConfigProvider) Name() string {
	return m.name
}
