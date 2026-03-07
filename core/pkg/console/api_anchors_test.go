package console

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/pack"
	"github.com/stretchr/testify/assert"
)

func TestHandleRegistryAnchorsAPI(t *testing.T) {
	// Setup
	// We need a dummy registry for NewVerifier
	// But NewVerifier takes PackRegistry interface.
	// Since we can't easily implement the interface with "interface{}" arguments in Mock above without importing context correctly,
	// I'll rely on the real verifier with a nil registry if acceptable, or a minimal mock.
	// Actually, Verifier doesn't use registry for AddTrustAnchor.
	// But NewVerifier might store it.

	verifier := pack.NewVerifier(nil) // nil registry should be safe if we don't call Verify
	srv := &Server{
		packVerifier: verifier,
	}

	// Payload
	anchor := pack.TrustAnchor{
		AnchorID:   "anchor-1",
		Name:       "Test Anchor",
		PublicKey:  "deadbeef",
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().Add(24 * time.Hour),
		TrustLevel: 5,
	}
	body, _ := json.Marshal(anchor)

	req := httptest.NewRequest("POST", "/api/registry/anchors", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	srv.handleRegistryAnchorsAPI(w, req)

	// Verify Response
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "anchor added", result["status"])

	// Verify State (checking private field trustAnchors via reflection or if we exported a getter?
	// Or just trust the test passed if no error. pack.Verifier doesn't export TrustAnchors.
	// But since we are in `console` package, `pack` is external.
	// We can trust it works if we trust `verifier.AddTrustAnchor` works (unit tested elsewhere).
	// This test confirms the API wiring.)
}
