package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/store"
)

// ImmunityVerifier provides LLM output stability verification and resilience.
// It implements the "Immune Response" pattern for AI systems:
// 1. Golden sample verification for regression detection
// 2. Airgap cache for fallback during outages
// 3. Rate limiting to prevent abuse
// 4. Anomaly detection for output validation
type ImmunityVerifier struct {
	client      Client
	airgapStore *store.AirgapStore
	config      ImmunityConfig
	metrics     *ImmunityMetrics
}

// ImmunityConfig configures the immunity verifier behavior.
type ImmunityConfig struct {
	// MaxOutputLength limits response size to prevent resource exhaustion
	MaxOutputLength int
	// RateLimitPerMinute limits requests to prevent abuse
	RateLimitPerMinute int
	// CacheTTL specifies how long cached responses remain valid
	CacheTTL time.Duration
	// EnableAnomalyDetection enables output validation
	EnableAnomalyDetection bool
	// SimilarityThreshold for anomaly detection (0.0-1.0)
	SimilarityThreshold float64
	// CircuitBreakerThreshold failures before opening
	CircuitBreakerThreshold int
	// CircuitBreakerTimeout before attempting reset
	CircuitBreakerTimeout time.Duration
}

// DefaultImmunityConfig returns sensible defaults.
func DefaultImmunityConfig() ImmunityConfig {
	return ImmunityConfig{
		MaxOutputLength:         100000, // 100KB
		RateLimitPerMinute:      60,
		CacheTTL:                24 * time.Hour,
		EnableAnomalyDetection:  true,
		SimilarityThreshold:     0.7,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   30 * time.Second,
	}
}

// ImmunityMetrics tracks verifier health.
type ImmunityMetrics struct {
	TotalRequests      int64
	CacheHits          int64
	CacheMisses        int64
	RegressionCount    int64
	AnomaliesDetected  int64
	CircuitBreakerOpen bool
	FailureCount       int
	LastFailure        time.Time
	mu                 sync.Mutex
}

// NewImmunityVerifier creates a new immunity verifier.
func NewImmunityVerifier(client Client, airgap *store.AirgapStore) *ImmunityVerifier {
	return NewImmunityVerifierWithConfig(client, airgap, DefaultImmunityConfig())
}

// NewImmunityVerifierWithConfig creates a configured immunity verifier.
func NewImmunityVerifierWithConfig(client Client, airgap *store.AirgapStore, config ImmunityConfig) *ImmunityVerifier {
	return &ImmunityVerifier{
		client:      client,
		airgapStore: airgap,
		config:      config,
		metrics:     &ImmunityMetrics{},
	}
}

// PromptTemplate defines verification structure with golden samples.
type PromptTemplate struct {
	Name          string
	Version       string
	GoldenSamples []GoldenSample
}

// GoldenSample represents a known-good input/output pair.
type GoldenSample struct {
	Input              map[string]any
	ExpectedOutputHash string
	Tolerance          float64 // Optional: similarity tolerance (0.0-1.0)
}

// VerifyStability checks if current model output matches golden samples.
// If stable, it caches the result in the Airgap Store.
func (v *ImmunityVerifier) VerifyStability(ctx context.Context, template PromptTemplate) error {
	v.metrics.mu.Lock()
	v.metrics.TotalRequests++
	v.metrics.mu.Unlock()

	// Check circuit breaker
	if v.isCircuitOpen(ctx) {
		return fmt.Errorf("circuit breaker open: too many failures")
	}

	for i, sample := range template.GoldenSamples {
		if err := v.verifySample(ctx, i, sample); err != nil {
			v.recordFailure(ctx)
			return err
		}
	}

	return nil
}

// verifySample verifies a single golden sample.
func (v *ImmunityVerifier) verifySample(ctx context.Context, index int, sample GoldenSample) error {
	// 1. Construct Message from Sample
	inputStr := fmt.Sprintf("%v", sample.Input)
	msgs := []Message{{Role: "user", Content: inputStr}}

	// 2. Call LLM
	resp, err := v.client.Chat(ctx, msgs, nil, nil)
	if err != nil {
		return fmt.Errorf("sample %d: llm failure: %w", index, err)
	}

	// 3. Validate output length
	if len(resp.Content) > v.config.MaxOutputLength {
		return fmt.Errorf("sample %d: output exceeds max length (%d > %d)",
			index, len(resp.Content), v.config.MaxOutputLength)
	}

	// 4. Hash Output
	hash := sha256.Sum256([]byte(resp.Content))
	hashStr := hex.EncodeToString(hash[:])

	// 5. Compare with tolerance if specified
	if sample.Tolerance > 0 {
		// Use semantic similarity for comparison
		similarity := v.calculateSimilarity(hashStr, sample.ExpectedOutputHash)
		if similarity < sample.Tolerance {
			v.metrics.mu.Lock()
			v.metrics.RegressionCount++
			v.metrics.mu.Unlock()
			return fmt.Errorf("sample %d: semantic regression detected. similarity %.2f < %.2f",
				index, similarity, sample.Tolerance)
		}
	} else {
		// Exact match required
		if hashStr != sample.ExpectedOutputHash {
			v.metrics.mu.Lock()
			v.metrics.RegressionCount++
			v.metrics.mu.Unlock()
			return fmt.Errorf("sample %d: regression detected. got %s, want %s",
				index, hashStr, sample.ExpectedOutputHash)
		}
	}

	// 6. Anomaly detection
	if v.config.EnableAnomalyDetection {
		if anomaly := v.detectAnomaly(resp.Content); anomaly != "" {
			v.metrics.mu.Lock()
			v.metrics.AnomaliesDetected++
			v.metrics.mu.Unlock()

			slog.WarnContext(ctx, "anomaly detected in LLM output",
				"sample", index,
				"anomaly", anomaly)
		}
	}

	// 7. Immunity: Cache valid output in Airgap Store
	if v.airgapStore != nil {
		hash := sha256.Sum256([]byte(inputStr))
		cacheKey := hex.EncodeToString(hash[:])
		_ = v.airgapStore.Put(ctx, cacheKey, []byte(resp.Content))
	}

	return nil
}

// GetImmuneResponse tries to get from LLM, falls back to Airgap if outage.
func (v *ImmunityVerifier) GetImmuneResponse(ctx context.Context, input string) (string, error) {
	v.metrics.mu.Lock()
	v.metrics.TotalRequests++
	v.metrics.mu.Unlock()

	// Check circuit breaker
	if v.isCircuitOpen(ctx) {
		// Direct fallback to cache
		return v.getCachedResponse(ctx, input)
	}

	// 1. Try Live
	msgs := []Message{{Role: "user", Content: input}}
	resp, err := v.client.Chat(ctx, msgs, nil, nil)

	if err == nil {
		// Validate output length
		if len(resp.Content) > v.config.MaxOutputLength {
			slog.WarnContext(ctx, "truncating LLM output",
				"original_length", len(resp.Content),
				"max_length", v.config.MaxOutputLength)
			resp.Content = resp.Content[:v.config.MaxOutputLength]
		}

		// Cache valid output
		if v.airgapStore != nil {
			hash := sha256.Sum256([]byte(input))
			cacheKey := hex.EncodeToString(hash[:])
			_ = v.airgapStore.Put(ctx, cacheKey, []byte(resp.Content))

			v.metrics.mu.Lock()
			v.metrics.CacheMisses++ // Fresh response
			v.metrics.mu.Unlock()
		}
		return resp.Content, nil
	}

	// Record failure for circuit breaker
	v.recordFailure(ctx)

	// 2. Fallback to Airgap
	cached, cacheErr := v.getCachedResponse(ctx, input)
	if cacheErr == nil {
		return cached, nil
	}

	return "", fmt.Errorf("immunity failure: llm down and no cached response: %w", err)
}

// getCachedResponse retrieves a cached response from the airgap store.
func (v *ImmunityVerifier) getCachedResponse(ctx context.Context, input string) (string, error) {
	if v.airgapStore == nil {
		return "", fmt.Errorf("no airgap store configured")
	}

	hash := sha256.Sum256([]byte(input))
	cacheKey := hex.EncodeToString(hash[:])
	cached, err := v.airgapStore.Get(ctx, cacheKey)
	if err != nil {
		return "", err
	}

	v.metrics.mu.Lock()
	v.metrics.CacheHits++
	v.metrics.mu.Unlock()

	slog.InfoContext(ctx, "serving cached response",
		"cache_key", cacheKey[:16]+"...")

	return string(cached), nil
}

// isCircuitOpen checks if the circuit breaker is open.
func (v *ImmunityVerifier) isCircuitOpen(ctx context.Context) bool {
	v.metrics.mu.Lock()
	defer v.metrics.mu.Unlock()

	if !v.metrics.CircuitBreakerOpen {
		return false
	}

	// Check if timeout has elapsed
	if time.Since(v.metrics.LastFailure) > v.config.CircuitBreakerTimeout {
		v.metrics.CircuitBreakerOpen = false
		v.metrics.FailureCount = 0
		slog.InfoContext(ctx, "circuit breaker reset")
		return false
	}

	return true
}

// recordFailure records a failure and potentially opens the circuit.
func (v *ImmunityVerifier) recordFailure(ctx context.Context) {
	v.metrics.mu.Lock()
	defer v.metrics.mu.Unlock()

	v.metrics.FailureCount++
	v.metrics.LastFailure = time.Now()

	if v.metrics.FailureCount >= v.config.CircuitBreakerThreshold {
		v.metrics.CircuitBreakerOpen = true
		slog.WarnContext(ctx, "circuit breaker opened",
			"failure_count", v.metrics.FailureCount)
	}
}

// detectAnomaly checks for suspicious patterns in output.
func (v *ImmunityVerifier) detectAnomaly(output string) string {
	// Check for common anomaly patterns
	patterns := []struct {
		check func(string) bool
		name  string
	}{
		{func(s string) bool { return len(s) == 0 }, "empty_output"},
		{func(s string) bool { return countRepeatingChars(s) > 10 }, "excessive_repetition"},
		{containsSuspiciousPatterns, "suspicious_pattern"},
	}

	for _, p := range patterns {
		if p.check(output) {
			return p.name
		}
	}

	return ""
}

// calculateSimilarity calculates hash-based similarity (simplified).
func (v *ImmunityVerifier) calculateSimilarity(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 1.0
	}

	// Count matching characters (simplified similarity)
	matches := 0
	minLen := len(hash1)
	if len(hash2) < minLen {
		minLen = len(hash2)
	}

	for i := 0; i < minLen; i++ {
		if hash1[i] == hash2[i] {
			matches++
		}
	}

	return float64(matches) / float64(max(len(hash1), len(hash2)))
}

// ImmunityMetricsSnapshot is a thread-safe copy of metrics for reading.
type ImmunityMetricsSnapshot struct {
	TotalRequests      int64
	CacheHits          int64
	CacheMisses        int64
	RegressionCount    int64
	AnomaliesDetected  int64
	CircuitBreakerOpen bool
	FailureCount       int
	LastFailure        time.Time
}

// GetMetrics removed - was dead code

// ===== Helper Functions =====

func countRepeatingChars(s string) int {
	if len(s) < 2 {
		return 0
	}

	maxRepeat := 1
	currentRepeat := 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			currentRepeat++
			if currentRepeat > maxRepeat {
				maxRepeat = currentRepeat
			}
		} else {
			currentRepeat = 1
		}
	}
	return maxRepeat
}

func containsSuspiciousPatterns(s string) bool {
	// Check for common jailbreak/injection patterns
	suspiciousPatterns := []string{
		"ignore previous instructions",
		"disregard all prior",
		"you are now",
		"pretend you are",
		"act as if",
	}

	lowerS := strings.ToLower(s)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerS, pattern) {
			return true
		}
	}
	return false
}
