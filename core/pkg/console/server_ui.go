package console

import (
	"encoding/json"
	"net/http"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/api"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/auth"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/console/ui"
)

// Generative UI Handler (AGUI)

func (s *Server) handleUIRender(w http.ResponseWriter, r *http.Request) {
	// 1. Auth Check
	ctx := r.Context()
	principal, err := auth.GetPrincipal(ctx)
	if err != nil {
		api.WriteUnauthorized(w, "Authentication required")
		return
	}

	// 2. Determine Context (e.g. via query param or path)
	// For now, we render the "Home" or "Admin" based on context param
	uiContext := r.URL.Query().Get("context")

	var spec ui.UISpec

	switch uiContext {
	case "admin_dashboard":
		// RBAC Check for Admin
		isAdmin := false
		for _, role := range principal.GetRoles() {
			if role == "admin" {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			api.WriteForbidden(w, "Admin only")
			return
		}
		spec = s.buildAdminDashboardSpec()
	case "audit_log":
		spec = s.buildAuditLogSpec()
	default:
		spec = s.buildHomeSpec(principal)
	}

	// 3. Render Receipt
	receipt, err := s.uiAdapter.Render(ctx, spec)
	if err != nil {
		api.WriteInternal(w, err)
		return
	}

	// 4. Return Response (Spec + Receipt)
	resp := map[string]interface{}{
		"spec":    spec,
		"receipt": receipt,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Spec Builders

func (s *Server) buildAdminDashboardSpec() ui.UISpec {
	return ui.UISpec{
		Version: "1.0",
		Theme:   "dark_enterprise",
		Components: []ui.UIComponentCall{
			{
				ID:            "admin-header",
				ComponentName: "Header",
				Props: map[string]any{
					"title": "Enterprise Administration",
				},
			},
			{
				ID:            "tenant-list",
				ComponentName: "TenantTable", // Needs to be added to AllowedComponents
				Props: map[string]any{
					"source": "/api/admin/tenants",
				},
			},
			{
				ID:            "audit-widget",
				ComponentName: "AuditLogViewer", // Needs to be added to AllowedComponents
				Props: map[string]any{
					"limit": 5,
				},
			},
		},
	}
}

func (s *Server) buildAuditLogSpec() ui.UISpec {
	return ui.UISpec{
		Version: "1.0",
		Components: []ui.UIComponentCall{
			{
				ID:            "audit-full",
				ComponentName: "AuditLogViewer",
				Props: map[string]any{
					"limit":          100,
					"export_enabled": true,
				},
			},
		},
	}
}

func (s *Server) buildHomeSpec(p auth.Principal) ui.UISpec {
	return ui.UISpec{
		Version: "1.0",
		Components: []ui.UIComponentCall{
			{
				ID:            "welcome-banner",
				ComponentName: "Banner",
				Props: map[string]any{
					"message": "Welcome back, " + p.GetID(),
				},
			},
			{
				ID:            "quick-actions",
				ComponentName: "ActionGrid",
				Props: map[string]any{
					"actions": []string{"deploy", "status", "audit"},
				},
			},
		},
	}
}
