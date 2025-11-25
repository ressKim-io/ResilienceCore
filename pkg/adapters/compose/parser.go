package compose

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
	"gopkg.in/yaml.v3"
)

// File represents a docker-compose.yml file structure
type File struct {
	Version  string                 `yaml:"version"`
	Services map[string]Service     `yaml:"services"`
	Networks map[string]interface{} `yaml:"networks,omitempty"`
	Volumes  map[string]interface{} `yaml:"volumes,omitempty"`
}

// Service represents a service definition in docker-compose.yml
type Service struct {
	Image       string                 `yaml:"image,omitempty"`
	Build       interface{}            `yaml:"build,omitempty"`
	Command     interface{}            `yaml:"command,omitempty"`
	Entrypoint  interface{}            `yaml:"entrypoint,omitempty"`
	Environment interface{}            `yaml:"environment,omitempty"`
	Ports       []string               `yaml:"ports,omitempty"`
	Volumes     []string               `yaml:"volumes,omitempty"`
	Networks    interface{}            `yaml:"networks,omitempty"`
	DependsOn   interface{}            `yaml:"depends_on,omitempty"`
	Restart     string                 `yaml:"restart,omitempty"`
	HealthCheck *HealthCheck           `yaml:"healthcheck,omitempty"`
	Labels      map[string]string      `yaml:"labels,omitempty"`
	Deploy      *Deploy                `yaml:"deploy,omitempty"`
	Extra       map[string]interface{} `yaml:",inline"`
}

// HealthCheck represents health check configuration
type HealthCheck struct {
	Test     interface{} `yaml:"test,omitempty"`
	Interval string      `yaml:"interval,omitempty"`
	Timeout  string      `yaml:"timeout,omitempty"`
	Retries  int         `yaml:"retries,omitempty"`
}

// Deploy represents deploy configuration
type Deploy struct {
	Replicas *int32                 `yaml:"replicas,omitempty"`
	Extra    map[string]interface{} `yaml:",inline"`
}

// ParseFile parses a docker-compose.yml file
func ParseFile(path string) (*File, error) {
	// #nosec G304 - path is provided by user configuration, not external input
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var compose File
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	return &compose, nil
}

// ServiceToResourceSpec converts a Service to a ResourceSpec
func ServiceToResourceSpec(name string, service Service) types.ResourceSpec {
	spec := types.ResourceSpec{
		Image:       service.Image,
		Ports:       parsePorts(service.Ports),
		Environment: parseEnvironment(service.Environment),
		Volumes:     parseVolumes(service.Volumes),
		Command:     parseCommand(service.Command),
		Args:        parseArgs(service.Entrypoint),
	}

	// Parse replicas from deploy section
	if service.Deploy != nil && service.Deploy.Replicas != nil {
		spec.Replicas = service.Deploy.Replicas
	}

	// Parse health check
	if service.HealthCheck != nil {
		spec.HealthCheck = parseHealthCheck(service.HealthCheck)
	}

	return spec
}

// parsePorts converts docker-compose port definitions to Port structs
func parsePorts(ports []string) []types.Port {
	var result []types.Port
	for _, p := range ports {
		port := parsePort(p)
		if port != nil {
			result = append(result, *port)
		}
	}
	return result
}

// parsePort parses a single port definition
// Supports formats: "8080", "8080:80", "127.0.0.1:8080:80", "8080:80/tcp"
func parsePort(portStr string) *types.Port {
	// Remove protocol suffix if present
	protocol := "TCP"
	if strings.Contains(portStr, "/") {
		parts := strings.Split(portStr, "/")
		portStr = parts[0]
		if len(parts) > 1 {
			protocol = strings.ToUpper(parts[1])
		}
	}

	// Split by colon
	parts := strings.Split(portStr, ":")

	var hostPort, containerPort int32

	switch len(parts) {
	case 1:
		// Just container port: "8080"
		if p, err := strconv.ParseInt(parts[0], 10, 32); err == nil {
			containerPort = int32(p)
			hostPort = int32(p)
		}
	case 2:
		// Host:Container: "8080:80"
		if hp, err := strconv.ParseInt(parts[0], 10, 32); err == nil {
			hostPort = int32(hp)
		}
		if cp, err := strconv.ParseInt(parts[1], 10, 32); err == nil {
			containerPort = int32(cp)
		}
	case 3:
		// IP:Host:Container: "127.0.0.1:8080:80" - ignore IP for now
		if hp, err := strconv.ParseInt(parts[1], 10, 32); err == nil {
			hostPort = int32(hp)
		}
		if cp, err := strconv.ParseInt(parts[2], 10, 32); err == nil {
			containerPort = int32(cp)
		}
	default:
		return nil
	}

	if containerPort == 0 {
		return nil
	}

	return &types.Port{
		ContainerPort: containerPort,
		HostPort:      hostPort,
		Protocol:      protocol,
	}
}

// parseEnvironment converts docker-compose environment to map
func parseEnvironment(env interface{}) map[string]string {
	result := make(map[string]string)

	switch e := env.(type) {
	case map[string]interface{}:
		// Object format: {KEY: value}
		for k, v := range e {
			result[k] = fmt.Sprintf("%v", v)
		}
	case []interface{}:
		// Array format: ["KEY=value"]
		for _, item := range e {
			if str, ok := item.(string); ok {
				parts := strings.SplitN(str, "=", 2)
				if len(parts) == 2 {
					result[parts[0]] = parts[1]
				} else if len(parts) == 1 {
					// Environment variable without value
					result[parts[0]] = ""
				}
			}
		}
	}

	return result
}

// parseVolumes converts docker-compose volumes to Volume structs
func parseVolumes(volumes []string) []types.Volume {
	var result []types.Volume
	for i, v := range volumes {
		vol := parseVolume(i, v)
		if vol != nil {
			result = append(result, *vol)
		}
	}
	return result
}

// parseVolume parses a single volume definition
// Supports formats: "/host/path:/container/path", "/host/path:/container/path:ro", "volume_name:/container/path"
func parseVolume(index int, volStr string) *types.Volume {
	parts := strings.Split(volStr, ":")
	if len(parts) < 2 {
		return nil
	}

	readOnly := false
	if len(parts) >= 3 && parts[2] == "ro" {
		readOnly = true
	}

	source := parts[0]
	mountPath := parts[1]

	vol := &types.Volume{
		Name:      fmt.Sprintf("volume-%d", index),
		MountPath: mountPath,
		ReadOnly:  readOnly,
	}

	// Determine volume source type
	if strings.HasPrefix(source, "/") || strings.HasPrefix(source, ".") {
		// Host path
		vol.Source.HostPath = &source
	} else {
		// Named volume - treat as empty dir for now
		vol.Source.EmptyDir = true
	}

	return vol
}

// parseCommand converts docker-compose command to string slice
func parseCommand(cmd interface{}) []string {
	switch c := cmd.(type) {
	case string:
		// Shell form: "command arg1 arg2"
		return strings.Fields(c)
	case []interface{}:
		// Exec form: ["command", "arg1", "arg2"]
		var result []string
		for _, item := range c {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	}
	return nil
}

// parseArgs converts docker-compose entrypoint to string slice
func parseArgs(entrypoint interface{}) []string {
	switch e := entrypoint.(type) {
	case string:
		return strings.Fields(e)
	case []interface{}:
		var result []string
		for _, item := range e {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	}
	return nil
}

// parseHealthCheck converts docker-compose health check to HealthCheckConfig
func parseHealthCheck(hc *HealthCheck) *types.HealthCheckConfig {
	if hc == nil {
		return nil
	}

	config := &types.HealthCheckConfig{
		Retries: hc.Retries,
	}

	// Parse test command
	switch t := hc.Test.(type) {
	case string:
		// Simple string command
		switch {
		case strings.HasPrefix(t, "CMD-SHELL "):
			config.Type = "exec"
			config.Endpoint = strings.TrimPrefix(t, "CMD-SHELL ")
		case strings.HasPrefix(t, "CMD "):
			config.Type = "exec"
			config.Endpoint = strings.TrimPrefix(t, "CMD ")
		default:
			config.Type = "exec"
			config.Endpoint = t
		}
	case []interface{}:
		// Array format: ["CMD", "curl", "-f", "http://localhost"]
		if len(t) > 0 {
			if first, ok := t[0].(string); ok {
				if first == "NONE" {
					return nil
				}
				if first == "CMD" || first == "CMD-SHELL" {
					config.Type = "exec"
					var parts []string
					for i := 1; i < len(t); i++ {
						parts = append(parts, fmt.Sprintf("%v", t[i]))
					}
					config.Endpoint = strings.Join(parts, " ")
				}
			}
		}
	}

	// Parse interval and timeout
	// Docker compose uses duration strings like "30s", "1m30s"
	// In a real implementation, we'd parse these properly and set them on the config
	// For now, we skip this as it's not critical for the basic implementation
	_ = hc.Interval // Acknowledge we're intentionally not using this yet

	return config
}
