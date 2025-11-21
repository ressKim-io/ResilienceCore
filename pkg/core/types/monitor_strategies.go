// Package types provides core type definitions and interfaces for the Infrastructure Resilience Engine.
// This file contains built-in health check strategy implementations.
package types

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// HTTPHealthCheckStrategy Implementation
// ============================================================================

// Check implements HealthCheckStrategy for HTTPHealthCheckStrategy
func (h *HTTPHealthCheckStrategy) Check(ctx context.Context, resource Resource) (HealthStatus, error) {
	if resource.Spec.HealthCheck == nil {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: "no health check configuration",
		}, fmt.Errorf("resource has no health check configuration")
	}

	if resource.Spec.HealthCheck.Type != "http" {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: "health check type is not http",
		}, fmt.Errorf("health check type mismatch: expected http, got %s", resource.Spec.HealthCheck.Type)
	}

	endpoint := resource.Spec.HealthCheck.Endpoint
	if endpoint == "" {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: "no endpoint specified",
		}, fmt.Errorf("no endpoint specified for HTTP health check")
	}

	// Create HTTP client with timeout
	timeout := resource.Spec.HealthCheck.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return HealthStatus{
			Status:  HealthStatusUnhealthy,
			Message: fmt.Sprintf("failed to create request: %v", err),
			Checks: []HealthCheck{
				{
					Name:    "http",
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("request creation failed: %v", err),
				},
			},
		}, err
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return HealthStatus{
			Status:  HealthStatusUnhealthy,
			Message: fmt.Sprintf("HTTP request failed: %v", err),
			Checks: []HealthCheck{
				{
					Name:    "http",
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("request failed: %v", err),
				},
			},
		}, err
	}
	defer resp.Body.Close()

	// Check status code
	expectedCodes := h.ExpectedStatusCodes
	if len(expectedCodes) == 0 {
		expectedCodes = []int{200}
	}

	statusOK := false
	for _, code := range expectedCodes {
		if resp.StatusCode == code {
			statusOK = true
			break
		}
	}

	if !statusOK {
		return HealthStatus{
			Status:  HealthStatusUnhealthy,
			Message: fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
			Checks: []HealthCheck{
				{
					Name:    "http",
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("status code %d not in expected codes %v", resp.StatusCode, expectedCodes),
				},
			},
		}, nil
	}

	// Check body if expected
	if h.ExpectedBody != "" {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return HealthStatus{
				Status:  HealthStatusUnhealthy,
				Message: fmt.Sprintf("failed to read response body: %v", err),
				Checks: []HealthCheck{
					{
						Name:    "http",
						Status:  HealthStatusUnhealthy,
						Message: fmt.Sprintf("body read failed: %v", err),
					},
				},
			}, err
		}

		if !strings.Contains(string(body), h.ExpectedBody) {
			return HealthStatus{
				Status:  HealthStatusUnhealthy,
				Message: "response body does not contain expected content",
				Checks: []HealthCheck{
					{
						Name:    "http",
						Status:  HealthStatusUnhealthy,
						Message: fmt.Sprintf("body does not contain '%s'", h.ExpectedBody),
					},
				},
			}, nil
		}
	}

	return HealthStatus{
		Status:  HealthStatusHealthy,
		Message: "HTTP health check passed",
		Checks: []HealthCheck{
			{
				Name:    "http",
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("status code %d", resp.StatusCode),
			},
		},
	}, nil
}

// Name implements HealthCheckStrategy for HTTPHealthCheckStrategy
func (h *HTTPHealthCheckStrategy) Name() string {
	return "http"
}

// ============================================================================
// TCPHealthCheckStrategy Implementation
// ============================================================================

// Check implements HealthCheckStrategy for TCPHealthCheckStrategy
func (t *TCPHealthCheckStrategy) Check(ctx context.Context, resource Resource) (HealthStatus, error) {
	if resource.Spec.HealthCheck == nil {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: "no health check configuration",
		}, fmt.Errorf("resource has no health check configuration")
	}

	if resource.Spec.HealthCheck.Type != "tcp" {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: "health check type is not tcp",
		}, fmt.Errorf("health check type mismatch: expected tcp, got %s", resource.Spec.HealthCheck.Type)
	}

	endpoint := resource.Spec.HealthCheck.Endpoint
	if endpoint == "" {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: "no endpoint specified",
		}, fmt.Errorf("no endpoint specified for TCP health check")
	}

	// Set dial timeout
	timeout := t.DialTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	// Attempt to connect
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return HealthStatus{
			Status:  HealthStatusUnhealthy,
			Message: fmt.Sprintf("TCP connection failed: %v", err),
			Checks: []HealthCheck{
				{
					Name:    "tcp",
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("connection failed: %v", err),
				},
			},
		}, err
	}
	defer conn.Close()

	return HealthStatus{
		Status:  HealthStatusHealthy,
		Message: "TCP connection successful",
		Checks: []HealthCheck{
			{
				Name:    "tcp",
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("connected to %s", endpoint),
			},
		},
	}, nil
}

// Name implements HealthCheckStrategy for TCPHealthCheckStrategy
func (t *TCPHealthCheckStrategy) Name() string {
	return "tcp"
}

// ============================================================================
// ExecHealthCheckStrategy Implementation
// ============================================================================

// Check implements HealthCheckStrategy for ExecHealthCheckStrategy
func (e *ExecHealthCheckStrategy) Check(ctx context.Context, resource Resource) (HealthStatus, error) {
	if resource.Spec.HealthCheck == nil {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: "no health check configuration",
		}, fmt.Errorf("resource has no health check configuration")
	}

	if resource.Spec.HealthCheck.Type != "exec" {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: "health check type is not exec",
		}, fmt.Errorf("health check type mismatch: expected exec, got %s", resource.Spec.HealthCheck.Type)
	}

	endpoint := resource.Spec.HealthCheck.Endpoint
	if endpoint == "" {
		return HealthStatus{
			Status:  HealthStatusUnknown,
			Message: "no command specified",
		}, fmt.Errorf("no command specified for exec health check")
	}

	// Note: This is a placeholder implementation
	// The actual execution would require an EnvironmentAdapter to execute the command
	// For now, we return unknown status as we don't have access to the adapter here
	return HealthStatus{
		Status:  HealthStatusUnknown,
		Message: "exec health check requires adapter integration",
		Checks: []HealthCheck{
			{
				Name:    "exec",
				Status:  HealthStatusUnknown,
				Message: "exec health check not yet implemented (requires adapter)",
			},
		},
	}, fmt.Errorf("exec health check requires EnvironmentAdapter integration")
}

// Name implements HealthCheckStrategy for ExecHealthCheckStrategy
func (e *ExecHealthCheckStrategy) Name() string {
	return "exec"
}
