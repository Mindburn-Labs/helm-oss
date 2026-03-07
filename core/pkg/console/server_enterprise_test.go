package console

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/audit"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/auth"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/console/ui"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/governance"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/identity"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/manifest"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/metering"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/pack"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Enterprise Mock Registry
type entMockRegistry struct {
	mock.Mock
}

func (m *entMockRegistry) Register(bundle *manifest.Bundle) error                   { return nil }
func (m *entMockRegistry) Get(name string) (*manifest.Bundle, error)                { return nil, nil }
func (m *entMockRegistry) GetForUser(name, userID string) (*manifest.Bundle, error) { return nil, nil }
func (m *entMockRegistry) SetRollout(name string, canaryBundle *manifest.Bundle, percentage int) error {
	return nil
}
func (m *entMockRegistry) List() []*manifest.Bundle     { return nil }
func (m *entMockRegistry) Unregister(name string) error { return nil }
func (m *entMockRegistry) Install(tenantID, packID string) error {
	args := m.Called(tenantID, packID)
	return args.Error(0)
}

// Enterprise Mock Receipt Store
type entMockReceiptStore struct {
	mock.Mock
}

func (m *entMockReceiptStore) Get(ctx context.Context, decisionID string) (*contracts.Receipt, error) {
	return nil, nil
}
func (m *entMockReceiptStore) GetByReceiptID(ctx context.Context, receiptID string) (*contracts.Receipt, error) {
	return nil, nil
}
func (m *entMockReceiptStore) List(ctx context.Context, limit int) ([]*contracts.Receipt, error) {
	return nil, nil
}
func (m *entMockReceiptStore) Store(ctx context.Context, receipt *contracts.Receipt) error {
	return nil
}

func (m *entMockReceiptStore) GetLastForSession(ctx context.Context, sessionID string) (*contracts.Receipt, error) {
	return nil, nil
}

// Enterprise Mock Meter
type entMockMeter struct{}

func (m *entMockMeter) Record(ctx context.Context, evt metering.Event) error           { return nil }
func (m *entMockMeter) RecordBatch(ctx context.Context, events []metering.Event) error { return nil }
func (m *entMockMeter) Init(ctx context.Context) error                                 { return nil }
func (m *entMockMeter) GetUsage(ctx context.Context, tenantID string, period metering.Period) (*metering.Usage, error) {
	return nil, nil
}
func (m *entMockMeter) GetUsageByType(ctx context.Context, tenantID string, eventType metering.EventType, period metering.Period) (int64, error) {
	return 0, nil
}

func TestEnterpriseFoundation(t *testing.T) {
	// Setup Dependencies
	reg := &entMockRegistry{}
	// led := &ledger.MockLedger{} // Removed usage
	recStore := &entMockReceiptStore{}
	uiAdapter := ui.NewAGUIAdapter(nil)
	meter := &entMockMeter{}
	verifier := &pack.Verifier{}

	pol, _ := governance.NewPolicyEngine()
	// Load test policies
	_ = pol.LoadPolicy("admin_policy", "principal.endsWith('admin-user')")
	_ = pol.LoadPolicy("status_policy", "true") // Allow status check
	audStore := store.NewAuditStore()
	aud := audit.NewStoreLogger(audStore)
	audExporter := audit.NewExporter(audStore)

	srv := &Server{
		// ledger:       led, // Removed
		registry:     reg,
		uiAdapter:    uiAdapter,
		receiptStore: recStore,
		meter:        meter,
		packVerifier: verifier,
		policyEngine: pol,
		auditLogger:  aud,
		auditStore:   audStore,
		auditExport:  audExporter,
		// errorBudget:  100.0,
		// systemStatus: "HEALTHY",
	}

	// create router and wrap
	mux := http.NewServeMux()
	mux.HandleFunc("/api/registry/install", srv.handleRegistryInstallAPI)
	mux.HandleFunc("/api/admin/audit/export", srv.handleAuditExportAPI)

	// Setup Auth with KeySet
	ks, _ := identity.NewInMemoryKeySet()
	validator := auth.NewJWTValidator(ks)

	handler := auth.NewMiddleware(validator)(mux)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()

	// Helper to generate tokens
	genToken := func(subject, tenant string, roles ...string) string {
		claims := auth.HelmClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: subject,
			},
			TenantID: tenant,
			Roles:    roles,
		}
		token, _ := ks.Sign(context.Background(), claims)
		return token
	}

	demoToken := genToken("demo-user", "demo-tenant", "viewer")
	adminToken := genToken("admin-user", "system", "admin")

	t.Run("Auth Enforcement", func(t *testing.T) {
		req, _ := http.NewRequest("POST", server.URL+"/api/registry/install", strings.NewReader(`{}`))
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("Cross-Tenant Denial", func(t *testing.T) {
		// demo-token maps to "demo-tenant"
		reqBody := `{"pack_id":"p1", "tenant_id":"other-tenant"}`
		req, _ := http.NewRequest("POST", server.URL+"/api/registry/install", strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+demoToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("Policy Denial (Export as Viewer)", func(t *testing.T) {
		// demo-token has "viewer" role. Export requires "admin" (assumed from previous logic)
		req, _ := http.NewRequest("GET", server.URL+"/api/admin/audit/export", nil)
		req.Header.Set("Authorization", "Bearer "+demoToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		// Should be 403 due to RBAC/ABAC
		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("Successful Install Denial for Viewer", func(t *testing.T) {
		// demo-token has "viewer" role. Install needs "install" permission (Editor).
		// Wait, demo-token (viewer) should FAIL install too.

		// reg.On("Install", "demo-tenant", "p1").Return(nil) // Should not be called if policy fails

		reqBody := `{"pack_id":"p1", "tenant_id":"demo-tenant"}`
		req, _ := http.NewRequest("POST", server.URL+"/api/registry/install", strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+demoToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Policy check: viewer does NOT have install.
		assert.Equal(t, 403, resp.StatusCode, "Viewer should not be able to install")
	})

	t.Run("Admin Access", func(t *testing.T) {
		// admin-token maps to "system" tenant, "admin" role.
		req, _ := http.NewRequest("GET", server.URL+"/api/admin/audit/export", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)

		ct := resp.Header.Get("Content-Type")
		assert.Equal(t, "application/zip", ct)
	})
}
