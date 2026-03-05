package kernel

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewSwarmOrchestrator(t *testing.T) {
	s := NewSwarmOrchestrator(nil)
	require.NotNil(t, s)
	require.NotNil(t, s.GetMetrics())
}

func TestDefaultSwarmConfig(t *testing.T) {
	config := DefaultSwarmConfig()
	require.Equal(t, 100, config.MaxAgents)
	require.Equal(t, 1500, config.MaxToolCallsPerAgent)
	require.True(t, config.EnableCriticalPath)
	require.Equal(t, ModeHybrid, config.CoordinationMode)
}

func TestSwarmOrchestrator_ExecuteEmpty(t *testing.T) {
	s := NewSwarmOrchestrator(nil)

	execution, err := s.Execute(context.Background(), []AgentTask{})
	require.NoError(t, err)
	require.NotNil(t, execution)
	require.Equal(t, StatusCompleted, execution.Status)
	require.Empty(t, execution.Results)
}

func TestSwarmOrchestrator_RegisterHandler(t *testing.T) {
	s := NewSwarmOrchestrator(nil)

	handler := func(ctx context.Context, task AgentTask) (*AgentResult, error) {
		return &AgentResult{Status: StatusCompleted}, nil
	}

	s.RegisterHandler("test-type", handler)

	tasks := []AgentTask{
		CreateTask("task-1", "test-type", 1, nil, nil),
	}

	execution, err := s.Execute(context.Background(), tasks)
	require.NoError(t, err)
	require.Len(t, execution.Results, 1)
	require.Equal(t, StatusCompleted, execution.Results[0].Status)
}

func TestSwarmOrchestrator_ParallelExecution(t *testing.T) {
	config := DefaultSwarmConfig()
	config.CoordinationMode = ModeParallel
	s := NewSwarmOrchestrator(config)

	var executed int32
	handler := func(ctx context.Context, task AgentTask) (*AgentResult, error) {
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&executed, 1)
		return &AgentResult{Status: StatusCompleted, ToolCalls: 5}, nil
	}

	s.RegisterHandler("parallel-type", handler)

	tasks := make([]AgentTask, 10)
	for i := range tasks {
		tasks[i] = CreateTask("task-"+string(rune('0'+i)), "parallel-type", 1, nil, nil)
	}

	start := time.Now()
	execution, err := s.Execute(context.Background(), tasks)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Equal(t, int32(10), atomic.LoadInt32(&executed))
	require.Len(t, execution.Results, 10)

	// Should complete faster than sequential (10 * 10ms = 100ms)
	require.Less(t, elapsed.Milliseconds(), int64(100))
}

func TestSwarmOrchestrator_WaterfallExecution(t *testing.T) {
	config := DefaultSwarmConfig()
	config.CoordinationMode = ModeWaterfall
	s := NewSwarmOrchestrator(config)

	order := make([]string, 0)
	handler := func(ctx context.Context, task AgentTask) (*AgentResult, error) {
		order = append(order, task.TaskID)
		return &AgentResult{Status: StatusCompleted}, nil
	}

	s.RegisterHandler("waterfall-type", handler)

	// Create dependency chain: task-3 -> task-2 -> task-1
	tasks := []AgentTask{
		CreateTask("task-1", "waterfall-type", 1, nil, nil),
		CreateTask("task-2", "waterfall-type", 2, []string{"task-1"}, nil),
		CreateTask("task-3", "waterfall-type", 3, []string{"task-2"}, nil),
	}

	execution, err := s.Execute(context.Background(), tasks)
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, execution.Status)

	// Should execute in dependency order
	require.Equal(t, []string{"task-1", "task-2", "task-3"}, order)
}

func TestSwarmOrchestrator_HybridExecution(t *testing.T) {
	config := DefaultSwarmConfig()
	config.CoordinationMode = ModeHybrid
	s := NewSwarmOrchestrator(config)

	var level0Count, level1Count int32
	handler := func(ctx context.Context, task AgentTask) (*AgentResult, error) {
		if task.TaskID == "task-1" || task.TaskID == "task-2" {
			atomic.AddInt32(&level0Count, 1)
		} else {
			atomic.AddInt32(&level1Count, 1)
		}
		return &AgentResult{Status: StatusCompleted}, nil
	}

	s.RegisterHandler("hybrid-type", handler)

	// task-1 and task-2 are independent (level 0)
	// task-3 depends on both (level 1)
	tasks := []AgentTask{
		CreateTask("task-1", "hybrid-type", 1, nil, nil),
		CreateTask("task-2", "hybrid-type", 2, nil, nil),
		CreateTask("task-3", "hybrid-type", 3, []string{"task-1", "task-2"}, nil),
	}

	execution, err := s.Execute(context.Background(), tasks)
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, execution.Status)
	require.Equal(t, int32(2), atomic.LoadInt32(&level0Count))
	require.Equal(t, int32(1), atomic.LoadInt32(&level1Count))
}

func TestSwarmOrchestrator_HandlerError(t *testing.T) {
	s := NewSwarmOrchestrator(nil)

	handler := func(ctx context.Context, task AgentTask) (*AgentResult, error) {
		return nil, errors.New("handler failed")
	}

	s.RegisterHandler("error-type", handler)

	tasks := []AgentTask{
		CreateTask("task-1", "error-type", 1, nil, nil),
	}

	execution, err := s.Execute(context.Background(), tasks)
	require.Error(t, err)
	require.Equal(t, StatusFailed, execution.Status)
}

func TestSwarmOrchestrator_MissingHandler(t *testing.T) {
	s := NewSwarmOrchestrator(nil)

	tasks := []AgentTask{
		CreateTask("task-1", "missing-type", 1, nil, nil),
	}

	execution, err := s.Execute(context.Background(), tasks)
	require.Error(t, err)
	require.Equal(t, StatusFailed, execution.Status)
	require.Contains(t, err.Error(), "no handler")
}

func TestSwarmOrchestrator_Metrics(t *testing.T) {
	s := NewSwarmOrchestrator(nil)

	handler := func(ctx context.Context, task AgentTask) (*AgentResult, error) {
		return &AgentResult{Status: StatusCompleted, ToolCalls: 10}, nil
	}

	s.RegisterHandler("metric-type", handler)

	tasks := []AgentTask{
		CreateTask("task-1", "metric-type", 1, nil, nil),
		CreateTask("task-2", "metric-type", 2, nil, nil),
	}

	_, _ = s.Execute(context.Background(), tasks)

	metrics := s.GetMetrics()
	require.Equal(t, int64(1), metrics.TotalExecutions)
	require.Equal(t, int64(2), metrics.TotalTasksProcessed)
	require.Equal(t, int64(20), metrics.TotalToolCalls)
}

func TestSwarmOrchestrator_Hash(t *testing.T) {
	s := NewSwarmOrchestrator(nil)

	hash1 := s.Hash()
	require.NotEmpty(t, hash1)

	// Hash should be deterministic
	hash2 := s.Hash()
	require.Equal(t, hash1, hash2)
}

func TestCreateTask(t *testing.T) {
	task := CreateTask("id-1", "type-1", 5, []string{"dep-1"}, map[string]interface{}{"key": "value"})

	require.Equal(t, "id-1", task.TaskID)
	require.Equal(t, "type-1", task.TaskType)
	require.Equal(t, 5, task.Priority)
	require.Equal(t, []string{"dep-1"}, task.Dependencies)
	require.Equal(t, map[string]interface{}{"key": "value"}, task.Payload)
}

func TestSwarmOrchestrator_DependencyLevels(t *testing.T) {
	config := DefaultSwarmConfig()
	config.CoordinationMode = ModeHybrid
	s := NewSwarmOrchestrator(config)

	handler := func(ctx context.Context, task AgentTask) (*AgentResult, error) {
		return &AgentResult{Status: StatusCompleted}, nil
	}

	s.RegisterHandler("level-type", handler)

	// Complex dependency graph:
	// Level 0: A, B
	// Level 1: C (depends on A), D (depends on B)
	// Level 2: E (depends on C, D)
	tasks := []AgentTask{
		CreateTask("A", "level-type", 1, nil, nil),
		CreateTask("B", "level-type", 1, nil, nil),
		CreateTask("C", "level-type", 2, []string{"A"}, nil),
		CreateTask("D", "level-type", 2, []string{"B"}, nil),
		CreateTask("E", "level-type", 3, []string{"C", "D"}, nil),
	}

	execution, err := s.Execute(context.Background(), tasks)
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, execution.Status)
	require.Len(t, execution.Results, 5)

	// All should complete
	for _, r := range execution.Results {
		require.Equal(t, StatusCompleted, r.Status)
	}
}

func TestSwarmOrchestrator_MaxAgentsLimit(t *testing.T) {
	config := &SwarmConfig{
		MaxAgents:        2, // Limit to 2 concurrent
		AgentTimeout:     5 * time.Second,
		CoordinationMode: ModeParallel,
	}
	s := NewSwarmOrchestrator(config)

	var maxConcurrent int32
	var current int32

	handler := func(ctx context.Context, task AgentTask) (*AgentResult, error) {
		cur := atomic.AddInt32(&current, 1)
		if cur > atomic.LoadInt32(&maxConcurrent) {
			atomic.StoreInt32(&maxConcurrent, cur)
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&current, -1)
		return &AgentResult{Status: StatusCompleted}, nil
	}

	s.RegisterHandler("limit-type", handler)

	tasks := make([]AgentTask, 5)
	for i := range tasks {
		tasks[i] = CreateTask("task-"+string(rune('0'+i)), "limit-type", 1, nil, nil)
	}

	_, err := s.Execute(context.Background(), tasks)
	require.NoError(t, err)

	// Max concurrent should not exceed 2
	require.LessOrEqual(t, int(atomic.LoadInt32(&maxConcurrent)), 2)
}
