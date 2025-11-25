package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// DefaultExecutionEngine implements the ExecutionEngine interface
type DefaultExecutionEngine struct {
	// Plugin registry
	plugins map[string]types.Plugin
	mu      sync.RWMutex

	// Execution tracking
	executions   map[string]*executionState
	executionsMu sync.RWMutex

	// Resource locking to prevent concurrent execution on same resource
	resourceLocks   map[string]*sync.Mutex
	resourceLocksMu sync.Mutex

	// Configuration
	concurrencyLimit int
	defaultTimeout   time.Duration
	semaphore        chan struct{}

	// Components
	monitor  types.Monitor
	reporter types.Reporter
	eventBus types.EventBus
	logger   types.Logger
	tracer   types.Tracer
	auth     types.AuthorizationProvider
	secrets  types.SecretProvider
}

// executionState tracks the state of an ongoing execution
type executionState struct {
	id       string
	status   types.ExecutionStatus
	cancel   context.CancelFunc
	result   types.ExecutionResult
	resultMu sync.RWMutex
}

// NewDefaultExecutionEngine creates a new execution engine
func NewDefaultExecutionEngine() *DefaultExecutionEngine {
	return &DefaultExecutionEngine{
		plugins:          make(map[string]types.Plugin),
		executions:       make(map[string]*executionState),
		resourceLocks:    make(map[string]*sync.Mutex),
		concurrencyLimit: 100, // Default concurrency limit
		defaultTimeout:   5 * time.Minute,
		semaphore:        make(chan struct{}, 100),
	}
}

// SetComponents sets the core components for the engine
func (e *DefaultExecutionEngine) SetComponents(
	monitor types.Monitor,
	reporter types.Reporter,
	eventBus types.EventBus,
	logger types.Logger,
	tracer types.Tracer,
	auth types.AuthorizationProvider,
	secrets types.SecretProvider,
) {
	e.monitor = monitor
	e.reporter = reporter
	e.eventBus = eventBus
	e.logger = logger
	e.tracer = tracer
	e.auth = auth
	e.secrets = secrets
}

// RegisterPlugin registers a plugin with the engine
func (e *DefaultExecutionEngine) RegisterPlugin(plugin types.Plugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}

	metadata := plugin.Metadata()
	if metadata.Name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.plugins[metadata.Name]; exists {
		return fmt.Errorf("plugin %s already registered", metadata.Name)
	}

	e.plugins[metadata.Name] = plugin
	return nil
}

// UnregisterPlugin removes a plugin from the engine
func (e *DefaultExecutionEngine) UnregisterPlugin(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	delete(e.plugins, name)
	return nil
}

// GetPlugin retrieves a plugin by name
func (e *DefaultExecutionEngine) GetPlugin(name string) (types.Plugin, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	plugin, exists := e.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// ListPlugins returns a list of registered plugins matching the filter
func (e *DefaultExecutionEngine) ListPlugins(filter types.PluginFilter) ([]types.PluginMetadata, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []types.PluginMetadata

	for _, plugin := range e.plugins {
		metadata := plugin.Metadata()

		// Apply filters
		if !matchesFilter(metadata, filter) {
			continue
		}

		result = append(result, metadata)
	}

	return result, nil
}

// matchesFilter checks if metadata matches the filter criteria
func matchesFilter(metadata types.PluginMetadata, filter types.PluginFilter) bool {
	// Filter by names
	if len(filter.Names) > 0 {
		found := false
		for _, name := range filter.Names {
			if metadata.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by versions
	if len(filter.Versions) > 0 {
		found := false
		for _, version := range filter.Versions {
			if metadata.Version == version {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by supported kinds
	if len(filter.SupportedKinds) > 0 {
		found := false
		for _, filterKind := range filter.SupportedKinds {
			for _, metaKind := range metadata.SupportedKinds {
				if metaKind == filterKind {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by capabilities
	if len(filter.Capabilities) > 0 {
		for _, filterCap := range filter.Capabilities {
			found := false
			for _, metaCap := range metadata.RequiredCapabilities {
				if metaCap == filterCap {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

// Execute executes a plugin synchronously
func (e *DefaultExecutionEngine) Execute(ctx context.Context, request types.ExecutionRequest) (types.ExecutionResult, error) {
	// Get plugin
	plugin, err := e.GetPlugin(request.PluginName)
	if err != nil {
		return types.ExecutionResult{
			Status: types.StatusFailed,
			Error:  err,
		}, fmt.Errorf("failed to get plugin: %w", err)
	}

	// Check authorization if auth provider is set
	if e.auth != nil && request.Principal != "" {
		action := types.Action{
			Verb:     "execute",
			Resource: request.Resource.Kind,
		}
		if err := e.auth.Authorize(ctx, request.Principal, action, request.Resource); err != nil {
			return types.ExecutionResult{
				Status: types.StatusFailed,
				Error:  err,
			}, fmt.Errorf("authorization failed: %w", err)
		}
	}

	// Acquire resource lock to prevent concurrent execution on same resource
	resourceLock := e.getResourceLock(request.Resource.ID)
	resourceLock.Lock()
	defer resourceLock.Unlock()

	// Acquire semaphore for concurrency control
	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	case <-ctx.Done():
		return types.ExecutionResult{
			Status: types.StatusCanceled,
			Error:  ctx.Err(),
		}, ctx.Err()
	}

	// Use strategy if provided, otherwise use simple execution
	if request.Strategy != nil {
		return request.Strategy.Execute(ctx, plugin, request.Resource)
	}

	// Execute with default strategy
	return e.executePlugin(ctx, plugin, request.Resource, request.Principal)
}

// executePlugin executes a plugin with full lifecycle management
func (e *DefaultExecutionEngine) executePlugin(ctx context.Context, plugin types.Plugin, resource types.Resource, principal string) (types.ExecutionResult, error) {
	startTime := time.Now()
	executionID := uuid.New().String()

	// Create plugin context
	progressChan := make(chan types.ProgressUpdate, 10)
	defer close(progressChan)

	// Create context with timeout
	timeout := e.defaultTimeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pluginCtx := types.PluginContext{
		Context:     execCtx,
		ExecutionID: executionID,
		Timeout:     timeout,
		Monitor:     e.monitor,
		Reporter:    e.reporter,
		EventBus:    e.eventBus,
		Logger:      e.logger,
		Tracer:      e.tracer,
		Progress:    progressChan,
		Principal:   principal,
		Auth:        e.auth,
		Secrets:     e.secrets,
	}

	var snapshot types.Snapshot
	var execErr error
	status := types.StatusSuccess

	// Lifecycle: Validate
	if err := plugin.Validate(pluginCtx, resource); err != nil {
		return types.ExecutionResult{
			Status:    types.StatusFailed,
			StartTime: startTime,
			EndTime:   time.Now(),
			Duration:  time.Since(startTime),
			Error:     fmt.Errorf("validation failed: %w", err),
		}, fmt.Errorf("validation failed: %w", err)
	}

	// Lifecycle: PreExecute (create snapshot)
	snapshot, err := plugin.PreExecute(pluginCtx, resource)
	if err != nil {
		return types.ExecutionResult{
			Status:    types.StatusFailed,
			StartTime: startTime,
			EndTime:   time.Now(),
			Duration:  time.Since(startTime),
			Error:     fmt.Errorf("pre-execute failed: %w", err),
		}, fmt.Errorf("pre-execute failed: %w", err)
	}

	// Lifecycle: Execute
	execErr = plugin.Execute(pluginCtx, resource)
	if execErr != nil {
		status = types.StatusFailed
		if execErr == context.DeadlineExceeded {
			status = types.StatusTimeout
		} else if execErr == context.Canceled {
			status = types.StatusCanceled
		}

		// Attempt rollback if plugin supports it and snapshot exists
		if snapshot != nil {
			if rollbackErr := plugin.Rollback(pluginCtx, resource, snapshot); rollbackErr != nil {
				// Log rollback error but don't fail the execution result
				if e.logger != nil {
					e.logger.Error("rollback failed", types.Field{Key: "error", Value: rollbackErr})
				}
			}
		}
	}

	endTime := time.Now()
	result := types.ExecutionResult{
		Status:    status,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Error:     execErr,
		Metadata:  make(map[string]interface{}),
	}

	// Lifecycle: PostExecute (always run, even on failure)
	if postErr := plugin.PostExecute(pluginCtx, resource, result); postErr != nil {
		if e.logger != nil {
			e.logger.Error("post-execute failed", types.Field{Key: "error", Value: postErr})
		}
	}

	// Lifecycle: Cleanup (always run)
	if cleanupErr := plugin.Cleanup(pluginCtx, resource); cleanupErr != nil {
		if e.logger != nil {
			e.logger.Error("cleanup failed", types.Field{Key: "error", Value: cleanupErr})
		}
	}

	// Record execution if reporter is available
	if e.reporter != nil {
		record := types.ExecutionRecord{
			ID:           executionID,
			PluginName:   plugin.Metadata().Name,
			ResourceID:   resource.ID,
			ResourceName: resource.Name,
			StartTime:    startTime,
			EndTime:      endTime,
			Duration:     endTime.Sub(startTime),
			Status:       status,
			Principal:    principal,
			Metadata:     result.Metadata,
		}
		if execErr != nil {
			record.Error = execErr.Error()
		}
		if err := e.reporter.RecordExecution(ctx, record); err != nil {
			if e.logger != nil {
				e.logger.Error("failed to record execution", types.Field{Key: "error", Value: err})
			}
		}
	}

	return result, execErr
}

// getResourceLock gets or creates a lock for a resource
func (e *DefaultExecutionEngine) getResourceLock(resourceID string) *sync.Mutex {
	e.resourceLocksMu.Lock()
	defer e.resourceLocksMu.Unlock()

	if lock, exists := e.resourceLocks[resourceID]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	e.resourceLocks[resourceID] = lock
	return lock
}

// ExecuteAsync executes a plugin asynchronously and returns a Future
func (e *DefaultExecutionEngine) ExecuteAsync(ctx context.Context, request types.ExecutionRequest) (types.Future, error) {
	// Create execution state
	executionID := uuid.New().String()
	execCtx, cancel := context.WithCancel(ctx)

	state := &executionState{
		id:     executionID,
		status: types.StatusSuccess,
		cancel: cancel,
	}

	e.executionsMu.Lock()
	e.executions[executionID] = state
	e.executionsMu.Unlock()

	// Create future
	future := &defaultFuture{
		id:     executionID,
		engine: e,
		done:   make(chan struct{}),
	}

	// Execute in goroutine
	go func() {
		defer close(future.done)
		defer cancel()

		result, err := e.Execute(execCtx, request)
		if err != nil && e.logger != nil {
			e.logger.Error("async execution failed", types.Field{Key: "error", Value: err})
		}

		state.resultMu.Lock()
		state.result = result
		state.status = result.Status
		state.resultMu.Unlock()
	}()

	return future, nil
}

// Cancel cancels an ongoing execution
func (e *DefaultExecutionEngine) Cancel(executionID string) error {
	e.executionsMu.RLock()
	state, exists := e.executions[executionID]
	e.executionsMu.RUnlock()

	if !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	state.cancel()
	return nil
}

// GetStatus returns the status of an execution
func (e *DefaultExecutionEngine) GetStatus(executionID string) (types.ExecutionStatus, error) {
	e.executionsMu.RLock()
	state, exists := e.executions[executionID]
	e.executionsMu.RUnlock()

	if !exists {
		return "", fmt.Errorf("execution %s not found", executionID)
	}

	state.resultMu.RLock()
	defer state.resultMu.RUnlock()

	return state.status, nil
}

// SetConcurrencyLimit sets the maximum number of concurrent executions
func (e *DefaultExecutionEngine) SetConcurrencyLimit(limit int) {
	if limit <= 0 {
		limit = 1
	}

	e.concurrencyLimit = limit
	e.semaphore = make(chan struct{}, limit)
}

// SetDefaultTimeout sets the default timeout for plugin executions
func (e *DefaultExecutionEngine) SetDefaultTimeout(timeout time.Duration) {
	e.defaultTimeout = timeout
}

// ExecuteWorkflow executes a workflow with dependency management and error handling
func (e *DefaultExecutionEngine) ExecuteWorkflow(ctx context.Context, workflow types.Workflow) (types.WorkflowResult, error) {
	startTime := time.Now()

	// Build dependency graph and detect cycles
	graph, err := e.buildDependencyGraph(workflow)
	if err != nil {
		return types.WorkflowResult{
			Status:      types.StatusFailed,
			StepResults: make(map[string]types.ExecutionResult),
			StartTime:   startTime,
			EndTime:     time.Now(),
			Duration:    time.Since(startTime),
		}, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Detect circular dependencies
	if cycleErr := e.detectCycles(graph); cycleErr != nil {
		return types.WorkflowResult{
			Status:      types.StatusFailed,
			StepResults: make(map[string]types.ExecutionResult),
			StartTime:   startTime,
			EndTime:     time.Now(),
			Duration:    time.Since(startTime),
		}, fmt.Errorf("circular dependency detected: %w", cycleErr)
	}

	// Topological sort to determine execution order
	executionOrder, err := e.topologicalSort(graph)
	if err != nil {
		return types.WorkflowResult{
			Status:      types.StatusFailed,
			StepResults: make(map[string]types.ExecutionResult),
			StartTime:   startTime,
			EndTime:     time.Now(),
			Duration:    time.Since(startTime),
		}, fmt.Errorf("failed to sort workflow steps: %w", err)
	}

	// Execute workflow steps
	stepResults := make(map[string]types.ExecutionResult)
	overallStatus := types.StatusSuccess

	// Track completed steps for dependency checking
	completedSteps := make(map[string]bool)
	stepResultsMu := sync.Mutex{}

	// Process steps in topological order
	for _, level := range executionOrder {
		// Check if we should abort before processing this level
		if overallStatus == types.StatusFailed {
			// Mark all remaining steps as skipped
			for _, stepName := range level {
				stepResultsMu.Lock()
				if _, exists := stepResults[stepName]; !exists {
					stepResults[stepName] = types.ExecutionResult{
						Status:    types.StatusSkipped,
						StartTime: time.Now(),
						EndTime:   time.Now(),
						Duration:  0,
					}
				}
				stepResultsMu.Unlock()
			}
			continue
		}

		// Separate parallel and sequential steps at this level
		// Preserve original order from workflow definition
		var parallelSteps, sequentialSteps []types.WorkflowStep
		for _, step := range workflow.Steps {
			// Check if this step is in the current level
			inLevel := false
			for _, stepName := range level {
				if step.Name == stepName {
					inLevel = true
					break
				}
			}
			if !inLevel {
				continue
			}

			if step.Parallel {
				parallelSteps = append(parallelSteps, step)
			} else {
				sequentialSteps = append(sequentialSteps, step)
			}
		}

		// Execute sequential steps first
		for _, step := range sequentialSteps {
			// Check if we should abort
			if overallStatus == types.StatusFailed {
				// Mark this and remaining steps as skipped
				stepResultsMu.Lock()
				stepResults[step.Name] = types.ExecutionResult{
					Status:    types.StatusSkipped,
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Duration:  0,
				}
				completedSteps[step.Name] = true
				stepResultsMu.Unlock()
				continue
			}

			// Check conditional execution
			if !e.shouldExecuteStep(step, stepResults) {
				stepResultsMu.Lock()
				stepResults[step.Name] = types.ExecutionResult{
					Status:    types.StatusSkipped,
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Duration:  0,
				}
				completedSteps[step.Name] = true
				stepResultsMu.Unlock()
				continue
			}

			// Wait for dependencies
			if err := e.waitForDependencies(step, completedSteps); err != nil {
				stepResultsMu.Lock()
				stepResults[step.Name] = types.ExecutionResult{
					Status:    types.StatusFailed,
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Duration:  0,
					Error:     err,
				}
				stepResultsMu.Unlock()
				overallStatus = types.StatusFailed
				break
			}

			// Execute step
			result, err := e.executeWorkflowStep(ctx, step)
			stepResultsMu.Lock()
			stepResults[step.Name] = result
			completedSteps[step.Name] = true
			stepResultsMu.Unlock()

			// Handle errors
			if err != nil {
				action := e.handleStepError(ctx, step, err)
				switch action {
				case types.ErrorActionAbort:
					overallStatus = types.StatusFailed
					goto done
				case types.ErrorActionRetry:
					// Retry the step once
					result, err = e.executeWorkflowStep(ctx, step)
					stepResultsMu.Lock()
					stepResults[step.Name] = result
					stepResultsMu.Unlock()
					if err != nil {
						overallStatus = types.StatusFailed
						goto done
					}
				case types.ErrorActionRollback:
					// Mark for rollback but continue
					overallStatus = types.StatusFailed
					goto done
				case types.ErrorActionContinue:
					// Continue to next step
					if e.logger != nil {
						e.logger.Warn("step failed but continuing",
							types.Field{Key: "step", Value: step.Name},
							types.Field{Key: "error", Value: err})
					}
				}
			}
		}

		// Execute parallel steps concurrently
		if len(parallelSteps) > 0 {
			var wg sync.WaitGroup
			errorsChan := make(chan error, len(parallelSteps))

			for _, step := range parallelSteps {
				// Check if we should abort
				if overallStatus == types.StatusFailed {
					break
				}

				// Check conditional execution
				if !e.shouldExecuteStep(step, stepResults) {
					stepResultsMu.Lock()
					stepResults[step.Name] = types.ExecutionResult{
						Status:    types.StatusSkipped,
						StartTime: time.Now(),
						EndTime:   time.Now(),
						Duration:  0,
					}
					completedSteps[step.Name] = true
					stepResultsMu.Unlock()
					continue
				}

				wg.Add(1)
				go func(s types.WorkflowStep) {
					defer wg.Done()

					// Wait for dependencies
					if err := e.waitForDependencies(s, completedSteps); err != nil {
						stepResultsMu.Lock()
						stepResults[s.Name] = types.ExecutionResult{
							Status:    types.StatusFailed,
							StartTime: time.Now(),
							EndTime:   time.Now(),
							Duration:  0,
							Error:     err,
						}
						stepResultsMu.Unlock()
						errorsChan <- err
						return
					}

					// Execute step
					result, err := e.executeWorkflowStep(ctx, s)
					stepResultsMu.Lock()
					stepResults[s.Name] = result
					completedSteps[s.Name] = true
					stepResultsMu.Unlock()

					if err != nil {
						errorsChan <- err
					}
				}(step)
			}

			wg.Wait()
			close(errorsChan)

			// Check for errors in parallel execution
			for err := range errorsChan {
				if err != nil {
					overallStatus = types.StatusFailed
					// Note: For parallel steps, we use a simplified error handling
					// All parallel steps complete, then we check if any failed
				}
			}
		}
	}

done:
	endTime := time.Now()

	return types.WorkflowResult{
		Status:      overallStatus,
		StepResults: stepResults,
		StartTime:   startTime,
		EndTime:     endTime,
		Duration:    endTime.Sub(startTime),
	}, nil
}

// buildDependencyGraph builds a dependency graph from workflow steps
func (e *DefaultExecutionEngine) buildDependencyGraph(workflow types.Workflow) (map[string][]string, error) {
	graph := make(map[string][]string)

	// Initialize all nodes
	for _, step := range workflow.Steps {
		if step.Name == "" {
			return nil, fmt.Errorf("step name cannot be empty")
		}
		graph[step.Name] = []string{}
	}

	// Add edges (dependencies)
	for _, step := range workflow.Steps {
		for _, dep := range step.DependsOn {
			if _, exists := graph[dep]; !exists {
				return nil, fmt.Errorf("dependency %s not found for step %s", dep, step.Name)
			}
			graph[step.Name] = append(graph[step.Name], dep)
		}
	}

	return graph, nil
}

// detectCycles detects circular dependencies using DFS
func (e *DefaultExecutionEngine) detectCycles(graph map[string][]string) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string) error
	dfs = func(node string) error {
		visited[node] = true
		recStack[node] = true

		for _, dep := range graph[node] {
			if !visited[dep] {
				if err := dfs(dep); err != nil {
					return err
				}
			} else if recStack[dep] {
				return fmt.Errorf("circular dependency detected: %s -> %s", node, dep)
			}
		}

		recStack[node] = false
		return nil
	}

	for node := range graph {
		if !visited[node] {
			if err := dfs(node); err != nil {
				return err
			}
		}
	}

	return nil
}

// topologicalSort performs topological sort using Kahn's algorithm
// Returns levels of steps that can be executed in parallel
func (e *DefaultExecutionEngine) topologicalSort(graph map[string][]string) ([][]string, error) {
	// Calculate in-degree for each node
	// In-degree = number of dependencies a node has
	inDegree := make(map[string]int)
	for node := range graph {
		inDegree[node] = len(graph[node])
	}

	// Find all nodes with in-degree 0 (no dependencies)
	var queue []string
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	var result [][]string
	visited := 0

	for len(queue) > 0 {
		// All nodes in current queue can be executed in parallel
		level := make([]string, len(queue))
		copy(level, queue)
		result = append(result, level)

		// Process current level
		nextQueue := []string{}
		for _, node := range queue {
			visited++

			// For each node that depends on the current node, reduce its in-degree
			for dependent, deps := range graph {
				for _, dep := range deps {
					if dep == node {
						inDegree[dependent]--
						if inDegree[dependent] == 0 {
							nextQueue = append(nextQueue, dependent)
						}
						break
					}
				}
			}
		}

		queue = nextQueue
	}

	if visited != len(graph) {
		return nil, fmt.Errorf("graph contains a cycle")
	}

	return result, nil
}

// findStep finds a step by name in the workflow
//
//nolint:unused // Used in workflow execution
func (e *DefaultExecutionEngine) findStep(workflow types.Workflow, name string) *types.WorkflowStep {
	for i := range workflow.Steps {
		if workflow.Steps[i].Name == name {
			return &workflow.Steps[i]
		}
	}
	return nil
}

// shouldExecuteStep checks if a step should execute based on its condition
func (e *DefaultExecutionEngine) shouldExecuteStep(step types.WorkflowStep, stepResults map[string]types.ExecutionResult) bool {
	if step.Condition == nil {
		return true
	}

	switch step.Condition.Type {
	case types.ConditionAlways:
		return true
	case types.ConditionOnSuccess:
		// Check if all dependencies succeeded
		for _, dep := range step.DependsOn {
			if result, exists := stepResults[dep]; exists {
				if result.Status != types.StatusSuccess {
					return false
				}
			}
		}
		return true
	case types.ConditionOnFailure:
		// Check if any dependency failed
		for _, dep := range step.DependsOn {
			if result, exists := stepResults[dep]; exists {
				if result.Status == types.StatusFailed {
					return true
				}
			}
		}
		return false
	case types.ConditionCustom:
		// Custom conditions would need to be evaluated
		// For now, default to true
		return true
	default:
		return true
	}
}

// waitForDependencies waits for all dependencies to complete
func (e *DefaultExecutionEngine) waitForDependencies(step types.WorkflowStep, completedSteps map[string]bool) error {
	// In our implementation, dependencies are guaranteed to be completed
	// by the topological sort, but we check anyway
	for _, dep := range step.DependsOn {
		if !completedSteps[dep] {
			return fmt.Errorf("dependency %s not completed for step %s", dep, step.Name)
		}
	}
	return nil
}

// executeWorkflowStep executes a single workflow step
func (e *DefaultExecutionEngine) executeWorkflowStep(ctx context.Context, step types.WorkflowStep) (types.ExecutionResult, error) {
	// Execute plugin
	request := types.ExecutionRequest{
		PluginName: step.PluginName,
		Resource:   step.Resource,
		Config:     step.Config,
	}

	return e.Execute(ctx, request)
}

// handleStepError handles errors during step execution
func (e *DefaultExecutionEngine) handleStepError(ctx context.Context, step types.WorkflowStep, err error) types.ErrorAction {
	if step.OnError == nil {
		// Default action is abort
		return types.ErrorActionAbort
	}

	return step.OnError.Handle(ctx, step, err)
}

// defaultFuture implements the Future interface
type defaultFuture struct {
	id     string
	engine *DefaultExecutionEngine
	done   chan struct{}
}

// Wait waits for the execution to complete and returns the result
func (f *defaultFuture) Wait() (types.ExecutionResult, error) {
	<-f.done

	f.engine.executionsMu.RLock()
	state, exists := f.engine.executions[f.id]
	f.engine.executionsMu.RUnlock()

	if !exists {
		return types.ExecutionResult{}, fmt.Errorf("execution %s not found", f.id)
	}

	state.resultMu.RLock()
	defer state.resultMu.RUnlock()

	return state.result, state.result.Error
}

// Cancel cancels the execution
func (f *defaultFuture) Cancel() error {
	return f.engine.Cancel(f.id)
}

// Done returns a channel that is closed when the execution completes
func (f *defaultFuture) Done() <-chan struct{} {
	return f.done
}
