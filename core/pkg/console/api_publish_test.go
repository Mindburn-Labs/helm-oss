package console

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/manifest"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/pack"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleRegistryPublishAPI(t *testing.T) {
	// 1. Setup Trust Infrastructure
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	pubHex := hex.EncodeToString(pub)
	privHex := hex.EncodeToString(priv)
	anchorID := "trusted-partner-1"

	// Mock Registry (InMemory for simplicity, but using Adapter logic internally if we could)
	// Here we just test the API wiring and Verifier logic.
	// We need a dummy registry implementation that satisfies `registry.Registry`.
	// Since `console` tests might not have easy access to `registry` mocks without circular deps or import issues,
	// We will rely on nil registry check in Adapter or mock it if possible.
	// Actually, `NewRegistryAdapter` takes `registry.Registry`.
	// We can use `InMemoryRegistry` from `pack` package if it implemented `registry.Registry`. It does NOT.
	// `pack.InMemoryRegistry` implements `pack.PackRegistry`.
	// `registry.PostgresRegistry` implements `registry.Registry`.
	// We need to mock `registry.Registry`.

	mockReg := &MockLegacyRegistry{}
	verifier := pack.NewVerifier(nil) // We don't need registry for verification, only for lookup which we won't do here
	verifier.AddTrustAnchor(pack.TrustAnchor{
		AnchorID:   anchorID,
		Name:       "Trusted Partner",
		PublicKey:  pubHex,
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().Add(1 * time.Hour),
		TrustLevel: 5,
	})

	srv := &Server{
		packVerifier: verifier,
		registry:     mockReg, // Inject Mock Legacy Registry
	}

	// 2. Build Signed Pack
	p, err := sdk.NewPack("com.partner.pack", "1.0.0", "Partner Pack").
		WithDescription("A verified pack").
		WithSignature(anchorID, privHex).
		Build()
	require.NoError(t, err)

	// 3. Test Success: Publish Valid Pack
	body, _ := json.Marshal(p)
	req := httptest.NewRequest("POST", "/api/registry/publish", bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.handleRegistryPublishAPI(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var res map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&res)
	assert.Equal(t, "published", res["status"])
	assert.True(t, mockReg.registered) // Verify it hit the registry

	// 4. Test Failure: Tampered Pack
	pTampered := *p
	pTampered.ContentHash = "deadbeef" // Tamper hash
	bodyTampered, _ := json.Marshal(pTampered)
	reqTampered := httptest.NewRequest("POST", "/api/registry/publish", bytes.NewReader(bodyTampered))
	wTampered := httptest.NewRecorder()

	srv.handleRegistryPublishAPI(wTampered, reqTampered)
	assert.Equal(t, 400, wTampered.Result().StatusCode) // Bad Request (Hash mismatch)
}

// MockLegacyRegistry satisfies registry.Registry interface
type MockLegacyRegistry struct {
	registered bool
}

func (m *MockLegacyRegistry) Register(bundle *manifest.Bundle) error {
	m.registered = true
	return nil
}

func (m *MockLegacyRegistry) Get(name string) (*manifest.Bundle, error) {
	return nil, nil
}
func (m *MockLegacyRegistry) GetForUser(name, userID string) (*manifest.Bundle, error) {
	return nil, nil
}
func (m *MockLegacyRegistry) SetRollout(name string, canaryBundle *manifest.Bundle, percentage int) error {
	return nil
}
func (m *MockLegacyRegistry) List() []*manifest.Bundle {
	return []*manifest.Bundle{}
}
func (m *MockLegacyRegistry) Unregister(name string) error {
	return nil
}
func (m *MockLegacyRegistry) Install(tenantID, packID string) error {
	return nil
}
