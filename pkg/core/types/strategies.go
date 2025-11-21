// Package types provides core type definitions and interfaces for the Infrastructure Resilience Engine.
// This file contains built-in execution strategy implementations.
package types

import (
	"context"
	"time"
)

// ============================================================================
// Built-in Strategy Implementations
// ============================================================================

// Execute implements ExecutionStrategy for SimpleStrategy
func (s *SimpleStrategy) Execute(ctx context.Context, plugin Plugin, resource Resource) (ExecutionResult, error) {
	startTime := time.Now()

	pluginCtx := PluginContext{
		Context:     ctx,
		ExecutionID: "simple-exec",
		Timeout:     30 * time.Second,
	}

	// Execute plugin
	err := plugin.Execute(pluginCtx, resource)

	endTime := time.Now()
	status := StatusSuccess
	if err != nil {
		switch err {
		case context.DeadlineExceeded:
			status = StatusTimeout
		case context.Canceled:
			status = StatusCanceled
		default:
			status = StatusFailed
		}
	}

	return ExecutionResult{
		Status:    status,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Error:     err,
	}, nil
}

// Name implements ExecutionStrategy for SimpleStrategy
func (s *SimpleStrategy) Name() string {
	return "simple"
}

// Execute implements ExecutionStrategy for RetryStrategy
func (r *RetryStrategy) Execute(ctx context.Context, plugin Plugin, resource Resource) (ExecutionResult, error) {
	var lastErr error
	var result ExecutionResult

	for attempt := 0; attempt <= r.MaxRetries; attempt++ {
		startTime := time.Now()

		pluginCtx := PluginContext{
			Context:     ctx,
			ExecutionID: "retry-exec",
			Timeout:     30 * time.Second,
		}

		// Execute plugin
		err := plugin.Execute(pluginCtx, resource)

		endTime := time.Now()

		if err == nil {
			return ExecutionResult{
				Status:    StatusSuccess,
				StartTime: startTime,
				EndTime:   endTime,
				Duration:  endTime.Sub(startTime),
			}, nil
		}

		lastErr = err
		result = ExecutionResult{
			Status:    StatusFailed,
			StartTime: startTime,
			EndTime:   endTime,
			Duration:  endTime.Sub(startTime),
			Error:     err,
		}

		// If not the last attempt, wait before retrying
		if attempt < r.MaxRetries && r.Backoff != nil {
			delay := r.Backoff.NextDelay(attempt)
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return ExecutionResult{
					Status: StatusCanceled,
					Error:  ctx.Err(),
				}, ctx.Err()
			}
		}
	}

	return result, lastErr
}

// Name implements ExecutionStrategy for RetryStrategy
func (r *RetryStrategy) Name() string {
	return "retry"
}

// Execute implements ExecutionStrategy for CircuitBreakerStrategy
func (c *CircuitBreakerStrategy) Execute(ctx context.Context, plugin Plugin, resource Resource) (ExecutionResult, error) {
	// Simplified circuit breaker implementation for interface compliance
	startTime := time.Now()

	pluginCtx := PluginContext{
		Context:     ctx,
		ExecutionID: "circuit-breaker-exec",
		Timeout:     c.Timeout,
	}

	// Execute plugin with timeout
	execCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	pluginCtx.Context = execCtx

	err := plugin.Execute(pluginCtx, resource)

	endTime := time.Now()
	status := StatusSuccess
	if err != nil {
		switch err {
		case context.DeadlineExceeded:
			status = StatusTimeout
		default:
			status = StatusFailed
		}
	}

	return ExecutionResult{
		Status:    status,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Error:     err,
	}, nil
}

// Name implements ExecutionStrategy for CircuitBreakerStrategy
func (c *CircuitBreakerStrategy) Name() string {
	return "circuit-breaker"
}

// Execute implements ExecutionStrategy for RateLimitStrategy
func (r *RateLimitStrategy) Execute(ctx context.Context, plugin Plugin, resource Resource) (ExecutionResult, error) {
	// Simplified rate limiter implementation for interface compliance
	startTime := time.Now()

	pluginCtx := PluginContext{
		Context:     ctx,
		ExecutionID: "rate-limit-exec",
		Timeout:     30 * time.Second,
	}

	// Execute plugin
	err := plugin.Execute(pluginCtx, resource)

	endTime := time.Now()
	status := StatusSuccess
	if err != nil {
		status = StatusFailed
	}

	return ExecutionResult{
		Status:    status,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Error:     err,
	}, nil
}

// Name implements ExecutionStrategy for RateLimitStrategy
func (r *RateLimitStrategy) Name() string {
	return "rate-limit"
}
