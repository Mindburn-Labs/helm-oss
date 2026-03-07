package metering_test

import (
	"context"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/metering"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMeter implements Meter for testing
type MockMeter struct {
	events []metering.Event
}

func NewMockMeter() *MockMeter {
	return &MockMeter{events: make([]metering.Event, 0)}
}

func (m *MockMeter) Record(ctx context.Context, event metering.Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	m.events = append(m.events, event)
	return nil
}

func (m *MockMeter) RecordBatch(ctx context.Context, events []metering.Event) error {
	for _, e := range events {
		if err := m.Record(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockMeter) GetUsage(ctx context.Context, tenantID string, period metering.Period) (*metering.Usage, error) {
	usage := &metering.Usage{
		TenantID:   tenantID,
		Period:     period,
		Totals:     make(map[metering.EventType]int64),
		LastUpdate: time.Now().UTC(),
	}

	for _, e := range m.events {
		if e.TenantID == tenantID && !e.Timestamp.Before(period.Start) && e.Timestamp.Before(period.End) {
			usage.Totals[e.EventType] += e.Quantity
		}
	}

	return usage, nil
}

func (m *MockMeter) GetUsageByType(ctx context.Context, tenantID string, eventType metering.EventType, period metering.Period) (int64, error) {
	usage, err := m.GetUsage(ctx, tenantID, period)
	if err != nil {
		return 0, err
	}
	return usage.Totals[eventType], nil
}

func TestMeter_RecordAndGetUsage(t *testing.T) {
	meter := NewMockMeter()
	ctx := context.Background()
	tenantID := "tenant-123"

	// Record various events
	events := []metering.Event{
		{TenantID: tenantID, EventType: metering.EventRequest, Quantity: 1},
		{TenantID: tenantID, EventType: metering.EventRequest, Quantity: 1},
		{TenantID: tenantID, EventType: metering.EventLLMToken, Quantity: 1500},
		{TenantID: tenantID, EventType: metering.EventToolCall, Quantity: 3},
	}

	for _, e := range events {
		err := meter.Record(ctx, e)
		require.NoError(t, err)
	}

	// Get usage
	usage, err := meter.GetUsage(ctx, tenantID, metering.DailyPeriod())
	require.NoError(t, err)

	assert.Equal(t, tenantID, usage.TenantID)
	assert.Equal(t, int64(2), usage.Totals[metering.EventRequest])
	assert.Equal(t, int64(1500), usage.Totals[metering.EventLLMToken])
	assert.Equal(t, int64(3), usage.Totals[metering.EventToolCall])
}

func TestMeter_GetUsageByType(t *testing.T) {
	meter := NewMockMeter()
	ctx := context.Background()
	tenantID := "tenant-456"

	// Record events
	err := meter.RecordBatch(ctx, []metering.Event{
		{TenantID: tenantID, EventType: metering.EventExecution, Quantity: 10},
		{TenantID: tenantID, EventType: metering.EventExecution, Quantity: 5},
		{TenantID: tenantID, EventType: metering.EventRequest, Quantity: 100},
	})
	require.NoError(t, err)

	// Query specific type
	executions, err := meter.GetUsageByType(ctx, tenantID, metering.EventExecution, metering.DailyPeriod())
	require.NoError(t, err)
	assert.Equal(t, int64(15), executions)
}

func TestMeter_TenantIsolation(t *testing.T) {
	meter := NewMockMeter()
	ctx := context.Background()

	// Record for different tenants
	_ = meter.Record(ctx, metering.Event{TenantID: "tenant-a", EventType: metering.EventRequest, Quantity: 100})
	_ = meter.Record(ctx, metering.Event{TenantID: "tenant-b", EventType: metering.EventRequest, Quantity: 50})

	// Verify isolation
	usageA, _ := meter.GetUsage(ctx, "tenant-a", metering.DailyPeriod())
	usageB, _ := meter.GetUsage(ctx, "tenant-b", metering.DailyPeriod())

	assert.Equal(t, int64(100), usageA.Totals[metering.EventRequest])
	assert.Equal(t, int64(50), usageB.Totals[metering.EventRequest])
}

func TestPeriods(t *testing.T) {
	daily := metering.DailyPeriod()
	assert.True(t, daily.End.Sub(daily.Start) == 24*time.Hour)

	monthly := metering.MonthlyPeriod()
	assert.True(t, monthly.Start.Day() == 1)
	assert.True(t, monthly.End.After(monthly.Start))
}
