// Package types provides property-based tests for the Observability interfaces
package types

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty53_LogsContainStructuredFields verifies that all log entries
// contain structured fields including Timestamp, Level, Component, ExecutionID, and ResourceID
// Feature: infrastructure-resilience-engine, Property 53: Logs contain structured fields
// Validates: Requirements 11.1
func TestProperty53_LogsContainStructuredFields(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("logs contain all required structured fields", prop.ForAll(
		func(msg string, component string, executionID string, resourceID string) bool {
			// Create a mock logger that captures log entries
			logger := NewMockLogger()

			// Create structured fields
			fields := []Field{
				{Key: "component", Value: component},
				{Key: "execution_id", Value: executionID},
				{Key: "resource_id", Value: resourceID},
				{Key: "timestamp", Value: time.Now()},
			}

			// Log at different levels
			logger.Debug(msg, fields...)
			logger.Info(msg, fields...)
			logger.Warn(msg, fields...)
			logger.Error(msg, fields...)

			// Verify all log entries contain the required fields
			for _, entry := range logger.Entries {
				// Check that all required fields are present
				hasComponent := false
				hasExecutionID := false
				hasResourceID := false
				hasTimestamp := false

				for _, field := range entry.Fields {
					switch field.Key {
					case "component":
						hasComponent = true
						if field.Value != component {
							return false
						}
					case "execution_id":
						hasExecutionID = true
						if field.Value != executionID {
							return false
						}
					case "resource_id":
						hasResourceID = true
						if field.Value != resourceID {
							return false
						}
					case "timestamp":
						hasTimestamp = true
						if _, ok := field.Value.(time.Time); !ok {
							return false
						}
					}
				}

				// All required fields must be present
				if !hasComponent || !hasExecutionID || !hasResourceID || !hasTimestamp {
					return false
				}

				// Verify the message is present
				if entry.Message != msg {
					return false
				}

				// Verify the level is set
				if entry.Level == "" {
					return false
				}
			}

			return len(logger.Entries) == 4 // We logged 4 times
		},
		gen.AlphaString(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty55_TraceSpanIsCreatedAndPropagated verifies that when an execution starts,
// a trace span is created and propagated through the execution context
// Feature: infrastructure-resilience-engine, Property 55: Trace span is created and propagated
// Validates: Requirements 11.3
func TestProperty55_TraceSpanIsCreatedAndPropagated(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("trace span is created and propagated in context", prop.ForAll(
		func(spanName string, tagKey string, tagValue string) bool {
			// Create a mock tracer
			tracer := NewMockTracer()

			// Start a span
			ctx := context.Background()
			newCtx, span := tracer.StartSpan(ctx, spanName)

			// Verify span was created
			if span == nil {
				return false
			}

			// Verify span is in the context
			if newCtx == ctx {
				// Context should be different (contains span)
				return false
			}

			// Set a tag on the span
			span.SetTag(tagKey, tagValue)

			// Verify the tag was set
			mockSpan, ok := span.(*MockSpan)
			if !ok {
				return false
			}

			if mockSpan.Tags[tagKey] != tagValue {
				return false
			}

			// Verify span name
			if mockSpan.Name != spanName {
				return false
			}

			// Verify span can be finished
			span.Finish()
			if !mockSpan.Finished {
				return false
			}

			// Verify span context can be retrieved
			spanCtx := span.Context()
			return spanCtx != nil
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty57_CustomObservabilityProvidersAreSupported verifies that custom
// ObservabilityProvider implementations can be used without Core modification
// Feature: infrastructure-resilience-engine, Property 57: Custom observability providers are supported
// Validates: Requirements 11.5
func TestProperty57_CustomObservabilityProvidersAreSupported(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("custom observability providers work without Core modification", prop.ForAll(
		func(providerName string) bool {
			// Create different types of custom observability providers
			var provider ObservabilityProvider

			switch providerName {
			case "stdout":
				provider = &MockStdoutObservabilityProvider{}
			case "prometheus":
				provider = &MockPrometheusObservabilityProvider{}
			case "jaeger":
				provider = &MockJaegerObservabilityProvider{}
			case "opentelemetry":
				provider = &MockOpenTelemetryObservabilityProvider{}
			default:
				provider = &MockObservabilityProvider{name: providerName}
			}

			// Verify all components are accessible
			logger := provider.Logger()
			if logger == nil {
				return false
			}

			metrics := provider.Metrics()
			if metrics == nil {
				return false
			}

			tracer := provider.Tracer()
			if tracer == nil {
				return false
			}

			// Verify logger works
			logger.Info("test message", Field{Key: "test", Value: "value"})

			// Verify metrics work
			counter := metrics.Counter("test_counter", map[string]string{"label": "value"})
			if counter == nil {
				return false
			}
			counter.Inc()

			gauge := metrics.Gauge("test_gauge", map[string]string{"label": "value"})
			if gauge == nil {
				return false
			}
			gauge.Set(42.0)

			histogram := metrics.Histogram("test_histogram", map[string]string{"label": "value"})
			if histogram == nil {
				return false
			}
			histogram.Observe(1.5)

			// Verify tracer works
			ctx, span := tracer.StartSpan(context.Background(), "test_span")
			if span == nil || ctx == nil {
				return false
			}
			span.SetTag("key", "value")
			span.Finish()

			return true
		},
		gen.OneConstOf("stdout", "prometheus", "jaeger", "opentelemetry", "custom"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ============================================================================
// Platform-specific mock observability providers
// ============================================================================

type MockStdoutObservabilityProvider struct {
	MockObservabilityProvider
}

type MockPrometheusObservabilityProvider struct {
	MockObservabilityProvider
}

type MockJaegerObservabilityProvider struct {
	MockObservabilityProvider
}

type MockOpenTelemetryObservabilityProvider struct {
	MockObservabilityProvider
}
