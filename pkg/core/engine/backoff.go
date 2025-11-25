package engine

import (
	"math"
	"time"
)

// ExponentialBackoff implements exponential backoff strategy
type ExponentialBackoff struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// NewExponentialBackoff creates a new exponential backoff strategy with sensible defaults
func NewExponentialBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
	}
}

// NextDelay calculates the next delay based on the attempt number
func (e *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Calculate delay: initialDelay * multiplier^attempt
	delay := float64(e.InitialDelay) * math.Pow(e.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(e.MaxDelay) {
		return e.MaxDelay
	}

	return time.Duration(delay)
}
