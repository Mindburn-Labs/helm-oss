package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/store"
)

// MockClient (Redefined here if not exported, or reuse)
type mockClientImmunity struct {
	response string
	err      error
}

func (m *mockClientImmunity) Chat(ctx context.Context, msgs []Message, tools []ToolDefinition, options *SamplingOptions) (*Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Response{Content: m.response}, nil
}

func TestVerifyStability_I32(t *testing.T) {
	// 1. Golden Sample
	expectedContent := "Safe Response"
	h := sha256.Sum256([]byte(expectedContent))
	expectedHash := hex.EncodeToString(h[:])

	template := PromptTemplate{
		GoldenSamples: []GoldenSample{
			{
				Input:              map[string]any{"q": "test"},
				ExpectedOutputHash: expectedHash,
			},
		},
	}

	// 2. Setup Verifier
	airgap, err := store.NewAirgapStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create airgap store: %v", err)
	}
	mock := &mockClientImmunity{response: expectedContent}
	verifier := NewImmunityVerifier(mock, airgap)

	// 3. Test Success
	if err := verifier.VerifyStability(context.Background(), template); err != nil {
		t.Errorf("VerifyStability failed: %v", err)
	}

	// 4. Verify Cache Population
	// inputStr := "map[q:test]"
	// In real app, we use canonical JSON. For now, we verified the logic via the public method.
}

func TestGetImmuneResponse_Fallback(t *testing.T) {
	airgap, err := store.NewAirgapStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create airgap store: %v", err)
	}

	// 1. Pre-seed cache
	input := "test-input"
	// Let's rely on the Verifier to populate it first.

	// 1. Populate via Verify (or direct Put if we knew the key alg)
	// Let's use a "Live" client first to populate.
	onlineMock := &mockClientImmunity{response: "Cached Response"}
	verifier := NewImmunityVerifier(onlineMock, airgap)

	// Trigger population (using GetImmuneResponse with live client)
	val, err := verifier.GetImmuneResponse(context.Background(), input)
	if err != nil || val != "Cached Response" {
		t.Fatalf("Setup failed: %v", err)
	}

	// 2. Simulate Outage
	offlineMock := &mockClientImmunity{err: errors.New("network down")}
	verifierOffline := NewImmunityVerifier(offlineMock, airgap)

	// 3. Test Fallback
	val, err = verifierOffline.GetImmuneResponse(context.Background(), input)
	if err != nil {
		t.Errorf("Fallback failed: %v", err)
	}
	if val != "Cached Response" {
		t.Errorf("Got %s, want Cached Response", val)
	}
}
