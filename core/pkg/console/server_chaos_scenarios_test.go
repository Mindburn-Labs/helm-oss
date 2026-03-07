package console

import (
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/auth"
)

// TEST-004: Chaos testing scenarios for the console server.
// These tests verify that the server handles concurrent and malformed
// requests without panicking or corrupting state.

func TestChaos_ConcurrentChaosInjection(t *testing.T) {
	srv := &Server{}

	// Slam the chaos endpoint concurrently to verify no data races.
	// Run with -race flag to detect race conditions.
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/api/ops/chaos/inject", nil)
			ctx := auth.WithPrincipal(req.Context(), &auth.BasePrincipal{
				ID:    "admin-concurrent",
				Roles: []string{"admin"},
			})
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()
			srv.handleChaosInjectAPI(w, req)
		}()
	}
	wg.Wait()
}

func TestChaos_UnauthenticatedChaos(t *testing.T) {
	srv := &Server{}

	// Unauthenticated chaos requests should return 401, not panic
	req := httptest.NewRequest("POST", "/api/ops/chaos/inject", nil)
	w := httptest.NewRecorder()
	srv.handleChaosInjectAPI(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestChaos_NonAdminChaos(t *testing.T) {
	srv := &Server{}

	// Non-admin chaos requests should return 403
	req := httptest.NewRequest("POST", "/api/ops/chaos/inject", nil)
	ctx := auth.WithPrincipal(req.Context(), &auth.BasePrincipal{
		ID:    "non-admin",
		Roles: []string{"viewer"},
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	srv.handleChaosInjectAPI(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestChaos_MalformedChaosRequest(t *testing.T) {
	srv := &Server{}

	// Admin with wrong HTTP method — should still not panic
	methods := []string{"GET", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/ops/chaos/inject", nil)
			ctx := auth.WithPrincipal(req.Context(), &auth.BasePrincipal{
				ID:    "chaos-admin",
				Roles: []string{"admin"},
			})
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			// Must not panic regardless of method
			srv.handleChaosInjectAPI(w, req)

			// Verify valid JSON response
			var result map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil && w.Body.Len() > 0 {
				t.Errorf("%s: response is not valid JSON: %v", method, err)
			}
		})
	}
}
