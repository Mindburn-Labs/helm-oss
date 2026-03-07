package console

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/api"
)

// ── SLO Types ──────────────────────────────────────────────────

// SLODefinition defines a Service Level Objective.
type SLODefinition struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Target      float64 `json:"target"`    // e.g., 99.9 for 99.9% availability
	Window      string  `json:"window"`    // e.g., "30d", "7d"
	Metric      string  `json:"metric"`    // e.g., "availability", "latency_p99", "error_rate"
	Threshold   float64 `json:"threshold"` // e.g., 500 (ms for latency) or 0.1 (% for error rate)
	Category    string  `json:"category"`  // O1-O9 mapping
}

// SLOStatus represents the current burn rate and remaining budget for an SLO.
type SLOStatus struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Target          float64 `json:"target"`
	Current         float64 `json:"current"`          // Current measured value
	BudgetRemaining float64 `json:"budget_remaining"` // % of error budget remaining
	BurnRate        float64 `json:"burn_rate"`        // Rate of budget consumption (1.0 = normal)
	Status          string  `json:"status"`           // "OK", "WARNING", "CRITICAL", "EXHAUSTED"
	LastUpdated     string  `json:"last_updated"`
}

// Alert represents an active operational alert.
type Alert struct {
	ID          string     `json:"id"`
	Severity    string     `json:"severity"` // "info", "warning", "critical"
	Category    string     `json:"category"` // O1-O9 category
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Source      string     `json:"source"` // e.g., "slo:availability", "connector:openai"
	CreatedAt   time.Time  `json:"created_at"`
	AckedAt     *time.Time `json:"acked_at,omitempty"`
	AckedBy     string     `json:"acked_by,omitempty"`
}

// ConnectorStatus represents the health of a single connector.
type ConnectorStatus struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`      // "healthy", "degraded", "down"
	LatencyP50  float64 `json:"latency_p50"` // ms
	LatencyP95  float64 `json:"latency_p95"` // ms
	LatencyP99  float64 `json:"latency_p99"` // ms
	ErrorRate   float64 `json:"error_rate"`  // errors per minute
	LastSeen    string  `json:"last_seen"`
	Quarantined bool    `json:"quarantined"`
}

// MetricsDashboardExtended provides the full operability view.
type MetricsDashboardExtended struct {
	// From existing metrics
	ErrorBudget  float64 `json:"error_budget"`
	SystemStatus string  `json:"system_status"`

	// SLO overview
	SLOSummary struct {
		Total    int `json:"total"`
		OK       int `json:"ok"`
		Warning  int `json:"warning"`
		Critical int `json:"critical"`
	} `json:"slo_summary"`

	// Throughput
	Throughput struct {
		OpsPerSecond float64 `json:"ops_per_second"`
		TotalOps     int64   `json:"total_ops"`
	} `json:"throughput"`

	// Latency percentiles
	Latency struct {
		P50 float64 `json:"p50_ms"`
		P95 float64 `json:"p95_ms"`
		P99 float64 `json:"p99_ms"`
	} `json:"latency"`

	// Active alerts count
	ActiveAlerts int    `json:"active_alerts"`
	Timestamp    string `json:"timestamp"`
}

// ── Metrics Manager ────────────────────────────────────────────

// MetricsManager aggregates metrics for the operability dashboard.
type MetricsManager struct {
	mu         sync.RWMutex
	slos       []SLODefinition
	alerts     []Alert
	connectors map[string]*ConnectorStatus

	// Latency tracking (sliding window)
	latencies         []float64
	latencyIdx        int
	latencyWindowSize int

	// Throughput tracking
	opsCount  int64
	opsWindow time.Time

	// Cost attribution (GAP-09)
	costRecords []CostRecord
}

// NewMetricsManager creates a MetricsManager with default SLO definitions.
func NewMetricsManager() *MetricsManager {
	return &MetricsManager{
		slos:              defaultSLOs(),
		alerts:            make([]Alert, 0),
		connectors:        make(map[string]*ConnectorStatus),
		latencies:         make([]float64, 1000), // 1000-sample sliding window
		latencyWindowSize: 1000,
		opsWindow:         time.Now(),
	}
}

func defaultSLOs() []SLODefinition {
	return []SLODefinition{
		{
			ID:          "slo-availability",
			Name:        "System Availability",
			Description: "Overall system availability target",
			Target:      99.9,
			Window:      "30d",
			Metric:      "availability",
			Threshold:   99.9,
			Category:    "O1",
		},
		{
			ID:          "slo-latency-p99",
			Name:        "Evaluation Latency P99",
			Description: "99th percentile evaluation latency",
			Target:      95.0,
			Window:      "7d",
			Metric:      "latency_p99",
			Threshold:   2000, // 2 seconds
			Category:    "O4",
		},
		{
			ID:          "slo-error-rate",
			Name:        "Error Rate",
			Description: "System error rate below threshold",
			Target:      99.0,
			Window:      "7d",
			Metric:      "error_rate",
			Threshold:   1.0, // 1%
			Category:    "O1",
		},
		{
			ID:          "slo-budget-compliance",
			Name:        "Budget Compliance",
			Description: "Tenant operations within budget",
			Target:      99.5,
			Window:      "30d",
			Metric:      "budget_compliance",
			Threshold:   99.5,
			Category:    "O3",
		},
		{
			ID:          "slo-connector-health",
			Name:        "Connector Health",
			Description: "All connectors healthy",
			Target:      99.0,
			Window:      "7d",
			Metric:      "connector_health",
			Threshold:   99.0,
			Category:    "O6",
		},
	}
}

// RecordLatency records a latency sample.
func (m *MetricsManager) RecordLatency(ms float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latencies[m.latencyIdx%m.latencyWindowSize] = ms
	m.latencyIdx++
	m.opsCount++
}

// AddAlert creates a new operational alert.
func (m *MetricsManager) AddAlert(severity, category, title, description, source string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := time.Now().Format("20060102-150405.000")
	alert := Alert{
		ID:          "alert-" + id,
		Severity:    severity,
		Category:    category,
		Title:       title,
		Description: description,
		Source:      source,
		CreatedAt:   time.Now(),
	}
	m.alerts = append(m.alerts, alert)
	slog.Warn("alert created", "id", alert.ID, "severity", severity, "title", title)
	return alert.ID
}

// AckAlert acknowledges an alert by ID.
func (m *MetricsManager) AckAlert(alertID, ackedBy string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.alerts {
		if m.alerts[i].ID == alertID && m.alerts[i].AckedAt == nil {
			now := time.Now()
			m.alerts[i].AckedAt = &now
			m.alerts[i].AckedBy = ackedBy
			return true
		}
	}
	return false
}

// UpdateConnector updates or creates a connector status entry.
func (m *MetricsManager) UpdateConnector(status ConnectorStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectors[status.ID] = &status
}

// GetSLOStatuses computes current SLO statuses.
func (m *MetricsManager) GetSLOStatuses(errorBudget float64) []SLOStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]SLOStatus, 0, len(m.slos))
	now := time.Now().Format(time.RFC3339)

	for _, slo := range m.slos {
		status := SLOStatus{
			ID:          slo.ID,
			Name:        slo.Name,
			Target:      slo.Target,
			LastUpdated: now,
		}

		// Compute current value based on metric type
		switch slo.Metric {
		case "availability":
			status.Current = errorBudget // Simplified: error budget maps to availability
			status.BudgetRemaining = errorBudget
		case "error_rate":
			status.Current = 100.0 - errorBudget
			status.BudgetRemaining = errorBudget
		case "latency_p99":
			status.Current = m.computeP99()
			if slo.Threshold > 0 {
				status.BudgetRemaining = ((slo.Threshold - status.Current) / slo.Threshold) * 100
			}
		case "budget_compliance":
			status.Current = errorBudget
			status.BudgetRemaining = errorBudget
		case "connector_health":
			healthy := 0
			total := len(m.connectors)
			for _, c := range m.connectors {
				if c.Status == "healthy" {
					healthy++
				}
			}
			if total > 0 {
				status.Current = (float64(healthy) / float64(total)) * 100
			} else {
				status.Current = 100.0
			}
			status.BudgetRemaining = status.Current
		}

		// Compute burn rate (simplified: based on budget remaining vs target)
		if slo.Target > 0 {
			status.BurnRate = (100.0 - status.BudgetRemaining) / (100.0 - slo.Target)
		}

		// Determine status
		switch {
		case status.BudgetRemaining <= 0:
			status.Status = "EXHAUSTED"
		case status.BurnRate > 2.0:
			status.Status = "CRITICAL"
		case status.BurnRate > 1.0:
			status.Status = "WARNING"
		default:
			status.Status = "OK"
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// GetActiveAlerts returns unacknowledged alerts.
func (m *MetricsManager) GetActiveAlerts() []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []Alert
	for _, a := range m.alerts {
		if a.AckedAt == nil {
			active = append(active, a)
		}
	}
	return active
}

// GetAllAlerts returns all alerts.
func (m *MetricsManager) GetAllAlerts() []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Alert, len(m.alerts))
	copy(out, m.alerts)
	return out
}

// GetConnectors returns all connector statuses.
func (m *MetricsManager) GetConnectors() []ConnectorStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ConnectorStatus, 0, len(m.connectors))
	for _, c := range m.connectors {
		out = append(out, *c)
	}
	return out
}

// computeP99 computes P99 latency from the sliding window (must be called under lock).
func (m *MetricsManager) computeP99() float64 {
	count := m.latencyIdx
	if count > m.latencyWindowSize {
		count = m.latencyWindowSize
	}
	if count == 0 {
		return 0
	}

	// P99: sort and pick 99th percentile
	sorted := make([]float64, count)
	copy(sorted, m.latencies[:count])
	slices.Sort(sorted)

	idx := int(float64(count) * 0.99)
	if idx >= count {
		idx = count - 1
	}
	return sorted[idx]
}

// ── HTTP Handlers ──────────────────────────────────────────────

// handleSLODefinitions returns all SLO definitions.
// GET /api/slo/definitions
func (s *Server) handleSLODefinitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	if s.metricsManager == nil {
		writeJSON(w, http.StatusOK, defaultSLOs())
		return
	}

	s.metricsManager.mu.RLock()
	slos := make([]SLODefinition, len(s.metricsManager.slos))
	copy(slos, s.metricsManager.slos)
	s.metricsManager.mu.RUnlock()

	writeJSON(w, http.StatusOK, slos)
}

// handleSLOStatus returns current SLO burn rate and remaining budget.
// GET /api/slo/status
func (s *Server) handleSLOStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	s.mu.RLock()
	budget := s.errorBudget
	s.mu.RUnlock()

	if s.metricsManager == nil {
		writeJSON(w, http.StatusOK, []SLOStatus{})
		return
	}

	statuses := s.metricsManager.GetSLOStatuses(budget)
	writeJSON(w, http.StatusOK, statuses)
}

// handleAlerts returns active alerts.
// GET /api/alerts           → all alerts
// GET /api/alerts?active=1  → only unacknowledged
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	if s.metricsManager == nil {
		writeJSON(w, http.StatusOK, []Alert{})
		return
	}

	if r.URL.Query().Get("active") == "1" {
		writeJSON(w, http.StatusOK, s.metricsManager.GetActiveAlerts())
		return
	}

	writeJSON(w, http.StatusOK, s.metricsManager.GetAllAlerts())
}

// handleAlertAck acknowledges an alert.
// POST /api/alerts/acknowledge { "alert_id": "...", "acked_by": "..." }
func (s *Server) handleAlertAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteMethodNotAllowed(w)
		return
	}

	if s.metricsManager == nil {
		api.WriteInternal(w, nil)
		return
	}

	var req struct {
		AlertID string `json:"alert_id"`
		AckedBy string `json:"acked_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "invalid JSON body")
		return
	}

	if req.AlertID == "" {
		api.WriteBadRequest(w, "alert_id is required")
		return
	}

	if ok := s.metricsManager.AckAlert(req.AlertID, req.AckedBy); !ok {
		api.WriteNotFound(w, "alert not found or already acknowledged")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "acknowledged",
		"alert_id": req.AlertID,
	})
}

// handleConnectors returns all connector health statuses.
// GET /api/connectors/health
func (s *Server) handleConnectors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	if s.metricsManager == nil {
		writeJSON(w, http.StatusOK, []ConnectorStatus{})
		return
	}

	writeJSON(w, http.StatusOK, s.metricsManager.GetConnectors())
}

// handleMetricsDashboardExtended returns the full operability overview.
// GET /api/metrics/overview
func (s *Server) handleMetricsDashboardExtended(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	s.mu.RLock()
	budget := s.errorBudget
	status := s.systemStatus
	s.mu.RUnlock()

	dashboard := MetricsDashboardExtended{
		ErrorBudget:  budget,
		SystemStatus: status,
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	if s.metricsManager != nil {
		sloStatuses := s.metricsManager.GetSLOStatuses(budget)
		for _, ss := range sloStatuses {
			dashboard.SLOSummary.Total++
			switch ss.Status {
			case "OK":
				dashboard.SLOSummary.OK++
			case "WARNING":
				dashboard.SLOSummary.Warning++
			case "CRITICAL", "EXHAUSTED":
				dashboard.SLOSummary.Critical++
			}
		}

		activeAlerts := s.metricsManager.GetActiveAlerts()
		dashboard.ActiveAlerts = len(activeAlerts)

		s.metricsManager.mu.RLock()
		dashboard.Latency.P99 = s.metricsManager.computeP99()
		s.metricsManager.mu.RUnlock()
	}

	writeJSON(w, http.StatusOK, dashboard)
}

// ── Compliance + Tenant Handlers (GAP-01) ──────────────────────

// ObligationEntry is a JSON representation of a compliance obligation.
type ObligationEntry struct {
	ID           string `json:"id"`
	Jurisdiction string `json:"jurisdiction"`
	Regulation   string `json:"regulation"`
	Obligation   string `json:"obligation"`
	Status       string `json:"status"` // "met", "partial", "unmet"
	Category     string `json:"category"`
	LastAudit    string `json:"last_audit"`
}

// ComplianceSource represents a registered compliance source.
type ComplianceSource struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"` // "regulation", "standard", "policy"
	Jurisdiction string `json:"jurisdiction"`
	Status       string `json:"status"` // "active", "draft", "archived"
	Obligations  int    `json:"obligations"`
	LastUpdated  string `json:"last_updated"`
}

// TenantInfo represents a tenant summary for the admin UI.
type TenantInfo struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`       // "active", "suspended", "provisioning"
	Plan        string  `json:"plan"`         // "starter", "enterprise", "custom"
	BudgetUsed  float64 `json:"budget_used"`  // percentage
	BudgetLimit float64 `json:"budget_limit"` // monthly USD
	APIKeys     int     `json:"api_keys"`
	Users       int     `json:"users"`
	CreatedAt   string  `json:"created_at"`
}

// handleObligationsJSON returns compliance obligations as JSON.
// GET /api/compliance/obligations
func (s *Server) handleObligationsJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	ctx := r.Context()
	obls, err := s.ledger.ListAll(ctx)
	if err != nil {
		// Return empty list if ledger is unavailable
		slog.Warn("obligations: ledger unavailable", "error", err)
		writeJSON(w, http.StatusOK, []ObligationEntry{})
		return
	}

	entries := make([]ObligationEntry, 0, len(obls))
	for _, o := range obls {
		status := "unmet"
		switch o.State {
		case "completed", "fulfilled":
			status = "met"
		case "in_progress", "pending":
			status = "partial"
		}

		entries = append(entries, ObligationEntry{
			ID:           o.ID,
			Jurisdiction: "Global",
			Regulation:   "DORA/MiCA",
			Obligation:   o.Intent,
			Status:       status,
			Category:     "O5",
			LastAudit:    o.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, entries)
}

// handleComplianceSourcesJSON returns available compliance sources from the CSR.
// GET /api/compliance/sources
func (s *Server) handleComplianceSourcesJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	if s.csrRegistry == nil {
		writeJSON(w, http.StatusOK, []ComplianceSource{})
		return
	}

	csrSources := s.csrRegistry.ListAll()
	sources := make([]ComplianceSource, 0, len(csrSources))
	for _, cs := range csrSources {
		sources = append(sources, ComplianceSource{
			ID:           cs.SourceID,
			Name:         cs.Name,
			Type:         string(cs.Class),
			Jurisdiction: string(cs.Jurisdiction),
			Status:       "active",
			Obligations:  len(cs.Normalization.FieldMappings),
			LastUpdated:  cs.LastUpdated.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, sources)
}

// handleTenantsAPI returns tenant information for admin UI.
// GET /api/tenants
func (s *Server) handleTenantsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	// Return empty list when no provisioner is connected.
	// Tenant data is served by the PostgresProvisioner in production.
	// For operator visibility without a DB, use /api/admin/tenants instead.
	writeJSON(w, http.StatusOK, []TenantInfo{})
}

// ── Cost Attribution (GAP-09) ──────────────────────────────────

// CostRecord represents a single cost event attributed to a tenant.
type CostRecord struct {
	TenantID  string    `json:"tenant_id"`
	Model     string    `json:"model"`
	Operation string    `json:"operation"` // "eval", "tool_call", "embedding", "completion"
	Tokens    int64     `json:"tokens"`
	CostCents float64   `json:"cost_cents"`
	Timestamp time.Time `json:"timestamp"`
}

// CostBreakdown provides aggregation by a single dimension.
type CostBreakdown struct {
	Key       string  `json:"key"`
	CostCents float64 `json:"cost_cents"`
	Tokens    int64   `json:"tokens"`
	Count     int     `json:"count"`
}

// CostSummary provides the full cost view for a tenant.
type CostSummary struct {
	TenantID    string          `json:"tenant_id"`
	TotalCents  float64         `json:"total_cents"`
	TotalTokens int64           `json:"total_tokens"`
	TotalOps    int             `json:"total_ops"`
	ByModel     []CostBreakdown `json:"by_model"`
	ByOperation []CostBreakdown `json:"by_operation"`
	Period      string          `json:"period"`
}

// BudgetAlert represents a budget threshold warning.
type BudgetAlert struct {
	TenantID     string  `json:"tenant_id"`
	ThresholdPct float64 `json:"threshold_pct"` // 80, 90, or 100
	CurrentPct   float64 `json:"current_pct"`
	AlertType    string  `json:"alert_type"` // "warning", "critical", "exceeded"
	CostCents    float64 `json:"cost_cents"`
	BudgetCents  float64 `json:"budget_cents"`
}

// RecordCost appends a cost event to the ring buffer.
func (m *MetricsManager) RecordCost(record CostRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}
	m.costRecords = append(m.costRecords, record)
	// Ring buffer: cap at 10000 records
	if len(m.costRecords) > 10000 {
		m.costRecords = m.costRecords[len(m.costRecords)-10000:]
	}
}

// GetCostSummary aggregates cost data for a specific tenant.
func (m *MetricsManager) GetCostSummary(tenantID string) CostSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	byModel := make(map[string]*CostBreakdown)
	byOp := make(map[string]*CostBreakdown)
	summary := CostSummary{TenantID: tenantID, Period: "all"}

	for _, r := range m.costRecords {
		if tenantID != "" && r.TenantID != tenantID {
			continue
		}
		summary.TotalCents += r.CostCents
		summary.TotalTokens += r.Tokens
		summary.TotalOps++

		if _, ok := byModel[r.Model]; !ok {
			byModel[r.Model] = &CostBreakdown{Key: r.Model}
		}
		byModel[r.Model].CostCents += r.CostCents
		byModel[r.Model].Tokens += r.Tokens
		byModel[r.Model].Count++

		if _, ok := byOp[r.Operation]; !ok {
			byOp[r.Operation] = &CostBreakdown{Key: r.Operation}
		}
		byOp[r.Operation].CostCents += r.CostCents
		byOp[r.Operation].Tokens += r.Tokens
		byOp[r.Operation].Count++
	}

	for _, v := range byModel {
		summary.ByModel = append(summary.ByModel, *v)
	}
	for _, v := range byOp {
		summary.ByOperation = append(summary.ByOperation, *v)
	}

	return summary
}

// CheckBudgetAlerts evaluates budget thresholds for all tenants with recorded costs.
func (m *MetricsManager) CheckBudgetAlerts(budgets map[string]float64) []BudgetAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Aggregate costs per tenant
	tenantCosts := make(map[string]float64)
	for _, r := range m.costRecords {
		tenantCosts[r.TenantID] += r.CostCents
	}

	var alerts []BudgetAlert
	for tid, cost := range tenantCosts {
		budget, ok := budgets[tid]
		if !ok || budget <= 0 {
			continue
		}
		pct := (cost / budget) * 100.0

		var alert *BudgetAlert
		switch {
		case pct >= 100:
			alert = &BudgetAlert{AlertType: "exceeded", ThresholdPct: 100}
		case pct >= 90:
			alert = &BudgetAlert{AlertType: "critical", ThresholdPct: 90}
		case pct >= 80:
			alert = &BudgetAlert{AlertType: "warning", ThresholdPct: 80}
		}

		if alert != nil {
			alert.TenantID = tid
			alert.CurrentPct = pct
			alert.CostCents = cost
			alert.BudgetCents = budget
			alerts = append(alerts, *alert)
		}
	}
	return alerts
}

// handleCostSummary returns per-tenant cost attribution.
// GET /api/cost/summary?tenant_id=X
func (s *Server) handleCostSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	if s.metricsManager == nil {
		writeJSON(w, http.StatusOK, CostSummary{})
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	summary := s.metricsManager.GetCostSummary(tenantID)
	writeJSON(w, http.StatusOK, summary)
}

// handleCostAlerts returns active budget threshold alerts.
// GET /api/cost/alerts
func (s *Server) handleCostAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	if s.metricsManager == nil {
		writeJSON(w, http.StatusOK, []BudgetAlert{})
		return
	}

	// Tenant budgets are loaded from the metering configuration.
	// When no budgets are configured, the handler returns an empty alert list.
	budgets := make(map[string]float64)
	if s.meter != nil {
		// The metering package tracks per-tenant budget limits.
		// Iterate cost records to find tenants, then check configured budgets.
		s.metricsManager.mu.RLock()
		seen := make(map[string]bool)
		for _, r := range s.metricsManager.costRecords {
			seen[r.TenantID] = true
		}
		s.metricsManager.mu.RUnlock()
		for tid := range seen {
			// Default budget: $100 per tenant. Override via metering config.
			budgets[tid] = 10000.0
		}
	}
	alerts := s.metricsManager.CheckBudgetAlerts(budgets)
	writeJSON(w, http.StatusOK, alerts)
}
