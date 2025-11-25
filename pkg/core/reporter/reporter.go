package reporter

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// DefaultReporter is the default implementation of the Reporter interface
type DefaultReporter struct {
	mu         sync.RWMutex
	storage    types.StorageBackend
	formatters map[string]types.ReportFormatter
}

// NewDefaultReporter creates a new DefaultReporter with in-memory storage
func NewDefaultReporter() *DefaultReporter {
	r := &DefaultReporter{
		storage:    NewInMemoryStorage(),
		formatters: make(map[string]types.ReportFormatter),
	}
	
	// Register built-in formatters
	r.RegisterFormatter("json", NewJSONFormatter())
	r.RegisterFormatter("markdown", NewMarkdownFormatter())
	r.RegisterFormatter("html", NewHTMLFormatter())
	
	return r
}

// RecordEvent records an event
func (r *DefaultReporter) RecordEvent(ctx context.Context, event types.Event) error {
	key := fmt.Sprintf("events:%s", event.ID)
	return r.storage.Save(ctx, key, event)
}

// RecordExecution records an execution
func (r *DefaultReporter) RecordExecution(ctx context.Context, exec types.ExecutionRecord) error {
	key := fmt.Sprintf("executions:%s", exec.ID)
	return r.storage.Save(ctx, key, exec)
}

// QueryEvents queries events based on the given criteria
func (r *DefaultReporter) QueryEvents(ctx context.Context, query types.EventQuery) ([]types.Event, error) {
	// Build storage query
	filter := make(map[string]interface{})
	
	// Note: This is a simplified implementation. In a real system, you'd need
	// more sophisticated filtering logic for arrays and time ranges
	
	storageQuery := types.Query{
		Collection: "events",
		Filter:     filter,
		Limit:      query.Limit,
		Offset:     query.Offset,
		Sort: []types.SortField{
			{Field: "Timestamp", Descending: true},
		},
	}
	
	results, err := r.storage.Query(ctx, storageQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	
	// Convert results to events and apply additional filtering
	var events []types.Event
	for _, result := range results {
		event, ok := result.(types.Event)
		if !ok {
			continue
		}
		
		// Apply filters
		if !matchesEventQuery(event, query) {
			continue
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// QueryExecutions queries executions based on the given criteria
func (r *DefaultReporter) QueryExecutions(ctx context.Context, query types.ExecutionQuery) ([]types.ExecutionRecord, error) {
	// Build storage query
	filter := make(map[string]interface{})
	
	storageQuery := types.Query{
		Collection: "executions",
		Filter:     filter,
		Limit:      query.Limit,
		Offset:     query.Offset,
		Sort: []types.SortField{
			{Field: "StartTime", Descending: true},
		},
	}
	
	results, err := r.storage.Query(ctx, storageQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query executions: %w", err)
	}
	
	// Convert results to execution records and apply additional filtering
	var executions []types.ExecutionRecord
	for _, result := range results {
		exec, ok := result.(types.ExecutionRecord)
		if !ok {
			continue
		}
		
		// Apply filters
		if !matchesExecutionQuery(exec, query) {
			continue
		}
		
		executions = append(executions, exec)
	}
	
	return executions, nil
}

// ComputeStatistics computes statistics based on the given filter
func (r *DefaultReporter) ComputeStatistics(ctx context.Context, filter types.StatisticsFilter) (types.Statistics, error) {
	// Query all executions matching the filter
	query := types.ExecutionQuery{
		PluginNames: filter.PluginNames,
		ResourceIDs: filter.ResourceIDs,
		StartTime:   filter.StartTime,
		EndTime:     filter.EndTime,
	}
	
	executions, err := r.QueryExecutions(ctx, query)
	if err != nil {
		return types.Statistics{}, fmt.Errorf("failed to query executions: %w", err)
	}
	
	if len(executions) == 0 {
		return types.Statistics{}, nil
	}
	
	stats := types.Statistics{
		TotalExecutions: len(executions),
		ByPlugin:        make(map[string]types.PluginStatistics),
		ByResource:      make(map[string]types.ResourceStatistics),
	}
	
	// Collect durations and count statuses
	var durations []time.Duration
	var recoveryTimes []time.Duration
	var lastFailureTime time.Time
	
	for _, exec := range executions {
		durations = append(durations, exec.Duration)
		
		switch exec.Status {
		case types.StatusSuccess:
			stats.SuccessCount++
			// If there was a previous failure, calculate recovery time
			if !lastFailureTime.IsZero() {
				recoveryTime := exec.EndTime.Sub(lastFailureTime)
				recoveryTimes = append(recoveryTimes, recoveryTime)
				lastFailureTime = time.Time{} // Reset
			}
		case types.StatusFailed:
			stats.FailureCount++
			lastFailureTime = exec.EndTime
		case types.StatusTimeout:
			stats.TimeoutCount++
			lastFailureTime = exec.EndTime
		case types.StatusCanceled:
			stats.CanceledCount++
		}
		
		// Update plugin statistics
		pluginStats, ok := stats.ByPlugin[exec.PluginName]
		if !ok {
			pluginStats = types.PluginStatistics{
				PluginName: exec.PluginName,
			}
		}
		pluginStats.TotalExecutions++
		if exec.Status == types.StatusSuccess {
			pluginStats.SuccessCount++
		} else {
			pluginStats.FailureCount++
		}
		stats.ByPlugin[exec.PluginName] = pluginStats
		
		// Update resource statistics
		resourceKey := exec.ResourceID
		if resourceKey == "" {
			resourceKey = exec.ResourceName
		}
		resourceStats, ok := stats.ByResource[resourceKey]
		if !ok {
			resourceStats = types.ResourceStatistics{
				ResourceID:   exec.ResourceID,
				ResourceName: exec.ResourceName,
			}
		}
		resourceStats.TotalExecutions++
		if exec.Status == types.StatusSuccess {
			resourceStats.SuccessCount++
		} else {
			resourceStats.FailureCount++
		}
		stats.ByResource[resourceKey] = resourceStats
	}
	
	// Calculate success rate
	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalExecutions)
	}
	
	// Calculate MTTR (Mean Time To Recovery)
	if len(recoveryTimes) > 0 {
		var totalRecoveryTime time.Duration
		for _, rt := range recoveryTimes {
			totalRecoveryTime += rt
		}
		stats.MTTR = totalRecoveryTime / time.Duration(len(recoveryTimes))
	}
	
	// Calculate duration statistics
	if len(durations) > 0 {
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})
		
		var totalDuration time.Duration
		for _, d := range durations {
			totalDuration += d
		}
		stats.AverageDuration = totalDuration / time.Duration(len(durations))
		stats.MinDuration = durations[0]
		stats.MaxDuration = durations[len(durations)-1]
		
		// Calculate percentiles
		stats.P50Duration = durations[len(durations)*50/100]
		stats.P95Duration = durations[len(durations)*95/100]
		stats.P99Duration = durations[len(durations)*99/100]
	}
	
	// Calculate per-plugin statistics
	for pluginName, pluginStats := range stats.ByPlugin {
		if pluginStats.TotalExecutions > 0 {
			pluginStats.SuccessRate = float64(pluginStats.SuccessCount) / float64(pluginStats.TotalExecutions)
		}
		
		// Calculate average duration for this plugin
		var pluginDurations []time.Duration
		for _, exec := range executions {
			if exec.PluginName == pluginName {
				pluginDurations = append(pluginDurations, exec.Duration)
			}
		}
		if len(pluginDurations) > 0 {
			var total time.Duration
			for _, d := range pluginDurations {
				total += d
			}
			pluginStats.AverageDuration = total / time.Duration(len(pluginDurations))
		}
		
		stats.ByPlugin[pluginName] = pluginStats
	}
	
	// Calculate per-resource statistics
	for resourceKey, resourceStats := range stats.ByResource {
		if resourceStats.TotalExecutions > 0 {
			resourceStats.SuccessRate = float64(resourceStats.SuccessCount) / float64(resourceStats.TotalExecutions)
		}
		
		// Calculate average duration for this resource
		var resourceDurations []time.Duration
		for _, exec := range executions {
			execResourceKey := exec.ResourceID
			if execResourceKey == "" {
				execResourceKey = exec.ResourceName
			}
			if execResourceKey == resourceKey {
				resourceDurations = append(resourceDurations, exec.Duration)
			}
		}
		if len(resourceDurations) > 0 {
			var total time.Duration
			for _, d := range resourceDurations {
				total += d
			}
			resourceStats.AverageDuration = total / time.Duration(len(resourceDurations))
		}
		
		stats.ByResource[resourceKey] = resourceStats
	}
	
	return stats, nil
}

// GenerateReport generates a report in the specified format
func (r *DefaultReporter) GenerateReport(ctx context.Context, format string, filter types.ReportFilter) ([]byte, error) {
	r.mu.RLock()
	formatter, ok := r.formatters[format]
	r.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("formatter %s not found", format)
	}
	
	// Compute statistics based on the filter
	statsFilter := types.StatisticsFilter{
		PluginNames: filter.PluginNames,
		ResourceIDs: filter.ResourceIDs,
		StartTime:   filter.StartTime,
		EndTime:     filter.EndTime,
	}
	
	stats, err := r.ComputeStatistics(ctx, statsFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to compute statistics: %w", err)
	}
	
	// Format the statistics
	return formatter.Format(ctx, stats)
}

// SetStorage sets the storage backend
func (r *DefaultReporter) SetStorage(storage types.StorageBackend) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.storage = storage
	return nil
}

// RegisterFormatter registers a report formatter
func (r *DefaultReporter) RegisterFormatter(name string, formatter types.ReportFormatter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.formatters[name] = formatter
	return nil
}

// Helper functions

func matchesEventQuery(event types.Event, query types.EventQuery) bool {
	// Check types
	if len(query.Types) > 0 {
		found := false
		for _, t := range query.Types {
			if event.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check sources
	if len(query.Sources) > 0 {
		found := false
		for _, s := range query.Sources {
			if event.Source == s {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check resources
	if len(query.Resources) > 0 {
		found := false
		for _, r := range query.Resources {
			if event.Resource.ID == r || event.Resource.Name == r {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check time range
	if !query.StartTime.IsZero() && event.Timestamp.Before(query.StartTime) {
		return false
	}
	if !query.EndTime.IsZero() && event.Timestamp.After(query.EndTime) {
		return false
	}
	
	return true
}

func matchesExecutionQuery(exec types.ExecutionRecord, query types.ExecutionQuery) bool {
	// Check plugin names
	if len(query.PluginNames) > 0 {
		found := false
		for _, p := range query.PluginNames {
			if exec.PluginName == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check resource IDs
	if len(query.ResourceIDs) > 0 {
		found := false
		for _, r := range query.ResourceIDs {
			if exec.ResourceID == r {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check statuses
	if len(query.Statuses) > 0 {
		found := false
		for _, s := range query.Statuses {
			if exec.Status == s {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check time range
	if !query.StartTime.IsZero() && exec.StartTime.Before(query.StartTime) {
		return false
	}
	if !query.EndTime.IsZero() && exec.EndTime.After(query.EndTime) {
		return false
	}
	
	return true
}
