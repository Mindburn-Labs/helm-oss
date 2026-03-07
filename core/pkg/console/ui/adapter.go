package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/artifacts"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/crypto"
)

// UIAdapter defines the protocol for Generative UI interaction.
type UIAdapter interface {
	// Name returns the adapter type (e.g. "AGUI", "Declarative").
	Name() string

	// ValidateSpec ensures the UI specification complies with safety rules.
	ValidateSpec(spec UISpec) error

	// Render produces a receipt for a given UI specification.
	// It does NOT execute side effects, only prepares the view.
	Render(ctx context.Context, spec UISpec) (*UIRenderReceipt, error)

	// HandleInteraction processes user input and may produce an OperatorIntent.
	// This is the bridge to the Kernel's EffectProposal system.
	HandleInteraction(ctx context.Context, interaction UIInteraction) (*OperatorIntent, error)
}

// OperatorIntent captures the meaning of a user's action before it becomes a Proposal.
type OperatorIntent struct {
	IntentID       string                     `json:"intent_id"`
	InteractionRef string                     `json:"interaction_ref"`
	ProposalHash   string                     `json:"proposal_hash"` // Hash of the interaction payload
	Summary        string                     `json:"summary"`
	Params         map[string]any             `json:"params,omitempty"`
	ToProposal     func() *contracts.Proposal `json:"-"` // Conversion logic
}

// AGUIAdapter implements the "Static Generative UI" pattern (bounded, safe).
// It only allows pre-approved components.
type AGUIAdapter struct {
	allowedComponents map[string]bool
	artifactStore     artifacts.Store
}

func NewAGUIAdapter(store artifacts.Store) *AGUIAdapter {
	return &AGUIAdapter{
		allowedComponents: map[string]bool{
			"ApprovalForm":           true,
			"DangerousActionConfirm": true,
			"EvidenceViewer":         true,
			"Timeline":               true,
			"DiffViewer":             true,
			"CompilerWorkbench":      true,
			"Header":                 true,
			"TenantTable":            true,
			"AuditLogViewer":         true,
			"Banner":                 true,
			"ActionGrid":             true,
		},
		artifactStore: store,
	}
}

func (a *AGUIAdapter) Name() string { return "AGUI-Static" }

func (a *AGUIAdapter) ValidateSpec(spec UISpec) error {
	for _, comp := range spec.Components {
		if !a.allowedComponents[comp.ComponentName] {
			return fmt.Errorf("AGUI violation: component '%s' not in ALLOWLIST", comp.ComponentName)
		}
	}
	return nil
}

func (a *AGUIAdapter) Render(ctx context.Context, spec UISpec) (*UIRenderReceipt, error) {
	if err := a.ValidateSpec(spec); err != nil {
		return nil, err
	}

	// Canonicalize & Hash Spec
	specBytes, err := crypto.CanonicalMarshal(spec)
	if err != nil {
		return nil, err
	}

	specHash, err := a.artifactStore.Store(ctx, specBytes)
	if err != nil {
		return nil, err
	}

	return &UIRenderReceipt{
		RenderID:   fmt.Sprintf("render-%d", time.Now().UnixNano()),
		SpecHash:   specHash,
		RenderedAt: time.Now().UTC(),
	}, nil
}

func (a *AGUIAdapter) HandleInteraction(ctx context.Context, interaction UIInteraction) (*OperatorIntent, error) {
	// In a real system, this would look up the InteractionID matches a valid active Render.
	// For AGUI, we map specific actions to intents.

	// Compute ProposalHash from the interaction payload
	payloadBytes, err := crypto.CanonicalMarshal(interaction)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal interaction for hash: %w", err)
	}
	proposalHash, err := a.artifactStore.Store(ctx, payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to store interaction payload: %w", err)
	}

	return &OperatorIntent{
		IntentID:       fmt.Sprintf("intent-%s", interaction.InteractionID),
		InteractionRef: interaction.InteractionID,
		ProposalHash:   proposalHash,
		Summary:        fmt.Sprintf("AGUI Action: %s on %s", interaction.ActionType, interaction.ComponentID),
		Params:         interaction.Payload,
	}, nil
}
