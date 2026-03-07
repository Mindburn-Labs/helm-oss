package console

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/api"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/auth"
)

// Admin UI Handlers — JSON API with HTMX fallback.
// Tenant and role data is served by the PostgresProvisioner in production.
// When no provisioner is connected (local dev), handlers return empty collections.

func (s *Server) handleAdminTenantsAPI(w http.ResponseWriter, r *http.Request) {
	// In production, tenant data is loaded from the provisioner database.
	// This handler exposes the admin view; for operator-scoped tenant info,
	// use /api/tenants instead (which reads from MetricsManager state).
	type tenantEntry struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}

	// If the server has a metricsManager with tenant data, use it.
	// Otherwise return empty list — tenant provisioner is not connected.
	var tenants []tenantEntry
	if s.metricsManager != nil {
		// Collect unique tenant IDs from cost records as a visibility proxy.
		s.metricsManager.mu.RLock()
		seen := make(map[string]bool)
		for _, cr := range s.metricsManager.costRecords {
			if !seen[cr.TenantID] {
				seen[cr.TenantID] = true
				tenants = append(tenants, tenantEntry{
					ID:     cr.TenantID,
					Name:   cr.TenantID,
					Status: "ACTIVE",
				})
			}
		}
		s.metricsManager.mu.RUnlock()
	}

	if tenants == nil {
		tenants = []tenantEntry{}
	}

	// Content Negotiation
	accept := r.Header.Get("Accept")
	if accept == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tenants)
		return
	}

	// HTML (HTMX)
	html := `<table><tr><th>ID</th><th>Name</th><th>Status</th><th>Actions</th></tr>`
	for _, t := range tenants {
		html += fmt.Sprintf(`<tr>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>
				<button hx-delete="/api/admin/tenants/%s" hx-confirm="Are you sure?">Suspend</button>
			</td>
		</tr>`, t.ID, t.Name, t.Status, t.ID)
	}
	html += `</table>`

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		slog.Error("failed to write admin tenants response", "error", err)
	}
}

func (s *Server) handleAdminRolesAPI(w http.ResponseWriter, r *http.Request) {
	// Canonical RBAC roles defined by the authorization layer.
	// In production, the policy engine manages role definitions;
	// this endpoint returns the active role set for admin visibility.
	roles := []string{"admin", "editor", "viewer", "auditor"}

	// Content Negotiation
	accept := r.Header.Get("Accept")
	if accept == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(roles)
		return
	}

	html := `<ul>`
	for _, role := range roles {
		html += fmt.Sprintf(`<li>%s <button>Edit</button></li>`, role)
	}
	html += `</ul>`

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		slog.Error("failed to write admin roles response", "error", err)
	}
}

func (s *Server) handleAdminAuditUI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tid, _ := auth.GetTenantID(ctx)

	// Content Negotiation
	accept := r.Header.Get("Accept")
	if accept == "application/json" {
		// Query real audit records from the ledger (obligation log).
		obls, err := s.ledger.ListAll(ctx)
		if err != nil {
			api.WriteInternal(w, err)
			return
		}
		// Sort by time desc and limit to most recent 50
		sort.Slice(obls, func(i, j int) bool {
			return obls[i].CreatedAt.After(obls[j].CreatedAt)
		})
		if len(obls) > 50 {
			obls = obls[:50]
		}

		type auditEntry struct {
			Timestamp string `json:"timestamp"`
			Action    string `json:"action"`
			Actor     string `json:"actor"`
			Resource  string `json:"resource"`
		}
		entries := make([]auditEntry, 0, len(obls))
		for _, o := range obls {
			entries = append(entries, auditEntry{
				Timestamp: o.CreatedAt.Format(time.RFC3339),
				Action:    string(o.State),
				Actor:     o.LeasedBy,
				Resource:  o.ID,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(entries)
		return
	}

	// HTML fragment: recent audit log from the ledger
	obls, err := s.ledger.ListAll(ctx)
	if err != nil {
		html := fmt.Sprintf(`<div class="error">Failed to load audit log: %s</div>`, err.Error())
		w.Header().Set("Content-Type", "text/html")
		if _, werr := w.Write([]byte(html)); werr != nil {
			slog.Error("failed to write audit error response", "error", werr)
		}
		return
	}

	sort.Slice(obls, func(i, j int) bool {
		return obls[i].CreatedAt.After(obls[j].CreatedAt)
	})
	if len(obls) > 50 {
		obls = obls[:50]
	}

	html := fmt.Sprintf(`
		<div class="audit-controls">
			<h3>Audit Log for %s</h3>
			<button onclick="window.location.href='/api/admin/audit/export'">Export Evidence Pack</button>
		</div>
		<table>
			<tr><th>Time</th><th>Actor</th><th>Action</th><th>Resource</th></tr>`, tid)
	for _, o := range obls {
		html += fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			o.CreatedAt.Format(time.RFC3339), o.LeasedBy, o.State, o.ID)
	}
	html += `</table>`

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		slog.Error("failed to write audit UI response", "error", err)
	}
}
