package console

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/auth"
)

// TestChaosInjection_AdminOnly verifies CONSOLE-001: Chaos injection requires admin role.
func TestChaosInjection_AdminOnly(t *testing.T) {
	// Setup minimal server
	server := &Server{
		errorBudget:  100.0,
		systemStatus: "HEALTHY",
	}

	tests := []struct {
		name           string
		principal      auth.Principal
		wantStatus     int
		wantBudget     float64
		wantSystemStat string
	}{
		{
			name:       "No Principal (Unauthenticated)",
			principal:  nil,
			wantStatus: http.StatusUnauthorized,
			wantBudget: 100.0,
		},
		{
			name: "Non-Admin User (Forbidden)",
			principal: &auth.BasePrincipal{
				ID:    "user:alice",
				Roles: []string{"viewer", "editor"},
			},
			wantStatus: http.StatusForbidden,
			wantBudget: 100.0,
		},
		{
			name: "Admin User (Allowed)",
			principal: &auth.BasePrincipal{
				ID:    "user:admin",
				Roles: []string{"admin"},
			},
			wantStatus:     http.StatusOK,
			wantBudget:     70.0,      // 100 - 30
			wantSystemStat: "HEALTHY", // > 50 so still healthy
		},
		{
			name: "Admin User (Degrades System)",
			principal: &auth.BasePrincipal{
				ID:    "user:godmode",
				Roles: []string{"admin"},
			},
			wantStatus:     http.StatusOK, // Applied repeatedly to drop below 50
			wantBudget:     40.0,          // 70 - 30
			wantSystemStat: "DEGRADED",    // < 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Request
			req := httptest.NewRequest("POST", "/api/ops/chaos/inject", nil)

			// Inject Principal if present
			if tt.principal != nil {
				ctx := auth.WithPrincipal(req.Context(), tt.principal)
				req = req.WithContext(ctx)
			}

			// Capture Response
			w := httptest.NewRecorder()

			// Execute
			server.handleChaosInjectAPI(w, req)

			// Verify Status
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			// If success, verify internal state changes
			if tt.wantStatus == http.StatusOK {
				server.mu.RLock()
				if server.errorBudget != tt.wantBudget {
					t.Errorf("budget = %f, want %f", server.errorBudget, tt.wantBudget)
				}
				if tt.wantSystemStat != "" && server.systemStatus != tt.wantSystemStat {
					t.Errorf("systemStatus = %s, want %s", server.systemStatus, tt.wantSystemStat)
				}
				server.mu.RUnlock()

				// Verify JSON response
				var resp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if status, ok := resp["status"].(string); !ok || status != "chaos_injected" {
					t.Errorf("response status = %v, want chaos_injected", resp["status"])
				}
			}
		})
	}
}
