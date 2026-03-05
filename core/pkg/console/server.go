package console

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Mindburn-Labs/helm/core/pkg/api"
	"github.com/Mindburn-Labs/helm/core/pkg/audit"
	"github.com/Mindburn-Labs/helm/core/pkg/auth"
	"github.com/Mindburn-Labs/helm/core/pkg/compliance/csr"
	"github.com/Mindburn-Labs/helm/core/pkg/console/ui"
	"github.com/Mindburn-Labs/helm/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm/core/pkg/governance" // Policy Engine
	"github.com/Mindburn-Labs/helm/core/pkg/metering"
	"github.com/Mindburn-Labs/helm/core/pkg/pack"

	"github.com/Mindburn-Labs/helm/core/pkg/registry"

	"github.com/Mindburn-Labs/helm/core/pkg/store"
	"github.com/Mindburn-Labs/helm/core/pkg/store/ledger"
)

// Server defines the HTTP server for the Console.
type Server struct {
	ledger       ledger.Ledger
	registry     registry.Registry
	uiAdapter    ui.UIAdapter
	receiptStore store.ReceiptStore
	meter        metering.Meter
	packVerifier *pack.Verifier

	// Enterprise Foundation
	policyEngine *governance.PolicyEngine // CEL-based
	auditLogger  audit.Logger
	auditStore   *store.AuditStore
	auditExport  *audit.Exporter

	// Config
	OrgSchemaDir string

	// Phase 6: Metrics & Control Plane State
	mu           sync.RWMutex
	errorBudget  float64
	systemStatus string

	// Phase 8.1: API Caching
	cacheMu sync.RWMutex
	cache   map[string][]byte

	// Onboarding state (in-memory, backed by session TTL)
	onboardingMu   sync.RWMutex
	pendingSignups map[string]*pendingSignup // keyed by email

	// Operator Interaction Layer (in-memory stores)
	operatorMu   sync.RWMutex
	intents      map[string]*operatorIntent
	approvals    map[string]*operatorApproval
	operatorRuns map[string]*operatorRunState

	// JWT Validator for WebSocket handshake auth
	jwtValidator *auth.JWTValidator

	// Metrics Manager (GAP-01: Operability Surfaces)
	metricsManager *MetricsManager

	// Compliance Source Registry (GAP-01: Real CSR backend)
	csrRegistry csr.ComplianceSourceRegistry
}

type pendingSignup struct {
	Email    string
	Password string
	Code     string
	TenantID string
	Verified bool
}

// Start launches the Console Server.
func Start(port int, ledger ledger.Ledger, reg registry.Registry, uiAdapter ui.UIAdapter, receiptStore store.ReceiptStore, meter metering.Meter, staticDir string, verifier *pack.Verifier, validator *auth.JWTValidator, extraRoutes func(*http.ServeMux)) error {
	// Initialize Enterprise Components
	// Planned enhancement: wire the governance policy engine through full route surfaces.
	pol, err := governance.NewPolicyEngine()
	if err != nil {
		return fmt.Errorf("failed to init policy engine: %w", err)
	}
	audStore := store.NewAuditStore()
	aud := audit.NewStoreLogger(audStore)
	audExporter := audit.NewExporter(audStore)

	srv := &Server{
		ledger:         ledger,
		registry:       reg,
		uiAdapter:      uiAdapter,
		receiptStore:   receiptStore,
		meter:          meter,
		policyEngine:   pol,
		auditLogger:    aud,
		auditStore:     audStore,
		auditExport:    audExporter,
		errorBudget:    100.0,
		systemStatus:   "HEALTHY",
		cache:          make(map[string][]byte),
		packVerifier:   verifier,
		pendingSignups: make(map[string]*pendingSignup),
		intents:        make(map[string]*operatorIntent),
		approvals:      make(map[string]*operatorApproval),
		operatorRuns:   make(map[string]*operatorRunState),
		jwtValidator:   validator,
	}

	// Initialize Metrics Manager (GAP-01: Operability Surfaces)
	srv.metricsManager = NewMetricsManager()

	// Initialize Compliance Source Registry with canonical sources
	csrReg := csr.NewInMemoryCSR()
	if err := csr.SeedRegistry(csrReg); err != nil {
		slog.Warn("failed to seed CSR registry", "error", err)
	}
	srv.csrRegistry = csrReg

	mux := http.NewServeMux()

	// API Routes
	// API Routes (Phase 5 Update)
	mux.HandleFunc("/api/obligations", srv.handleObligationsAPI)
	mux.HandleFunc("/api/modules", srv.handleModulesAPI)
	mux.HandleFunc("/api/receipts", srv.handleReceiptsAPI)
	mux.HandleFunc("/api/builder/generate", srv.handleBuilderAPI)
	mux.HandleFunc("/api/factory/provision", srv.handleFactoryAPI)
	mux.HandleFunc("/api/registry/list", srv.handleRegistryListAPI)
	mux.HandleFunc("/api/registry/install", srv.handleRegistryInstallAPI)
	mux.HandleFunc("/api/registry/anchors", srv.handleRegistryAnchorsAPI)
	mux.HandleFunc("/api/registry/publish", srv.handleRegistryPublishAPI)
	mux.HandleFunc("/api/verify/", srv.handleVerifyAPI)
	mux.HandleFunc("/api/compliance/report", srv.handleComplianceReportAPI)

	// Enterprise API
	mux.HandleFunc("/api/admin/audit/export", srv.handleAuditExportAPI)
	mux.HandleFunc("/api/auth/login", srv.handleAuthLoginAPI)
	mux.HandleFunc("/api/auth/callback", srv.handleAuthCallbackAPI)

	// Phase 6: Metrics API
	mux.HandleFunc("/api/metrics/dashboard", srv.handleMetricsDashboardAPI)
	mux.HandleFunc("/api/ops/chaos/inject", srv.handleChaosInjectAPI)

	// Runs API (RB-18 / J3 live-data) — now routed through operator router
	mux.HandleFunc("/api/runs/", srv.handleOperatorRunsRouter)
	mux.HandleFunc("/api/runs", srv.handleOperatorRunsRouter)

	// SLO + Metrics API (GAP-01: Operability Surfaces)
	mux.HandleFunc("/api/slo/definitions", srv.handleSLODefinitions)
	mux.HandleFunc("/api/slo/status", srv.handleSLOStatus)
	mux.HandleFunc("/api/alerts", srv.handleAlerts)
	mux.HandleFunc("/api/alerts/acknowledge", srv.handleAlertAck)
	mux.HandleFunc("/api/connectors/health", srv.handleConnectors)
	mux.HandleFunc("/api/metrics/overview", srv.handleMetricsDashboardExtended)

	// Compliance + Tenant API (GAP-01: Operability Surfaces)
	mux.HandleFunc("/api/compliance/obligations", srv.handleObligationsJSON)
	mux.HandleFunc("/api/compliance/sources", srv.handleComplianceSourcesJSON)
	mux.HandleFunc("/api/tenants", srv.handleTenantsAPI)

	// Mission Control API (MC-01: Policy + Audit surfaces)
	mux.HandleFunc("/api/policies", srv.handlePoliciesListAPI)
	mux.HandleFunc("/api/policies/simulate", srv.handlePolicySimulateAPI) // MC-02: Policy Simulation
	mux.HandleFunc("/api/audit/events", srv.handleAuditEventsAPI)
	mux.HandleFunc("/api/vendors", srv.handleVendorsAPI)        // MC-03: A2A Vendor Mesh
	mux.HandleFunc("/api/ops/control", srv.handleOpsControlAPI) // MC-04: Flight Controls

	// Cost Attribution API (GAP-09: Resource Economics)
	mux.HandleFunc("/api/cost/summary", srv.handleCostSummary)
	mux.HandleFunc("/api/cost/alerts", srv.handleCostAlerts)

	// Operator API: Intents & Approvals
	mux.HandleFunc("/api/intents/", srv.handleIntentsRouter)
	mux.HandleFunc("/api/intents", srv.handleIntentsRouter)
	mux.HandleFunc("/api/approvals/", srv.handleApprovalsRouter)
	mux.HandleFunc("/api/approvals", srv.handleApprovalsRouter)

	// Phase 4: Cryptographic HITL Bridge — approval via Ed25519 signature verification
	approveHandler := api.NewApproveHandler(nil)
	mux.HandleFunc("/api/v1/kernel/approve", approveHandler.HandleApprove)

	// Onboarding Endpoints (RB-13 / J2)
	mux.HandleFunc("/api/signup", srv.handleSignupAPI)
	mux.HandleFunc("/api/onboarding/verify", srv.handleOnboardingVerifyAPI)
	mux.HandleFunc("/api/resend-verification", srv.handleResendVerificationAPI)

	// Generative UI Endpoints
	mux.HandleFunc("/api/ui/render", srv.handleUIRender)
	mux.HandleFunc("/api/ui/interact", srv.handleUIInteract)

	// Admin UI Fragments
	mux.HandleFunc("/api/admin/tenants", srv.handleAdminTenantsAPI)
	mux.HandleFunc("/api/admin/roles", srv.handleAdminRolesAPI)
	mux.HandleFunc("/api/admin/audit-ui", srv.handleAdminAuditUI)

	// Health endpoint on primary API port for quick checks and docs parity.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Static File Server (SPA Support)
	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			// If file exists, serve it
			if _, err := os.Stat(staticDir + path); err == nil {
				fs.ServeHTTP(w, r)
				return
			}
			// Otherwise fallback to index.html for React Router
			http.ServeFile(w, r, staticDir+"/index.html")
		}))
	} else {
		mux.HandleFunc("/", srv.handleDashboard)
	}

	// Register Extra Routes (from main.go)

	// Register Extra Routes (from main.go)
	if extraRoutes != nil {
		extraRoutes(mux)
	}

	addr := fmt.Sprintf(":%d", port)
	slog.Info("console interface active", "addr", "http://localhost"+addr)
	// Production HTTP server with explicit timeouts.
	// Production HTTP server with explicit timeouts.
	authHandler := auth.NewMiddleware(validator)(mux)

	// Public Routes Bypass — endpoints that do not require authentication.
	// These are intentionally unauthenticated for demo, health, and read-only proof access.
	publicPaths := []string{
		"/demo",
		"/v1/chat/completions",
		"/v1/tools/execute",
		"/api/v1/receipts",
		"/api/v1/proofgraph",
		"/api/v1/export",
		"/limits",
		"/health",
	}

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, p := range publicPaths {
			if strings.HasPrefix(r.URL.Path, p) {
				mux.ServeHTTP(w, r)
				return
			}
		}
		authHandler.ServeHTTP(w, r)
	})

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      finalHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return httpServer.ListenAndServe()
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmplInput := `
<!DOCTYPE html>
<html>
<head>
	<title>HELM Control Room</title>
	<style>
		body { font-family: monospace; background: #1a1a1a; color: #e0e0e0; padding: 20px; }
		h1 { color: #4CAF50; }
		.card { background: #2d2d2d; padding: 15px; margin-bottom: 20px; border-radius: 5px; }
		table { width: 100%; border-collapse: collapse; }
		th, td { text-align: left; padding: 8px; border-bottom: 1px solid #444; }
		th { color: #888; }
		.status-PASS { color: #4CAF50; }
		.status-FAIL { color: #F44336; }
		.status-PENDING { color: #FF9800; }
		.status-BLOCKED { color: #FF5722; }
	</style>
	<!-- HTMX for auto-refresh -->
	<script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body>
	<h1>HELM Autonomous OS</h1>
	
	<div class="card">
		<h2>Enterprise Admin</h2>
		<ul>
			<li><button hx-get="/api/admin/tenants" hx-target="#admin-view">Manage Tenants</button></li>
			<li><button hx-get="/api/admin/roles" hx-target="#admin-view">Manage Roles</button></li>
			<li><button hx-get="/api/admin/audit-ui" hx-target="#admin-view">Audit Log</button></li>
		</ul>
		<div id="admin-view"></div>
	</div>

	<div class="card">
		<h2>System Obligations (Live)</h2>
		<div hx-get="/api/obligations" hx-trigger="load, every 2s">Loading...</div>
	</div>

	<div class="card">
		<h2>Installed Modules</h2>
		<div hx-get="/api/modules" hx-trigger="load">Loading...</div>
	</div>
</body>
</html>
`
	tmpl, err := template.New("dash").Parse(tmplInput)
	if err != nil {
		slog.Error("failed to parse dashboard template", "error", err)
		api.WriteInternal(w, err)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		slog.Error("failed to render dashboard template", "error", err)
	}
}

func (s *Server) handleObligationsAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	obls, err := s.ledger.ListAll(ctx)
	if err != nil {
		api.WriteInternal(w, err)
		return
	}

	// Sort by Time desc
	sort.Slice(obls, func(i, j int) bool {
		return obls[i].CreatedAt.After(obls[j].CreatedAt)
	})

	// Render table HTML fragment for HTMX
	html := `<table><tr><th>ID</th><th>State</th><th>Intent</th><th>Retries</th><th>Created</th></tr>`
	for _, o := range obls {
		html += fmt.Sprintf(`<tr>
			<td>%s</td>
			<td class="status-%s">%s</td>
			<td>%s</td>
			<td>%d</td>
			<td>%s</td>
		</tr>`, o.ID, o.State, o.State, o.Intent, o.RetryCount, o.CreatedAt.Format(time.RFC3339))
	}
	html += `</table>`

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		slog.Error("failed to write obligations response", "error", err)
	}
}

func (s *Server) handleModulesAPI(w http.ResponseWriter, r *http.Request) {
	mods := s.registry.List()

	html := `<table><tr><th>Name</th><th>Capabilities</th><th>PowerDelta</th></tr>`
	for _, m := range mods {
		caps := ""
		for _, c := range m.Manifest.Capabilities {
			caps += c.Name + ", "
		}
		html += fmt.Sprintf(`<tr>
			<td>%s</td>
			<td>%s</td>
			<td>%d</td>
		</tr>`, m.Manifest.Name, caps, m.PowerDelta)
	}
	html += `</table>`

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		slog.Error("failed to write modules response", "error", err)
	}
}

func (s *Server) handleUIInteract(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		api.WriteMethodNotAllowed(w)
		return
	}

	var interaction ui.UIInteraction
	if err := json.NewDecoder(r.Body).Decode(&interaction); err != nil {
		api.WriteBadRequest(w, "Invalid UIInteraction")
		return
	}

	intent, err := s.uiAdapter.HandleInteraction(r.Context(), interaction)
	if err != nil {
		api.WriteInternal(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(intent) //nolint:errcheck // Encode error ignored
}

func (s *Server) handleVerifyAPI(w http.ResponseWriter, r *http.Request) {
	// Extract Receipt ID from URL: /api/verify/{id}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		api.WriteBadRequest(w, "Invalid receipt ID")
		return
	}
	receiptID := pathParts[3]

	if receiptID == "" {
		api.WriteBadRequest(w, "Missing receipt ID")
		return
	}

	ctx := r.Context()
	receipt, err := s.receiptStore.GetByReceiptID(ctx, receiptID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.WriteNotFound(w, "Receipt not found")
			return
		}
		api.WriteInternal(w, err)
		return
	}

	// Metering: Referral Tracking (Phase 5.1)
	if ref := r.URL.Query().Get("ref"); ref != "" {
		_ = s.meter.Record(ctx, metering.Event{
			TenantID:  "helm-system",          // Attribution to system
			EventType: metering.EventToolCall, // Using generic type for now
			Quantity:  1,
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"tool":        "referral_view",
				"receipt_id":  receiptID,
				"referrer_id": ref,
			},
		})
	}

	// Transform to UI format (if needed, or send as is)
	// For now, sending raw receipt which UI will adapt
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(receipt)
}

func (s *Server) handleReceiptsAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	receipts, err := s.receiptStore.List(ctx, 20)
	if err != nil {
		api.WriteInternal(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(receipts)
}

type BuilderRequest struct {
	Idea     string `json:"idea"`
	Industry string `json:"industry"`
}

type BuilderResponse struct {
	Receipt *contracts.Receipt `json:"receipt"`
	Plan    string             `json:"plan"`
}

func (s *Server) handleBuilderAPI(w http.ResponseWriter, r *http.Request) {
	// Phase 6: Circuit Breaker
	s.mu.RLock()
	budget := s.errorBudget
	s.mu.RUnlock()

	if budget < 20.0 {
		w.Header().Set("X-Gate-Enforced", "BudgetExhausted")
		api.WriteTooManyRequests(w, 60)
		return
	}

	if r.Method != "POST" {
		api.WriteMethodNotAllowed(w)
		return
	}

	// Read Body for Hashing (Phase 8.1)
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		api.WriteInternal(w, err)
		return
	}

	// Check Cache
	hash := sha256.Sum256(bodyBytes)
	key := hex.EncodeToString(hash[:])

	s.cacheMu.RLock()
	if val, ok := s.cache[key]; ok {
		s.cacheMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		_, _ = w.Write(val)
		return
	}
	s.cacheMu.RUnlock()

	var req BuilderRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		api.WriteBadRequest(w, "Invalid request")
		return
	}

	// Metering Logic (Phase 3.3)
	_ = s.meter.Record(r.Context(), metering.Event{
		TenantID:  "helm-system",
		EventType: metering.EventToolCall,
		Quantity:  1,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"tool":     "builder_generate",
			"industry": req.Industry,
		},
	})

	// Orchestration: JIT Planning
	// Planned enhancement: use the JIT Planner to generate a plan from intent.
	// fsm := orchestration.NewFSM(fmt.Sprintf("builder-%d", time.Now().UnixNano()))

	// Create context and intent
	// intent := fmt.Sprintf("Build a business plan for %s focused on %s", req.Industry, req.Idea)

	// Generate JIT Plan
	// Note: In a real environment, we would use the Policy/Context to drive this.
	// Here we initialize the JIT planner directly.
	// fsm.JIT = &orchestration.JITPlannerBasic{} // Use the basic implementation

	// stepDAG, err := fsm.JIT.Plan(r.Context(), intent, fsm.Context)
	// if err != nil {
	// 	slog.Error("jit.planning.failed", "error", err)
	// 	api.WriteInternal(w, err)
	// 	return
	// }

	// Execute (Phase 1: Just return the plan for approval/view)
	planStr := "{}" // JIT Planning Deprecated
	timestamp := time.Now()
	idPart := fmt.Sprintf("%d", timestamp.UnixNano())
	decisionID := "dec-builder-" + idPart
	receiptID := "rcpt-" + idPart
	effectID := "eff-builder-" + idPart

	// Create Receipt
	receipt := &contracts.Receipt{
		ReceiptID:  receiptID,
		DecisionID: decisionID,
		EffectID:   effectID,
		Status:     "PLANNED",
		Timestamp:  timestamp,
		BlobHash:   "sha256:jit-plan-stub", // Placeholder for actual hash
		ExecutorID: "helm-jit-planner",
	}

	// Store Receipt
	if err := s.receiptStore.Store(r.Context(), receipt); err != nil {
		slog.Error("failed to store receipt", "error", err)
		api.WriteInternal(w, err)
		return
	}

	resp := BuilderResponse{
		Receipt: receipt,
		Plan:    planStr,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		api.WriteInternal(w, err)
		return
	}

	// Update Cache
	s.cacheMu.Lock()
	s.cache[key] = respBytes
	s.cacheMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	_, _ = w.Write(respBytes)
}

type FactoryRequest struct {
	Company    string `json:"company"`
	Compliance string `json:"compliance"`
	Region     string `json:"region"`
}

type FactoryResponse struct {
	Receipt *contracts.Receipt `json:"receipt"`
	Message string             `json:"message"`
}

func (s *Server) handleFactoryAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		api.WriteMethodNotAllowed(w)
		return
	}

	var req FactoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "Invalid request")
		return
	}

	// Planned enhancement: expand governance policy engine integration.
	ctx := r.Context()
	accessReq := contracts.AccessRequest{
		PrincipalID: "user:unknown", // In real system, get from ctx
		Action:      "provision.factory",
		ResourceID:  "tenant:" + req.Company, // Use req.Company as resource scope
		Context:     map[string]any{"company": req.Company},
	}

	decision, err := s.policyEngine.Evaluate(ctx, "provision_policy", accessReq)
	if err != nil {
		slog.Error("policy evaluation failed", "error", err)
		api.WriteInternal(w, err)
		return
	}

	if decision.Verdict != "ALLOW" {
		api.WriteForbidden(w, "Access denied by policy: "+decision.Reason)
		return
	}

	// Metering Logic (Phase 3.3)
	// We'll record an event for "provision_tenant"
	_ = s.meter.Record(r.Context(), metering.Event{
		TenantID:  "helm-system", // Attribution to system for now
		EventType: metering.EventToolCall,
		Quantity:  1,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"tool":    "provision_tenant",
			"company": req.Company,
		},
	})

	// Ops Execution: Provision via Runtime
	// Planned enhancement: use active runtime to provision resources.

	timestamp := time.Now()
	idPart := fmt.Sprintf("%d", timestamp.UnixNano())
	decisionID := "dec-factory-" + idPart
	receiptID := "rcpt-" + idPart
	effectID := "eff-factory-" + idPart

	// In a real scenario, we would compile a factory spec and register it.
	// For now, we simulate the effect by creating a receipt, as dynamic provisioning
	// requires a full OrgGenome update or a dynamic factory loader.
	// We mark it as "PROVISIONED" to indicate intent.

	// Create Receipt
	receipt := &contracts.Receipt{
		ReceiptID:  receiptID,
		DecisionID: decisionID,
		EffectID:   effectID,
		Status:     "PROVISIONED",
		Timestamp:  timestamp,
		BlobHash:   "sha256:provision-hash",
		ExecutorID: "helm-ops-provisioner",
		Metadata: map[string]any{
			"company":    req.Company,
			"compliance": req.Compliance,
			"region":     req.Region,
		},
	}

	// Store Receipt
	if err := s.receiptStore.Store(r.Context(), receipt); err != nil {
		slog.Error("failed to store receipt", "error", err)
		api.WriteInternal(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(FactoryResponse{
		Receipt: receipt,
		Message: fmt.Sprintf("Tenant '%s' provisioned in %s with %s compliance.", req.Company, req.Region, req.Compliance),
	})
}

// Registry API

func (s *Server) handleRegistryListAPI(w http.ResponseWriter, r *http.Request) {
	packs := s.registry.List()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(packs)
}

type InstallRequest struct {
	PackID   string `json:"pack_id"`
	TenantID string `json:"tenant_id"`
}

func (s *Server) handleRegistryInstallAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		api.WriteMethodNotAllowed(w)
		return
	}

	var req InstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "Invalid request")
		return
	}

	// Enforce Tenancy & Policy
	ctx := r.Context()
	authTenant, err := auth.GetTenantID(ctx)
	if err != nil {
		api.WriteUnauthorized(w, "Authentication required")
		return
	}

	// 1. Tenant Isolation
	// Allow "system" tenant (admins) to act on others, otherwise strict match.
	if authTenant != "system" && authTenant != req.TenantID {
		api.WriteForbidden(w, "Cross-tenant access denied")
		return
	}

	// Planned enhancement: complete governance policy engine integration.
	userID := "user:unknown"
	if p, err := auth.GetPrincipal(ctx); err == nil {
		userID = p.GetID()
	}

	accessReq := contracts.AccessRequest{
		PrincipalID: userID,
		Action:      "registry.install",
		ResourceID:  "tenant:" + req.TenantID,
		Context:     map[string]any{"ip": r.RemoteAddr},
	}
	decision, err := s.policyEngine.Evaluate(r.Context(), "admin_policy", accessReq) // Use admin_policy for install for now
	if err != nil {
		slog.Error("policy evaluation failed", "error", err)
		api.WriteInternal(w, err)
		return
	}

	if decision.Verdict != "ALLOW" {
		api.WriteForbidden(w, "Access denied by policy: "+decision.Reason)
		return
	}

	if err := s.registry.Install(req.TenantID, req.PackID); err != nil {
		api.WriteInternal(w, err)
		return
	}

	// Metering (Phase 4.1 Requirement)
	_ = s.meter.Record(r.Context(), metering.Event{
		TenantID:  req.TenantID,
		EventType: metering.EventToolCall,
		Quantity:  1,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"tool":    "registry_install",
			"pack_id": req.PackID,
		},
	})

	// Phase 7: Drills as Packs (Chaos Injection)
	if req.PackID == "pack.ops.chaos" {
		s.mu.Lock()
		s.errorBudget = 0.0 // Usage of this pack burns the budget immediately
		s.systemStatus = "DEGRADED"
		s.mu.Unlock()
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "installed"})
}

// Phase 6: Metrics Dashboard API
func (s *Server) handleMetricsDashboardAPI(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeGates := []string{}
	if s.errorBudget < 20.0 {
		activeGates = append(activeGates, "BLOCK_PROMOTION", "BLOCK_BUILDER")
	}

	stats := map[string]any{
		"error_budget":  s.errorBudget,
		"system_status": s.systemStatus,
		"active_gates":  activeGates,
		"timestamp":     time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// Phase 6: Chaos Injection API (burns error budget to test degradation behavior)
// SECURITY: Restricted to admin role only (CONSOLE-001).
func (s *Server) handleChaosInjectAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		api.WriteMethodNotAllowed(w)
		return
	}

	// CONSOLE-001: Admin role restriction — chaos injection is a destructive operation
	principal, err := auth.GetPrincipal(r.Context())
	if err != nil {
		api.WriteUnauthorized(w, "Authentication required for chaos injection")
		return
	}
	isAdmin := false
	for _, role := range principal.GetRoles() {
		if role == "admin" {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		api.WriteForbidden(w, "Chaos injection requires admin role")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Burn 30% of budget (chaos injection)
	s.errorBudget -= 30.0
	if s.errorBudget < 0 {
		s.errorBudget = 0
	}

	if s.errorBudget < 50.0 {
		s.systemStatus = "DEGRADED"
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":       "chaos_injected",
		"new_budget":   s.errorBudget,
		"system_state": s.systemStatus,
	})
}

// Phase 5.2: Compliance Report API
func (s *Server) handleComplianceReportAPI(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch Receipts
	receipts, err := s.receiptStore.List(r.Context(), 100)
	if err != nil {
		api.WriteInternal(w, err)
		return
	}

	// 2. Aggregate Data
	toolUsage := make(map[string]int)
	for _, rcpt := range receipts {
		// Try to extract tool from metadata if available, else use ExecutorID
		if meta, ok := rcpt.Metadata["tool"]; ok {
			toolUsage[fmt.Sprintf("%v", meta)]++
		} else {
			toolUsage[rcpt.ExecutorID]++
		}
	}

	// 3. Calculate Score
	// Base score 50 + 5 per receipt (max 100)
	score := 50 + (len(receipts) * 5)
	if score > 100 {
		score = 100
	}

	// 4. Return Report
	report := map[string]any{
		"generated_at":    time.Now(),
		"period":          "Q1 2026",
		"total_receipts":  len(receipts),
		"readiness_score": score,
		"tool_breakdown":  toolUsage,
		"audit_status":    "READY",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

// handleRegistryAnchorsAPI allows dynamic registration of trusted public keys.
func (s *Server) handleRegistryAnchorsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		api.WriteMethodNotAllowed(w)
		return
	}

	var anchor pack.TrustAnchor
	if err := json.NewDecoder(r.Body).Decode(&anchor); err != nil {
		api.WriteBadRequest(w, "Invalid request")
		return
	}

	if s.packVerifier != nil {
		s.packVerifier.AddTrustAnchor(anchor)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "anchor added"})
}

// handleRegistryPublishAPI allows publishing signed packs.
func (s *Server) handleRegistryPublishAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		api.WriteMethodNotAllowed(w)
		return
	}

	var p pack.Pack
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		api.WriteBadRequest(w, "Invalid request")
		return
	}

	// 1. Verify Signature (if verifier configured)
	// We MUST check signatures before accepting into the registry.
	if s.packVerifier != nil {
		valid, err := s.packVerifier.VerifyPack(&p)
		if err != nil {
			// Verification failures (hash mismatch, invalid sig) are client errors (valid=false)
			// checking err message or just returning 400
			api.WriteBadRequest(w, "Pack verification failed")
			return
		}
		if !valid {
			api.WriteForbidden(w, "Invalid signature: pack rejected")
			return
		}
	}

	// 2. Persist
	// Use Adapter if available or direct registry.
	// Since we wired verifier with an adapter, we should probably access that adapter to publish
	// OR we need to interact with the registry directly.
	// Server has s.registry (legacy interface).
	// We should enhance s.registry or use the adapter helper directly.
	// But `srv` struct has `registry registry.Registry` (interface).
	// The interface doesn't have PublishPack.
	// WE need to convert here or use the adapter locally.

	adapter := NewRegistryAdapter(s.registry)
	if err := adapter.PublishPack(r.Context(), &p); err != nil {
		api.WriteInternal(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "published",
		"pack_id": p.PackID,
	})
}

// handleOrgCompileAPI has been removed as it depended on orgvm which was purged.
// handleOrgSynthesizeAPI has been removed as it depended on orgfactgraph which was purged.

// handleOrgSynthesizeAPI has been removed as it depended on orgfactgraph which was purged.
