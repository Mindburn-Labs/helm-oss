package sdk

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/pack"
)

// PackBuilder is a fluent helper for constructing Helm Packs.
type PackBuilder struct {
	manifest       pack.PackManifest
	pendingSigners []pendingSigner
	err            error
}

type pendingSigner struct {
	id  string
	key string
}

// NewPack starts building a new Pack with required fields.
func NewPack(id, version, name string) *PackBuilder {
	return &PackBuilder{
		manifest: pack.PackManifest{
			PackID:        id,
			Version:       version,
			Name:          name,
			SchemaVersion: "1.0.0",
			Capabilities:  []string{},
			ToolBindings:  []pack.ToolBinding{},
		},
	}
}

// WithDescription adds a description.
func (b *PackBuilder) WithDescription(desc string) *PackBuilder {
	b.manifest.Description = desc
	return b
}

// WithCapability adds a required capability.
func (b *PackBuilder) WithCapability(cap string) *PackBuilder {
	b.manifest.Capabilities = append(b.manifest.Capabilities, cap)
	return b
}

// WithToolBinding adds a tool binding constraint.
func (b *PackBuilder) WithToolBinding(toolID, versionConstraint string, required bool) *PackBuilder {
	b.manifest.ToolBindings = append(b.manifest.ToolBindings, pack.ToolBinding{
		ToolID:     toolID,
		Constraint: versionConstraint,
		Required:   required,
	})
	return b
}

// WithActivation sets the activation date.
func (b *PackBuilder) WithActivation(date time.Time) *PackBuilder {
	if b.manifest.Lifecycle == nil {
		b.manifest.Lifecycle = &pack.Lifecycle{}
	}
	b.manifest.Lifecycle.Activation = date
	return b
}

// WithSignature adds a signature to the pack upon building.
// privateKeyHex must be a hex-encoded Ed25519 private key (64 bytes).
func (b *PackBuilder) WithSignature(signerID, privateKeyHex string) *PackBuilder {
	b.pendingSigners = append(b.pendingSigners, pendingSigner{id: signerID, key: privateKeyHex})
	return b
}

// Build finalizes the Pack structure.
func (b *PackBuilder) Build() (*pack.Pack, error) {
	if b.err != nil {
		return nil, b.err
	}

	p := &pack.Pack{
		PackID:    b.manifest.PackID,
		Manifest:  b.manifest,
		CreatedAt: time.Now().UTC(),
	}

	// Calculate Content Hash
	p.ContentHash = pack.ComputePackHash(p)

	// Apply Signatures
	for _, s := range b.pendingSigners {
		privBytes, err := hex.DecodeString(s.key)
		if err != nil {
			return nil, fmt.Errorf("invalid private key hex for signer %s: %w", s.id, err)
		}
		if len(privBytes) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("invalid private key length for signer %s: expected %d, got %d", s.id, ed25519.PrivateKeySize, len(privBytes))
		}
		privKey := ed25519.PrivateKey(privBytes)

		// Sign the ContentHash
		sigBytes := ed25519.Sign(privKey, []byte(p.ContentHash))

		p.Manifest.Signatures = append(p.Manifest.Signatures, pack.Signature{
			SignerID:  s.id,
			Signature: hex.EncodeToString(sigBytes),
			SignedAt:  time.Now().UTC(),
			Algorithm: "ed25519",
		})
	}

	return p, nil
}
