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
