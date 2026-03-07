package api

import (
	"encoding/json"
	"net/http"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/escalation/ceremony"
)

// HandleApproval handles the POST /api/v1/kernel/approve endpoint.
// Validates ceremony requirements before accepting an approval.
func HandleApproval(policy ceremony.CeremonyPolicy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteMethodNotAllowed(w)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
		var req ceremony.CeremonyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteBadRequest(w, "Invalid request body")
			return
		}

		result := ceremony.ValidateCeremony(policy, req)
		if !result.Valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":  "ceremony_validation_failed",
				"reason": result.Reason,
			})
			return
		}

		// Ceremony passed — proceed with approval
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "approved",
			"decision_id": req.DecisionID,
			"lamport":     req.LamportHeight,
		})
	}
}
