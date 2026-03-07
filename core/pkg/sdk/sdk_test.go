package sdk_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/pack"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackBuilder_Build(t *testing.T) {
	// 1. Build a new Pack
	now := time.Now()
	p, err := sdk.NewPack("pack.ops.custom", "1.0.0", "My Custom Pack").
		WithDescription("A verified custom pack").
		WithCapability("scan_network").
		WithToolBinding("nmap", ">= 7.0", true).
		WithActivation(now).
		Build()

	require.NoError(t, err)

	// 2. Verify Fields
	assert.Equal(t, "pack.ops.custom", p.PackID)
	assert.Equal(t, "1.0.0", p.Manifest.Version)
	assert.Equal(t, "My Custom Pack", p.Manifest.Name)
	assert.Equal(t, "A verified custom pack", p.Manifest.Description)
	assert.Equal(t, "1.0.0", p.Manifest.SchemaVersion)
	assert.Contains(t, p.Manifest.Capabilities, "scan_network")

	require.Len(t, p.Manifest.ToolBindings, 1)
	assert.Equal(t, "nmap", p.Manifest.ToolBindings[0].ToolID)
	assert.Equal(t, ">= 7.0", p.Manifest.ToolBindings[0].Constraint)

	assert.True(t, p.Manifest.Lifecycle.Activation.Equal(now))
}

func TestPackSigning(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	privHex := hex.EncodeToString(priv)
	signerID := "key-1"

	p, err := sdk.NewPack("signed.pack", "1.0.0", "Signed").
		WithSignature(signerID, privHex).
		Build()

	require.NoError(t, err)
	require.Len(t, p.Manifest.Signatures, 1)
	assert.Equal(t, signerID, p.Manifest.Signatures[0].SignerID)
	assert.Equal(t, "ed25519", p.Manifest.Signatures[0].Algorithm)

	// Verify manually
	sigBytes, _ := hex.DecodeString(p.Manifest.Signatures[0].Signature)
	hash := pack.ComputePackHash(p) // This function must be exported or accessible. It is exported.
	assert.True(t, ed25519.Verify(pub, []byte(hash), sigBytes))
}
