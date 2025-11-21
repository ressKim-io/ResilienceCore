// Package types provides shared mock implementations for testing
package types

import (
	"context"
	"strings"
)

// ============================================================================
// Shared Mock Logger Implementation
// ============================================================================

// MockLogger is a mock implementation of Logger that captures log entries
type MockLogger struct {
	Entries []LogEntry
	Fields  []Field
}

type LogEntry struct {
	Level   string
	Message string
	Fields  []Field
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		Entries: make([]LogEntry, 0),
		Fields:  make([]Field, 0),
	}
}

func (m *MockLogger) Debug(msg string, fields ...Field) {
	m.Entries = append(m.Entries, LogEntry{
		Level:   "debug",
		Message: msg,
		Fields:  append(m.Fields, fields...),
	})
}

func (m *MockLogger) Info(msg string, fields ...Field) {
	m.Entries = append(m.Entries, LogEntry{
		Level:   "info",
		Message: msg,
		Fields:  append(m.Fields, fields...),
	})
}

func (m *MockLogger) Warn(msg string, fields ...Field) {
	m.Entries = append(m.Entries, LogEntry{
		Level:   "warn",
		Message: msg,
		Fields:  append(m.Fields, fields...),
	})
}

func (m *MockLogger) Error(msg string, fields ...Field) {
	m.Entries = append(m.Entries, LogEntry{
		Level:   "error",
		Message: msg,
		Fields:  append(m.Fields, fields...),
	})
}

func (m *MockLogger) With(fields ...Field) Logger {
	newLogger := &MockLogger{
		Entries: m.Entries,
		Fields:  append(m.Fields, fields...),
	}
	return newLogger
}

// ============================================================================
// Shared Mock Metrics Collector Implementation
// ============================================================================

// MockMetricsCollector is a mock implementation of MetricsCollector
type MockMetricsCollector struct {
	Counters   map[string]*MockCounter
	Gauges     map[string]*MockGauge
	Histograms map[string]*MockHistogram
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		Counters:   make(map[string]*MockCounter),
		Gauges:     make(map[string]*MockGauge),
		Histograms: make(map[string]*MockHistogram),
	}
}

func (m *MockMetricsCollector) Counter(name string, tags map[string]string) Counter {
	key := m.metricKey(name, tags)
	if _, exists := m.Counters[key]; !exists {
		m.Counters[key] = &MockCounter{Name: name, Tags: tags}
	}
	return m.Counters[key]
}

func (m *MockMetricsCollector) Gauge(name string, tags map[string]string) Gauge {
	key := m.metricKey(name, tags)
	if _, exists := m.Gauges[key]; !exists {
		m.Gauges[key] = &MockGauge{Name: name, Tags: tags}
	}
	return m.Gauges[key]
}

func (m *MockMetricsCollector) Histogram(name string, tags map[string]string) Histogram {
	key := m.metricKey(name, tags)
	if _, exists := m.Histograms[key]; !exists {
		m.Histograms[key] = &MockHistogram{Name: name, Tags: tags}
	}
	return m.Histograms[key]
}

func (m *MockMetricsCollector) metricKey(name string, tags map[string]string) string {
	var parts []string
	parts = append(parts, name)
	for k, v := range tags {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

// MockCounter is a mock implementation of Counter
type MockCounter struct {
	Name  string
	Tags  map[string]string
	Value float64
}

func (m *MockCounter) Inc() {
	m.Value++
}

func (m *MockCounter) Add(delta float64) {
	m.Value += delta
}

// MockGauge is a mock implementation of Gauge
type MockGauge struct {
	Name  string
	Tags  map[string]string
	Value float64
}

func (m *MockGauge) Set(value float64) {
	m.Value = value
}

func (m *MockGauge) Inc() {
	m.Value++
}

func (m *MockGauge) Dec() {
	m.Value--
}

func (m *MockGauge) Add(delta float64) {
	m.Value += delta
}

// MockHistogram is a mock implementation of Histogram
type MockHistogram struct {
	Name   string
	Tags   map[string]string
	Values []float64
}

func (m *MockHistogram) Observe(value float64) {
	m.Values = append(m.Values, value)
}

// ============================================================================
// Shared Mock Tracer Implementation
// ============================================================================

// MockTracer is a mock implementation of Tracer
type MockTracer struct {
	Spans []*MockSpan
}

func NewMockTracer() *MockTracer {
	return &MockTracer{
		Spans: make([]*MockSpan, 0),
	}
}

func (m *MockTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	config := &SpanConfig{
		Tags: make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(config)
	}

	span := &MockSpan{
		Name:     name,
		Tags:     config.Tags,
		Events:   make([]string, 0),
		Fields:   make([]Field, 0),
		Finished: false,
		ctx:      ctx,
	}

	m.Spans = append(m.Spans, span)

	// Create a new context with the span
	newCtx := context.WithValue(ctx, "span", span)

	return newCtx, span
}

// MockSpan is a mock implementation of Span
type MockSpan struct {
	Name     string
	Tags     map[string]interface{}
	Events   []string
	Fields   []Field
	Finished bool
	ctx      context.Context
}

func (m *MockSpan) SetTag(key string, value interface{}) {
	m.Tags[key] = value
}

func (m *MockSpan) LogEvent(event string) {
	m.Events = append(m.Events, event)
}

func (m *MockSpan) LogFields(fields ...Field) {
	m.Fields = append(m.Fields, fields...)
}

func (m *MockSpan) Finish() {
	m.Finished = true
}

func (m *MockSpan) Context() context.Context {
	return m.ctx
}

// ============================================================================
// Shared Mock Observability Provider Implementation
// ============================================================================

// MockObservabilityProvider is a mock implementation of ObservabilityProvider
type MockObservabilityProvider struct {
	name string
}

func (m *MockObservabilityProvider) Logger() Logger {
	return NewMockLogger()
}

func (m *MockObservabilityProvider) Metrics() MetricsCollector {
	return NewMockMetricsCollector()
}

func (m *MockObservabilityProvider) Tracer() Tracer {
	return NewMockTracer()
}
