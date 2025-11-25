# Observability Package

This package provides default implementations of the observability interfaces defined in `pkg/core/types`.

## Components

### StdoutLogger

A structured logger that outputs JSON-formatted logs to stdout (or any `io.Writer`).

**Features:**
- Structured logging with arbitrary fields
- JSON output format
- Support for persistent fields via `With()`
- Thread-safe

**Example:**
```go
logger := observability.NewStdoutLogger()

// Log with structured fields
logger.Info("plugin execution started",
    types.Field{Key: "component", Value: "engine"},
    types.Field{Key: "execution_id", Value: "exec-123"},
    types.Field{Key: "resource_id", Value: "res-456"},
)

// Create child logger with persistent fields
childLogger := logger.With(
    types.Field{Key: "component", Value: "engine"},
)
childLogger.Info("processing step 1")
childLogger.Info("processing step 2")
```

**Output:**
```json
{"timestamp":"2024-01-15T10:30:45.123Z","level":"INFO","message":"plugin execution started","component":"engine","execution_id":"exec-123","resource_id":"res-456"}
```

### NoOpMetricsCollector

A metrics collector that implements the `MetricsCollector` interface but does nothing. Useful for testing or when metrics are not needed.

**Example:**
```go
metrics := observability.NewNoOpMetricsCollector()

// All operations are no-ops
counter := metrics.Counter("executions", map[string]string{"status": "success"})
counter.Inc()

gauge := metrics.Gauge("active_plugins", nil)
gauge.Set(5)

histogram := metrics.Histogram("execution_duration", nil)
histogram.Observe(1.5)
```

### NoOpTracer

A tracer that implements the `Tracer` interface but does nothing. Useful for testing or when tracing is not needed.

**Example:**
```go
tracer := observability.NewNoOpTracer()

ctx, span := tracer.StartSpan(context.Background(), "plugin_execution")
defer span.Finish()

span.SetTag("plugin", "kill")
span.LogEvent("validation_complete")
```

### DefaultObservabilityProvider

Combines all three components into a single provider.

**Example:**
```go
// Use default components
provider := observability.NewDefaultObservabilityProvider()

logger := provider.Logger()
metrics := provider.Metrics()
tracer := provider.Tracer()

// Or create with custom components
customLogger := observability.NewStdoutLogger()
customMetrics := observability.NewNoOpMetricsCollector()
customTracer := observability.NewNoOpTracer()

provider := observability.NewObservabilityProvider(customLogger, customMetrics, customTracer)
```

## Design Principles

1. **Structured Logging**: All logs are JSON-formatted with structured fields for easy parsing and analysis
2. **No-Op Implementations**: Metrics and tracing have no-op implementations to minimize overhead when not needed
3. **Extensibility**: Custom implementations can be provided through the interfaces
4. **Thread Safety**: All implementations are thread-safe

## Property Validation

This package validates the following correctness properties:

- **Property 53**: Logs contain structured fields (timestamp, level, component, execution_id, resource_id)
- **Property 57**: Custom observability providers are supported without Core modification

## Testing

Run tests:
```bash
go test ./pkg/core/observability/...
```

Run with coverage:
```bash
go test -cover ./pkg/core/observability/...
```

## Future Enhancements

- **PrometheusMetricsCollector**: Export metrics to Prometheus
- **JaegerTracer**: Distributed tracing with Jaeger
- **OpenTelemetry**: Integration with OpenTelemetry
- **FileLogger**: Log to files with rotation
- **SyslogLogger**: Log to syslog
