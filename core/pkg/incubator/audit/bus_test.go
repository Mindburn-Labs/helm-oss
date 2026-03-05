package audit_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Mindburn-Labs/helm/core/pkg/incubator/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSink records calls for testing.
type mockSink struct {
	mu      sync.Mutex
	calls   int
	lastErr error
}

func (m *mockSink) Record(_ context.Context, _ audit.EventType, _, _ string, _ map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	return m.lastErr
}

func (m *mockSink) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestBus_FanOut(t *testing.T) {
	bus := audit.NewBus()
	s1, s2, s3 := &mockSink{}, &mockSink{}, &mockSink{}
	bus.AddSink(s1)
	bus.AddSink(s2)
	bus.AddSink(s3)

	err := bus.Record(context.Background(), audit.EventMutation, "deploy", "/clusters/prod", nil)
	require.NoError(t, err)

	assert.Equal(t, 1, s1.callCount())
	assert.Equal(t, 1, s2.callCount())
	assert.Equal(t, 1, s3.callCount())
}

func TestBus_FailClosed_NoSinks(t *testing.T) {
	bus := audit.NewBus()
	err := bus.Record(context.Background(), audit.EventAccess, "login", "/auth", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no sinks")
}

func TestBus_FailClosed_SinkError(t *testing.T) {
	bus := audit.NewBus()
	good := &mockSink{}
	bad := &mockSink{lastErr: fmt.Errorf("disk full")}
	bus.AddSink(good)
	bus.AddSink(bad)

	err := bus.Record(context.Background(), audit.EventMutation, "deploy", "/clusters/prod", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")

	// All sinks still called (best-effort fan-out)
	assert.Equal(t, 1, good.callCount())
	assert.Equal(t, 1, bad.callCount())
}

func TestBus_NilSink(t *testing.T) {
	bus := audit.NewBus()
	bus.AddSink(nil) // Should not panic
	assert.Equal(t, 0, bus.SinkCount())
}

func TestBus_ConcurrentSafety(t *testing.T) {
	bus := audit.NewBus()
	var counter int64
	sink := &mockSink{}
	bus.AddSink(sink)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bus.Record(context.Background(), audit.EventSystem, "ping", "/health", nil)
			atomic.AddInt64(&counter, 1)
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(100), counter)
	assert.Equal(t, 100, sink.callCount())
}
