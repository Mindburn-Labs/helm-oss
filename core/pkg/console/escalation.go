package console

import (
	"log/slog"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts"
)

// HandleEscalation processes a system-generated escalation intent.
// It converts the contract intent into a console operator intent and submits it for review.
// This bridges the gap between the automated ControlLoop and the Human Validation layer.
func (s *Server) HandleEscalation(intent *contracts.Intent) error {
	s.operatorMu.Lock()
	defer s.operatorMu.Unlock()

	// 1. Generate ID
	intentID := generateID("esc") // Escalation ID

	// 2. Map to OperatorIntent
	opIntent := &operatorIntent{
		IntentID:    intentID,
		Type:        "escalation", // System-defined type
		Description: intent.Description,
		Params:      intent.Metadata,
		Status:      IntentSubmitted, // Directly to SUBMITTED, bypassing DRAFT/PLANNED
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}

	// 3. Store
	s.intents[intentID] = opIntent

	// 4. Receipt
	s.storeOperatorReceipt("system_escalation", map[string]any{
		"intent_id":   intentID,
		"source":      intent.RequestedBy,
		"description": intent.Description,
		"metadata":    intent.Metadata,
	})

	slog.Info("escalation submitted", "description", intent.Description, "intent_id", intentID)
	return nil
}
