package memory

import (
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/kernel/ui"
)

// MockMemoryAdapter confirms interface satisfaction
type MockMemoryAdapter struct{}

func (m *MockMemoryAdapter) IngestSourceBundle(bundle SourceBundle) (MemoryBuildManifest, error) {
	return MemoryBuildManifest{
		ManifestID: "manifest1",
		BundleID:   bundle.BundleID,
		BuiltAt:    time.Now(),
		GraphRef:   DocumentGraphRef{Hash: "hash1"},
	}, nil
}

func (m *MockMemoryAdapter) QueryMemory(query string) ([]QueryResult, error) {
	return []QueryResult{{Content: "result", SourceURI: "doc1", Score: 0.9}}, nil
}

func (m *MockMemoryAdapter) Promote(ref DocumentGraphRef) error {
	return nil
}

func TestIntegrationInterfaces(t *testing.T) {
	// 1. Verify MemoryAdapter
	var _ MemoryAdapter = &MockMemoryAdapter{}

	adapter := &MockMemoryAdapter{}
	bundle := SourceBundle{BundleID: "bundle1", SourceURIs: []string{"doc1"}}
	manifest, err := adapter.IngestSourceBundle(bundle)
	if err != nil {
		t.Fatalf("IngestSourceBundle failed: %v", err)
	}
	results, err := adapter.QueryMemory("query")
	if err != nil {
		t.Fatalf("QueryMemory failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one query result")
	}
	if err := adapter.Promote(manifest.GraphRef); err != nil {
		t.Fatalf("Promote failed: %v", err)
	}

	// 2. Verify UI types
	interaction := ui.UIInteraction{
		InteractionID: "int1",
		Payload:       "click",
	}

	proposal := ui.Proposal{
		ProposalID:    "prop1",
		InteractionID: interaction.InteractionID,
		EffectType:    "UPDATE",
		Context:       ui.ProposalContext{JurisdictionID: "EU"},
	}

	if proposal.InteractionID != "int1" {
		t.Error("Proposal linkage broken")
	}
}
