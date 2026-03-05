package audit

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// AnomalyType categorizes detected anomalies.
type AnomalyType string

const (
	AnomalyBurstActivity    AnomalyType = "BURST_ACTIVITY"
	AnomalyChainDrift       AnomalyType = "CHAIN_DRIFT"
	AnomalyUnusualActor     AnomalyType = "UNUSUAL_ACTOR"
	AnomalyRepeatedFailure  AnomalyType = "REPEATED_FAILURE"
	AnomalyOffHoursActivity AnomalyType = "OFF_HOURS_ACTIVITY"
)

// Anomaly represents a detected anomalous pattern in the audit stream.
type Anomaly struct {
	Type        AnomalyType            `json:"type"`
	Severity    string                 `json:"severity"` // critical, high, medium, low
	Description string                 `json:"description"`
	DetectedAt  time.Time              `json:"detected_at"`
	Evidence    map[string]interface{} `json:"evidence,omitempty"`
}

// AnomalyHandler is called when an anomaly is detected.
type AnomalyHandler func(anomaly Anomaly)

// AnomalyDetector monitors the audit event stream for suspicious patterns.
//
// It uses a sliding window to track:
//   - Burst activity (too many events in a short period)
//   - Repeated failures from the same actor
//   - Off-hours activity
//   - Chain head drift (unexpected resets)
//
// Register it as a Bus sink for continuous monitoring:
//
//	detector := audit.NewAnomalyDetector(audit.AnomalyConfig{...})
//	bus.AddSink(detector)
type AnomalyDetector struct {
	mu       sync.Mutex
	config   AnomalyConfig
	window   []windowEntry
	handlers []AnomalyHandler
	lastHead string
}

// AnomalyConfig configures the detector thresholds.
type AnomalyConfig struct {
	// BurstThreshold: max events per BurstWindow before alerting.
	BurstThreshold int           `json:"burst_threshold"`
	BurstWindow    time.Duration `json:"burst_window"`

	// FailureThreshold: max failures from same actor before alerting.
	FailureThreshold int `json:"failure_threshold"`

	// OffHoursStart/End: UTC hours considered "off hours" (e.g., 22-06).
	OffHoursStart int `json:"off_hours_start"`
	OffHoursEnd   int `json:"off_hours_end"`
}

// DefaultAnomalyConfig returns sensible defaults.
func DefaultAnomalyConfig() AnomalyConfig {
	return AnomalyConfig{
		BurstThreshold:   100,
		BurstWindow:      1 * time.Minute,
		FailureThreshold: 10,
		OffHoursStart:    22,
		OffHoursEnd:      6,
	}
}

type windowEntry struct {
	timestamp time.Time
	actor     string
	action    string
	isFailure bool
}

// NewAnomalyDetector creates a detector with the given config.
func NewAnomalyDetector(config AnomalyConfig) *AnomalyDetector {
	return &AnomalyDetector{
		config: config,
		window: make([]windowEntry, 0, config.BurstThreshold*2),
	}
}

// OnAnomaly registers a handler for detected anomalies.
func (d *AnomalyDetector) OnAnomaly(handler AnomalyHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers = append(d.handlers, handler)
}

// Record implements the Logger interface so the detector can be a Bus sink.
func (d *AnomalyDetector) Record(ctx interface{}, eventType EventType, action, resource string, metadata map[string]interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC()
	actor := ""
	if metadata != nil {
		if a, ok := metadata["actor"].(string); ok {
			actor = a
		}
	}

	isFailure := eventType == EventDeny || eventType == EventViolation

	// Add to sliding window
	entry := windowEntry{
		timestamp: now,
		actor:     actor,
		action:    action,
		isFailure: isFailure,
	}
	d.window = append(d.window, entry)

	// Prune old entries
	cutoff := now.Add(-d.config.BurstWindow)
	pruned := make([]windowEntry, 0, len(d.window))
	for _, e := range d.window {
		if e.timestamp.After(cutoff) {
			pruned = append(pruned, e)
		}
	}
	d.window = pruned

	// --- Check anomalies ---

	// 1. Burst activity
	if len(d.window) >= d.config.BurstThreshold {
		d.emit(Anomaly{
			Type:        AnomalyBurstActivity,
			Severity:    "high",
			Description: fmt.Sprintf("Burst: %d events in %s (threshold: %d)", len(d.window), d.config.BurstWindow, d.config.BurstThreshold),
			DetectedAt:  now,
			Evidence:    map[string]interface{}{"window_size": len(d.window), "last_action": action},
		})
	}

	// 2. Repeated failures from same actor
	if isFailure && actor != "" {
		failCount := 0
		for _, e := range d.window {
			if e.actor == actor && e.isFailure {
				failCount++
			}
		}
		if failCount >= d.config.FailureThreshold {
			d.emit(Anomaly{
				Type:        AnomalyRepeatedFailure,
				Severity:    "critical",
				Description: fmt.Sprintf("Actor %q has %d failures in %s", actor, failCount, d.config.BurstWindow),
				DetectedAt:  now,
				Evidence:    map[string]interface{}{"actor": actor, "failures": failCount},
			})
		}
	}

	// 3. Off-hours activity
	hour := now.Hour()
	if (d.config.OffHoursStart > d.config.OffHoursEnd &&
		(hour >= d.config.OffHoursStart || hour < d.config.OffHoursEnd)) ||
		(d.config.OffHoursStart < d.config.OffHoursEnd &&
			hour >= d.config.OffHoursStart && hour < d.config.OffHoursEnd) {
		d.emit(Anomaly{
			Type:        AnomalyOffHoursActivity,
			Severity:    "medium",
			Description: fmt.Sprintf("Activity at %02d:00 UTC (off-hours: %02d:00-%02d:00)", hour, d.config.OffHoursStart, d.config.OffHoursEnd),
			DetectedAt:  now,
			Evidence:    map[string]interface{}{"actor": actor, "action": action, "hour": hour},
		})
	}

	return nil
}

func (d *AnomalyDetector) emit(anomaly Anomaly) {
	slog.Warn("audit anomaly detected",
		"type", anomaly.Type,
		"severity", anomaly.Severity,
		"description", anomaly.Description,
	)
	for _, h := range d.handlers {
		h(anomaly)
	}
}

// CheckChainDrift detects if the expected chain head changed unexpectedly.
// Call this periodically with the current chain head.
func (d *AnomalyDetector) CheckChainDrift(currentHead string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.lastHead != "" && d.lastHead != currentHead {
		// Could be normal (new entries appended) or abnormal (store reset)
		// Only alert if head went backwards to genesis
		if currentHead == "genesis" && d.lastHead != "genesis" {
			d.emit(Anomaly{
				Type:        AnomalyChainDrift,
				Severity:    "critical",
				Description: fmt.Sprintf("Chain head reset to genesis (was %s) — possible store restart or tampering", d.lastHead[:16]),
				DetectedAt:  time.Now().UTC(),
				Evidence:    map[string]interface{}{"previous_head": d.lastHead, "current_head": currentHead},
			})
		}
	}
	d.lastHead = currentHead
}
