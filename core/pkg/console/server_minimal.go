package console

import (
	"net/http"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/auth"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/console/ui"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/metering"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/pack"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/registry"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/store"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/store/ledger"
)

// NewMinimalServer constructs a console.Server with only the dependencies required
// for a small subset of routes. This enables hermetic tests without binding a
// fixed TCP port via console.Start.
func NewMinimalServer(l ledger.Ledger, reg registry.Registry, uiAdapter ui.UIAdapter, receiptStore store.ReceiptStore, meter metering.Meter, verifier *pack.Verifier) *Server {
	return &Server{
		ledger:       l,
		registry:     reg,
		uiAdapter:    uiAdapter,
		receiptStore: receiptStore,
		meter:        meter,
		packVerifier: verifier,
	}
}

// MinimalHandler wires a minimal route set used by proof tests.
func (s *Server) MinimalHandler(validator *auth.JWTValidator) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/obligations", s.handleObligationsAPI)
	mux.HandleFunc("/api/modules", s.handleModulesAPI)
	mux.HandleFunc("/", s.handleDashboard)
	return auth.NewMiddleware(validator)(mux)
}
