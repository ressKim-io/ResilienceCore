package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	testutil "github.com/yourusername/infrastructure-resilience-engine/pkg/core/testing"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

func TestDefaultMonitor_CheckHealth(t *testing.T) {
	tests := []struct {
		name           string
		resource       types.Resource
		setupAdapter   func(*testutil.MockEnvironmentAdapter)
		setupMonitor   func(*DefaultMonitor)
		wantStatus     types.HealthStatusType
		wantErr        bool
		wantErrMessage string
	}{
		{
			name: "HTTP health check with unreachable endpoint",
			resource: types.Resource{
				ID:   "resource-1",
				Name: "test-resource",
				Spec: types.ResourceSpec{
					HealthCheck: &types.HealthCheckConfig{
						Type:     "http",
						Endpoint: "http://localhost:8080/health",
						Timeout:  5 * time.Second,
					},
				},
			},
			setupMonitor: func(m *DefaultMonitor) {
				// HTTP strategy is registered by default
			},
			wantStatus: types.HealthStatusUnhealthy,
			wantErr:    true,
		},
		{
			name: "no health check configuration",
			resource: types.Resource{
				ID:   "resource-2",
				Name: "test-resource-2",
				Spec: types.ResourceSpec{
					HealthCheck: nil,
				},
			},
			wantStatus:     types.HealthStatusUnknown,
			wantErr:        true,
			wantErrMessage: "no health check configuration",
		},
		{
			name: "health check type not specified",
			resource: types.Resource{
				ID:   "resource-3",
				Name: "test-resource-3",
				Spec: types.ResourceSpec{
					HealthCheck: &types.HealthCheckConfig{
						Type:     "",
						Endpoint: "http://localhost:8080/health",
					},
				},
			},
			wantStatus:     types.HealthStatusUnknown,
			wantErr:        true,
			wantErrMessage: "health check type not specified",
		},
		{
			name: "strategy not found",
			resource: types.Resource{
				ID:   "resource-4",
				Name: "test-resource-4",
				Spec: types.ResourceSpec{
					HealthCheck: &types.HealthCheckConfig{
						Type:     "unknown",
						Endpoint: "http://localhost:8080/health",
					},
				},
			},
			wantStatus:     types.HealthStatusUnknown,
			wantErr:        true,
			wantErrMessage: "strategy not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := testutil.NewMockEnvironmentAdapter()
			if tt.setupAdapter != nil {
				tt.setupAdapter(adapter)
			}

			monitor := NewDefaultMonitor(adapter)
			if tt.setupMonitor != nil {
				tt.setupMonitor(monitor)
			}

			ctx := context.Background()
			status, err := monitor.CheckHealth(ctx, tt.resource)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckHealth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantErrMessage != "" {
				if err == nil || err.Error() == "" {
					t.Errorf("CheckHealth() expected error message containing %q, got nil", tt.wantErrMessage)
				}
			}

			if status.Status != tt.wantStatus {
				t.Errorf("CheckHealth() status = %v, want %v", status.Status, tt.wantStatus)
			}
		})
	}
}

func TestDefaultMonitor_WaitForCondition(t *testing.T) {
	tests := []struct {
		name           string
		resource       types.Resource
		condition      types.Condition
		timeout        time.Duration
		setupAdapter   func(*testutil.MockEnvironmentAdapter)
		wantErr        bool
		wantErrMessage string
	}{
		{
			name: "condition met immediately",
			resource: types.Resource{
				ID:   "resource-1",
				Name: "test-resource",
				Status: types.ResourceStatus{
					Conditions: []types.Condition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
				},
			},
			condition: types.Condition{
				Type:   "Ready",
				Status: "True",
			},
			timeout: 5 * time.Second,
			setupAdapter: func(adapter *testutil.MockEnvironmentAdapter) {
				adapter.AddResource(types.Resource{
					ID:   "resource-1",
					Name: "test-resource",
					Status: types.ResourceStatus{
						Conditions: []types.Condition{
							{
								Type:   "Ready",
								Status: "True",
							},
						},
					},
				})
			},
			wantErr: false,
		},
		{
			name: "timeout waiting for condition",
			resource: types.Resource{
				ID:   "resource-2",
				Name: "test-resource-2",
				Status: types.ResourceStatus{
					Conditions: []types.Condition{
						{
							Type:   "Ready",
							Status: "False",
						},
					},
				},
			},
			condition: types.Condition{
				Type:   "Ready",
				Status: "True",
			},
			timeout: 100 * time.Millisecond,
			setupAdapter: func(adapter *testutil.MockEnvironmentAdapter) {
				adapter.AddResource(types.Resource{
					ID:   "resource-2",
					Name: "test-resource-2",
					Status: types.ResourceStatus{
						Conditions: []types.Condition{
							{
								Type:   "Ready",
								Status: "False",
							},
						},
					},
				})
			},
			wantErr:        true,
			wantErrMessage: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := testutil.NewMockEnvironmentAdapter()
			if tt.setupAdapter != nil {
				tt.setupAdapter(adapter)
			}

			monitor := NewDefaultMonitor(adapter)
			monitor.SetPollInterval(10 * time.Millisecond)

			ctx := context.Background()
			err := monitor.WaitForCondition(ctx, tt.resource, tt.condition, tt.timeout)

			if (err != nil) != tt.wantErr {
				t.Errorf("WaitForCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantErrMessage != "" {
				if err == nil || err.Error() == "" {
					t.Errorf("WaitForCondition() expected error message containing %q, got nil", tt.wantErrMessage)
				}
			}
		})
	}
}

func TestDefaultMonitor_WaitForHealthy(t *testing.T) {
	tests := []struct {
		name           string
		resource       types.Resource
		timeout        time.Duration
		setupAdapter   func(*testutil.MockEnvironmentAdapter)
		wantErr        bool
		wantErrMessage string
	}{
		{
			name: "timeout waiting for healthy",
			resource: types.Resource{
				ID:   "resource-1",
				Name: "test-resource",
				Spec: types.ResourceSpec{
					HealthCheck: &types.HealthCheckConfig{
						Type:     "http",
						Endpoint: "http://localhost:8080/health",
					},
				},
			},
			timeout: 100 * time.Millisecond,
			setupAdapter: func(adapter *testutil.MockEnvironmentAdapter) {
				adapter.AddResource(types.Resource{
					ID:   "resource-1",
					Name: "test-resource",
					Spec: types.ResourceSpec{
						HealthCheck: &types.HealthCheckConfig{
							Type:     "http",
							Endpoint: "http://localhost:8080/health",
						},
					},
				})
			},
			wantErr:        true,
			wantErrMessage: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := testutil.NewMockEnvironmentAdapter()
			if tt.setupAdapter != nil {
				tt.setupAdapter(adapter)
			}

			monitor := NewDefaultMonitor(adapter)
			monitor.SetPollInterval(10 * time.Millisecond)

			ctx := context.Background()
			err := monitor.WaitForHealthy(ctx, tt.resource, tt.timeout)

			if (err != nil) != tt.wantErr {
				t.Errorf("WaitForHealthy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantErrMessage != "" {
				if err == nil || err.Error() == "" {
					t.Errorf("WaitForHealthy() expected error message containing %q, got nil", tt.wantErrMessage)
				}
			}
		})
	}
}

func TestDefaultMonitor_CollectMetrics(t *testing.T) {
	tests := []struct {
		name         string
		resource     types.Resource
		setupAdapter func(*testutil.MockEnvironmentAdapter)
		wantErr      bool
	}{
		{
			name: "successful metrics collection",
			resource: types.Resource{
				ID:   "resource-1",
				Name: "test-resource",
			},
			setupAdapter: func(adapter *testutil.MockEnvironmentAdapter) {
				adapter.AddResource(types.Resource{
					ID:   "resource-1",
					Name: "test-resource",
				})
				adapter.SetMetrics("resource-1", types.Metrics{
					ResourceID: "resource-1",
					CPU: types.CPUMetrics{
						UsagePercent: 75.0,
					},
				})
			},
			wantErr: false,
		},
		{
			name: "resource not found",
			resource: types.Resource{
				ID:   "resource-2",
				Name: "test-resource-2",
			},
			setupAdapter: func(adapter *testutil.MockEnvironmentAdapter) {
				adapter.GetMetricsError = fmt.Errorf("resource not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := testutil.NewMockEnvironmentAdapter()
			if tt.setupAdapter != nil {
				tt.setupAdapter(adapter)
			}

			monitor := NewDefaultMonitor(adapter)

			ctx := context.Background()
			metrics, err := monitor.CollectMetrics(ctx, tt.resource)

			if (err != nil) != tt.wantErr {
				t.Errorf("CollectMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && metrics.ResourceID != tt.resource.ID {
				t.Errorf("CollectMetrics() resourceID = %v, want %v", metrics.ResourceID, tt.resource.ID)
			}
		})
	}
}

func TestDefaultMonitor_RegisterHealthCheckStrategy(t *testing.T) {
	tests := []struct {
		name         string
		strategyName string
		strategy     types.HealthCheckStrategy
		wantErr      bool
	}{
		{
			name:         "successful registration",
			strategyName: "custom",
			strategy:     &types.HTTPHealthCheckStrategy{},
			wantErr:      false,
		},
		{
			name:         "empty strategy name",
			strategyName: "",
			strategy:     &types.HTTPHealthCheckStrategy{},
			wantErr:      true,
		},
		{
			name:         "nil strategy",
			strategyName: "custom",
			strategy:     nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := testutil.NewMockEnvironmentAdapter()
			monitor := NewDefaultMonitor(adapter)

			err := monitor.RegisterHealthCheckStrategy(tt.strategyName, tt.strategy)

			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterHealthCheckStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify strategy was registered
				strategy, err := monitor.GetHealthCheckStrategy(tt.strategyName)
				if err != nil {
					t.Errorf("GetHealthCheckStrategy() error = %v", err)
				}
				if strategy != tt.strategy {
					t.Errorf("GetHealthCheckStrategy() returned different strategy")
				}
			}
		})
	}
}

func TestDefaultMonitor_GetHealthCheckStrategy(t *testing.T) {
	tests := []struct {
		name         string
		strategyName string
		setup        func(*DefaultMonitor)
		wantErr      bool
	}{
		{
			name:         "get existing strategy",
			strategyName: "http",
			setup:        func(m *DefaultMonitor) {},
			wantErr:      false,
		},
		{
			name:         "strategy not found",
			strategyName: "nonexistent",
			setup:        func(m *DefaultMonitor) {},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := testutil.NewMockEnvironmentAdapter()
			monitor := NewDefaultMonitor(adapter)

			if tt.setup != nil {
				tt.setup(monitor)
			}

			strategy, err := monitor.GetHealthCheckStrategy(tt.strategyName)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetHealthCheckStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && strategy == nil {
				t.Errorf("GetHealthCheckStrategy() returned nil strategy")
			}
		})
	}
}
