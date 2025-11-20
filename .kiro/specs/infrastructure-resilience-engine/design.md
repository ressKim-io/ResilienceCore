# Design Document

## Overview

Infrastructure Resilience Engine is a completely immutable framework for building infrastructure resilience testing applications. The core design principle is **zero Core modification** - all extensions (new environments, plugins, storage backends, monitoring strategies) are added through well-defined interfaces without touching the Core codebase.

The framework consists of:
- **Core**: Immutable interfaces and default implementations
- **Adapters**: Environment-specific implementations (docker-compose, K8s, ECS, etc.)
- **Plugins**: Feature implementations (Kill, Restart, Backup, etc.)
- **Applications**: Compositions of Core + Adapters + Plugins (GrimOps, NecroOps, etc.)

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Core (Immutable)                          │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Execution    │  │   Monitor    │  │   Reporter   │     │
│  │   Engine     │  │              │  │              │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │  EventBus    │  │    Config    │  │ Plugin       │     │
│  │              │  │              │  │ Registry     │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │           Interface Definitions                     │    │
│  │  - EnvironmentAdapter                              │    │
│  │  - Plugin                                           │    │
│  │  - StorageBackend                                   │    │
│  │  - HealthCheckStrategy                              │    │
│  │  - ExecutionStrategy                                │    │
│  │  - ReportFormatter                                  │    │
│  │  - ConfigProvider                                   │    │
│  │  - AuthorizationProvider                            │    │
│  │  - ObservabilityProvider                            │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │ Implements
        ┌───────────────────┴───────────────────┐
        │                                       │
┌───────▼────────┐                    ┌────────▼────────┐
│   Adapters     │                    │    Plugins      │
│                │                    │                 │
│ - Compose      │                    │ - Kill          │
│ - Kubernetes   │                    │ - Restart       │
│ - ECS          │                    │ - Backup        │
│ - Nomad        │                    │ - Scale         │
└────────────────┘                    └─────────────────┘
        │                                       │
        └───────────────────┬───────────────────┘
                            │ Compose
                    ┌───────▼────────┐
                    │  Applications  │
                    │                │
                    │ - GrimOps      │
                    │ - NecroOps     │
                    │ - BackupOps    │
                    └────────────────┘
```


## Components and Interfaces

### 1. Resource Model (Environment-Agnostic)

The Resource model is the foundation of environment independence. It represents any infrastructure resource (container, pod, task, VM, etc.) in a unified way.

```go
// Resource represents any infrastructure resource across all environments
type Resource struct {
    // Core identification
    ID   string // Unique identifier (container ID, pod UID, task ARN, etc.)
    Name string // Human-readable name
    Kind string // Resource type (container, pod, task, instance, etc.) - free-form string
    
    // Organization
    Labels      map[string]string // Key-value labels for selection and grouping
    Annotations map[string]string // Additional metadata
    
    // State
    Status ResourceStatus
    
    // Specification
    Spec ResourceSpec
    
    // Environment-specific extensions
    Metadata map[string]interface{} // Arbitrary data for environment-specific needs
}

type ResourceStatus struct {
    Phase      string      // running, stopped, pending, failed, unknown (environment-defined)
    Conditions []Condition // Detailed conditions
    Message    string      // Human-readable status message
    Reason     string      // Machine-readable reason code
}

type Condition struct {
    Type    string    // Ready, Healthy, Initialized, etc.
    Status  string    // True, False, Unknown
    Reason  string    // CamelCase reason
    Message string    // Human-readable message
    LastTransitionTime time.Time
}

type ResourceSpec struct {
    Image       string            // Container/VM image
    Replicas    *int32            // Desired replica count (nil if not applicable)
    Ports       []Port            // Exposed ports
    Environment map[string]string // Environment variables
    Volumes     []Volume          // Mounted volumes
    Command     []string          // Entrypoint command
    Args        []string          // Command arguments
    
    // Health check configuration
    HealthCheck *HealthCheckConfig
}

type Port struct {
    Name          string
    ContainerPort int32
    HostPort      int32
    Protocol      string // TCP, UDP
}

type Volume struct {
    Name      string
    MountPath string
    ReadOnly  bool
    Source    VolumeSource
}

type VolumeSource struct {
    HostPath      *string // Host path
    ConfigMap     *string // ConfigMap name
    Secret        *string // Secret name
    EmptyDir      bool    // Temporary directory
}

type HealthCheckConfig struct {
    Type     string        // http, tcp, exec, custom
    Endpoint string        // URL for HTTP, address for TCP, command for exec
    Interval time.Duration // Check interval
    Timeout  time.Duration // Check timeout
    Retries  int           // Consecutive failures before unhealthy
}
```

### 2. EnvironmentAdapter Interface

The EnvironmentAdapter abstracts all environment-specific operations. Each infrastructure platform (docker-compose, K8s, ECS, etc.) implements this interface.

```go
type EnvironmentAdapter interface {
    // Lifecycle
    Initialize(ctx context.Context, config AdapterConfig) error
    Close() error
    
    // Resource discovery and management
    ListResources(ctx context.Context, filter ResourceFilter) ([]Resource, error)
    GetResource(ctx context.Context, id string) (Resource, error)
    
    // Resource operations
    StartResource(ctx context.Context, id string) error
    StopResource(ctx context.Context, id string, gracePeriod time.Duration) error
    RestartResource(ctx context.Context, id string) error
    DeleteResource(ctx context.Context, id string, options DeleteOptions) error
    CreateResource(ctx context.Context, spec ResourceSpec) (Resource, error)
    UpdateResource(ctx context.Context, id string, spec ResourceSpec) (Resource, error)
    
    // Real-time monitoring
    WatchResources(ctx context.Context, filter ResourceFilter) (<-chan ResourceEvent, error)
    
    // Execution
    ExecInResource(ctx context.Context, id string, cmd []string, options ExecOptions) (ExecResult, error)
    
    // Metrics
    GetMetrics(ctx context.Context, id string) (Metrics, error)
    
    // Metadata
    GetAdapterInfo() AdapterInfo
}

type AdapterConfig struct {
    // Environment-specific configuration
    // For Compose: docker socket path, compose file path
    // For K8s: kubeconfig path, namespace
    // For ECS: region, cluster name
    Config map[string]interface{}
}

type ResourceFilter struct {
    // Label selectors
    LabelSelector LabelSelector
    
    // Field selectors
    Kinds    []string // Filter by resource kind
    Statuses []string // Filter by status phase
    Names    []string // Filter by name (supports wildcards)
    
    // Pagination
    Limit  int
    Offset int
}

type LabelSelector struct {
    MatchLabels      map[string]string   // Equality-based (AND)
    MatchExpressions []SelectorRequirement // Set-based
}

type SelectorRequirement struct {
    Key      string
    Operator string // In, NotIn, Exists, DoesNotExist, Equals, NotEquals
    Values   []string
}

type ResourceEvent struct {
    Type     EventType // Added, Modified, Deleted
    Resource Resource
}

type EventType string

const (
    EventAdded    EventType = "Added"
    EventModified EventType = "Modified"
    EventDeleted  EventType = "Deleted"
)

type DeleteOptions struct {
    GracePeriod time.Duration
    Force       bool
}

type ExecOptions struct {
    Stdin  io.Reader
    Stdout io.Writer
    Stderr io.Writer
    TTY    bool
}

type ExecResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
}

type Metrics struct {
    Timestamp   time.Time
    ResourceID  string
    CPU         CPUMetrics
    Memory      MemoryMetrics
    Network     NetworkMetrics
    Disk        DiskMetrics
}

type CPUMetrics struct {
    UsagePercent float64
    UsageCores   float64
}

type MemoryMetrics struct {
    UsageBytes uint64
    LimitBytes uint64
    UsagePercent float64
}

type NetworkMetrics struct {
    RxBytes uint64
    TxBytes uint64
    RxPackets uint64
    TxPackets uint64
}

type DiskMetrics struct {
    ReadBytes  uint64
    WriteBytes uint64
    ReadOps    uint64
    WriteOps   uint64
}

type AdapterInfo struct {
    Name        string
    Version     string
    Environment string // compose, kubernetes, ecs, nomad, etc.
    Capabilities []string // List of supported features
}
```


### 3. Plugin Interface

The Plugin interface defines the contract for all feature implementations. Plugins are completely independent and can be developed without Core modification.

```go
type Plugin interface {
    // Metadata
    Metadata() PluginMetadata
    
    // Lifecycle hooks
    Initialize(config PluginConfig) error
    Validate(ctx PluginContext, resource Resource) error
    PreExecute(ctx PluginContext, resource Resource) (Snapshot, error)
    Execute(ctx PluginContext, resource Resource) error
    PostExecute(ctx PluginContext, resource Resource, result ExecutionResult) error
    Cleanup(ctx PluginContext, resource Resource) error
    
    // Optional: Rollback support
    Rollback(ctx PluginContext, resource Resource, snapshot Snapshot) error
}

type PluginMetadata struct {
    Name        string
    Version     string
    Description string
    Author      string
    
    // Compatibility
    SupportedKinds []string // Resource kinds this plugin supports (empty = all)
    RequiredCapabilities []Capability
    
    // Dependencies
    Dependencies []PluginDependency
    
    // Configuration schema
    ConfigSchema interface{} // JSON schema for plugin configuration
}

type Capability string

const (
    CapabilityExec    Capability = "exec"    // Requires exec in resources
    CapabilityNetwork Capability = "network" // Requires network manipulation
    CapabilityAdmin   Capability = "admin"   // Requires admin privileges
)

type PluginDependency struct {
    PluginName string
    Version    string // Semantic version constraint (e.g., ">=1.0.0", "^2.0.0")
}

type PluginConfig interface {
    Validate() error
}

type PluginContext struct {
    context.Context
    
    // Execution metadata
    ExecutionID string
    Timeout     time.Duration
    
    // Core components
    Env      EnvironmentAdapter
    Monitor  Monitor
    Reporter Reporter
    EventBus EventBus
    Logger   Logger
    Tracer   Tracer
    
    // Progress reporting
    Progress chan<- ProgressUpdate
    
    // Security
    Principal string
    Auth      AuthorizationProvider
    Secrets   SecretProvider
}

type ProgressUpdate struct {
    Percent int
    Message string
    Stage   string
}

type Snapshot interface {
    // Restore restores the resource to the state captured in this snapshot
    Restore(ctx context.Context) error
    
    // Serialize converts the snapshot to bytes for storage
    Serialize() ([]byte, error)
    
    // Deserialize reconstructs the snapshot from bytes
    Deserialize(data []byte) error
}

type ExecutionResult struct {
    Status    ExecutionStatus
    StartTime time.Time
    EndTime   time.Time
    Duration  time.Duration
    Error     error
    Metadata  map[string]interface{}
}

type ExecutionStatus string

const (
    StatusSuccess  ExecutionStatus = "success"
    StatusFailed   ExecutionStatus = "failed"
    StatusTimeout  ExecutionStatus = "timeout"
    StatusCanceled ExecutionStatus = "canceled"
    StatusSkipped  ExecutionStatus = "skipped"
)
```

### 4. ExecutionEngine Interface

The ExecutionEngine manages plugin execution with support for strategies, workflows, and concurrency control.

```go
type ExecutionEngine interface {
    // Plugin management
    RegisterPlugin(plugin Plugin) error
    UnregisterPlugin(name string) error
    GetPlugin(name string) (Plugin, error)
    ListPlugins(filter PluginFilter) ([]PluginMetadata, error)
    
    // Execution
    Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error)
    ExecuteAsync(ctx context.Context, request ExecutionRequest) (Future, error)
    
    // Workflow
    ExecuteWorkflow(ctx context.Context, workflow Workflow) (WorkflowResult, error)
    
    // Control
    Cancel(executionID string) error
    GetStatus(executionID string) (ExecutionStatus, error)
    
    // Configuration
    SetConcurrencyLimit(limit int)
    SetDefaultTimeout(timeout time.Duration)
}

type ExecutionRequest struct {
    PluginName string
    Resource   Resource
    Config     PluginConfig
    Strategy   ExecutionStrategy
    
    // Security context
    Principal string
}

type ExecutionStrategy interface {
    Execute(ctx context.Context, plugin Plugin, resource Resource) (ExecutionResult, error)
    Name() string
}

// Built-in strategies
type SimpleStrategy struct{}
type RetryStrategy struct {
    MaxRetries int
    Backoff    BackoffStrategy
}
type CircuitBreakerStrategy struct {
    FailureThreshold int
    Timeout          time.Duration
    ResetTimeout     time.Duration
}
type RateLimitStrategy struct {
    RequestsPerSecond int
}

type BackoffStrategy interface {
    NextDelay(attempt int) time.Duration
}

type Future interface {
    Wait() (ExecutionResult, error)
    Cancel() error
    Done() <-chan struct{}
}

type Workflow struct {
    Name  string
    Steps []WorkflowStep
}

type WorkflowStep struct {
    Name       string
    PluginName string
    Resource   Resource
    Config     PluginConfig
    
    // Dependencies
    DependsOn []string // Step names
    
    // Execution mode
    Parallel bool // Execute in parallel with other parallel steps
    
    // Conditional execution
    Condition *StepCondition
    
    // Error handling
    OnError ErrorHandler
}

type StepCondition struct {
    Type  ConditionType
    Value interface{}
}

type ConditionType string

const (
    ConditionAlways        ConditionType = "always"
    ConditionOnSuccess     ConditionType = "on_success"
    ConditionOnFailure     ConditionType = "on_failure"
    ConditionCustom        ConditionType = "custom"
)

type ErrorHandler interface {
    Handle(ctx context.Context, step WorkflowStep, err error) ErrorAction
}

type ErrorAction string

const (
    ErrorActionContinue ErrorAction = "continue"
    ErrorActionAbort    ErrorAction = "abort"
    ErrorActionRetry    ErrorAction = "retry"
    ErrorActionRollback ErrorAction = "rollback"
)

type WorkflowResult struct {
    Status      ExecutionStatus
    StepResults map[string]ExecutionResult
    StartTime   time.Time
    EndTime     time.Time
    Duration    time.Duration
}

type PluginFilter struct {
    Names        []string
    Versions     []string
    SupportedKinds []string
    Capabilities []Capability
}
```


### 5. Monitor Interface

The Monitor provides health checking and metrics collection with pluggable strategies.

```go
type Monitor interface {
    // Health checking
    CheckHealth(ctx context.Context, resource Resource) (HealthStatus, error)
    WaitForCondition(ctx context.Context, resource Resource, condition Condition, timeout time.Duration) error
    WaitForHealthy(ctx context.Context, resource Resource, timeout time.Duration) error
    
    // Metrics
    CollectMetrics(ctx context.Context, resource Resource) (Metrics, error)
    
    // Events
    WatchEvents(ctx context.Context, resource Resource) (<-chan Event, error)
    
    // Strategy management
    RegisterHealthCheckStrategy(name string, strategy HealthCheckStrategy) error
    GetHealthCheckStrategy(name string) (HealthCheckStrategy, error)
}

type HealthCheckStrategy interface {
    Check(ctx context.Context, resource Resource) (HealthStatus, error)
    Name() string
}

type HealthStatus struct {
    Status  HealthStatusType
    Message string
    Checks  []HealthCheck
}

type HealthStatusType string

const (
    HealthStatusHealthy   HealthStatusType = "healthy"
    HealthStatusUnhealthy HealthStatusType = "unhealthy"
    HealthStatusUnknown   HealthStatusType = "unknown"
)

type HealthCheck struct {
    Name    string
    Status  HealthStatusType
    Message string
}

// Built-in health check strategies
type HTTPHealthCheckStrategy struct {
    ExpectedStatusCodes []int
    ExpectedBody        string
}

type TCPHealthCheckStrategy struct {
    DialTimeout time.Duration
}

type ExecHealthCheckStrategy struct {
    ExpectedExitCode int
}

type Event struct {
    ID        string
    Type      string
    Source    string
    Timestamp time.Time
    Resource  Resource
    Data      map[string]interface{}
    Metadata  map[string]string
}
```

### 6. Reporter Interface

The Reporter records execution results and generates reports with pluggable storage and formatters.

```go
type Reporter interface {
    // Recording
    RecordEvent(ctx context.Context, event Event) error
    RecordExecution(ctx context.Context, exec ExecutionRecord) error
    
    // Querying
    QueryEvents(ctx context.Context, query EventQuery) ([]Event, error)
    QueryExecutions(ctx context.Context, query ExecutionQuery) ([]ExecutionRecord, error)
    
    // Statistics
    ComputeStatistics(ctx context.Context, filter StatisticsFilter) (Statistics, error)
    
    // Reporting
    GenerateReport(ctx context.Context, format string, filter ReportFilter) ([]byte, error)
    
    // Storage management
    SetStorage(storage StorageBackend) error
    
    // Formatter management
    RegisterFormatter(name string, formatter ReportFormatter) error
}

type ExecutionRecord struct {
    ID          string
    PluginName  string
    ResourceID  string
    ResourceName string
    StartTime   time.Time
    EndTime     time.Time
    Duration    time.Duration
    Status      ExecutionStatus
    Error       string
    Metadata    map[string]interface{}
    
    // Security
    Principal string
}

type EventQuery struct {
    Types     []string
    Sources   []string
    Resources []string
    StartTime time.Time
    EndTime   time.Time
    Limit     int
    Offset    int
}

type ExecutionQuery struct {
    PluginNames  []string
    ResourceIDs  []string
    Statuses     []ExecutionStatus
    StartTime    time.Time
    EndTime      time.Time
    Limit        int
    Offset       int
}

type StatisticsFilter struct {
    PluginNames []string
    ResourceIDs []string
    StartTime   time.Time
    EndTime     time.Time
}

type Statistics struct {
    TotalExecutions int
    SuccessCount    int
    FailureCount    int
    TimeoutCount    int
    CanceledCount   int
    SuccessRate     float64
    
    // Time to recovery
    MTTR time.Duration // Mean Time To Recovery
    
    // Duration statistics
    AverageDuration time.Duration
    P50Duration     time.Duration
    P95Duration     time.Duration
    P99Duration     time.Duration
    MinDuration     time.Duration
    MaxDuration     time.Duration
    
    // Breakdown
    ByPlugin   map[string]PluginStatistics
    ByResource map[string]ResourceStatistics
}

type PluginStatistics struct {
    PluginName      string
    TotalExecutions int
    SuccessCount    int
    FailureCount    int
    SuccessRate     float64
    AverageDuration time.Duration
}

type ResourceStatistics struct {
    ResourceID      string
    ResourceName    string
    TotalExecutions int
    SuccessCount    int
    FailureCount    int
    SuccessRate     float64
    AverageDuration time.Duration
}

type ReportFilter struct {
    PluginNames []string
    ResourceIDs []string
    StartTime   time.Time
    EndTime     time.Time
}

type StorageBackend interface {
    Save(ctx context.Context, key string, value interface{}) error
    Load(ctx context.Context, key string) (interface{}, error)
    Query(ctx context.Context, query Query) ([]interface{}, error)
    Delete(ctx context.Context, key string) error
    Close() error
}

type Query struct {
    Collection string
    Filter     map[string]interface{}
    Sort       []SortField
    Limit      int
    Offset     int
}

type SortField struct {
    Field      string
    Descending bool
}

type ReportFormatter interface {
    Format(ctx context.Context, data interface{}) ([]byte, error)
    ContentType() string
    Name() string
}

// Built-in formatters
type JSONFormatter struct{}
type MarkdownFormatter struct{}
type HTMLFormatter struct{}
```


### 7. EventBus Interface

The EventBus enables loose coupling between components through publish-subscribe messaging.

```go
type EventBus interface {
    // Publishing
    Publish(ctx context.Context, event Event) error
    PublishAsync(ctx context.Context, event Event) error
    
    // Subscribing
    Subscribe(ctx context.Context, filter EventFilter) (Subscription, error)
    
    // Lifecycle
    Close() error
}

type EventFilter struct {
    Types    []string
    Sources  []string
    Metadata map[string]string
}

type Subscription interface {
    ID() string
    Events() <-chan Event
    Unsubscribe() error
}
```

### 8. Config Interface

The Config system supports multiple configuration sources with precedence and hot-reloading.

```go
type Config interface {
    // Access
    Get(key string) (interface{}, error)
    GetString(key string) (string, error)
    GetInt(key string) (int, error)
    GetBool(key string) (bool, error)
    GetDuration(key string) (time.Duration, error)
    
    // Modification
    Set(key string, value interface{}) error
    
    // Watching
    Watch(key string) (<-chan interface{}, error)
    
    // Provider management
    AddProvider(provider ConfigProvider, priority int) error
}

type ConfigProvider interface {
    Load(ctx context.Context) (map[string]interface{}, error)
    Watch(ctx context.Context) (<-chan ConfigChange, error)
    Name() string
}

type ConfigChange struct {
    Key      string
    OldValue interface{}
    NewValue interface{}
}

// Built-in providers
type YAMLConfigProvider struct {
    FilePath string
}

type EnvironmentConfigProvider struct {
    Prefix string
}

type ConsulConfigProvider struct {
    Address string
    Prefix  string
}

type VaultConfigProvider struct {
    Address string
    Token   string
    Path    string
}
```

### 9. Security Interfaces

Security components provide authorization, audit logging, and secret management.

```go
type AuthorizationProvider interface {
    Authorize(ctx context.Context, principal string, action Action, resource Resource) error
}

type Action struct {
    Verb     string // get, list, create, update, delete, execute
    Resource string // Resource kind or specific resource ID
}

type AuditLogger interface {
    LogAction(ctx context.Context, entry AuditEntry) error
}

type AuditEntry struct {
    Timestamp time.Time
    Principal string
    Action    Action
    Resource  Resource
    Result    ActionResult
    Metadata  map[string]interface{}
}

type ActionResult struct {
    Success bool
    Error   string
    Duration time.Duration
}

type SecretProvider interface {
    GetSecret(ctx context.Context, key string) (string, error)
    SetSecret(ctx context.Context, key string, value string) error
    DeleteSecret(ctx context.Context, key string) error
}
```

### 10. Observability Interfaces

Observability components provide logging, metrics, and tracing.

```go
type ObservabilityProvider interface {
    Logger() Logger
    Metrics() MetricsCollector
    Tracer() Tracer
}

type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    With(fields ...Field) Logger
}

type Field struct {
    Key   string
    Value interface{}
}

type MetricsCollector interface {
    Counter(name string, tags map[string]string) Counter
    Gauge(name string, tags map[string]string) Gauge
    Histogram(name string, tags map[string]string) Histogram
}

type Counter interface {
    Inc()
    Add(delta float64)
}

type Gauge interface {
    Set(value float64)
    Inc()
    Dec()
    Add(delta float64)
}

type Histogram interface {
    Observe(value float64)
}

type Tracer interface {
    StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
}

type Span interface {
    SetTag(key string, value interface{})
    LogEvent(event string)
    LogFields(fields ...Field)
    Finish()
    Context() context.Context
}

type SpanOption func(*SpanConfig)

type SpanConfig struct {
    Tags   map[string]interface{}
    Parent Span
}
```

### 11. Plugin Registry Interface

The Plugin Registry manages plugin discovery, validation, and lifecycle.

```go
type PluginRegistry interface {
    // Registration
    Register(plugin Plugin) error
    Unregister(name string) error
    
    // Discovery
    Discover(path string) error
    DiscoverFromDirectory(dir string) error
    
    // Retrieval
    Get(name string) (Plugin, error)
    List(filter PluginFilter) ([]PluginMetadata, error)
    
    // Validation
    Validate(plugin Plugin) error
    ValidateDependencies(plugin Plugin) error
    
    // Hot-reload
    Reload(name string) error
    
    // Loader management
    SetLoader(loader PluginLoader) error
}

type PluginLoader interface {
    Load(path string) (Plugin, error)
    Unload(plugin Plugin) error
}

type PluginValidator interface {
    Validate(plugin Plugin) error
}
```

## Data Models

### Resource Lifecycle States

```
┌─────────┐
│ Pending │
└────┬────┘
     │
     ▼
┌─────────┐     ┌─────────┐
│ Running │────▶│ Stopped │
└────┬────┘     └────┬────┘
     │               │
     │               │
     ▼               ▼
┌─────────┐     ┌─────────┐
│ Failed  │     │ Deleted │
└─────────┘     └─────────┘
```

### Plugin Execution Flow

```
1. Validate
   ├─ Success → Continue
   └─ Failure → Return Error

2. PreExecute
   ├─ Success → Create Snapshot → Continue
   └─ Failure → Cleanup → Return Error

3. Execute
   ├─ Success → Continue
   └─ Failure → Rollback (if supported) → Cleanup → Return Error

4. PostExecute
   ├─ Success → Continue
   └─ Failure → Log Error → Continue

5. Cleanup
   └─ Always execute (even on failure)

6. Return Result
```

### Workflow Execution Flow

```
1. Parse workflow definition
2. Build dependency graph
3. Detect cycles (fail if found)
4. Topological sort
5. Execute steps:
   - Sequential: one at a time
   - Parallel: concurrent execution
   - Wait for dependencies
6. Handle errors per step configuration
7. Return workflow result
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Core Immutability Properties

**Property 1: Core remains unchanged when adding new environments**
*For any* new EnvironmentAdapter implementation, registering and using the adapter should not require any modifications to Core interfaces or implementations.
**Validates: Requirements 1.1**

**Property 2: Core remains unchanged when adding new plugins**
*For any* new Plugin implementation, registering and executing the plugin should not require any modifications to Core interfaces or implementations.
**Validates: Requirements 1.2**

**Property 3: Core remains unchanged when adding new storage backends**
*For any* new StorageBackend implementation, using the storage backend should not require any modifications to Core interfaces or implementations.
**Validates: Requirements 1.3**

**Property 4: Core remains unchanged when adding new monitoring strategies**
*For any* new HealthCheckStrategy implementation, using the strategy should not require any modifications to Core interfaces or implementations.
**Validates: Requirements 1.4**

**Property 5: Core contains no environment-specific implementations**
*For all* types and functions in the Core package, none should contain concrete references to specific environments (docker, kubernetes, ecs, etc.).
**Validates: Requirements 1.5**

### Resource Model Properties

**Property 6: Resource contains environment-agnostic fields**
*For any* Resource instance, it must contain ID, Name, Kind, Labels, Status, and Spec fields that are not tied to any specific environment.
**Validates: Requirements 2.1**

**Property 7: Adapter conversion produces valid Resources**
*For any* EnvironmentAdapter and any environment-specific resource, calling ListResources or GetResource must return Resource objects that conform to the unified model.
**Validates: Requirements 2.2**

**Property 8: Resource supports arbitrary metadata**
*For any* Resource instance and any key-value pair, setting and retrieving the pair through the Metadata map should preserve the data.
**Validates: Requirements 2.3**

**Property 9: Resource status is normalized across environments**
*For any* two EnvironmentAdapters and any resource in each environment, the ResourceStatus structure should follow the same format (Phase, Conditions, Message, Reason).
**Validates: Requirements 2.4**

**Property 10: Resource model contains no environment-specific enums**
*For all* fields in the Resource struct, none should be enums that require Core modification when adding new environments.
**Validates: Requirements 2.5**

### Plugin System Properties

**Property 11: Plugin metadata is complete**
*For any* Plugin registration, the PluginMetadata must contain Name, Version, Description, SupportedKinds, RequiredCapabilities, and Dependencies.
**Validates: Requirements 3.1**

**Property 12: Plugin lifecycle hooks execute in order**
*For any* plugin execution, the ExecutionEngine must invoke hooks in the order: Validate → PreExecute → Execute → PostExecute → Cleanup.
**Validates: Requirements 3.2**

**Property 13: Rollback is invoked on failure**
*For any* plugin execution that fails and implements Rollback, the ExecutionEngine must invoke Rollback with the snapshot from PreExecute.
**Validates: Requirements 3.3**

**Property 14: Progress is reported through channel**
*For any* plugin execution, progress updates sent by the plugin must be delivered through the PluginContext.Progress channel.
**Validates: Requirements 3.4**

**Property 15: Events are published through EventBus**
*For any* plugin that emits events, the events must be published through the PluginContext.EventBus.
**Validates: Requirements 3.5**

**Property 16: Plugin supports sync and async execution**
*For any* plugin, both Execute (synchronous) and ExecuteAsync (asynchronous) methods must work correctly.
**Validates: Requirements 3.6**

### EnvironmentAdapter Properties

**Property 17: Adapter provides all required methods**
*For any* EnvironmentAdapter implementation, it must provide working implementations of ListResources, GetResource, StartResource, StopResource, RestartResource, DeleteResource, WatchResources, ExecInResource, and GetMetrics.
**Validates: Requirements 4.1**

**Property 18: Watch emits ResourceEvent objects**
*For any* EnvironmentAdapter, calling WatchResources must return a channel that emits ResourceEvent objects when resources change.
**Validates: Requirements 4.2**

**Property 19: Exec returns structured result**
*For any* EnvironmentAdapter and any command execution, ExecInResource must return an ExecResult with Stdout, Stderr, and ExitCode fields.
**Validates: Requirements 4.3**

**Property 20: Metrics are normalized**
*For any* EnvironmentAdapter, GetMetrics must return Metrics with CPU, Memory, Network, and Disk fields in the normalized format.
**Validates: Requirements 4.4**

**Property 21: Adapter interface is sufficient for all platforms**
*For any* orchestration platform (docker-compose, K8s, ECS, Nomad), it must be possible to implement a fully functional EnvironmentAdapter without Core modification.
**Validates: Requirements 4.5**

### ExecutionEngine Properties

**Property 22: Execution strategy is applied**
*For any* ExecutionRequest with a specified ExecutionStrategy, the ExecutionEngine must apply that strategy during execution.
**Validates: Requirements 5.1**

**Property 23: Concurrency limits are respected**
*For any* set of concurrent plugin executions, the ExecutionEngine must not exceed the configured concurrency limit.
**Validates: Requirements 5.2**

**Property 24: Timeout cancels execution**
*For any* plugin execution that exceeds the timeout, the ExecutionEngine must cancel the context and invoke cleanup hooks.
**Validates: Requirements 5.3**

**Property 25: Workflow respects dependencies**
*For any* Workflow with step dependencies, the ExecutionEngine must execute steps only after all dependencies complete.
**Validates: Requirements 5.4**

**Property 26: Workflow error handler is invoked**
*For any* WorkflowStep that fails, the ExecutionEngine must invoke the configured ErrorHandler.
**Validates: Requirements 5.5**

**Property 27: Custom execution strategies are supported**
*For any* custom ExecutionStrategy implementation, it must be usable without Core modification.
**Validates: Requirements 5.6**

### Monitor Properties

**Property 28: Health check uses registered strategy**
*For any* Resource with a HealthCheckConfig, the Monitor must use the registered HealthCheckStrategy matching the config type.
**Validates: Requirements 6.1**

**Property 29: Condition waiting polls until met or timeout**
*For any* Resource and Condition, WaitForCondition must poll the resource state until the condition is met or timeout occurs.
**Validates: Requirements 6.2**

**Property 30: Metrics collection delegates to adapter**
*For any* Resource, CollectMetrics must delegate to the EnvironmentAdapter and return normalized Metrics.
**Validates: Requirements 6.3**

**Property 31: Event watching provides channel**
*For any* Resource, WatchEvents must return a channel that delivers Event objects.
**Validates: Requirements 6.4**

**Property 32: Custom health check strategies are supported**
*For any* custom HealthCheckStrategy implementation, it must be usable without Core modification.
**Validates: Requirements 6.5**

### Reporter Properties

**Property 33: Execution record is complete**
*For any* completed execution, the ExecutionRecord must contain PluginName, ResourceID, StartTime, EndTime, Duration, Status, and Error (if failed).
**Validates: Requirements 7.1**

**Property 34: Event record is complete**
*For any* recorded event, the Event must contain Type, Source, Timestamp, and Data.
**Validates: Requirements 7.2**

**Property 35: Statistics are calculated correctly**
*For any* set of ExecutionRecords, ComputeStatistics must calculate TotalExecutions, SuccessCount, FailureCount, SuccessRate, MTTR, and duration percentiles (P50, P95, P99).
**Validates: Requirements 7.3**

**Property 36: Report uses registered formatter**
*For any* report generation request, the Reporter must use the ReportFormatter registered for the requested format.
**Validates: Requirements 7.4**

**Property 37: Custom storage backends are supported**
*For any* custom StorageBackend implementation, it must be usable without Core modification.
**Validates: Requirements 7.5**

### EventBus Properties

**Property 38: Events are delivered to all matching subscribers**
*For any* published Event, all Subscriptions with matching EventFilters must receive the event asynchronously.
**Validates: Requirements 8.1**

**Property 39: Subscription receives only matching events**
*For any* Subscription with an EventFilter, only events matching the filter criteria must be delivered.
**Validates: Requirements 8.2**

**Property 40: Unsubscribe stops event delivery**
*For any* Subscription, calling Unsubscribe must stop event delivery to that subscription.
**Validates: Requirements 8.3**

**Property 41: Shutdown closes all channels gracefully**
*For any* EventBus, calling Close must close all subscription channels without panics.
**Validates: Requirements 8.4**

**Property 42: Event filtering supports type, source, and metadata**
*For any* EventFilter with Type, Source, or Metadata criteria, only events matching all criteria must pass through.
**Validates: Requirements 8.5**

### Config Properties

**Property 43: Multiple config providers are supported**
*For any* set of ConfigProvider implementations (YAML, Environment, Consul, Vault), all must be usable simultaneously.
**Validates: Requirements 9.1**

**Property 44: Config merge follows precedence**
*For any* set of config sources with defined priorities, Get must return the value from the highest priority source.
**Validates: Requirements 9.2**

**Property 45: Config values are type-converted**
*For any* config key and requested type, Get{Type} methods must return the value converted to the requested type or an error.
**Validates: Requirements 9.3**

**Property 46: Config changes notify watchers**
*For any* config key being watched, changes to that key must be delivered through the watch channel.
**Validates: Requirements 9.4**

**Property 47: Custom config providers are supported**
*For any* custom ConfigProvider implementation, it must be usable without Core modification.
**Validates: Requirements 9.5**

### Security Properties

**Property 48: Authorization is checked before execution**
*For any* plugin execution, the ExecutionEngine must call AuthorizationProvider.Authorize before invoking the plugin.
**Validates: Requirements 10.1**

**Property 49: Audit log is complete**
*For any* action performed, the AuditEntry must contain Principal, Action, Resource, Timestamp, and Result.
**Validates: Requirements 10.2**

**Property 50: Secrets are not exposed in logs**
*For any* secret retrieved through SecretProvider, the secret value must not appear in any log output.
**Validates: Requirements 10.3**

**Property 51: Authorization failure prevents execution**
*For any* plugin execution where authorization fails, the ExecutionEngine must return an error and must not execute the plugin.
**Validates: Requirements 10.4**

**Property 52: Custom security providers are supported**
*For any* custom AuthorizationProvider, AuditLogger, or SecretProvider implementation, it must be usable without Core modification.
**Validates: Requirements 10.5**

### Observability Properties

**Property 53: Logs contain structured fields**
*For any* log entry, it must include Timestamp, Level, Component, ExecutionID, and ResourceID fields.
**Validates: Requirements 11.1**

**Property 54: Metrics include type and tags**
*For any* metric emission, it must include the correct type (Counter, Gauge, Histogram) and appropriate tags.
**Validates: Requirements 11.2**

**Property 55: Trace span is created and propagated**
*For any* execution start, a trace span must be created and propagated through the execution context.
**Validates: Requirements 11.3**

**Property 56: Trace span is finished on completion**
*For any* execution completion, the trace span must be finished with appropriate tags and events.
**Validates: Requirements 11.4**

**Property 57: Custom observability providers are supported**
*For any* custom ObservabilityProvider implementation, it must be usable without Core modification.
**Validates: Requirements 11.5**

### Testing Properties

**Property 58: Mock implementations exist for all interfaces**
*For all* Core interfaces, a mock implementation must exist in the testing package.
**Validates: Requirements 12.5**

### Plugin Registry Properties

**Property 59: Plugins are discovered from directories**
*For any* directory containing valid plugins, calling Discover must find and register all plugins.
**Validates: Requirements 13.1**

**Property 60: Plugin validation occurs on registration**
*For any* plugin registration, the PluginRegistry must validate metadata and interface compliance.
**Validates: Requirements 13.2**

**Property 61: Plugin retrieval returns plugin or error**
*For any* plugin name, Get must return the Plugin instance if registered, or an error if not found.
**Validates: Requirements 13.3**

**Property 62: Plugin listing supports filtering**
*For any* PluginFilter criteria, List must return only plugins matching all criteria.
**Validates: Requirements 13.4**

**Property 63: Hot-reload updates plugins without restart**
*For any* plugin, calling Reload must update the plugin without restarting the application.
**Validates: Requirements 13.5**

### Error Handling Properties

**Property 64: Retryable errors trigger retry with backoff**
*For any* retryable error, the ExecutionEngine must retry the operation with exponential backoff up to MaxRetries.
**Validates: Requirements 14.1**

**Property 65: Fatal errors abort immediately**
*For any* fatal error, the ExecutionEngine must immediately abort and invoke cleanup hooks.
**Validates: Requirements 14.2**

**Property 66: Failure triggers rollback if supported**
*For any* plugin execution that fails and implements Rollback, the ExecutionEngine must attempt rollback using the snapshot.
**Validates: Requirements 14.3**

**Property 67: Errors are wrapped with context**
*For any* error returned, it must be wrapped using fmt.Errorf with %w to preserve the error chain.
**Validates: Requirements 14.4**

**Property 68: Errors are correctly classified**
*For any* error, the system must correctly classify it as retryable or fatal based on error type.
**Validates: Requirements 14.5**

### Concurrency Properties

**Property 69: Shared state uses synchronization**
*For any* component with shared state accessed by multiple goroutines, appropriate synchronization primitives (Mutex, RWMutex, channels) must be used.
**Validates: Requirements 15.1**

**Property 70: Goroutines cleanup properly**
*For any* spawned goroutine, it must terminate when the context is cancelled, verified using WaitGroups.
**Validates: Requirements 15.2**

**Property 71: Channel creators close channels**
*For any* channel created by a component, that component must be responsible for closing the channel.
**Validates: Requirements 15.3**

**Property 72: Graceful shutdown completes within timeout**
*For any* system shutdown, all goroutines must terminate within the shutdown timeout.
**Validates: Requirements 15.4**

**Property 73: Resource locking prevents concurrent execution**
*For any* resource, the ExecutionEngine must prevent concurrent plugin execution on the same resource.
**Validates: Requirements 15.5**

### Workflow Properties

**Property 74: Workflow supports sequential and parallel execution**
*For any* Workflow, steps marked as sequential must execute one at a time, and steps marked as parallel must execute concurrently.
**Validates: Requirements 16.1**

**Property 75: Dependencies are waited for**
*For any* WorkflowStep with dependencies, execution must wait for all dependency steps to complete.
**Validates: Requirements 16.2**

**Property 76: Step error handler is executed**
*For any* WorkflowStep that fails, the configured ErrorHandler must be executed.
**Validates: Requirements 16.3**

**Property 77: Workflow result contains all step results**
*For any* completed Workflow, the WorkflowResult must contain individual results for all executed steps.
**Validates: Requirements 16.4**

**Property 78: Conditional execution works correctly**
*For any* WorkflowStep with a Condition, the step must execute only if the condition evaluates to true.
**Validates: Requirements 16.5**

### Snapshot and Rollback Properties

**Property 79: PreExecute creates valid snapshot**
*For any* plugin PreExecute call, a valid Snapshot must be created containing sufficient state information.
**Validates: Requirements 17.1**

**Property 80: Rollback restores state (Round-trip)**
*For any* resource, creating a snapshot, modifying the resource, and then rolling back must restore the resource to its original state.
**Validates: Requirements 17.2**

**Property 81: Snapshot is stored in context**
*For any* created Snapshot, it must be accessible from the ExecutionContext during rollback.
**Validates: Requirements 17.3**

**Property 82: Rollback failure doesn't prevent cleanup**
*For any* rollback that fails, cleanup hooks must still execute.
**Validates: Requirements 17.4**

**Property 83: Snapshot serialization round-trip**
*For any* Snapshot, serializing and then deserializing must preserve all snapshot data.
**Validates: Requirements 17.5**

### Plugin Dependency Properties

**Property 84: Dependencies are declared in metadata**
*For any* plugin with dependencies, the dependencies must be declared in PluginMetadata.Dependencies.
**Validates: Requirements 18.1**

**Property 85: Dependencies are verified on load**
*For any* plugin being loaded, the PluginRegistry must verify all dependencies are satisfied.
**Validates: Requirements 18.2**

**Property 86: Missing dependencies prevent registration**
*For any* plugin with unsatisfied dependencies, the PluginRegistry must return an error and not register the plugin.
**Validates: Requirements 18.3**

**Property 87: Circular dependencies are detected**
*For any* set of plugins with circular dependencies, the PluginRegistry must detect the cycle and return an error.
**Validates: Requirements 18.4**

**Property 88: Version constraints are validated**
*For any* plugin dependency with version constraints, the PluginRegistry must validate that the constraint is satisfied.
**Validates: Requirements 18.5**

### Resource Filtering Properties

**Property 89: Filters support multiple criteria**
*For any* ResourceFilter with labels, kinds, statuses, or names, only resources matching all criteria must be returned.
**Validates: Requirements 19.1**

**Property 90: Filters support logical operators**
*For any* ResourceFilter with AND, OR, or NOT operators, the filter must correctly apply the logical operations.
**Validates: Requirements 19.2**

**Property 91: Label selectors support all operators**
*For any* LabelSelector with equality, inequality, set membership, or existence operators, the selector must work correctly.
**Validates: Requirements 19.3**

**Property 92: Empty results return empty list**
*For any* ResourceFilter that matches no resources, the operation must return an empty list without error.
**Validates: Requirements 19.4**

**Property 93: Filter serialization round-trip**
*For any* ResourceFilter, serializing and then deserializing must preserve all filter criteria.
**Validates: Requirements 19.5**

### Performance and Scalability Properties

**Property 94: Pagination prevents memory exhaustion**
*For any* large result set, ListResources with pagination must not load all resources into memory at once.
**Validates: Requirements 20.1**

**Property 95: Worker pool limits resource consumption**
*For any* large number of concurrent executions, the ExecutionEngine must use a worker pool to bound resource usage.
**Validates: Requirements 20.2**

**Property 96: Writes are batched**
*For any* sequence of execution records, the Reporter must batch writes to reduce I/O overhead.
**Validates: Requirements 20.3**

**Property 97: Watch uses event streaming**
*For any* EnvironmentAdapter watch implementation, it must use efficient event streaming (informers, websockets) instead of polling.
**Validates: Requirements 20.4**

**Property 98: Distributed coordination works correctly**
*For any* multiple framework instances, coordination through distributed locks must prevent conflicts.
**Validates: Requirements 20.5**


## Error Handling

### Error Classification

Errors are classified into three categories:

1. **Retryable Errors**: Temporary failures that may succeed on retry
   - Network timeouts
   - Temporary resource unavailability
   - Rate limit exceeded
   - Connection refused (transient)

2. **Fatal Errors**: Permanent failures that will not succeed on retry
   - Authorization denied
   - Resource not found
   - Invalid configuration
   - Unsupported operation

3. **Ignorable Errors**: Expected conditions that are not failures
   - Resource already in desired state
   - Idempotent operation already applied

### Error Wrapping

All errors must be wrapped with context using `fmt.Errorf` with `%w`:

```go
if err != nil {
    return fmt.Errorf("failed to execute plugin %s on resource %s: %w", 
        pluginName, resourceID, err)
}
```

This preserves the error chain for inspection using `errors.Is` and `errors.As`.

### Retry Strategy

Retryable errors trigger exponential backoff retry:

```
Attempt 1: immediate
Attempt 2: 1s delay
Attempt 3: 2s delay
Attempt 4: 4s delay
Attempt 5: 8s delay
...
Max delay: 60s
```

### Rollback on Failure

When a plugin execution fails:

1. If the plugin implements `Rollback`, invoke it with the snapshot from `PreExecute`
2. If rollback fails, log the error but continue to cleanup
3. Always invoke `Cleanup` hooks regardless of rollback success

## Testing Strategy

### Unit Testing

Each Core component must have unit tests with:
- Mock implementations of all dependencies
- Test coverage ≥ 80%
- Table-driven tests for multiple scenarios
- Edge case coverage

Example:
```go
func TestExecutionEngine_Execute(t *testing.T) {
    tests := []struct {
        name    string
        plugin  Plugin
        resource Resource
        wantErr bool
    }{
        {"valid execution", validPlugin, validResource, false},
        {"invalid resource", validPlugin, invalidResource, true},
        {"plugin validation fails", invalidPlugin, validResource, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := NewExecutionEngine()
            err := engine.Execute(context.Background(), ExecutionRequest{
                PluginName: tt.plugin.Metadata().Name,
                Resource:   tt.resource,
            })
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Property-Based Testing

Property-based testing will be used to verify correctness properties using **gopter** (Go property testing library).

Each correctness property must have a corresponding property-based test:

```go
func TestProperty_CoreImmutability(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("adding new adapter doesn't modify Core", prop.ForAll(
        func(adapterName string) bool {
            // Create a new test adapter
            adapter := &TestAdapter{name: adapterName}
            
            // Get Core package hash before
            hashBefore := getCorePackageHash()
            
            // Register adapter
            engine.RegisterAdapter(adapterName, adapter)
            
            // Get Core package hash after
            hashAfter := getCorePackageHash()
            
            // Core should be unchanged
            return hashBefore == hashAfter
        },
        gen.Identifier(),
    ))
    
    properties.TestingRun(t)
}
```

### Integration Testing

Integration tests verify component interactions:

- Use real EnvironmentAdapter implementations with test containers (testcontainers-go)
- Test complete execution flows (plugin registration → execution → reporting)
- Verify event propagation through EventBus
- Test workflow execution with dependencies

### End-to-End Testing

E2E tests verify complete scenarios:

1. **Chaos + Healing Cycle**:
   - Start test environment (docker-compose with redis)
   - Execute Kill plugin on redis
   - Verify NecroOps detects failure
   - Verify NecroOps restarts redis
   - Verify redis is healthy
   - Verify MTTR is calculated

2. **Multi-Environment Test**:
   - Execute same plugin on docker-compose and K8s
   - Verify both succeed
   - Verify results are consistent

### Test Utilities

The `testing` package provides:

```go
// Mock implementations
type MockEnvironmentAdapter struct {
    mock.Mock
}

type MockPlugin struct {
    mock.Mock
}

// Test harness
type TestHarness struct {
    Engine    ExecutionEngine
    Monitor   Monitor
    Reporter  Reporter
    EventBus  EventBus
    Config    Config
}

func NewTestHarness() *TestHarness {
    // Returns fully configured test environment
}

// Builders
type ResourceBuilder struct {
    resource Resource
}

func NewResourceBuilder() *ResourceBuilder {
    return &ResourceBuilder{
        resource: Resource{
            Labels:      make(map[string]string),
            Annotations: make(map[string]string),
            Metadata:    make(map[string]interface{}),
        },
    }
}

func (b *ResourceBuilder) WithName(name string) *ResourceBuilder {
    b.resource.Name = name
    return b
}

func (b *ResourceBuilder) WithKind(kind string) *ResourceBuilder {
    b.resource.Kind = kind
    return b
}

func (b *ResourceBuilder) WithLabel(key, value string) *ResourceBuilder {
    b.resource.Labels[key] = value
    return b
}

func (b *ResourceBuilder) Build() Resource {
    return b.resource
}

// Workflow test runner
type WorkflowTestRunner struct {
    engine ExecutionEngine
}

func (r *WorkflowTestRunner) Run(workflow Workflow) WorkflowResult {
    // Execute workflow and return result
}

func (r *WorkflowTestRunner) AssertStepExecuted(stepName string) {
    // Assert step was executed
}

func (r *WorkflowTestRunner) AssertStepSkipped(stepName string) {
    // Assert step was skipped
}
```

## Implementation Phases

### Phase 1: Core Interfaces and Models (Week 1)

**Goal**: Define all Core interfaces and data models

**Tasks**:
1. Define Resource model and related types
2. Define all Core interfaces (EnvironmentAdapter, Plugin, ExecutionEngine, Monitor, Reporter, EventBus, Config)
3. Define security interfaces (AuthorizationProvider, AuditLogger, SecretProvider)
4. Define observability interfaces (Logger, MetricsCollector, Tracer)
5. Create testing package with mock implementations
6. Write unit tests for data models

**Deliverables**:
- `pkg/core/types/` - All data models
- `pkg/core/interfaces/` - All interface definitions
- `pkg/core/testing/` - Mock implementations and test utilities

### Phase 2: Default Implementations (Week 2)

**Goal**: Implement default Core components

**Tasks**:
1. Implement DefaultExecutionEngine
2. Implement DefaultMonitor
3. Implement DefaultReporter with InMemoryStorage
4. Implement InMemoryEventBus
5. Implement DefaultConfig with YAMLConfigProvider
6. Implement DefaultPluginRegistry
7. Write unit tests for all implementations

**Deliverables**:
- `pkg/core/engine/` - ExecutionEngine implementation
- `pkg/core/monitor/` - Monitor implementation
- `pkg/core/reporter/` - Reporter implementation
- `pkg/core/eventbus/` - EventBus implementation
- `pkg/core/config/` - Config implementation
- `pkg/core/registry/` - PluginRegistry implementation

### Phase 3: Adapters (Week 3)

**Goal**: Implement environment adapters

**Tasks**:
1. Implement ComposeAdapter (docker-compose)
2. Implement K8sAdapter (Kubernetes)
3. Write integration tests for each adapter
4. Verify adapters work without Core modification

**Deliverables**:
- `pkg/adapters/compose/` - docker-compose adapter
- `pkg/adapters/k8s/` - Kubernetes adapter

### Phase 4: Example Plugins (Week 4)

**Goal**: Implement example plugins to validate plugin system

**Tasks**:
1. Implement KillPlugin (Chaos)
2. Implement RestartPlugin (Healing)
3. Implement DelayPlugin (Chaos)
4. Implement HealthMonitorPlugin (Healing)
5. Write unit and integration tests for plugins
6. Verify plugins work without Core modification

**Deliverables**:
- `pkg/plugins/kill/` - Kill plugin
- `pkg/plugins/restart/` - Restart plugin
- `pkg/plugins/delay/` - Network delay plugin
- `pkg/plugins/healthmonitor/` - Health monitor plugin

### Phase 5: Applications (Week 5)

**Goal**: Build example applications using Core

**Tasks**:
1. Build GrimOps (Chaos application)
2. Build NecroOps (Healing application)
3. Create CLI for both applications
4. Write E2E tests
5. Create demo scenarios

**Deliverables**:
- `cmd/grimops/` - GrimOps application
- `cmd/necroops/` - NecroOps application
- Demo docker-compose environment
- Demo scenarios

### Phase 6: Documentation and Polish (Week 6)

**Goal**: Complete documentation and polish

**Tasks**:
1. Write comprehensive README
2. Write plugin development guide
3. Write adapter development guide
4. Create API documentation
5. Add examples and tutorials
6. Performance testing and optimization

**Deliverables**:
- README.md
- docs/plugin-development.md
- docs/adapter-development.md
- docs/api-reference.md
- examples/

## Success Criteria

### Functional Requirements

- [ ] All 20 requirements are implemented
- [ ] All 98 correctness properties pass property-based tests
- [ ] GrimOps and NecroOps work without Core modification
- [ ] New environment (e.g., ECS adapter) can be added without Core modification
- [ ] New plugin can be added without Core modification

### Quality Requirements

- [ ] Unit test coverage ≥ 80%
- [ ] All property-based tests pass with 100+ iterations
- [ ] Integration tests pass for docker-compose and K8s
- [ ] E2E tests pass for complete scenarios
- [ ] No race conditions detected by `go test -race`
- [ ] All code passes `golangci-lint`

### Documentation Requirements

- [ ] All public interfaces have godoc comments
- [ ] README with quick start guide
- [ ] Plugin development guide
- [ ] Adapter development guide
- [ ] API reference documentation
- [ ] At least 3 example applications

### Performance Requirements

- [ ] Plugin execution overhead < 10ms
- [ ] Event delivery latency < 1ms
- [ ] Support 1000+ concurrent plugin executions
- [ ] Support 10,000+ resources per adapter
- [ ] Memory usage < 100MB for idle framework

## Future Enhancements

### Phase 7+: Advanced Features

- **Web UI**: Dashboard for monitoring and control
- **Distributed Execution**: Multi-node coordination
- **Advanced Workflows**: Conditional branching, loops, parallel fan-out
- **Plugin Marketplace**: Registry for sharing plugins
- **Cloud Adapters**: AWS ECS, Azure Container Instances, Google Cloud Run
- **Advanced Monitoring**: Prometheus metrics, Jaeger tracing
- **Policy Engine**: Define and enforce resilience policies
- **Chaos Scenarios**: Pre-built chaos engineering scenarios
- **AI-Powered Healing**: ML-based failure prediction and auto-healing

