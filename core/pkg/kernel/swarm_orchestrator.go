// Package kernel provides Agent Swarm Orchestration.
// Implements Kimi K2.5's 100-agent swarm paradigm for HELM.
package kernel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Mindburn-Labs/helm/core/pkg/orgdna/types"
)

// SwarmConfig configures the agent swarm orchestrator.
type SwarmConfig struct {
	// MaxAgents is the maximum number of concurrent sub-agents (K2.5 supports 100)
	MaxAgents int `json:"max_agents"`

	// MaxToolCallsPerAgent limits tool calls per sub-agent
	MaxToolCallsPerAgent int `json:"max_tool_calls_per_agent"`

	// EnableCriticalPath enables PARL critical path tracking
	EnableCriticalPath bool `json:"enable_critical_path"`

	// AgentTimeout is the maximum duration for a sub-agent
	AgentTimeout time.Duration `json:"agent_timeout"`

	// CoordinationMode determines how agents coordinate
	CoordinationMode CoordinationMode `json:"coordination_mode"`

	// OrgPhenotype defines the organizational structure (L1/L2)
	// Added to link Kernel to OrgDNA.
	OrgPhenotype *types.OrgPhenotype `json:"-"`
}

// CoordinationMode determines agent coordination strategy.
type CoordinationMode string

const (
	// ModeParallel executes all agents in parallel
	ModeParallel CoordinationMode = "parallel"
	// ModeWaterfall executes agents in dependency order
	ModeWaterfall CoordinationMode = "waterfall"
	// ModeHybrid combines parallel and waterfall based on dependencies
	ModeHybrid CoordinationMode = "hybrid"
)

// DefaultSwarmConfig returns production defaults.
func DefaultSwarmConfig() *SwarmConfig {
	return &SwarmConfig{
		MaxAgents:            100,  // K2.5's documented maximum
		MaxToolCallsPerAgent: 1500, // K2.5's documented maximum
		EnableCriticalPath:   true,
		AgentTimeout:         5 * time.Minute,
		CoordinationMode:     ModeHybrid,
	}
}

// AgentTask represents a task to be executed by a sub-agent.
type AgentTask struct {
	TaskID       string                 `json:"task_id"`
	TaskType     string                 `json:"task_type"`
	Priority     int                    `json:"priority"` // Lower = higher priority
	Dependencies []string               `json:"dependencies,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// AgentResult represents the result from a sub-agent.
type AgentResult struct {
	TaskID      string                 `json:"task_id"`
	AgentID     string                 `json:"agent_id"`
	Status      AgentStatus            `json:"status"`
	Output      map[string]interface{} `json:"output,omitempty"`
	ToolCalls   int                    `json:"tool_calls"`
	DurationMs  int64                  `json:"duration_ms"`
	Error       string                 `json:"error,omitempty"`
	CompletedAt time.Time              `json:"completed_at"`
}

// AgentStatus represents the execution status of an agent.
type AgentStatus string

const (
	StatusPending   AgentStatus = "pending"
	StatusRunning   AgentStatus = "running"
	StatusCompleted AgentStatus = "completed"
	StatusFailed    AgentStatus = "failed"
	StatusCancelled AgentStatus = "canceled"
	StatusTimedOut  AgentStatus = "timed_out"
)

// SubAgent represents a single sub-agent in the swarm.
type SubAgent struct {
	AgentID     string      `json:"agent_id"`
	TaskID      string      `json:"task_id"`
	Status      AgentStatus `json:"status"`
	ToolCalls   int         `json:"tool_calls"`
	StartedAt   time.Time   `json:"started_at,omitempty"`
	CompletedAt time.Time   `json:"completed_at,omitempty"`
}

// SwarmExecution represents a swarm execution session.
type SwarmExecution struct {
	ExecutionID    string        `json:"execution_id"`
	Status         AgentStatus   `json:"status"`
	Tasks          []AgentTask   `json:"tasks"`
	Results        []AgentResult `json:"results"`
	Agents         []SubAgent    `json:"agents"`
	CriticalSteps  int           `json:"critical_steps"`
	TotalToolCalls int           `json:"total_tool_calls"`
	StartedAt      time.Time     `json:"started_at"`
	CompletedAt    time.Time     `json:"completed_at,omitempty"`
	DurationMs     int64         `json:"duration_ms,omitempty"`
}

// SwarmMetrics tracks swarm performance.
type SwarmMetrics struct {
	mu                   sync.RWMutex
	TotalExecutions      int64   `json:"total_executions"`
	SuccessfulExecutions int64   `json:"successful_executions"`
	FailedExecutions     int64   `json:"failed_executions"`
	TotalTasksProcessed  int64   `json:"total_tasks_processed"`
	TotalToolCalls       int64   `json:"total_tool_calls"`
	AvgCriticalPath      float64 `json:"avg_critical_path"`
	MaxConcurrentAgents  int     `json:"max_concurrent_agents"`
	AvgTaskDuration      float64 `json:"avg_task_duration_ms"`
}

// TaskHandler is a function that executes a task.
type TaskHandler func(ctx context.Context, task AgentTask) (*AgentResult, error)

// SwarmOrchestrator coordinates multiple sub-agents.
type SwarmOrchestrator struct {
	mu           sync.RWMutex
	config       *SwarmConfig
	metrics      *SwarmMetrics
	handlers     map[string]TaskHandler
	activeAgents int32
}

// NewSwarmOrchestrator creates a new swarm orchestrator.
func NewSwarmOrchestrator(config *SwarmConfig) *SwarmOrchestrator {
	if config == nil {
		config = DefaultSwarmConfig()
	}

	return &SwarmOrchestrator{
		config:   config,
		metrics:  &SwarmMetrics{},
		handlers: make(map[string]TaskHandler),
	}
}

// RegisterHandler registers a task handler for a task type.
func (s *SwarmOrchestrator) RegisterHandler(taskType string, handler TaskHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[taskType] = handler
}

// Execute runs a swarm execution with the given tasks.
func (s *SwarmOrchestrator) Execute(ctx context.Context, tasks []AgentTask) (*SwarmExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	execution := &SwarmExecution{
		ExecutionID: fmt.Sprintf("swarm-%d", time.Now().UnixNano()),
		Status:      StatusRunning,
		Tasks:       tasks,
		Results:     make([]AgentResult, 0, len(tasks)),
		Agents:      make([]SubAgent, 0),
		StartedAt:   time.Now(),
	}

	if len(tasks) == 0 {
		execution.Status = StatusCompleted
		execution.CompletedAt = time.Now()
		return execution, nil
	}

	// Build dependency graph
	depGraph := s.buildDependencyGraph(tasks)

	// Execute based on coordination mode
	var results []AgentResult
	var err error

	switch s.config.CoordinationMode {
	case ModeParallel:
		results, err = s.executeParallel(ctx, tasks, execution)
	case ModeWaterfall:
		results, err = s.executeWaterfall(ctx, tasks, depGraph, execution)
	case ModeHybrid:
		results, err = s.executeHybrid(ctx, tasks, depGraph, execution)
	default:
		results, err = s.executeParallel(ctx, tasks, execution)
	}

	if err != nil {
		execution.Status = StatusFailed
		atomic.AddInt64(&s.metrics.FailedExecutions, 1)
		return execution, err
	}

	execution.Results = results
	execution.Status = StatusCompleted
	execution.CompletedAt = time.Now()
	execution.DurationMs = time.Since(execution.StartedAt).Milliseconds()

	// Calculate totals
	for _, r := range results {
		execution.TotalToolCalls += r.ToolCalls
	}

	// Update metrics
	s.updateMetrics(execution)

	return execution, nil
}

// executeParallel runs all tasks in parallel with semaphore.
func (s *SwarmOrchestrator) executeParallel(ctx context.Context, tasks []AgentTask, execution *SwarmExecution) ([]AgentResult, error) {
	results := make([]AgentResult, len(tasks))
	sem := make(chan struct{}, s.config.MaxAgents)
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t AgentTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			atomic.AddInt32(&s.activeAgents, 1)
			defer atomic.AddInt32(&s.activeAgents, -1)

			agentCtx, cancel := context.WithTimeout(ctx, s.config.AgentTimeout)
			defer cancel()

			result := s.executeTask(agentCtx, t)
			results[idx] = result

			if result.Status == StatusFailed && firstErr == nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("task %s failed: %s", t.TaskID, result.Error)
				}
				errMu.Unlock()
			}
		}(i, task)
	}

	wg.Wait()

	// Track max concurrent
	current := int(atomic.LoadInt32(&s.activeAgents))
	if current > s.metrics.MaxConcurrentAgents {
		s.metrics.MaxConcurrentAgents = current
	}

	return results, firstErr
}

// executeWaterfall runs tasks in dependency order.
func (s *SwarmOrchestrator) executeWaterfall(ctx context.Context, tasks []AgentTask, depGraph map[string][]string, execution *SwarmExecution) ([]AgentResult, error) {
	results := make(map[string]AgentResult)
	completed := make(map[string]bool)
	taskMap := make(map[string]AgentTask)

	for _, t := range tasks {
		taskMap[t.TaskID] = t
	}

	// Topological sort
	order := s.topologicalSort(tasks, depGraph)

	// Execute in order
	for _, taskID := range order {
		task := taskMap[taskID]

		// Check dependencies are complete
		for _, depID := range task.Dependencies {
			if !completed[depID] {
				return nil, fmt.Errorf("dependency %s not completed for task %s", depID, taskID)
			}
			if results[depID].Status == StatusFailed {
				// Skip if dependency failed
				results[taskID] = AgentResult{
					TaskID:      taskID,
					Status:      StatusCancelled,
					Error:       "dependency failed",
					CompletedAt: time.Now(),
				}
				completed[taskID] = true
				continue
			}
		}

		agentCtx, cancel := context.WithTimeout(ctx, s.config.AgentTimeout)
		result := s.executeTask(agentCtx, task)
		cancel()

		results[taskID] = result
		completed[taskID] = true
	}

	// Convert to slice
	resultSlice := make([]AgentResult, len(tasks))
	for i, t := range tasks {
		resultSlice[i] = results[t.TaskID]
	}

	return resultSlice, nil
}

// executeHybrid combines parallel and waterfall based on dependencies.
func (s *SwarmOrchestrator) executeHybrid(ctx context.Context, tasks []AgentTask, depGraph map[string][]string, execution *SwarmExecution) ([]AgentResult, error) {
	results := make(map[string]AgentResult)
	completed := make(map[string]bool)
	taskMap := make(map[string]AgentTask)

	for _, t := range tasks {
		taskMap[t.TaskID] = t
	}

	// Group tasks by dependency level
	levels := s.groupByLevel(tasks, depGraph)

	// Execute each level in parallel
	for _, level := range levels {
		levelResults, err := s.executeParallel(ctx, level, execution)
		if err != nil {
			return nil, err
		}

		for i, t := range level {
			results[t.TaskID] = levelResults[i]
			completed[t.TaskID] = true
		}
	}

	// Convert to slice
	resultSlice := make([]AgentResult, len(tasks))
	for i, t := range tasks {
		resultSlice[i] = results[t.TaskID]
	}

	return resultSlice, nil
}

// executeTask runs a single task.
func (s *SwarmOrchestrator) executeTask(ctx context.Context, task AgentTask) AgentResult {
	start := time.Now()

	agent := SubAgent{
		AgentID:   fmt.Sprintf("agent-%s", task.TaskID),
		TaskID:    task.TaskID,
		Status:    StatusRunning,
		StartedAt: start,
	}

	handler, exists := s.handlers[task.TaskType]
	if !exists {
		return AgentResult{
			TaskID:      task.TaskID,
			AgentID:     agent.AgentID,
			Status:      StatusFailed,
			Error:       fmt.Sprintf("no handler for task type: %s", task.TaskType),
			CompletedAt: time.Now(),
			DurationMs:  time.Since(start).Milliseconds(),
		}
	}

	result, err := handler(ctx, task)
	if err != nil {
		return AgentResult{
			TaskID:      task.TaskID,
			AgentID:     agent.AgentID,
			Status:      StatusFailed,
			Error:       err.Error(),
			CompletedAt: time.Now(),
			DurationMs:  time.Since(start).Milliseconds(),
		}
	}

	if result == nil {
		result = &AgentResult{
			TaskID:      task.TaskID,
			AgentID:     agent.AgentID,
			Status:      StatusCompleted,
			CompletedAt: time.Now(),
			DurationMs:  time.Since(start).Milliseconds(),
		}
	} else {
		result.TaskID = task.TaskID
		result.AgentID = agent.AgentID
		result.CompletedAt = time.Now()
		result.DurationMs = time.Since(start).Milliseconds()
		if result.Status == "" {
			result.Status = StatusCompleted
		}
	}

	return *result
}

// buildDependencyGraph creates a dependency map.
func (s *SwarmOrchestrator) buildDependencyGraph(tasks []AgentTask) map[string][]string {
	graph := make(map[string][]string)
	for _, t := range tasks {
		graph[t.TaskID] = t.Dependencies
	}
	return graph
}

// topologicalSort returns tasks in dependency order.
func (s *SwarmOrchestrator) topologicalSort(tasks []AgentTask, depGraph map[string][]string) []string {
	visited := make(map[string]bool)
	result := make([]string, 0, len(tasks))
	var visit func(string)

	visit = func(taskID string) {
		if visited[taskID] {
			return
		}
		visited[taskID] = true

		for _, depID := range depGraph[taskID] {
			visit(depID)
		}

		result = append(result, taskID)
	}

	// Sort tasks by priority for determinism
	sorted := make([]AgentTask, len(tasks))
	copy(sorted, tasks)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	for _, t := range sorted {
		visit(t.TaskID)
	}

	return result
}

// groupByLevel groups tasks by dependency level.
func (s *SwarmOrchestrator) groupByLevel(tasks []AgentTask, depGraph map[string][]string) [][]AgentTask {
	levels := make(map[string]int)
	taskMap := make(map[string]AgentTask)

	for _, t := range tasks {
		taskMap[t.TaskID] = t
	}

	var getLevel func(string) int
	getLevel = func(taskID string) int {
		if lvl, exists := levels[taskID]; exists {
			return lvl
		}

		maxDepLevel := -1
		for _, depID := range depGraph[taskID] {
			depLevel := getLevel(depID)
			if depLevel > maxDepLevel {
				maxDepLevel = depLevel
			}
		}

		levels[taskID] = maxDepLevel + 1
		return levels[taskID]
	}

	for _, t := range tasks {
		getLevel(t.TaskID)
	}

	// Group by level
	maxLevel := 0
	for _, lvl := range levels {
		if lvl > maxLevel {
			maxLevel = lvl
		}
	}

	groups := make([][]AgentTask, maxLevel+1)
	for i := range groups {
		groups[i] = make([]AgentTask, 0)
	}

	for _, t := range tasks {
		lvl := levels[t.TaskID]
		groups[lvl] = append(groups[lvl], t)
	}

	return groups
}

// updateMetrics updates swarm metrics.
func (s *SwarmOrchestrator) updateMetrics(execution *SwarmExecution) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	atomic.AddInt64(&s.metrics.TotalExecutions, 1)
	atomic.AddInt64(&s.metrics.SuccessfulExecutions, 1)
	atomic.AddInt64(&s.metrics.TotalTasksProcessed, int64(len(execution.Tasks)))
	atomic.AddInt64(&s.metrics.TotalToolCalls, int64(execution.TotalToolCalls))

	// Update averages
	totalExec := float64(atomic.LoadInt64(&s.metrics.TotalExecutions))
	if totalExec > 0 {
		s.metrics.AvgCriticalPath = (s.metrics.AvgCriticalPath*(totalExec-1) + float64(execution.CriticalSteps)) / totalExec
		s.metrics.AvgTaskDuration = (s.metrics.AvgTaskDuration*(totalExec-1) + float64(execution.DurationMs)) / totalExec
	}
}

// GetMetrics returns swarm metrics.
func (s *SwarmOrchestrator) GetMetrics() *SwarmMetrics {
	return s.metrics
}

// GetActiveAgents returns the current number of active agents.
func (s *SwarmOrchestrator) GetActiveAgents() int {
	return int(atomic.LoadInt32(&s.activeAgents))
}

// Hash returns a deterministic hash of the orchestrator state.
func (s *SwarmOrchestrator) Hash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, _ := json.Marshal(struct {
		Config  *SwarmConfig
		Metrics *SwarmMetrics
	}{
		Config:  s.config,
		Metrics: s.metrics,
	})

	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// CreateTask is a helper to create an AgentTask.
func CreateTask(taskID, taskType string, priority int, deps []string, payload map[string]interface{}) AgentTask {
	return AgentTask{
		TaskID:       taskID,
		TaskType:     taskType,
		Priority:     priority,
		Dependencies: deps,
		Payload:      payload,
		CreatedAt:    time.Now(),
	}
}

// OrchestrateAdaptive uses TaskClassifier to automatically select the optimal
// coordination mode and then executes tasks accordingly.
//
// This implements adaptive swarm selection, applying
// Google's scaling laws for multi-agent coordination:
//   - High parallelizable fraction (≥0.7) → parallel mode
//   - Medium fraction (0.3–0.7) → hybrid mode
//   - Low fraction (<0.3) or deep sequential chains → waterfall
//   - High error amplification risk → hybrid with centralized validation
func (s *SwarmOrchestrator) OrchestrateAdaptive(ctx context.Context, tasks []AgentTask) (*SwarmExecution, error) {
	if len(tasks) == 0 {
		return s.Execute(ctx, tasks)
	}

	// Extract task IDs and dependency map for the classifier
	taskIDs := make([]string, len(tasks))
	deps := make(map[string][]string, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.TaskID
		deps[t.TaskID] = t.Dependencies
	}

	// Analyze task properties using the classifier
	classifier := DefaultTaskClassifier()
	props := classifier.ClassifyFromDAG(taskIDs, deps)

	// Select coordination mode (returns mode + rationale)
	selectedMode, _ := classifier.SelectMode(props)

	// Temporarily override the coordination mode
	s.mu.Lock()
	originalMode := s.config.CoordinationMode
	s.config.CoordinationMode = selectedMode
	s.mu.Unlock()

	// Execute with selected mode
	result, err := s.Execute(ctx, tasks)

	// Restore original mode
	s.mu.Lock()
	s.config.CoordinationMode = originalMode
	s.mu.Unlock()

	return result, err
}
