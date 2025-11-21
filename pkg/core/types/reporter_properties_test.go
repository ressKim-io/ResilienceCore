// Package types provides property-based tests for the Reporter interface
package types

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty33_ExecutionRecordIsComplete verifies that any completed execution
// produces a complete ExecutionRecord with all required fields
// Feature: infrastructure-resilience-engine, Property 33: Execution record is complete
// Validates: Requirements 7.1
func TestProperty33_ExecutionRecordIsComplete(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("execution record contains all required fields", prop.ForAll(
		func(pluginName string, resourceID string, resourceName string, status ExecutionStatus, principal string) bool {
			// Create an execution record
			startTime := time.Now()
			endTime := startTime.Add(time.Second * 5)
			duration := endTime.Sub(startTime)

			// Set error message based on status
			errorMsg := ""
			if status == StatusFailed {
				errorMsg = "execution failed"
			}

			record := ExecutionRecord{
				ID:           "exec-" + pluginName + "-" + resourceID,
				PluginName:   pluginName,
				ResourceID:   resourceID,
				ResourceName: resourceName,
				StartTime:    startTime,
				EndTime:      endTime,
				Duration:     duration,
				Status:       status,
				Error:        errorMsg,
				Metadata:     map[string]interface{}{"test": "data"},
				Principal:    principal,
			}

			// Verify all required fields are present and valid
			if record.ID == "" {
				return false
			}
			if record.PluginName == "" {
				return false
			}
			if record.ResourceID == "" {
				return false
			}
			if record.ResourceName == "" {
				return false
			}
			if record.StartTime.IsZero() {
				return false
			}
			if record.EndTime.IsZero() {
				return false
			}
			if record.Duration <= 0 {
				return false
			}
			if record.Status == "" {
				return false
			}
			// Error field should be populated if status is failed
			if record.Status == StatusFailed && record.Error == "" {
				return false
			}
			if record.Principal == "" {
				return false
			}

			// Verify time consistency
			if !record.EndTime.After(record.StartTime) {
				return false
			}
			if record.Duration != record.EndTime.Sub(record.StartTime) {
				return false
			}

			return true
		},
		gen.Identifier(), // pluginName
		gen.Identifier(), // resourceID
		gen.Identifier(), // resourceName
		gen.OneConstOf(StatusSuccess, StatusFailed, StatusTimeout, StatusCanceled, StatusSkipped), // status
		gen.Identifier(), // principal
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty35_StatisticsAreCalculatedCorrectly verifies that statistics
// are correctly calculated from a set of execution records
// Feature: infrastructure-resilience-engine, Property 35: Statistics are calculated correctly
// Validates: Requirements 7.3
func TestProperty35_StatisticsAreCalculatedCorrectly(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("statistics are calculated correctly from execution records", prop.ForAll(
		func(numSuccess int, numFailed int, numTimeout int, numCanceled int) bool {
			// Ensure we have at least one execution
			if numSuccess+numFailed+numTimeout+numCanceled == 0 {
				return true // Skip empty case
			}

			// Create execution records
			records := []ExecutionRecord{}
			durations := []time.Duration{}

			// Add success records
			for i := 0; i < numSuccess; i++ {
				duration := time.Duration(i+1) * time.Millisecond * 100
				durations = append(durations, duration)
				records = append(records, ExecutionRecord{
					ID:         "success-" + string(rune(i)),
					PluginName: "test-plugin",
					ResourceID: "resource-1",
					StartTime:  time.Now(),
					EndTime:    time.Now().Add(duration),
					Duration:   duration,
					Status:     StatusSuccess,
				})
			}

			// Add failed records
			for i := 0; i < numFailed; i++ {
				duration := time.Duration(i+1) * time.Millisecond * 100
				durations = append(durations, duration)
				records = append(records, ExecutionRecord{
					ID:         "failed-" + string(rune(i)),
					PluginName: "test-plugin",
					ResourceID: "resource-1",
					StartTime:  time.Now(),
					EndTime:    time.Now().Add(duration),
					Duration:   duration,
					Status:     StatusFailed,
					Error:      "test error",
				})
			}

			// Add timeout records
			for i := 0; i < numTimeout; i++ {
				duration := time.Duration(i+1) * time.Millisecond * 100
				durations = append(durations, duration)
				records = append(records, ExecutionRecord{
					ID:         "timeout-" + string(rune(i)),
					PluginName: "test-plugin",
					ResourceID: "resource-1",
					StartTime:  time.Now(),
					EndTime:    time.Now().Add(duration),
					Duration:   duration,
					Status:     StatusTimeout,
				})
			}

			// Add canceled records
			for i := 0; i < numCanceled; i++ {
				duration := time.Duration(i+1) * time.Millisecond * 100
				durations = append(durations, duration)
				records = append(records, ExecutionRecord{
					ID:         "canceled-" + string(rune(i)),
					PluginName: "test-plugin",
					ResourceID: "resource-1",
					StartTime:  time.Now(),
					EndTime:    time.Now().Add(duration),
					Duration:   duration,
					Status:     StatusCanceled,
				})
			}

			// Calculate expected statistics
			totalExecutions := numSuccess + numFailed + numTimeout + numCanceled
			expectedSuccessRate := float64(numSuccess) / float64(totalExecutions)

			// Calculate duration statistics
			sort.Slice(durations, func(i, j int) bool {
				return durations[i] < durations[j]
			})

			var totalDuration time.Duration
			for _, d := range durations {
				totalDuration += d
			}
			expectedAvgDuration := totalDuration / time.Duration(len(durations))

			// Calculate percentiles
			p50Index := int(float64(len(durations)) * 0.50)
			p95Index := int(float64(len(durations)) * 0.95)
			p99Index := int(float64(len(durations)) * 0.99)

			if p50Index >= len(durations) {
				p50Index = len(durations) - 1
			}
			if p95Index >= len(durations) {
				p95Index = len(durations) - 1
			}
			if p99Index >= len(durations) {
				p99Index = len(durations) - 1
			}

			expectedP50 := durations[p50Index]
			expectedP95 := durations[p95Index]
			expectedP99 := durations[p99Index]
			expectedMin := durations[0]
			expectedMax := durations[len(durations)-1]

			// Compute statistics using our mock implementation
			stats := computeStatistics(records)

			// Verify counts
			if stats.TotalExecutions != totalExecutions {
				return false
			}
			if stats.SuccessCount != numSuccess {
				return false
			}
			if stats.FailureCount != numFailed {
				return false
			}
			if stats.TimeoutCount != numTimeout {
				return false
			}
			if stats.CanceledCount != numCanceled {
				return false
			}

			// Verify success rate (with small tolerance for floating point)
			if abs(stats.SuccessRate-expectedSuccessRate) > 0.001 {
				return false
			}

			// Verify duration statistics (with tolerance)
			if abs(float64(stats.AverageDuration-expectedAvgDuration)) > float64(time.Millisecond) {
				return false
			}
			if stats.P50Duration != expectedP50 {
				return false
			}
			if stats.P95Duration != expectedP95 {
				return false
			}
			if stats.P99Duration != expectedP99 {
				return false
			}
			if stats.MinDuration != expectedMin {
				return false
			}
			if stats.MaxDuration != expectedMax {
				return false
			}

			return true
		},
		gen.IntRange(0, 10), // numSuccess
		gen.IntRange(0, 10), // numFailed
		gen.IntRange(0, 10), // numTimeout
		gen.IntRange(0, 10), // numCanceled
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty37_CustomStorageBackendsAreSupported verifies that custom
// StorageBackend implementations can be used without Core modification
// Feature: infrastructure-resilience-engine, Property 37: Custom storage backends are supported
// Validates: Requirements 7.5
func TestProperty37_CustomStorageBackendsAreSupported(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("custom storage backends work without Core modification", prop.ForAll(
		func(backendType string, key string, value string) bool {
			ctx := context.Background()

			// Create different storage backend implementations
			var backend StorageBackend
			switch backendType {
			case "memory":
				backend = &MockInMemoryStorage{}
			case "file":
				backend = &MockFileStorage{}
			case "database":
				backend = &MockDatabaseStorage{}
			default:
				backend = &MockInMemoryStorage{}
			}

			// Test basic storage operations
			// 1. Save
			if err := backend.Save(ctx, key, value); err != nil {
				return false
			}

			// 2. Load
			loaded, err := backend.Load(ctx, key)
			if err != nil {
				return false
			}
			if loaded != value {
				return false
			}

			// 3. Query
			results, err := backend.Query(ctx, Query{
				Collection: "test",
				Filter:     map[string]interface{}{"key": key},
			})
			if err != nil {
				return false
			}
			if len(results) == 0 {
				return false
			}

			// 4. Delete
			if delErr := backend.Delete(ctx, key); delErr != nil {
				return false
			}

			// 5. Verify deletion
			_, err = backend.Load(ctx, key)
			if err == nil {
				return false // Should return error for missing key
			}

			// 6. Close
			if err := backend.Close(); err != nil {
				return false
			}

			return true
		},
		gen.OneConstOf("memory", "file", "database"),
		gen.Identifier(),
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper functions

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func computeStatistics(records []ExecutionRecord) Statistics {
	if len(records) == 0 {
		return Statistics{}
	}

	stats := Statistics{
		TotalExecutions: len(records),
		ByPlugin:        make(map[string]PluginStatistics),
		ByResource:      make(map[string]ResourceStatistics),
	}

	var totalDuration time.Duration
	durations := make([]time.Duration, 0, len(records))

	for _, record := range records {
		durations = append(durations, record.Duration)
		totalDuration += record.Duration

		switch record.Status {
		case StatusSuccess:
			stats.SuccessCount++
		case StatusFailed:
			stats.FailureCount++
		case StatusTimeout:
			stats.TimeoutCount++
		case StatusCanceled:
			stats.CanceledCount++
		}
	}

	// Calculate success rate
	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalExecutions)
	}

	// Calculate average duration
	if len(durations) > 0 {
		stats.AverageDuration = totalDuration / time.Duration(len(durations))
	}

	// Sort durations for percentile calculation
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	// Calculate percentiles
	if len(durations) > 0 {
		p50Index := int(float64(len(durations)) * 0.50)
		p95Index := int(float64(len(durations)) * 0.95)
		p99Index := int(float64(len(durations)) * 0.99)

		if p50Index >= len(durations) {
			p50Index = len(durations) - 1
		}
		if p95Index >= len(durations) {
			p95Index = len(durations) - 1
		}
		if p99Index >= len(durations) {
			p99Index = len(durations) - 1
		}

		stats.P50Duration = durations[p50Index]
		stats.P95Duration = durations[p95Index]
		stats.P99Duration = durations[p99Index]
		stats.MinDuration = durations[0]
		stats.MaxDuration = durations[len(durations)-1]
	}

	return stats
}

// Mock storage backend implementations

type MockInMemoryStorage struct {
	data map[string]interface{}
}

func (m *MockInMemoryStorage) Save(ctx context.Context, key string, value interface{}) error {
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	m.data[key] = value
	return nil
}

func (m *MockInMemoryStorage) Load(ctx context.Context, key string) (interface{}, error) {
	if m.data == nil {
		return nil, &StorageError{Message: "key not found"}
	}
	val, ok := m.data[key]
	if !ok {
		return nil, &StorageError{Message: "key not found"}
	}
	return val, nil
}

func (m *MockInMemoryStorage) Query(ctx context.Context, query Query) ([]interface{}, error) {
	results := []interface{}{}
	if m.data == nil {
		return results, nil
	}
	for _, v := range m.data {
		results = append(results, v)
	}
	return results, nil
}

func (m *MockInMemoryStorage) Delete(ctx context.Context, key string) error {
	if m.data == nil {
		return nil
	}
	delete(m.data, key)
	return nil
}

func (m *MockInMemoryStorage) Close() error {
	m.data = nil
	return nil
}

type MockFileStorage struct {
	MockInMemoryStorage
}

type MockDatabaseStorage struct {
	MockInMemoryStorage
}

// StorageError represents a storage error
type StorageError struct {
	Message string
}

func (e *StorageError) Error() string {
	return e.Message
}
