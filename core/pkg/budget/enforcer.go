package budget

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Storage handles persistence of budget data.
// In a real implementation, this would be a Postgres/Redis backing.
type Storage interface {
	Get(ctx context.Context, tenantID string) (*Budget, error)
	Set(ctx context.Context, budget *Budget) error
	Limits(ctx context.Context, tenantID string) (daily, monthly int64, err error)
	SetLimits(ctx context.Context, tenantID string, daily, monthly int64) error
}

// SimpleEnforcer implements fail-closed budget enforcement.
type SimpleEnforcer struct {
	storage Storage
}

// NewSimpleEnforcer creates a new enforcer with the given storage.
func NewSimpleEnforcer(s Storage) *SimpleEnforcer {
	return &SimpleEnforcer{
		storage: s,
	}
}

func (e *SimpleEnforcer) GetBudget(ctx context.Context, tenantID string) (*Budget, error) {
	return e.storage.Get(ctx, tenantID)
}

func (e *SimpleEnforcer) SetLimits(ctx context.Context, tenantID string, daily, monthly int64) error {
	return e.storage.SetLimits(ctx, tenantID, daily, monthly)
}

func (e *SimpleEnforcer) RecordSpend(ctx context.Context, tenantID string, cost Cost) error {
	// For SimpleEnforcer, Check() already reserves the budget.
	// We might implement adjustment logic here later.
	return nil
}

// Check verifies if a cost can be incurred. Fails closed on errors.
func (e *SimpleEnforcer) Check(ctx context.Context, tenantID string, cost Cost) (*Decision, error) {
	// FAIL-CLOSED: Any error results in denial.
	b, err := e.storage.Get(ctx, tenantID)
	if err != nil {
		// Log error here in real impl
		slog.Warn("budget check failed", "tenant_id", tenantID, "error", err)
		return &Decision{
			Allowed:   false,
			Reason:    fmt.Sprintf("check failed: %v", err),
			Remaining: nil,
			Receipt:   e.createReceipt(tenantID, "denied", cost.Amount, "internal_error"),
		}, err
	}

	// 1. Check Default Limits if budget is new
	if b == nil {
		daily, monthly, err := e.storage.Limits(ctx, tenantID)
		if err != nil {
			slog.Warn("budget limits fetch failed", "tenant_id", tenantID, "error", err)
			return &Decision{
				Allowed: false,
				Reason:  "failed to fetch limits",
				Receipt: e.createReceipt(tenantID, "denied", cost.Amount, "limit_fetch_error"),
			}, err
		}
		b = &Budget{
			TenantID:     tenantID,
			DailyLimit:   daily,
			MonthlyLimit: monthly,
			LastUpdated:  time.Now(),
		}
	}

	// 2. Reset counters if new period (MVP logic: naive time check)
	// In production, period management is complex (timezone, exact reset time).
	// Here we assume UTC resets.
	now := time.Now().UTC()
	if now.Day() != b.LastUpdated.Day() {
		b.DailyUsed = 0
	}
	if now.Month() != b.LastUpdated.Month() {
		b.MonthlyUsed = 0
	}

	// 3. Check Limits
	// NOTE: Currency conversion is out of scope for v0.1. All amounts are assumed to be in the same base unit (cents/USD).
	newDaily := b.DailyUsed + cost.Amount
	newMonthly := b.MonthlyUsed + cost.Amount

	if newDaily > b.DailyLimit {
		slog.Warn("budget daily limit exceeded", "tenant_id", tenantID, "new_daily", newDaily, "daily_limit", b.DailyLimit)
		return &Decision{
			Allowed:   false,
			Reason:    fmt.Sprintf("daily limit exceeded: %d > %d", newDaily, b.DailyLimit),
			Remaining: b,
			Receipt:   e.createReceipt(tenantID, "denied", cost.Amount, "daily_limit_exceeded"),
		}, nil
	}

	if newMonthly > b.MonthlyLimit {
		slog.Warn("budget monthly limit exceeded", "tenant_id", tenantID, "new_monthly", newMonthly, "monthly_limit", b.MonthlyLimit)
		return &Decision{
			Allowed:   false,
			Reason:    fmt.Sprintf("monthly limit exceeded: %d > %d", newMonthly, b.MonthlyLimit),
			Remaining: b,
			Receipt:   e.createReceipt(tenantID, "denied", cost.Amount, "monthly_limit_exceeded"),
		}, nil
	}

	// 4. Update usage (optimistic locking would be needed here for concurrency)
	b.DailyUsed = newDaily
	b.MonthlyUsed = newMonthly
	b.LastUpdated = now

	if err := e.storage.Set(ctx, b); err != nil {
		// FAIL-CLOSED on write failure
		slog.Error("budget usage persistence failed", "tenant_id", tenantID, "error", err)
		return &Decision{
			Allowed: false,
			Reason:  "failed to persist usage",
			Receipt: e.createReceipt(tenantID, "denied", cost.Amount, "persistence_error"),
		}, err
	}

	return &Decision{
		Allowed:   true,
		Reason:    "within limits",
		Remaining: b,
		Receipt:   e.createReceipt(tenantID, "allowed", cost.Amount, "ok"),
	}, nil
}

func (e *SimpleEnforcer) createReceipt(tenantID, action string, cost int64, reason string) *EnforcementReceipt {
	return &EnforcementReceipt{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Action:    action,
		CostCents: cost,
		Reason:    reason,
		Timestamp: time.Now().UTC(),
	}
}
