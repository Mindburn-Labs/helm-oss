package ui_test

import (
	"context"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/artifacts"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/console/ui"
)

func TestAGUIAdapter(t *testing.T) {
	// Setup Token Store (InMemory)
	store, _ := artifacts.NewFileStore("/tmp/helm_test_artifacts") // Clean up?
	adapter := ui.NewAGUIAdapter(store)

	ctx := context.Background()

	t.Run("Refuse Unapproved Component", func(t *testing.T) {
		spec := ui.UISpec{
			Version: "1.0",
			Components: []ui.UIComponentCall{
				{ComponentName: "MaliciousScript"},
			},
		}
		_, err := adapter.Render(ctx, spec)
		if err == nil {
			t.Error("Expected error for malicious component, got nil")
		}
	})

	t.Run("Render Approved Component", func(t *testing.T) {
		spec := ui.UISpec{
			Version: "1.0",
			Components: []ui.UIComponentCall{
				{ComponentName: "ApprovalForm", Props: map[string]any{"target": "deploy"}},
			},
		}
		receipt, err := adapter.Render(ctx, spec)
		if err != nil {
			t.Errorf("Render failed: %v", err)
		}
		if receipt.SpecHash == "" {
			t.Error("Receipt missing SpecHash")
		}
	})

	t.Run("Handle Interaction", func(t *testing.T) {
		interaction := ui.UIInteraction{
			InteractionID: "int-1",
			ActionType:    "CLICK",
			ComponentID:   "btn-approve",
			Timestamp:     time.Now(),
		}
		intent, err := adapter.HandleInteraction(ctx, interaction)
		if err != nil {
			t.Errorf("Interaction failed: %v", err)
		}
		if intent.IntentID == "" {
			t.Error("Intent missing ID")
		}
	})
}
