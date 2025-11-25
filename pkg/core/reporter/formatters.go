// Package reporter provides report formatting implementations for the Infrastructure Resilience Engine.
package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// ============================================================================
// JSONFormatter
// ============================================================================

// JSONFormatter formats reports as JSON
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSONFormatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format formats the data as JSON
func (f *JSONFormatter) Format(ctx context.Context, data interface{}) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

// ContentType returns the content type for JSON
func (f *JSONFormatter) ContentType() string {
	return "application/json"
}

// Name returns the formatter name
func (f *JSONFormatter) Name() string {
	return "json"
}

// ============================================================================
// MarkdownFormatter
// ============================================================================

// MarkdownFormatter formats reports as Markdown
type MarkdownFormatter struct{}

// NewMarkdownFormatter creates a new MarkdownFormatter
func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{}
}

// Format formats the data as Markdown
func (f *MarkdownFormatter) Format(ctx context.Context, data interface{}) ([]byte, error) {
	var sb strings.Builder

	switch v := data.(type) {
	case types.Statistics:
		f.formatStatistics(&sb, v)
	case []types.ExecutionRecord:
		f.formatExecutionRecords(&sb, v)
	case []types.Event:
		f.formatEvents(&sb, v)
	default:
		return nil, fmt.Errorf("unsupported data type for markdown formatting: %T", data)
	}

	return []byte(sb.String()), nil
}

// ContentType returns the content type for Markdown
func (f *MarkdownFormatter) ContentType() string {
	return "text/markdown"
}

// Name returns the formatter name
func (f *MarkdownFormatter) Name() string {
	return "markdown"
}

func (f *MarkdownFormatter) formatStatistics(sb *strings.Builder, stats types.Statistics) {
	sb.WriteString("# Execution Statistics\n\n")

	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Executions**: %d\n", stats.TotalExecutions))
	sb.WriteString(fmt.Sprintf("- **Success Count**: %d\n", stats.SuccessCount))
	sb.WriteString(fmt.Sprintf("- **Failure Count**: %d\n", stats.FailureCount))
	sb.WriteString(fmt.Sprintf("- **Timeout Count**: %d\n", stats.TimeoutCount))
	sb.WriteString(fmt.Sprintf("- **Canceled Count**: %d\n", stats.CanceledCount))
	sb.WriteString(fmt.Sprintf("- **Success Rate**: %.2f%%\n", stats.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("- **MTTR**: %s\n\n", stats.MTTR))

	sb.WriteString("## Duration Statistics\n\n")
	sb.WriteString(fmt.Sprintf("- **Average**: %s\n", stats.AverageDuration))
	sb.WriteString(fmt.Sprintf("- **P50**: %s\n", stats.P50Duration))
	sb.WriteString(fmt.Sprintf("- **P95**: %s\n", stats.P95Duration))
	sb.WriteString(fmt.Sprintf("- **P99**: %s\n", stats.P99Duration))
	sb.WriteString(fmt.Sprintf("- **Min**: %s\n", stats.MinDuration))
	sb.WriteString(fmt.Sprintf("- **Max**: %s\n\n", stats.MaxDuration))

	if len(stats.ByPlugin) > 0 {
		sb.WriteString("## By Plugin\n\n")
		sb.WriteString("| Plugin | Executions | Success | Failure | Success Rate | Avg Duration |\n")
		sb.WriteString("|--------|------------|---------|---------|--------------|-------------|\n")
		for _, ps := range stats.ByPlugin {
			sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %.2f%% | %s |\n",
				ps.PluginName, ps.TotalExecutions, ps.SuccessCount, ps.FailureCount,
				ps.SuccessRate*100, ps.AverageDuration))
		}
		sb.WriteString("\n")
	}

	if len(stats.ByResource) > 0 {
		sb.WriteString("## By Resource\n\n")
		sb.WriteString("| Resource | Executions | Success | Failure | Success Rate | Avg Duration |\n")
		sb.WriteString("|----------|------------|---------|---------|--------------|-------------|\n")
		for _, rs := range stats.ByResource {
			sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %.2f%% | %s |\n",
				rs.ResourceName, rs.TotalExecutions, rs.SuccessCount, rs.FailureCount,
				rs.SuccessRate*100, rs.AverageDuration))
		}
		sb.WriteString("\n")
	}
}

func (f *MarkdownFormatter) formatExecutionRecords(sb *strings.Builder, records []types.ExecutionRecord) {
	sb.WriteString("# Execution Records\n\n")
	sb.WriteString("| ID | Plugin | Resource | Status | Duration | Start Time | Error |\n")
	sb.WriteString("|----|--------|----------|--------|----------|------------|-------|\n")

	for _, rec := range records {
		errorMsg := ""
		if rec.Error != "" {
			errorMsg = rec.Error
			if len(errorMsg) > 50 {
				errorMsg = errorMsg[:47] + "..."
			}
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n",
			rec.ID[:8], rec.PluginName, rec.ResourceName, rec.Status,
			rec.Duration, rec.StartTime.Format(time.RFC3339), errorMsg))
	}
}

func (f *MarkdownFormatter) formatEvents(sb *strings.Builder, events []types.Event) {
	sb.WriteString("# Events\n\n")
	sb.WriteString("| ID | Type | Source | Timestamp | Resource |\n")
	sb.WriteString("|----|------|--------|-----------|----------|\n")

	for _, evt := range events {
		resourceName := ""
		if evt.Resource.Name != "" {
			resourceName = evt.Resource.Name
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			evt.ID[:8], evt.Type, evt.Source, evt.Timestamp.Format(time.RFC3339), resourceName))
	}
}

// ============================================================================
// HTMLFormatter
// ============================================================================

// HTMLFormatter formats reports as HTML
type HTMLFormatter struct{}

// NewHTMLFormatter creates a new HTMLFormatter
func NewHTMLFormatter() *HTMLFormatter {
	return &HTMLFormatter{}
}

// Format formats the data as HTML
func (f *HTMLFormatter) Format(ctx context.Context, data interface{}) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html>\n<head>\n")
	sb.WriteString("<title>Infrastructure Resilience Report</title>\n")
	sb.WriteString("<style>\n")
	sb.WriteString("body { font-family: Arial, sans-serif; margin: 20px; }\n")
	sb.WriteString("table { border-collapse: collapse; width: 100%; margin: 20px 0; }\n")
	sb.WriteString("th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }\n")
	sb.WriteString("th { background-color: #4CAF50; color: white; }\n")
	sb.WriteString("tr:nth-child(even) { background-color: #f2f2f2; }\n")
	sb.WriteString(".summary { background-color: #f9f9f9; padding: 15px; border-radius: 5px; }\n")
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")

	switch v := data.(type) {
	case types.Statistics:
		f.formatStatistics(&sb, v)
	case []types.ExecutionRecord:
		f.formatExecutionRecords(&sb, v)
	case []types.Event:
		f.formatEvents(&sb, v)
	default:
		return nil, fmt.Errorf("unsupported data type for HTML formatting: %T", data)
	}

	sb.WriteString("</body>\n</html>")
	return []byte(sb.String()), nil
}

// ContentType returns the content type for HTML
func (f *HTMLFormatter) ContentType() string {
	return "text/html"
}

// Name returns the formatter name
func (f *HTMLFormatter) Name() string {
	return "html"
}

func (f *HTMLFormatter) formatStatistics(sb *strings.Builder, stats types.Statistics) {
	sb.WriteString("<h1>Execution Statistics</h1>\n")

	sb.WriteString("<div class='summary'>\n")
	sb.WriteString("<h2>Summary</h2>\n")
	sb.WriteString(fmt.Sprintf("<p><strong>Total Executions:</strong> %d</p>\n", stats.TotalExecutions))
	sb.WriteString(fmt.Sprintf("<p><strong>Success Count:</strong> %d</p>\n", stats.SuccessCount))
	sb.WriteString(fmt.Sprintf("<p><strong>Failure Count:</strong> %d</p>\n", stats.FailureCount))
	sb.WriteString(fmt.Sprintf("<p><strong>Timeout Count:</strong> %d</p>\n", stats.TimeoutCount))
	sb.WriteString(fmt.Sprintf("<p><strong>Canceled Count:</strong> %d</p>\n", stats.CanceledCount))
	sb.WriteString(fmt.Sprintf("<p><strong>Success Rate:</strong> %.2f%%</p>\n", stats.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("<p><strong>MTTR:</strong> %s</p>\n", stats.MTTR))
	sb.WriteString("</div>\n")

	sb.WriteString("<h2>Duration Statistics</h2>\n")
	sb.WriteString("<table>\n")
	sb.WriteString("<tr><th>Metric</th><th>Value</th></tr>\n")
	sb.WriteString(fmt.Sprintf("<tr><td>Average</td><td>%s</td></tr>\n", stats.AverageDuration))
	sb.WriteString(fmt.Sprintf("<tr><td>P50</td><td>%s</td></tr>\n", stats.P50Duration))
	sb.WriteString(fmt.Sprintf("<tr><td>P95</td><td>%s</td></tr>\n", stats.P95Duration))
	sb.WriteString(fmt.Sprintf("<tr><td>P99</td><td>%s</td></tr>\n", stats.P99Duration))
	sb.WriteString(fmt.Sprintf("<tr><td>Min</td><td>%s</td></tr>\n", stats.MinDuration))
	sb.WriteString(fmt.Sprintf("<tr><td>Max</td><td>%s</td></tr>\n", stats.MaxDuration))
	sb.WriteString("</table>\n")

	if len(stats.ByPlugin) > 0 {
		sb.WriteString("<h2>By Plugin</h2>\n")
		sb.WriteString("<table>\n")
		sb.WriteString("<tr><th>Plugin</th><th>Executions</th><th>Success</th><th>Failure</th><th>Success Rate</th><th>Avg Duration</th></tr>\n")
		for _, ps := range stats.ByPlugin {
			sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%.2f%%</td><td>%s</td></tr>\n",
				ps.PluginName, ps.TotalExecutions, ps.SuccessCount, ps.FailureCount,
				ps.SuccessRate*100, ps.AverageDuration))
		}
		sb.WriteString("</table>\n")
	}

	if len(stats.ByResource) > 0 {
		sb.WriteString("<h2>By Resource</h2>\n")
		sb.WriteString("<table>\n")
		sb.WriteString("<tr><th>Resource</th><th>Executions</th><th>Success</th><th>Failure</th><th>Success Rate</th><th>Avg Duration</th></tr>\n")
		for _, rs := range stats.ByResource {
			sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%.2f%%</td><td>%s</td></tr>\n",
				rs.ResourceName, rs.TotalExecutions, rs.SuccessCount, rs.FailureCount,
				rs.SuccessRate*100, rs.AverageDuration))
		}
		sb.WriteString("</table>\n")
	}
}

func (f *HTMLFormatter) formatExecutionRecords(sb *strings.Builder, records []types.ExecutionRecord) {
	sb.WriteString("<h1>Execution Records</h1>\n")
	sb.WriteString("<table>\n")
	sb.WriteString("<tr><th>ID</th><th>Plugin</th><th>Resource</th><th>Status</th><th>Duration</th><th>Start Time</th><th>Error</th></tr>\n")

	for _, rec := range records {
		errorMsg := ""
		if rec.Error != "" {
			errorMsg = rec.Error
		}
		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
			rec.ID[:8], rec.PluginName, rec.ResourceName, rec.Status,
			rec.Duration, rec.StartTime.Format(time.RFC3339), errorMsg))
	}
	sb.WriteString("</table>\n")
}

func (f *HTMLFormatter) formatEvents(sb *strings.Builder, events []types.Event) {
	sb.WriteString("<h1>Events</h1>\n")
	sb.WriteString("<table>\n")
	sb.WriteString("<tr><th>ID</th><th>Type</th><th>Source</th><th>Timestamp</th><th>Resource</th></tr>\n")

	for _, evt := range events {
		resourceName := ""
		if evt.Resource.Name != "" {
			resourceName = evt.Resource.Name
		}
		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
			evt.ID[:8], evt.Type, evt.Source, evt.Timestamp.Format(time.RFC3339), resourceName))
	}
	sb.WriteString("</table>\n")
}
