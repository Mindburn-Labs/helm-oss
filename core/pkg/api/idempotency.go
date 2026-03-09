package api

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

// cachedResponse stores a previously-seen response for idempotent replay.
type cachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	CachedAt   time.Time
}

// IdempotencyStorer defines the interface for idempotency backends.
type IdempotencyStorer interface {
	Check(key string) (*cachedResponse, bool)
	Set(key string, statusCode int, headers http.Header, body []byte) error
}

// MemoryIdempotencyStore holds cached responses keyed by idempotency key (in-memory).
type MemoryIdempotencyStore struct {
	mu      sync.RWMutex
	entries map[string]*cachedResponse
	ttl     time.Duration
}

// NewIdempotencyStore creates a new in-memory idempotency store.
func NewIdempotencyStore(ttl time.Duration) *MemoryIdempotencyStore {
	s := &MemoryIdempotencyStore{
		entries: make(map[string]*cachedResponse),
		ttl:     ttl,
	}
	// Background cleanup of expired entries
	go s.cleanup()
	return s
}

func (s *MemoryIdempotencyStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, v := range s.entries {
			if now.Sub(v.CachedAt) > s.ttl {
				delete(s.entries, k)
			}
		}
		s.mu.Unlock()
	}
}

// Check returns a cached response if existing and valid.
func (s *MemoryIdempotencyStore) Check(key string) (*cachedResponse, bool) {
	s.mu.RLock()
	cached, exists := s.entries[key]
	s.mu.RUnlock()

	if exists && time.Since(cached.CachedAt) < s.ttl {
		return cached, true
	}
	return nil, false
}

// Set stores a response.
func (s *MemoryIdempotencyStore) Set(key string, statusCode int, headers http.Header, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = &cachedResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
		CachedAt:   time.Now(),
	}
	return nil
}

// responseCapture wraps http.ResponseWriter to capture the response.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.statusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	rc.body.Write(b)
	return rc.ResponseWriter.Write(b)
}

// IdempotencyMiddleware ensures that mutating requests with an Idempotency-Key
// header are processed exactly once. Duplicate requests receive the cached response.
func IdempotencyMiddleware(store IdempotencyStorer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to mutating methods
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				// No idempotency key — process normally
				next.ServeHTTP(w, r)
				return
			}

			// Check cache
			// store.mu.RLock() // Interface doesn't have mu
			cached, exists := store.Check(key)
			// store.mu.RUnlock()

			if exists { // check implemented inside store
				// Replay cached response
				for k, vals := range cached.Headers {
					for _, v := range vals {
						w.Header().Set(k, v)
					}
				}
				w.WriteHeader(cached.StatusCode)
				_, _ = w.Write(cached.Body)
				return
			}

			// Capture response
			capture := &responseCapture{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(capture, r)

			// Cache successful responses (2xx) — fail-closed: if cache write fails, return 500
			if capture.statusCode >= 200 && capture.statusCode < 300 {
				if err := store.Set(key, capture.statusCode, w.Header().Clone(), capture.body.Bytes()); err != nil {
					// Idempotency persistence failure — client must retry
					http.Error(w, "idempotency persistence failed", http.StatusInternalServerError)
					return
				}
			}
		})
	}
}
