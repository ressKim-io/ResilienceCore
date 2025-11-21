package types

import (
	"context"
	"io"
	"time"
)

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

// ResourceStatus represents the current state of a resource
type ResourceStatus struct {
	Phase      string      // running, stopped, pending, failed, unknown (environment-defined)
	Conditions []Condition // Detailed conditions
	Message    string      // Human-readable status message
	Reason     string      // Machine-readable reason code
}

// Condition represents a detailed condition of a resource
type Condition struct {
	Type               string // Ready, Healthy, Initialized, etc.
	Status             string // True, False, Unknown
	Reason             string // CamelCase reason
	Message            string // Human-readable message
	LastTransitionTime time.Time
}

// ResourceSpec defines the desired state of a resource
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

// Port represents a network port configuration
type Port struct {
	Name          string
	ContainerPort int32
	HostPort      int32
	Protocol      string // TCP, UDP
}

// Volume represents a volume mount
type Volume struct {
	Name      string
	MountPath string
	ReadOnly  bool
	Source    VolumeSource
}

// VolumeSource defines the source of a volume
type VolumeSource struct {
	HostPath  *string // Host path
	ConfigMap *string // ConfigMap name
	Secret    *string // Secret name
	EmptyDir  bool    // Temporary directory
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	Type     string        // http, tcp, exec, custom
	Endpoint string        // URL for HTTP, address for TCP, command for exec
	Interval time.Duration // Check interval
	Timeout  time.Duration // Check timeout
	Retries  int           // Consecutive failures before unhealthy
}

// Metrics represents resource metrics
type Metrics struct {
	Timestamp  time.Time
	ResourceID string
	CPU        CPUMetrics
	Memory     MemoryMetrics
	Network    NetworkMetrics
	Disk       DiskMetrics
}

// CPUMetrics represents CPU usage metrics
type CPUMetrics struct {
	UsagePercent float64
	UsageCores   float64
}

// MemoryMetrics represents memory usage metrics
type MemoryMetrics struct {
	UsageBytes   uint64
	LimitBytes   uint64
	UsagePercent float64
}

// NetworkMetrics represents network usage metrics
type NetworkMetrics struct {
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
}

// DiskMetrics represents disk usage metrics
type DiskMetrics struct {
	ReadBytes  uint64
	WriteBytes uint64
	ReadOps    uint64
	WriteOps   uint64
}

// Event represents a system event
type Event struct {
	ID        string
	Type      string
	Source    string
	Timestamp time.Time
	Resource  Resource
	Data      map[string]interface{}
	Metadata  map[string]string
}

// EventType represents the type of resource event
type EventType string

// Event type constants
const (
	EventAdded    EventType = "Added"
	EventModified EventType = "Modified"
	EventDeleted  EventType = "Deleted"
)

// ResourceEvent represents a resource change event
type ResourceEvent struct {
	Type     EventType // Added, Modified, Deleted
	Resource Resource
}

// DeleteOptions defines options for resource deletion
type DeleteOptions struct {
	GracePeriod time.Duration
	Force       bool
}

// ExecOptions defines options for command execution
type ExecOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	TTY    bool
}

// ExecResult represents the result of command execution
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// EnvironmentAdapter abstracts all environment-specific operations
// Each infrastructure platform (docker-compose, K8s, ECS, etc.) implements this interface
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

// AdapterConfig contains environment-specific configuration
type AdapterConfig struct {
	// Environment-specific configuration
	// For Compose: docker socket path, compose file path
	// For K8s: kubeconfig path, namespace
	// For ECS: region, cluster name
	Config map[string]interface{}
}

// ResourceFilter defines criteria for filtering resources
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

// LabelSelector defines label-based selection criteria
type LabelSelector struct {
	MatchLabels      map[string]string     // Equality-based (AND)
	MatchExpressions []SelectorRequirement // Set-based
}

// SelectorRequirement defines a single selector requirement
type SelectorRequirement struct {
	Key      string
	Operator string // In, NotIn, Exists, DoesNotExist, Equals, NotEquals
	Values   []string
}

// AdapterInfo provides metadata about an adapter
type AdapterInfo struct {
	Name         string
	Version      string
	Environment  string   // compose, kubernetes, ecs, nomad, etc.
	Capabilities []string // List of supported features
}

// ============================================================================
// Plugin System Types
// ============================================================================

// Plugin defines the contract for all feature implementations
// Plugins are completely independent and can be developed without Core modification
type Plugin interface {
	// Metadata returns plugin metadata
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

// PluginMetadata contains plugin metadata
type PluginMetadata struct {
	Name        string
	Version     string
	Description string
	Author      string

	// Compatibility
	SupportedKinds       []string     // Resource kinds this plugin supports (empty = all)
	RequiredCapabilities []Capability // Required adapter capabilities

	// Dependencies
	Dependencies []PluginDependency

	// Configuration schema
	ConfigSchema interface{} // JSON schema for plugin configuration
}

// Capability represents a required capability
type Capability string

// Capability constants
const (
	CapabilityExec    Capability = "exec"    // Requires exec in resources
	CapabilityNetwork Capability = "network" // Requires network manipulation
	CapabilityAdmin   Capability = "admin"   // Requires admin privileges
)

// PluginDependency represents a plugin dependency
type PluginDependency struct {
	PluginName string
	Version    string // Semantic version constraint (e.g., ">=1.0.0", "^2.0.0")
}

// PluginConfig is the interface for plugin configuration
type PluginConfig interface {
	Validate() error
}

// PluginContext provides context for plugin execution
type PluginContext struct {
	context.Context

	// Execution metadata
	ExecutionID string
	Timeout     time.Duration

	// Core components (will be defined in later tasks)
	Env      EnvironmentAdapter
	Monitor  interface{} // Monitor interface - to be defined
	Reporter interface{} // Reporter interface - to be defined
	EventBus interface{} // EventBus interface - to be defined
	Logger   interface{} // Logger interface - to be defined
	Tracer   interface{} // Tracer interface - to be defined

	// Progress reporting
	Progress chan<- ProgressUpdate

	// Security
	Principal string
	Auth      interface{} // AuthorizationProvider - to be defined
	Secrets   interface{} // SecretProvider - to be defined
}

// ProgressUpdate represents a progress update from a plugin
type ProgressUpdate struct {
	Percent int
	Message string
	Stage   string
}

// Snapshot captures resource state for rollback
type Snapshot interface {
	// Restore restores the resource to the state captured in this snapshot
	Restore(ctx context.Context) error

	// Serialize converts the snapshot to bytes for storage
	Serialize() ([]byte, error)

	// Deserialize reconstructs the snapshot from bytes
	Deserialize(data []byte) error
}

// ExecutionResult represents the result of a plugin execution
type ExecutionResult struct {
	Status    ExecutionStatus
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Error     error
	Metadata  map[string]interface{}
}

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

// ExecutionStatus constants
const (
	StatusSuccess  ExecutionStatus = "success"
	StatusFailed   ExecutionStatus = "failed"
	StatusTimeout  ExecutionStatus = "timeout"
	StatusCanceled ExecutionStatus = "canceled"
	StatusSkipped  ExecutionStatus = "skipped"
)

// ============================================================================
// ExecutionEngine Types
// ============================================================================

// ExecutionEngine manages plugin execution with support for strategies, workflows, and concurrency control
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

// ExecutionRequest represents a request to execute a plugin
type ExecutionRequest struct {
	PluginName string
	Resource   Resource
	Config     PluginConfig
	Strategy   ExecutionStrategy

	// Security context
	Principal string
}

// ExecutionStrategy defines how a plugin should be executed
type ExecutionStrategy interface {
	Execute(ctx context.Context, plugin Plugin, resource Resource) (ExecutionResult, error)
	Name() string
}

// Future represents an asynchronous execution result
type Future interface {
	Wait() (ExecutionResult, error)
	Cancel() error
	Done() <-chan struct{}
}

// Workflow represents a multi-step execution workflow
type Workflow struct {
	Name  string
	Steps []WorkflowStep
}

// WorkflowStep represents a single step in a workflow
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

// StepCondition defines when a step should execute
type StepCondition struct {
	Type  ConditionType
	Value interface{}
}

// ConditionType represents the type of condition
type ConditionType string

// ConditionType constants
const (
	ConditionAlways    ConditionType = "always"
	ConditionOnSuccess ConditionType = "on_success"
	ConditionOnFailure ConditionType = "on_failure"
	ConditionCustom    ConditionType = "custom"
)

// ErrorHandler handles errors during workflow execution
type ErrorHandler interface {
	Handle(ctx context.Context, step WorkflowStep, err error) ErrorAction
}

// ErrorAction defines what action to take on error
type ErrorAction string

// ErrorAction constants
const (
	ErrorActionContinue ErrorAction = "continue"
	ErrorActionAbort    ErrorAction = "abort"
	ErrorActionRetry    ErrorAction = "retry"
	ErrorActionRollback ErrorAction = "rollback"
)

// WorkflowResult represents the result of a workflow execution
type WorkflowResult struct {
	Status      ExecutionStatus
	StepResults map[string]ExecutionResult
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
}

// PluginFilter defines criteria for filtering plugins
type PluginFilter struct {
	Names          []string
	Versions       []string
	SupportedKinds []string
	Capabilities   []Capability
}

// ============================================================================
// Built-in Execution Strategies
// ============================================================================

// SimpleStrategy executes a plugin without any special handling
type SimpleStrategy struct{}

// RetryStrategy retries failed executions with exponential backoff
type RetryStrategy struct {
	MaxRetries int
	Backoff    BackoffStrategy
}

// BackoffStrategy defines how to calculate retry delays
type BackoffStrategy interface {
	NextDelay(attempt int) time.Duration
}

// CircuitBreakerStrategy implements circuit breaker pattern
type CircuitBreakerStrategy struct {
	FailureThreshold int
	Timeout          time.Duration
	ResetTimeout     time.Duration
}

// RateLimitStrategy limits execution rate
type RateLimitStrategy struct {
	RequestsPerSecond int
}

// ============================================================================
// Monitor Types
// ============================================================================

// Monitor provides health checking and metrics collection with pluggable strategies
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

// HealthCheckStrategy defines how to check resource health
type HealthCheckStrategy interface {
	Check(ctx context.Context, resource Resource) (HealthStatus, error)
	Name() string
}

// HealthStatus represents the health status of a resource
type HealthStatus struct {
	Status  HealthStatusType
	Message string
	Checks  []HealthCheck
}

// HealthStatusType represents the type of health status
type HealthStatusType string

// HealthStatusType constants
const (
	HealthStatusHealthy   HealthStatusType = "healthy"
	HealthStatusUnhealthy HealthStatusType = "unhealthy"
	HealthStatusUnknown   HealthStatusType = "unknown"
)

// HealthCheck represents a single health check result
type HealthCheck struct {
	Name    string
	Status  HealthStatusType
	Message string
}

// ============================================================================
// Built-in Health Check Strategies
// ============================================================================

// HTTPHealthCheckStrategy checks health via HTTP endpoint
type HTTPHealthCheckStrategy struct {
	ExpectedStatusCodes []int
	ExpectedBody        string
}

// TCPHealthCheckStrategy checks health via TCP connection
type TCPHealthCheckStrategy struct {
	DialTimeout time.Duration
}

// ExecHealthCheckStrategy checks health via command execution
type ExecHealthCheckStrategy struct {
	ExpectedExitCode int
}

// ============================================================================
// Reporter Types
// ============================================================================

// Reporter records execution results and generates reports with pluggable storage and formatters
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

// ExecutionRecord represents a recorded plugin execution
type ExecutionRecord struct {
	ID           string
	PluginName   string
	ResourceID   string
	ResourceName string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Status       ExecutionStatus
	Error        string
	Metadata     map[string]interface{}

	// Security
	Principal string
}

// EventQuery defines criteria for querying events
type EventQuery struct {
	Types     []string
	Sources   []string
	Resources []string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Offset    int
}

// ExecutionQuery defines criteria for querying executions
type ExecutionQuery struct {
	PluginNames []string
	ResourceIDs []string
	Statuses    []ExecutionStatus
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Offset      int
}

// StatisticsFilter defines criteria for computing statistics
type StatisticsFilter struct {
	PluginNames []string
	ResourceIDs []string
	StartTime   time.Time
	EndTime     time.Time
}

// Statistics represents aggregated execution statistics
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

// PluginStatistics represents statistics for a specific plugin
type PluginStatistics struct {
	PluginName      string
	TotalExecutions int
	SuccessCount    int
	FailureCount    int
	SuccessRate     float64
	AverageDuration time.Duration
}

// ResourceStatistics represents statistics for a specific resource
type ResourceStatistics struct {
	ResourceID      string
	ResourceName    string
	TotalExecutions int
	SuccessCount    int
	FailureCount    int
	SuccessRate     float64
	AverageDuration time.Duration
}

// ReportFilter defines criteria for generating reports
type ReportFilter struct {
	PluginNames []string
	ResourceIDs []string
	StartTime   time.Time
	EndTime     time.Time
}

// StorageBackend defines the interface for pluggable storage backends
type StorageBackend interface {
	Save(ctx context.Context, key string, value interface{}) error
	Load(ctx context.Context, key string) (interface{}, error)
	Query(ctx context.Context, query Query) ([]interface{}, error)
	Delete(ctx context.Context, key string) error
	Close() error
}

// Query defines a query for the storage backend
type Query struct {
	Collection string
	Filter     map[string]interface{}
	Sort       []SortField
	Limit      int
	Offset     int
}

// SortField defines a field to sort by
type SortField struct {
	Field      string
	Descending bool
}

// ReportFormatter defines the interface for report formatters
type ReportFormatter interface {
	Format(ctx context.Context, data interface{}) ([]byte, error)
	ContentType() string
	Name() string
}

// ============================================================================
// Built-in Report Formatters
// ============================================================================

// JSONFormatter formats reports as JSON
type JSONFormatter struct{}

// MarkdownFormatter formats reports as Markdown
type MarkdownFormatter struct{}

// HTMLFormatter formats reports as HTML
type HTMLFormatter struct{}

// ============================================================================
// EventBus Types
// ============================================================================

// EventBus enables loose coupling between components through publish-subscribe messaging
type EventBus interface {
	// Publishing
	Publish(ctx context.Context, event Event) error
	PublishAsync(ctx context.Context, event Event) error

	// Subscribing
	Subscribe(ctx context.Context, filter EventFilter) (Subscription, error)

	// Lifecycle
	Close() error
}

// EventFilter defines criteria for filtering events
type EventFilter struct {
	Types    []string          // Filter by event types
	Sources  []string          // Filter by event sources
	Metadata map[string]string // Filter by metadata key-value pairs
}

// Subscription represents an active event subscription
type Subscription interface {
	ID() string
	Events() <-chan Event
	Unsubscribe() error
}

// ============================================================================
// Config Types
// ============================================================================

// Config supports multiple configuration sources with precedence and hot-reloading
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

// ConfigProvider defines the interface for configuration sources
type ConfigProvider interface {
	Load(ctx context.Context) (map[string]interface{}, error)
	Watch(ctx context.Context) (<-chan ConfigChange, error)
	Name() string
}

// ConfigChange represents a configuration change event
type ConfigChange struct {
	Key      string
	OldValue interface{}
	NewValue interface{}
}

// ============================================================================
// Built-in Config Providers
// ============================================================================

// YAMLConfigProvider loads configuration from YAML files
type YAMLConfigProvider struct {
	FilePath string
}

// EnvironmentConfigProvider loads configuration from environment variables
type EnvironmentConfigProvider struct {
	Prefix string // Prefix for environment variables (e.g., "APP_")
}

// ConsulConfigProvider loads configuration from HashiCorp Consul
type ConsulConfigProvider struct {
	Address string // Consul address
	Prefix  string // Key prefix in Consul
}

// VaultConfigProvider loads configuration from HashiCorp Vault
type VaultConfigProvider struct {
	Address string // Vault address
	Token   string // Vault token
	Path    string // Secret path in Vault
}
