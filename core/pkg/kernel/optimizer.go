package kernel

import (
	"context"
	"fmt"
	"log/slog"
)

// Mode represents the agent's operating mode.
type Mode string

const (
	ModeSmart Mode = "SMART" // High cost, high quality (e.g. specialized models)
	ModeFast  Mode = "FAST"  // Low cost, lower quality (e.g. distilled models)
)

// OptimizationStrategy defines how the optimizer selects modes.
type OptimizationStrategy string

const (
	StrategyPerformance OptimizationStrategy = "PERFORMANCE" // Always prefer Smart unless hard limited
	StrategyCostAware   OptimizationStrategy = "COST_AWARE"  // Downgrade to Fast if budget low
	StrategyLoadAware   OptimizationStrategy = "LOAD_AWARE"  // Downgrade to Fast if system load high
)

// Optimizer adjusts the agent's strategy based on system pressure and policy.
// It implements a self-improving optimization loop (Gap G).
type Optimizer struct {
	currentMode Mode
	strategy    OptimizationStrategy
	policy      BackpressurePolicy
}

// NewOptimizer creates a new optimizer with the given policy and strategy.
func NewOptimizer(policy BackpressurePolicy, strategy OptimizationStrategy) *Optimizer {
	if strategy == "" {
		strategy = StrategyPerformance
	}
	return &Optimizer{
		currentMode: ModeSmart,
		strategy:    strategy,
		policy:      policy,
	}
}

// CheckAndAttenuate checks if backpressure or policy requires a mode downgrade.
// GAP-18: Dynamic Strategy Selection & Retry-After.
func (o *Optimizer) CheckAndAttenuate(ctx context.Context, actorID string, store LimiterStore, metrics map[string]float64) (Mode, error) {
	// 1. Check Rate Limit (Hard Constraint)
	allowed, err := store.Allow(ctx, actorID, o.policy, 1)
	if err != nil {
		// Rate limiter error (e.g. storage failure) - fail closed
		return "", fmt.Errorf("limiter error: %w", err)
	}
	if !allowed {
		// Hard limit hit
		return "", fmt.Errorf("rate limit exceeded for %s", actorID)
	}

	// 2. Soft Constraints (Optimization)
	targetMode := ModeSmart

	switch o.strategy {
	case StrategyCostAware:
		// Check budget metric
		if budgetRemaining, ok := metrics["budget_remaining"]; ok && budgetRemaining < 0.2 {
			// Low budget (<20%), switch to Fast
			targetMode = ModeFast
		}
	case StrategyLoadAware:
		// Check load metric (0.0 to 1.0)
		if systemLoad, ok := metrics["system_load"]; ok && systemLoad > 0.8 {
			// High load (>80%), switch to Fast
			targetMode = ModeFast
		}
	case StrategyPerformance:
		// Keep Smart
	}

	// 3. Apply Decision
	if targetMode != o.currentMode {
		slog.Info(
			"optimizer mode switch",
			"from", o.currentMode,
			"to", targetMode,
			"actor_id", actorID,
			"strategy", o.strategy,
		)
		o.currentMode = targetMode
	}

	return o.currentMode, nil
}

// SetStrategy updates the optimization strategy at runtime.
func (o *Optimizer) SetStrategy(s OptimizationStrategy) {
	o.strategy = s
}
