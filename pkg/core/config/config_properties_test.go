package config

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
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

			config := NewDefaultConfig()
			defer config.Close()

			// Add multiple providers with different priorities
			for i := 0; i < numProviders; i++ {
				provider := &testConfigProvider{
					name: fmt.Sprintf("provider-%d", i),
					values: map[string]interface{}{
						key: values[i],
					},
				}
				if err := config.AddProvider(provider, i); err != nil {
					return false
				}
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

			config := NewDefaultConfig()
			defer config.Close()

			// Add low priority provider
			lowProvider := &testConfigProvider{
				name: "low-priority",
				values: map[string]interface{}{
					key: lowPriorityValue,
				},
			}
			if err := config.AddProvider(lowProvider, 1); err != nil {
				return false
			}

			// Add high priority provider
			highProvider := &testConfigProvider{
				name: "high-priority",
				values: map[string]interface{}{
					key: highPriorityValue,
				},
			}
			if err := config.AddProvider(highProvider, 10); err != nil {
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

// TestProperty45_ConfigValuesAreTypeConverted verifies that
// for any config key and requested type, Get{Type} methods return the value converted to the requested type or an error
// Feature: infrastructure-resilience-engine, Property 45: Config values are type-converted
// Validates: Requirements 9.3
func TestProperty45_ConfigValuesAreTypeConverted(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("config values are type-converted", prop.ForAll(
		func(key string, intValue int) bool {
			config := NewDefaultConfig()
			defer config.Close()

			// Set an integer value
			if err := config.Set(key, intValue); err != nil {
				return false
			}

			// Get as int
			gotInt, err := config.GetInt(key)
			if err != nil {
				return false
			}
			if gotInt != intValue {
				return false
			}

			// Get as string (should convert)
			gotString, err := config.GetString(key)
			if err != nil {
				return false
			}
			expectedString := fmt.Sprintf("%d", intValue)
			if gotString != expectedString {
				return false
			}

			// Set a string value
			stringKey := key + "_string"
			stringValue := "123"
			if err := config.Set(stringKey, stringValue); err != nil {
				return false
			}

			// Get as int (should convert)
			gotIntFromString, err := config.GetInt(stringKey)
			if err != nil {
				return false
			}
			if gotIntFromString != 123 {
				return false
			}

			// Set a bool value
			boolKey := key + "_bool"
			if err := config.Set(boolKey, true); err != nil {
				return false
			}

			// Get as bool
			gotBool, err := config.GetBool(boolKey)
			if err != nil {
				return false
			}
			if !gotBool {
				return false
			}

			// Get as string (should convert)
			gotBoolString, err := config.GetString(boolKey)
			if err != nil {
				return false
			}
			if gotBoolString != "true" {
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.Int(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty46_ConfigChangesNotifyWatchers verifies that
// for any config key being watched, changes to that key are delivered through the watch channel
// Feature: infrastructure-resilience-engine, Property 46: Config changes notify watchers
// Validates: Requirements 9.4
func TestProperty46_ConfigChangesNotifyWatchers(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("config changes notify watchers", prop.ForAll(
		func(key string, initialValue string, newValue string) bool {
			// Skip if values are the same
			if initialValue == newValue {
				return true
			}

			config := NewDefaultConfig()
			defer config.Close()

			// Set initial value
			if err := config.Set(key, initialValue); err != nil {
				return false
			}

			// Start watching
			ch, err := config.Watch(key)
			if err != nil {
				return false
			}

			// Change value in goroutine
			done := make(chan bool)
			go func() {
				time.Sleep(50 * time.Millisecond)
				config.Set(key, newValue)
				done <- true
			}()

			// Wait for notification
			select {
			case value := <-ch:
				<-done // Wait for goroutine to finish
				return value == newValue
			case <-time.After(1 * time.Second):
				<-done // Wait for goroutine to finish
				return false
			}
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

			config := NewDefaultConfig()
			defer config.Close()

			// Create a custom config provider
			customData := make(map[string]interface{})
			for k, v := range configData {
				customData[k] = v
			}

			customProvider := &testConfigProvider{
				name:   providerName,
				values: customData,
			}

			// Add the custom provider
			if err := config.AddProvider(customProvider, 1); err != nil {
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
// Test Helper Types
// ============================================================================

// testConfigProvider is a simple test implementation of ConfigProvider
type testConfigProvider struct {
	name   string
	values map[string]interface{}
}

func (p *testConfigProvider) Load(ctx context.Context) (map[string]interface{}, error) {
	return p.values, nil
}

func (p *testConfigProvider) Watch(ctx context.Context) (<-chan types.ConfigChange, error) {
	ch := make(chan types.ConfigChange)
	// For testing, we don't need to implement watching
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
}

func (p *testConfigProvider) Name() string {
	return p.name
}
