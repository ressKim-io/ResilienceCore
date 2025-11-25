package reporter

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

func TestDefaultReporter_RecordEvent(t *testing.T) {
	reporter := NewDefaultReporter()
	ctx := context.Background()

	event := types.Event{
		ID:        uuid.New().String(),
		Type:      "test.event",
		Source:    "test",
		Timestamp: time.Now(),
		Resource: types.Resource{
			ID:   "resource-1",
			Name: "test-resource",
		},
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	err := reporter.RecordEvent(ctx, event)
	if err != nil {
		t.Fatalf("RecordEvent failed: %v", err)
	}

	// Query the event back
	query := types.EventQuery{
		Types: []string{"test.event"},
	}
	events, err := reporter.QueryEvents(ctx, query)
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if events[0].ID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, events[0].ID)
	}
}

func TestDefaultReporter_RecordExecution(t *testing.T) {
	reporter := NewDefaultReporter()
	ctx := context.Background()

	exec := types.ExecutionRecord{
		ID:           uuid.New().String(),
		PluginName:   "test-plugin",
		ResourceID:   "resource-1",
		ResourceName: "test-resource",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(time.Second),
		Duration:     time.Second,
		Status:       types.StatusSuccess,
		Principal:    "test-user",
	}

	err := reporter.RecordExecution(ctx, exec)
	if err != nil {
		t.Fatalf("RecordExecution failed: %v", err)
	}

	// Query the execution back
	query := types.ExecutionQuery{
		PluginNames: []string{"test-plugin"},
	}
	executions, err := reporter.QueryExecutions(ctx, query)
	if err != nil {
		t.Fatalf("QueryExecutions failed: %v", err)
	}

	if len(executions) != 1 {
		t.Fatalf("Expected 1 execution, got %d", len(executions))
	}

	if executions[0].ID != exec.ID {
		t.Errorf("Expected execution ID %s, got %s", exec.ID, executions[0].ID)
	}
}

func TestDefaultReporter_QueryEvents(t *testing.T) {
	reporter := NewDefaultReporter()
	ctx := context.Background()

	// Record multiple events
	events := []types.Event{
		{
			ID:        uuid.New().String(),
			Type:      "type1",
			Source:    "source1",
			Timestamp: time.Now(),
			Resource:  types.Resource{ID: "res1"},
		},
		{
			ID:        uuid.New().String(),
			Type:      "type2",
			Source:    "source1",
			Timestamp: time.Now(),
			Resource:  types.Resource{ID: "res2"},
		},
		{
			ID:        uuid.New().String(),
			Type:      "type1",
			Source:    "source2",
			Timestamp: time.Now(),
			Resource:  types.Resource{ID: "res1"},
		},
	}

	for _, event := range events {
		if err := reporter.RecordEvent(ctx, event); err != nil {
			t.Fatalf("RecordEvent failed: %v", err)
		}
	}

	tests := []struct {
		name     string
		query    types.EventQuery
		expected int
	}{
		{
			name:     "query by type",
			query:    types.EventQuery{Types: []string{"type1"}},
			expected: 2,
		},
		{
			name:     "query by source",
			query:    types.EventQuery{Sources: []string{"source1"}},
			expected: 2,
		},
		{
			name:     "query by resource",
			query:    types.EventQuery{Resources: []string{"res1"}},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := reporter.QueryEvents(ctx, tt.query)
			if err != nil {
				t.Fatalf("QueryEvents failed: %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Expected %d events, got %d", tt.expected, len(results))
			}
		})
	}
}

func TestDefaultReporter_QueryExecutions(t *testing.T) {
	reporter := NewDefaultReporter()
	ctx := context.Background()

	// Record multiple executions
	executions := []types.ExecutionRecord{
		{
			ID:         uuid.New().String(),
			PluginName: "plugin1",
			ResourceID: "res1",
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(time.Second),
			Duration:   time.Second,
			Status:     types.StatusSuccess,
		},
		{
			ID:         uuid.New().String(),
			PluginName: "plugin2",
			ResourceID: "res1",
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(time.Second),
			Duration:   time.Second,
			Status:     types.StatusFailed,
		},
		{
			ID:         uuid.New().String(),
			PluginName: "plugin1",
			ResourceID: "res2",
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(time.Second),
			Duration:   time.Second,
			Status:     types.StatusSuccess,
		},
	}

	for _, exec := range executions {
		if err := reporter.RecordExecution(ctx, exec); err != nil {
			t.Fatalf("RecordExecution failed: %v", err)
		}
	}

	tests := []struct {
		name     string
		query    types.ExecutionQuery
		expected int
	}{
		{
			name:     "query by plugin",
			query:    types.ExecutionQuery{PluginNames: []string{"plugin1"}},
			expected: 2,
		},
		{
			name:     "query by resource",
			query:    types.ExecutionQuery{ResourceIDs: []string{"res1"}},
			expected: 2,
		},
		{
			name:     "query by status",
			query:    types.ExecutionQuery{Statuses: []types.ExecutionStatus{types.StatusSuccess}},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := reporter.QueryExecutions(ctx, tt.query)
			if err != nil {
				t.Fatalf("QueryExecutions failed: %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Expected %d executions, got %d", tt.expected, len(results))
			}
		})
	}
}

func TestDefaultReporter_ComputeStatistics(t *testing.T) {
	reporter := NewDefaultReporter()
	ctx := context.Background()

	// Record executions with different statuses and durations
	executions := []types.ExecutionRecord{
		{
			ID:         uuid.New().String(),
			PluginName: "plugin1",
			ResourceID: "res1",
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(100 * time.Millisecond),
			Duration:   100 * time.Millisecond,
			Status:     types.StatusSuccess,
		},
		{
			ID:         uuid.New().String(),
			PluginName: "plugin1",
			ResourceID: "res1",
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(200 * time.Millisecond),
			Duration:   200 * time.Millisecond,
			Status:     types.StatusSuccess,
		},
		{
			ID:         uuid.New().String(),
			PluginName: "plugin1",
			ResourceID: "res1",
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(150 * time.Millisecond),
			Duration:   150 * time.Millisecond,
			Status:     types.StatusFailed,
		},
		{
			ID:         uuid.New().String(),
			PluginName: "plugin2",
			ResourceID: "res2",
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(300 * time.Millisecond),
			Duration:   300 * time.Millisecond,
			Status:     types.StatusTimeout,
		},
	}

	for _, exec := range executions {
		if err := reporter.RecordExecution(ctx, exec); err != nil {
			t.Fatalf("RecordExecution failed: %v", err)
		}
	}

	// Compute statistics
	filter := types.StatisticsFilter{}
	stats, err := reporter.ComputeStatistics(ctx, filter)
	if err != nil {
		t.Fatalf("ComputeStatistics failed: %v", err)
	}

	// Verify total executions
	if stats.TotalExecutions != 4 {
		t.Errorf("Expected 4 total executions, got %d", stats.TotalExecutions)
	}

	// Verify success count
	if stats.SuccessCount != 2 {
		t.Errorf("Expected 2 successes, got %d", stats.SuccessCount)
	}

	// Verify failure count
	if stats.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", stats.FailureCount)
	}

	// Verify timeout count
	if stats.TimeoutCount != 1 {
		t.Errorf("Expected 1 timeout, got %d", stats.TimeoutCount)
	}

	// Verify success rate
	expectedRate := 0.5
	if stats.SuccessRate != expectedRate {
		t.Errorf("Expected success rate %.2f, got %.2f", expectedRate, stats.SuccessRate)
	}

	// Verify plugin statistics
	if len(stats.ByPlugin) != 2 {
		t.Errorf("Expected 2 plugins in statistics, got %d", len(stats.ByPlugin))
	}

	plugin1Stats, ok := stats.ByPlugin["plugin1"]
	if !ok {
		t.Fatal("Expected plugin1 in statistics")
	}
	if plugin1Stats.TotalExecutions != 3 {
		t.Errorf("Expected 3 executions for plugin1, got %d", plugin1Stats.TotalExecutions)
	}

	// Verify resource statistics
	if len(stats.ByResource) != 2 {
		t.Errorf("Expected 2 resources in statistics, got %d", len(stats.ByResource))
	}
}

func TestDefaultReporter_GenerateReport(t *testing.T) {
	reporter := NewDefaultReporter()
	ctx := context.Background()

	// Record some executions
	exec := types.ExecutionRecord{
		ID:         uuid.New().String(),
		PluginName: "test-plugin",
		ResourceID: "res1",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Second),
		Duration:   time.Second,
		Status:     types.StatusSuccess,
	}

	if err := reporter.RecordExecution(ctx, exec); err != nil {
		t.Fatalf("RecordExecution failed: %v", err)
	}

	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{
			name:    "json format",
			format:  "json",
			wantErr: false,
		},
		{
			name:    "markdown format",
			format:  "markdown",
			wantErr: false,
		},
		{
			name:    "html format",
			format:  "html",
			wantErr: false,
		},
		{
			name:    "unknown format",
			format:  "unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := types.ReportFilter{}
			report, err := reporter.GenerateReport(ctx, tt.format, filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(report) == 0 {
				t.Error("Expected non-empty report")
			}
		})
	}
}

func TestDefaultReporter_SetStorage(t *testing.T) {
	reporter := NewDefaultReporter()
	newStorage := NewInMemoryStorage()

	err := reporter.SetStorage(newStorage)
	if err != nil {
		t.Fatalf("SetStorage failed: %v", err)
	}

	// Verify the storage was changed by recording and querying
	ctx := context.Background()
	exec := types.ExecutionRecord{
		ID:         uuid.New().String(),
		PluginName: "test",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Second),
		Duration:   time.Second,
		Status:     types.StatusSuccess,
	}

	if err := reporter.RecordExecution(ctx, exec); err != nil {
		t.Fatalf("RecordExecution failed: %v", err)
	}

	query := types.ExecutionQuery{PluginNames: []string{"test"}}
	results, err := reporter.QueryExecutions(ctx, query)
	if err != nil {
		t.Fatalf("QueryExecutions failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 execution, got %d", len(results))
	}
}

func TestDefaultReporter_RegisterFormatter(t *testing.T) {
	reporter := NewDefaultReporter()

	// Register a custom formatter
	customFormatter := NewJSONFormatter()
	err := reporter.RegisterFormatter("custom", customFormatter)
	if err != nil {
		t.Fatalf("RegisterFormatter failed: %v", err)
	}

	// Verify the formatter was registered by generating a report
	ctx := context.Background()
	exec := types.ExecutionRecord{
		ID:         uuid.New().String(),
		PluginName: "test",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Second),
		Duration:   time.Second,
		Status:     types.StatusSuccess,
	}

	if err := reporter.RecordExecution(ctx, exec); err != nil {
		t.Fatalf("RecordExecution failed: %v", err)
	}

	filter := types.ReportFilter{}
	report, err := reporter.GenerateReport(ctx, "custom", filter)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	if len(report) == 0 {
		t.Error("Expected non-empty report")
	}
}
