package audit

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Bus is a fan-out audit event bus.
//
// Register sinks once at startup; all Record() calls emit to every sink.
// The bus is fail-closed: if any sink returns an error, the entire Record()
// call returns that error. This ensures no audit event is silently lost.
//
// Usage:
//
//	bus := audit.NewBus()
//	bus.AddSink(audit.NewLogger())                        // JSON stream
//	bus.AddSink(audit.NewStoreLogger(auditStore))          // Hash-chained store
//	// All future Record() calls go to both sinks
type Bus struct {
	mu    sync.RWMutex
	sinks []Logger
}

// NewBus creates a new audit event bus with no sinks.
func NewBus() *Bus {
	return &Bus{}
}

// AddSink registers an audit logger as a sink.
// All future Record() calls will also emit to this sink.
// Thread-safe — can be called at any time.
func (b *Bus) AddSink(l Logger) {
	if l == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sinks = append(b.sinks, l)
}

// Record emits an audit event to all registered sinks.
// Fail-closed: returns the first error encountered.
// All sinks are called even if one errors (best-effort fan-out with error reporting).
func (b *Bus) Record(ctx context.Context, eventType EventType, action, resource string, metadata map[string]interface{}) error {
	b.mu.RLock()
	sinks := make([]Logger, len(b.sinks))
	copy(sinks, b.sinks)
	b.mu.RUnlock()

	if len(sinks) == 0 {
		slog.Warn("audit bus has no sinks — event dropped (fail-closed)",
			"action", action,
			"resource", resource,
		)
		return fmt.Errorf("audit: bus has no sinks configured (fail-closed)")
	}

	var firstErr error
	for _, sink := range sinks {
		if err := sink.Record(ctx, eventType, action, resource, metadata); err != nil {
			slog.Error("audit sink error",
				"action", action,
				"error", err,
			)
			if firstErr == nil {
				firstErr = fmt.Errorf("audit: sink error: %w", err)
			}
		}
	}
	return firstErr
}

// SinkCount returns the number of registered sinks.
func (b *Bus) SinkCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.sinks)
}

// Compile-time assertion: Bus implements Logger.
var _ Logger = (*Bus)(nil)
