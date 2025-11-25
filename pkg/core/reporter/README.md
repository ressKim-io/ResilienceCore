# Reporter Package

This package implements the DefaultReporter and related components for the Infrastructure Resilience Engine.

## Components

### 1. DefaultReporter (`reporter.go`)

The main implementation of the Reporter interface that provides:

- **RecordEvent**: Records events to storage
- **RecordExecution**: Records execution records to storage
- **QueryEvents**: Queries events with filtering support
- **QueryExecutions**: Queries execution records with filtering support
- **ComputeStatistics**: Calculates comprehensive statistics including:
  - Total executions, success/failure/timeout/canceled counts
  - Success rate
  - MTTR (Mean Time To Recovery)
  - Duration statistics (average, P50, P95, P99, min, max)
  - Per-plugin and per-resource breakdowns
- **GenerateReport**: Generates reports in various formats using registered formatters
- **SetStorage**: Allows pluggable storage backends
- **RegisterFormatter**: Allows pluggable report formatters

### 2. InMemoryStorage (`storage.go`)

A thread-safe in-memory implementation of the StorageBackend interface:

- **Save**: Stores key-value pairs in memory
- **Load**: Retrieves values by key
- **Query**: Supports filtering, sorting, and pagination
- **Delete**: Removes values by key
- **Close**: Cleans up resources

Features:
- Collection-based organization (key format: "collection:id")
- Reflection-based filtering on struct fields
- Multi-field sorting support
- Thread-safe with RWMutex

### 3. Report Formatters (`formatters.go`)

Three built-in formatters for generating reports:

#### JSONFormatter
- Formats data as indented JSON
- Content-Type: `application/json`

#### MarkdownFormatter
- Formats statistics, execution records, and events as Markdown tables
- Content-Type: `text/markdown`
- Includes:
  - Summary section with key metrics
  - Duration statistics
  - Per-plugin breakdown table
  - Per-resource breakdown table

#### HTMLFormatter
- Formats data as styled HTML with tables
- Content-Type: `text/html`
- Includes CSS styling for professional appearance

## Usage Example

```go
// Create a reporter with default in-memory storage
reporter := reporter.NewDefaultReporter()

// Record an execution
ctx := context.Background()
exec := types.ExecutionRecord{
    ID:           "exec-123",
    PluginName:   "kill-plugin",
    ResourceID:   "container-1",
    ResourceName: "redis",
    StartTime:    time.Now(),
    EndTime:      time.Now().Add(time.Second),
    Duration:     time.Second,
    Status:       types.StatusSuccess,
    Principal:    "admin",
}
reporter.RecordExecution(ctx, exec)

// Query executions
query := types.ExecutionQuery{
    PluginNames: []string{"kill-plugin"},
}
executions, _ := reporter.QueryExecutions(ctx, query)

// Compute statistics
filter := types.StatisticsFilter{
    StartTime: time.Now().Add(-24 * time.Hour),
    EndTime:   time.Now(),
}
stats, _ := reporter.ComputeStatistics(ctx, filter)

// Generate a report
reportFilter := types.ReportFilter{
    PluginNames: []string{"kill-plugin"},
}
jsonReport, _ := reporter.GenerateReport(ctx, "json", reportFilter)
markdownReport, _ := reporter.GenerateReport(ctx, "markdown", reportFilter)
htmlReport, _ := reporter.GenerateReport(ctx, "html", reportFilter)

// Use custom storage backend
customStorage := NewCustomStorage()
reporter.SetStorage(customStorage)

// Register custom formatter
customFormatter := NewCustomFormatter()
reporter.RegisterFormatter("custom", customFormatter)
```

## Testing

### Unit Tests (`reporter_test.go`)

Comprehensive unit tests covering:
- Event recording and querying
- Execution recording and querying
- Statistics calculation
- Report generation in all formats
- Storage backend management
- Formatter registration

Run with:
```bash
go test ./pkg/core/reporter
```

### Property-Based Tests

Located in `pkg/core/types/reporter_properties_test.go`:

- **Property 33**: Execution record is complete (validates Requirements 7.1)
- **Property 34**: Event record is complete (validates Requirements 7.2)
- **Property 35**: Statistics are calculated correctly (validates Requirements 7.3)
- **Property 36**: Report uses registered formatter (validates Requirements 7.4)
- **Property 37**: Custom storage backends are supported (validates Requirements 7.5)

Run with:
```bash
go test -v -run "TestProperty3[3-7]" ./pkg/core/types
```

## Design Principles

1. **Immutable Core**: The Reporter interface is defined in the Core and never modified
2. **Pluggable Storage**: Any storage backend implementing the StorageBackend interface can be used
3. **Pluggable Formatters**: Custom formatters can be registered without Core modification
4. **Thread-Safe**: All operations are thread-safe using appropriate synchronization
5. **Comprehensive Statistics**: Calculates MTTR, percentiles, and per-plugin/resource breakdowns
6. **Flexible Querying**: Supports filtering by multiple criteria with time ranges

## Requirements Validation

This implementation satisfies all requirements from the design document:

- ✅ **Requirement 7.1**: Complete execution records with all required fields
- ✅ **Requirement 7.2**: Complete event records with type, source, timestamp, and data
- ✅ **Requirement 7.3**: Accurate statistics calculation including MTTR and percentiles
- ✅ **Requirement 7.4**: Multiple report formats with pluggable formatters
- ✅ **Requirement 7.5**: Pluggable storage backends without Core modification
