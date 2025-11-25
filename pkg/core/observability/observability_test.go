package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// ============================================================================
// StdoutLogger Tests
// ============================================================================

func TestStdoutLogger_LogsContainStructuredFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStdoutLoggerWithWriter(&buf)

	// Log with structured fields
	logger.Info("test message",
		types.Field{Key: "component", Value: "test"},
		types.Field{Key: "execution_id", Value: "exec-123"},
		types.Field{Key: "resource_id", Value: "res-456"},
	)

	// Parse the JSON output
	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	if err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"timestamp", "level", "message", "component", "execution_id", "resource_id"}
	for _, field := range requiredFields {
		if _, ok := entry[field]; !ok {
			t.Errorf("Log entry missing required field: %s", field)
		}
	}

	// Verify field values
	if entry["level"] != "INFO" {
		t.Errorf("Expected level=INFO, got %v", entry["level"])
	}
	if entry["message"] != "test message" {
		t.Errorf("Expected message='test message', got %v", entry["message"])
	}
	if entry["component"] != "test" {
		t.Errorf("Expected component='test', got %v", entry["component"])
	}
	if entry["execution_id"] != "exec-123" {
		t.Errorf("Expected execution_id='exec-123', got %v", entry["execution_id"])
	}
	if entry["resource_id"] != "res-456" {
		t.Errorf("Expected resource_id='res-456', got %v", entry["resource_id"])
	}
}

func TestStdoutLogger_AllLevels(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(types.Logger, string, ...types.Field)
		expected string
	}{
		{"Debug", func(l types.Logger, msg string, f ...types.Field) { l.Debug(msg, f...) }, "DEBUG"},
		{"Info", func(l types.Logger, msg string, f ...types.Field) { l.Info(msg, f...) }, "INFO"},
		{"Warn", func(l types.Logger, msg string, f ...types.Field) { l.Warn(msg, f...) }, "WARN"},
		{"Error", func(l types.Logger, msg string, f ...types.Field) { l.Error(msg, f...) }, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewStdoutLoggerWithWriter(&buf)

			tt.logFunc(logger, "test message")

			var entry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &entry)
			if err != nil {
				t.Fatalf("Failed to parse log output: %v", err)
			}

			if entry["level"] != tt.expected {
				t.Errorf("Expected level=%s, got %v", tt.expected, entry["level"])
			}
		})
	}
}

func TestStdoutLogger_With(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStdoutLoggerWithWriter(&buf)

	// Create a child logger with persistent fields
	childLogger := logger.With(
		types.Field{Key: "component", Value: "engine"},
		types.Field{Key: "execution_id", Value: "exec-789"},
	)

	// Log with the child logger
	childLogger.Info("child message", types.Field{Key: "resource_id", Value: "res-999"})

	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	if err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	// Verify persistent fields are present
	if entry["component"] != "engine" {
		t.Errorf("Expected component='engine', got %v", entry["component"])
	}
	if entry["execution_id"] != "exec-789" {
		t.Errorf("Expected execution_id='exec-789', got %v", entry["execution_id"])
	}
	if entry["resource_id"] != "res-999" {
		t.Errorf("Expected resource_id='res-999', got %v", entry["resource_id"])
	}
}

func TestStdoutLogger_WithDoesNotMutateParent(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	logger := NewStdoutLoggerWithWriter(&buf1)

	// Create child logger
	childLogger := logger.With(types.Field{Key: "child", Value: "true"})

	// Log with parent - should not have child field
	logger.Info("parent message")

	// Change writer for child to separate output
	childLogger = NewStdoutLoggerWithWriter(&buf2).With(types.Field{Key: "child", Value: "true"})
	childLogger.Info("child message")

	// Parse parent log
	var parentEntry map[string]interface{}
	err := json.Unmarshal(buf1.Bytes(), &parentEntry)
	if err != nil {
		t.Fatalf("Failed to parse parent log: %v", err)
	}

	// Parent should not have child field
	if _, ok := parentEntry["child"]; ok {
		t.Error("Parent logger should not have child field")
	}

	// Parse child log
	var childEntry map[string]interface{}
	err = json.Unmarshal(buf2.Bytes(), &childEntry)
	if err != nil {
		t.Fatalf("Failed to parse child log: %v", err)
	}

	// Child should have child field
	if childEntry["child"] != "true" {
		t.Error("Child logger should have child field")
	}
}

func TestStdoutLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStdoutLoggerWithWriter(&buf)

	logger.Info("test")

	// Should be valid JSON
	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	if err != nil {
		t.Errorf("Log output is not valid JSON: %v", err)
	}
}

// ============================================================================
// NoOpMetricsCollector Tests
// ============================================================================

func TestNoOpMetricsCollector_Counter(t *testing.T) {
	collector := NewNoOpMetricsCollector()
	counter := collector.Counter("test_counter", map[string]string{"tag": "value"})

	// Should not panic
	counter.Inc()
	counter.Add(5.0)
}

func TestNoOpMetricsCollector_Gauge(t *testing.T) {
	collector := NewNoOpMetricsCollector()
	gauge := collector.Gauge("test_gauge", map[string]string{"tag": "value"})

	// Should not panic
	gauge.Set(10.0)
	gauge.Inc()
	gauge.Dec()
	gauge.Add(5.0)
}

func TestNoOpMetricsCollector_Histogram(t *testing.T) {
	collector := NewNoOpMetricsCollector()
	histogram := collector.Histogram("test_histogram", map[string]string{"tag": "value"})

	// Should not panic
	histogram.Observe(1.5)
	histogram.Observe(2.5)
	histogram.Observe(3.5)
}

// ============================================================================
// NoOpTracer Tests
// ============================================================================

func TestNoOpTracer_StartSpan(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Should not panic
	newCtx, span := tracer.StartSpan(ctx, "test_span")

	if newCtx == nil {
		t.Error("StartSpan should return a non-nil context")
	}
	if span == nil {
		t.Error("StartSpan should return a non-nil span")
	}
}

func TestNoOpSpan_Operations(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, "test_span")

	// Should not panic
	span.SetTag("key", "value")
	span.LogEvent("test_event")
	span.LogFields(types.Field{Key: "field", Value: "value"})
	span.Finish()

	// Context should be preserved
	if span.Context() == nil {
		t.Error("Span context should not be nil")
	}
}

func TestNoOpSpan_WithOptions(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Create span with options
	opt := func(cfg *types.SpanConfig) {
		cfg.Tags = map[string]interface{}{"tag": "value"}
	}

	_, span := tracer.StartSpan(ctx, "test_span", opt)

	// Should not panic
	span.SetTag("another", "tag")
	span.Finish()
}

// ============================================================================
// DefaultObservabilityProvider Tests
// ============================================================================

func TestDefaultObservabilityProvider_Components(t *testing.T) {
	provider := NewDefaultObservabilityProvider()

	if provider.Logger() == nil {
		t.Error("Logger should not be nil")
	}
	if provider.Metrics() == nil {
		t.Error("Metrics should not be nil")
	}
	if provider.Tracer() == nil {
		t.Error("Tracer should not be nil")
	}
}

func TestDefaultObservabilityProvider_LoggerWorks(t *testing.T) {
	provider := NewDefaultObservabilityProvider()
	logger := provider.Logger()

	// Should not panic
	logger.Info("test message")
}

func TestDefaultObservabilityProvider_MetricsWork(t *testing.T) {
	provider := NewDefaultObservabilityProvider()
	metrics := provider.Metrics()

	// Should not panic
	counter := metrics.Counter("test", nil)
	counter.Inc()
}

func TestDefaultObservabilityProvider_TracerWorks(t *testing.T) {
	provider := NewDefaultObservabilityProvider()
	tracer := provider.Tracer()

	// Should not panic
	ctx, span := tracer.StartSpan(context.Background(), "test")
	span.Finish()
	if ctx == nil {
		t.Error("Context should not be nil")
	}
}

func TestNewObservabilityProvider_CustomComponents(t *testing.T) {
	var buf bytes.Buffer
	customLogger := NewStdoutLoggerWithWriter(&buf)
	customMetrics := NewNoOpMetricsCollector()
	customTracer := NewNoOpTracer()

	provider := NewObservabilityProvider(customLogger, customMetrics, customTracer)

	if provider.Logger() != customLogger {
		t.Error("Should use custom logger")
	}
	if provider.Metrics() != customMetrics {
		t.Error("Should use custom metrics")
	}
	if provider.Tracer() != customTracer {
		t.Error("Should use custom tracer")
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestObservability_Integration(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStdoutLoggerWithWriter(&buf)
	metrics := NewNoOpMetricsCollector()
	tracer := NewNoOpTracer()

	provider := NewObservabilityProvider(logger, metrics, tracer)

	// Simulate a plugin execution with observability
	ctx := context.Background()
	executionLogger := provider.Logger().With(
		types.Field{Key: "component", Value: "execution_engine"},
		types.Field{Key: "execution_id", Value: "exec-001"},
	)

	// Start trace
	ctx, span := provider.Tracer().StartSpan(ctx, "plugin_execution")
	defer span.Finish()

	// Log execution start
	executionLogger.Info("starting plugin execution",
		types.Field{Key: "plugin", Value: "kill"},
		types.Field{Key: "resource_id", Value: "container-123"},
	)

	// Record metrics
	counter := provider.Metrics().Counter("plugin_executions", map[string]string{
		"plugin": "kill",
		"status": "success",
	})
	counter.Inc()

	// Add span tags
	span.SetTag("plugin", "kill")
	span.SetTag("resource", "container-123")

	// Log execution complete
	executionLogger.Info("plugin execution completed")

	// Verify log output contains all fields
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines, got %d", len(lines))
	}

	// Parse first log line
	var entry1 map[string]interface{}
	err := json.Unmarshal([]byte(lines[0]), &entry1)
	if err != nil {
		t.Fatalf("Failed to parse first log line: %v", err)
	}

	// Verify structured fields
	if entry1["component"] != "execution_engine" {
		t.Error("Missing component field")
	}
	if entry1["execution_id"] != "exec-001" {
		t.Error("Missing execution_id field")
	}
	if entry1["plugin"] != "kill" {
		t.Error("Missing plugin field")
	}
	if entry1["resource_id"] != "container-123" {
		t.Error("Missing resource_id field")
	}
}
