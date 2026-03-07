package agent

import (
	"context"
	"os"
	"testing"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/crypto"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/executor"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/kernel"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/mcp"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/store/ledger"
)

// MockDriver implements executor.ToolDriver
type MockDriver struct {
	LastTool string
}

func (m *MockDriver) Execute(ctx context.Context, toolName string, params map[string]any) (any, error) {
	m.LastTool = toolName
	return "mock-out", nil
}

func TestBridge_Gating(t *testing.T) {
	_ = os.Remove("test.db")
	defer func() { _ = os.Remove("test.db") }()

	// 1. Setup
	ledger, _ := ledger.NewFileLedger("test.db")

	mockDriver := &MockDriver{}
	signer, _ := crypto.NewEd25519Signer("test-key")
	verifier, _ := crypto.NewEd25519Verifier(signer.PublicKeyBytes())

	// Use empty store for test
	// NewSafeExecutor(verifier, driver, receiptStore, artStore, outbox, hash, audit)
	exec := executor.NewSafeExecutor(verifier, signer, mockDriver, nil, nil, nil, "", nil, nil, nil, nil)
	catalog := mcp.NewInMemoryCatalog()

	// NewKernelBridge(ledger, planner, executor, catalog, guardian, verifier, limiter)
	// Guardian nil means "Unsafe" fallback in requestDecision, but "Blocked" in callMCPTool
	bridge := NewKernelBridge(ledger, exec, catalog, nil, verifier, kernel.NewInMemoryLimiterStore())
	ctx := context.Background()

	// 2. Test CreateObligation
	_, err := bridge.Dispatch(ctx, "create_obligation", map[string]any{
		"intent":          "Test Intent",
		"idempotency_key": "id-1",
	})
	// Current stub returns map or string?
	// If createObligation returns map[string]string{"id":...}
	// Let's assume stub behavior from older code or just check error for now.
	if err != nil {
		t.Fatalf("CreateObligation failed: %v", err)
	}
	// For MVP test, just checking it doesn't crash on dispatch.

	// 3. Test CallMCPTool - BLOCKED (No Guardian Configured)
	// bridge.go was updated to return error if guardian == nil in callMCPTool
	_, err = bridge.Dispatch(ctx, "call_mcp_tool", map[string]any{
		"tool_name":   "ls",
		"decision_id": "token",
	})
	if err == nil {
		t.Fatal("Expected error for missing guardian")
	}

	// 4. Test Tool Search
	catalog.Register(ctx, mcp.ToolRef{
		Name:        "git_checkout",
		Description: "Checkout a git branch",
	})

	resSearch, err := bridge.Dispatch(ctx, "mcp_tool_search", map[string]any{
		"query": "git",
	})
	if err != nil {
		t.Fatalf("Tool search failed: %v", err)
	}
	results := resSearch.([]mcp.ToolRef)
	if len(results) == 0 {
		t.Error("Expected search results")
	}
}
