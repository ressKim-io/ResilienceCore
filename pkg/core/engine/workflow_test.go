package engine

import (
	"context"
	"errors"
	"testing"

	coretesting "github.com/yourusername/infrastructure-resilience-engine/pkg/core/testing"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

func TestWorkflowAbortStopsExecution(t *testing.T) {
	engine := NewDefaultExecutionEngine()
	engine.SetComponents(nil, nil, nil, nil, nil, nil, nil)

	plugin1 := coretesting.NewMockPlugin("plugin1")
	plugin1.ExecuteError = errors.New("step failed")
	plugin2 := coretesting.NewMockPlugin("plugin2")

	engine.RegisterPlugin(plugin1)
	engine.RegisterPlugin(plugin2)

	resource := types.Resource{
		ID:   "test-resource",
		Name: "test-resource",
		Kind: "container",
	}

	// Create error handler that aborts
	errorHandler := &SimpleErrorHandler{Action: types.ErrorActionAbort}

	// Create workflow with error handler
	workflow := types.Workflow{
		Name: "error-workflow",
		Steps: []types.WorkflowStep{
			{
				Name:       "step1",
				PluginName: "plugin1",
				Resource:   resource,
				OnError:    errorHandler,
			},
			{
				Name:       "step2",
				PluginName: "plugin2",
				Resource:   resource,
			},
		},
	}

	ctx := context.Background()
	result, _ := engine.ExecuteWorkflow(ctx, workflow)

	t.Logf("Workflow status: %v", result.Status)
	t.Logf("Step results: %+v", result.StepResults)
	t.Logf("Plugin1 execute calls: %d", plugin1.ExecuteCalls)
	t.Logf("Plugin2 execute calls: %d", plugin2.ExecuteCalls)

	// Verify workflow failed
	if result.Status != types.StatusFailed {
		t.Errorf("Expected workflow to fail, got status: %v", result.Status)
	}

	// Verify step1 failed
	step1Result, step1Executed := result.StepResults["step1"]
	if !step1Executed {
		t.Error("Step1 should have executed")
	}
	if step1Result.Status != types.StatusFailed {
		t.Errorf("Step1 should have failed, got status: %v", step1Result.Status)
	}

	// Verify step2 was not executed or was skipped
	step2Result, step2Executed := result.StepResults["step2"]
	if step2Executed && step2Result.Status != types.StatusSkipped {
		t.Errorf("Step2 should not have executed or should be skipped, got status: %v", step2Result.Status)
	}

	// Verify plugin2 was not called
	if plugin2.ExecuteCalls > 0 {
		t.Errorf("Plugin2 should not have been called, but was called %d times", plugin2.ExecuteCalls)
	}
}
