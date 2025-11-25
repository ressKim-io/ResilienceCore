// Package observability provides default implementations of observability interfaces
package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// ============================================================================
// StdoutLogger - Structured logging to stdout
// ============================================================================

// StdoutLogger implements the Logger interface with structured logging to stdout
type StdoutLogger struct {
	mu     sync.Mutex
	writer io.Writer
	fields []types.Field
}

// NewStdoutLogger creates a new StdoutLogger
func NewStdoutLogger() *StdoutLogger {
	return &StdoutLogger{
		writer: os.Stdout,
		fields: make([]types.Field, 0),
	}
}

// NewStdoutLoggerWithWriter creates a new StdoutLogger with a custom writer
func NewStdoutLoggerWithWriter(w io.Writer) *StdoutLogger {
	return &StdoutLogger{
		writer: w,
		fields: make([]types.Field, 0),
	}
}

// Debug logs a debug message with structured fields
func (l *StdoutLogger) Debug(msg string, fields ...types.Field) {
	l.log("DEBUG", msg, fields...)
}

// Info logs an info message with structured fields
func (l *StdoutLogger) Info(msg string, fields ...types.Field) {
	l.log("INFO", msg, fields...)
}

// Warn logs a warning message with structured fields
func (l *StdoutLogger) Warn(msg string, fields ...types.Field) {
	l.log("WARN", msg, fields...)
}

// Error logs an error message with structured fields
func (l *StdoutLogger) Error(msg string, fields ...types.Field) {
	l.log("ERROR", msg, fields...)
}

// With returns a new logger with additional fields
func (l *StdoutLogger) With(fields ...types.Field) types.Logger {
	newFields := make([]types.Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &StdoutLogger{
		writer: l.writer,
		fields: newFields,
	}
}

// log writes a structured log entry
func (l *StdoutLogger) log(level, msg string, fields ...types.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Build log entry with all fields
	entry := make(map[string]interface{})
	entry["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	entry["level"] = level
	entry["message"] = msg

	// Add logger's persistent fields
	for _, f := range l.fields {
		entry[f.Key] = f.Value
	}

	// Add call-specific fields
	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple format if JSON marshaling fails
		fmt.Fprintf(l.writer, "[%s] %s %s\n", level, time.Now().Format(time.RFC3339), msg)
		return
	}

	fmt.Fprintf(l.writer, "%s\n", data)
}

// ============================================================================
// NoOpMetricsCollector - Metrics collector that does nothing
// ============================================================================

// NoOpMetricsCollector implements the MetricsCollector interface but does nothing
type NoOpMetricsCollector struct{}

// NewNoOpMetricsCollector creates a new NoOpMetricsCollector
func NewNoOpMetricsCollector() *NoOpMetricsCollector {
	return &NoOpMetricsCollector{}
}

// Counter returns a no-op counter
func (m *NoOpMetricsCollector) Counter(name string, tags map[string]string) types.Counter {
	return &noOpCounter{}
}

// Gauge returns a no-op gauge
func (m *NoOpMetricsCollector) Gauge(name string, tags map[string]string) types.Gauge {
	return &noOpGauge{}
}

// Histogram returns a no-op histogram
func (m *NoOpMetricsCollector) Histogram(name string, tags map[string]string) types.Histogram {
	return &noOpHistogram{}
}

// noOpCounter is a counter that does nothing
type noOpCounter struct{}

func (c *noOpCounter) Inc()              {}
func (c *noOpCounter) Add(delta float64) {}

// noOpGauge is a gauge that does nothing
type noOpGauge struct{}

func (g *noOpGauge) Set(value float64)   {}
func (g *noOpGauge) Inc()                {}
func (g *noOpGauge) Dec()                {}
func (g *noOpGauge) Add(delta float64)   {}

// noOpHistogram is a histogram that does nothing
type noOpHistogram struct{}

func (h *noOpHistogram) Observe(value float64) {}

// ============================================================================
// NoOpTracer - Tracer that does nothing
// ============================================================================

// NoOpTracer implements the Tracer interface but does nothing
type NoOpTracer struct{}

// NewNoOpTracer creates a new NoOpTracer
func NewNoOpTracer() *NoOpTracer {
	return &NoOpTracer{}
}

// StartSpan creates a no-op span
func (t *NoOpTracer) StartSpan(ctx context.Context, name string, opts ...types.SpanOption) (context.Context, types.Span) {
	return ctx, &noOpSpan{ctx: ctx}
}

// noOpSpan is a span that does nothing
type noOpSpan struct {
	ctx context.Context
}

func (s *noOpSpan) SetTag(key string, value interface{})  {}
func (s *noOpSpan) LogEvent(event string)                 {}
func (s *noOpSpan) LogFields(fields ...types.Field)       {}
func (s *noOpSpan) Finish()                               {}
func (s *noOpSpan) Context() context.Context              { return s.ctx }

// ============================================================================
// DefaultObservabilityProvider - Combines all observability components
// ============================================================================

// DefaultObservabilityProvider implements the ObservabilityProvider interface
type DefaultObservabilityProvider struct {
	logger  types.Logger
	metrics types.MetricsCollector
	tracer  types.Tracer
}

// NewDefaultObservabilityProvider creates a new DefaultObservabilityProvider
// with StdoutLogger, NoOpMetricsCollector, and NoOpTracer
func NewDefaultObservabilityProvider() *DefaultObservabilityProvider {
	return &DefaultObservabilityProvider{
		logger:  NewStdoutLogger(),
		metrics: NewNoOpMetricsCollector(),
		tracer:  NewNoOpTracer(),
	}
}

// NewObservabilityProvider creates a new ObservabilityProvider with custom components
func NewObservabilityProvider(logger types.Logger, metrics types.MetricsCollector, tracer types.Tracer) *DefaultObservabilityProvider {
	return &DefaultObservabilityProvider{
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
	}
}

// Logger returns the logger
func (p *DefaultObservabilityProvider) Logger() types.Logger {
	return p.logger
}

// Metrics returns the metrics collector
func (p *DefaultObservabilityProvider) Metrics() types.MetricsCollector {
	return p.metrics
}

// Tracer returns the tracer
func (p *DefaultObservabilityProvider) Tracer() types.Tracer {
	return p.tracer
}
