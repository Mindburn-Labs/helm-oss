package console

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts"
)

// inMemoryReceiptStore implements store.ReceiptStore for testing.
type inMemoryReceiptStore struct {
	receipts []*contracts.Receipt
}

func (s *inMemoryReceiptStore) Get(_ context.Context, decisionID string) (*contracts.Receipt, error) {
	for _, r := range s.receipts {
		if r.DecisionID == decisionID {
			return r, nil
		}
	}
	return nil, fmt.Errorf("receipt not found")
}

func (s *inMemoryReceiptStore) GetByReceiptID(_ context.Context, receiptID string) (*contracts.Receipt, error) {
	for _, r := range s.receipts {
		if r.ReceiptID == receiptID {
			return r, nil
		}
	}
	return nil, fmt.Errorf("receipt not found")
}

func (s *inMemoryReceiptStore) List(_ context.Context, limit int) ([]*contracts.Receipt, error) {
	if limit >= len(s.receipts) {
		return s.receipts, nil
	}
	return s.receipts[:limit], nil
}

func (s *inMemoryReceiptStore) Store(_ context.Context, receipt *contracts.Receipt) error {
	s.receipts = append(s.receipts, receipt)
	return nil
}

func (s *inMemoryReceiptStore) GetLastForSession(_ context.Context, sessionID string) (*contracts.Receipt, error) {
	return nil, nil
}

func newTestServerWithReceipts() *Server {
	store := &inMemoryReceiptStore{
		receipts: []*contracts.Receipt{
			{
				ReceiptID:  "rcpt-001",
				DecisionID: "dec-001",
				EffectID:   "eff-001",
				Status:     "SUCCESS",
				Timestamp:  time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC),
				BlobHash:   "sha256:abc123",
				ExecutorID: "helm-agent",
				Metadata:   map[string]any{"tool": "builder_generate"},
			},
			{
				ReceiptID:  "rcpt-002",
				DecisionID: "dec-002",
				EffectID:   "eff-002",
				Status:     "FAILURE",
				Timestamp:  time.Date(2026, 2, 7, 13, 0, 0, 0, time.UTC),
				ExecutorID: "helm-ops",
				Metadata:   map[string]any{"tool": "provision_tenant"},
			},
		},
	}

	return &Server{
		cache:          make(map[string][]byte),
		pendingSignups: make(map[string]*pendingSignup),
		errorBudget:    100.0,
		systemStatus:   "HEALTHY",
		receiptStore:   store,
		intents:        make(map[string]*operatorIntent),
		approvals:      make(map[string]*operatorApproval),
		operatorRuns:   make(map[string]*operatorRunState),
	}
}

func TestRunsListAPI(t *testing.T) {
	srv := newTestServerWithReceipts()

	t.Run("returns runs list with valid schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/runs", nil)
		w := httptest.NewRecorder()

		srv.handleRunsListAPI(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp runsListResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if resp.Total != 2 {
			t.Fatalf("expected total=2, got %d", resp.Total)
		}
		if len(resp.Runs) != 2 {
			t.Fatalf("expected 2 runs, got %d", len(resp.Runs))
		}
		if resp.Page != 1 {
			t.Fatalf("expected page=1, got %d", resp.Page)
		}

		// Validate first run shape
		run := resp.Runs[0]
		if run.RunID == "" {
			t.Fatal("run_id should not be empty")
		}
		if run.Status == "" {
			t.Fatal("status should not be empty")
		}
		if run.CurrentStage == "" {
			t.Fatal("current_stage should not be empty")
		}
		if len(run.Stages) == 0 {
			t.Fatal("stages should not be empty")
		}
		if len(run.Effects) == 0 {
			t.Fatal("effects should not be empty")
		}
	})

	t.Run("pagination works", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/runs?page=1&page_size=1", nil)
		w := httptest.NewRecorder()

		srv.handleRunsListAPI(w, req)

		var resp runsListResponse
		_ = json.NewDecoder(w.Body).Decode(&resp)

		if len(resp.Runs) != 1 {
			t.Fatalf("expected 1 run on page 1 with page_size=1, got %d", len(resp.Runs))
		}
		if resp.Total != 2 {
			t.Fatalf("expected total=2, got %d", resp.Total)
		}
		if resp.PageSize != 1 {
			t.Fatalf("expected page_size=1, got %d", resp.PageSize)
		}
	})

	t.Run("maps receipt status to run status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/runs", nil)
		w := httptest.NewRecorder()

		srv.handleRunsListAPI(w, req)

		var resp runsListResponse
		_ = json.NewDecoder(w.Body).Decode(&resp)

		// First receipt is SUCCESS → run status should be "complete"
		if resp.Runs[0].Status != "complete" {
			t.Fatalf("expected status=complete for SUCCESS receipt, got %s", resp.Runs[0].Status)
		}
		// Second receipt is FAILURE → run status should be "failed"
		if resp.Runs[1].Status != "failed" {
			t.Fatalf("expected status=failed for FAILURE receipt, got %s", resp.Runs[1].Status)
		}
	})

	t.Run("wrong method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/runs", nil)
		w := httptest.NewRecorder()

		srv.handleRunsListAPI(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", w.Code)
		}
	})
}

func TestRunDetailAPI(t *testing.T) {
	srv := newTestServerWithReceipts()

	t.Run("returns single run by ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/runs/rcpt-001", nil)
		w := httptest.NewRecorder()

		srv.handleRunDetailAPI(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var run apiRun
		if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if run.RunID != "rcpt-001" {
			t.Fatalf("expected run_id=rcpt-001, got %s", run.RunID)
		}
		if run.Status != "complete" {
			t.Fatalf("expected status=complete, got %s", run.Status)
		}
		if run.Receipt == nil {
			t.Fatal("receipt should not be nil")
		}
		if run.Receipt.ExecutorID != "helm-agent" {
			t.Fatalf("expected executor_id=helm-agent, got %s", run.Receipt.ExecutorID)
		}
	})

	t.Run("returns 404 for unknown run", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/runs/nonexistent", nil)
		w := httptest.NewRecorder()

		srv.handleRunDetailAPI(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/runs/rcpt-001", nil)
		w := httptest.NewRecorder()

		srv.handleRunDetailAPI(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", w.Code)
		}
	})
}
