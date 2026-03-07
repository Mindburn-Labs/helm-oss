package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/api"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/guardian"
	trustregistry "github.com/Mindburn-Labs/helm-oss/core/pkg/trust/registry"
)

// RegisterSubsystemRoutes registers all subsystem API routes on the given mux.
// This wires kernel-critical packages into the HTTP API surface.
// Non-TCB enterprise subsystems have been removed from OSS.
//
//nolint:gocyclo,gocognit // Route registration is linear and intentionally exhaustive.
func RegisterSubsystemRoutes(mux *http.ServeMux, svc *Services) {
	log.Println("[helm] routes: Registering API routes...")

	ctx := context.Background()
	versionInfo := map[string]any{
		"version":    displayVersion(),
		"commit":     displayCommit(),
		"build_time": displayBuildTime(),
		"go_version": runtime.Version(),
	}

	// --- OpenAI-Compatible Proxy (governed inference) ---
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteMethodNotAllowed(w)
			return
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			api.WriteBadRequest(w, "Invalid JSON body")
			return
		}

		// Governance Check (Guardian)
		if svc.Guardian != nil {
			model, _ := body["model"].(string)
			req := guardian.DecisionRequest{
				Principal: "developer",
				Action:    "LLM_INFERENCE",
				Resource:  model,
				Context:   body,
			}
			decision, err := svc.Guardian.EvaluateDecision(r.Context(), req)
			if err != nil {
				api.WriteInternal(w, err)
				return
			}
			if contracts.Verdict(decision.Verdict) != contracts.VerdictAllow {
				api.WriteError(w, http.StatusForbidden, "Governance Blocked", decision.Reason)
				return
			}
		}

		// Forward to Upstream (Mock for now)
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"id":      "chatcmpl-mock-" + fmt.Sprintf("%d", time.Now().Unix()),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   body["model"],
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "This is a governed response from the HELM Kernel.",
					},
					"finish_reason": "stop",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// --- Evidence Export ---
	mux.HandleFunc("/api/v1/evidence/soc2", func(w http.ResponseWriter, r *http.Request) {
		bundle, err := svc.Evidence.ExportSOC2(r.Context(), "trace-"+time.Now().Format("20060102"), nil)
		if err != nil {
			api.WriteInternal(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bundle)
	})

	// --- Merkle Root ---
	mux.HandleFunc("/api/v1/merkle/root", func(w http.ResponseWriter, r *http.Request) {
		root := "uninitialized"
		if svc.MerkleTree != nil {
			root = svc.MerkleTree.Root
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"root": root})
	})

	// --- Budget ---
	mux.HandleFunc("/api/v1/budget/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"enforcer": "postgres",
			"status":   "active",
		})
	})

	// --- Authz ---
	mux.HandleFunc("/api/v1/authz/check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"engine": "rebac",
			"status": "active",
		})
	})

	// --- Identity ---
	mux.HandleFunc("/api/v1/identity/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keyset": "active",
			"type":   "in-memory-rsa",
		})
	})

	// --- Tenants ---
	mux.HandleFunc("/api/v1/tenants/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"provisioner": "postgres",
			"status":      "active",
		})
	})

	// --- Tiers ---
	mux.HandleFunc("/api/v1/tiers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(svc.Tiers)
	})

	// --- SDK ---
	mux.HandleFunc("/api/v1/sdk/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sdk":     "helm-sdk",
			"version": "1.0.0",
		})
	})

	// --- Version ---
	versionHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(versionInfo)
	}
	mux.HandleFunc("/api/v1/version", versionHandler)
	mux.HandleFunc("/version", versionHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"version": displayVersion(),
		})
	})

	// --- Obligation ---
	mux.HandleFunc("/api/v1/obligation/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteMethodNotAllowed(w)
			return
		}
		var req struct {
			GoalSpec string `json:"goal_spec"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.WriteBadRequest(w, "Invalid body")
			return
		}
		obl, err := svc.Obligation.CreateObligation(req.GoalSpec)
		if err != nil {
			api.WriteInternal(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(obl)
	})

	// --- Boundary ---
	mux.HandleFunc("/api/v1/boundary/check", func(w http.ResponseWriter, r *http.Request) {
		if svc.BoundaryEnforcer == nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "disabled"})
			return
		}
		targetURL := r.URL.Query().Get("url")
		if targetURL != "" {
			err := svc.BoundaryEnforcer.CheckNetwork(r.Context(), targetURL)
			w.Header().Set("Content-Type", "application/json")
			if err != nil {
				_ = json.NewEncoder(w).Encode(map[string]any{"allowed": false, "reason": err.Error()})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]any{"allowed": true})
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"enforcer": "active", "status": "ready"})
	})

	// --- Sandbox ---
	mux.HandleFunc("/api/v1/sandbox/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"sandbox": "in-process", "status": "active"})
	})

	// --- Config ---
	mux.HandleFunc("/api/v1/config/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"port":      svc.Config.Port,
			"log_level": svc.Config.LogLevel,
		})
	})

	// --- Credentials ---
	if svc.Creds != nil {
		svc.Creds.RegisterRoutes(mux)
		log.Println("[helm] routes: Credential management routes registered")
	}

	// --- Trust Keys ---
	trustKeys := &api.TrustKeyHandler{Registry: trustregistry.NewTrustRegistry()}
	mux.HandleFunc("/api/v1/trust/keys/add", trustKeys.HandleAddKey)
	mux.HandleFunc("/api/v1/trust/keys/revoke", trustKeys.HandleRevokeKey)

	// --- MCP Gateway ---
	mcpGateway, err := newLocalMCPGateway()
	if err != nil {
		log.Printf("[helm] routes: MCP gateway unavailable: %v", err)
	} else {
		mcpGateway.RegisterRoutes(mux)
		log.Println("[helm] routes: MCP gateway routes registered")
	}

	// Suppress unused variable
	_ = ctx

	log.Println("[helm] routes: All subsystem routes registered")
}
