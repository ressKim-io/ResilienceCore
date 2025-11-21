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
