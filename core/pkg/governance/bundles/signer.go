package bundles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// SignedBundle is a policy bundle with a cryptographic signature.
type SignedBundle struct {
	Bundle       *PolicyBundle `json:"bundle"`
	Signature    string        `json:"signature"` // Ed25519 signature over ContentHash
	SignerKeyID  string        `json:"signer_key_id"`
	SignedAt     time.Time     `json:"signed_at"`
	SignatureAlg string        `json:"signature_alg"` // "ed25519"
}

// BundleSigner creates signed bundles.
type BundleSigner struct {
	sign  func(data []byte) (string, error)
	keyID string
}

// NewBundleSigner creates a signer with the given signing function and key ID.
func NewBundleSigner(sign func([]byte) (string, error), keyID string) *BundleSigner {
	return &BundleSigner{sign: sign, keyID: keyID}
}

// Sign signs a policy bundle, producing a SignedBundle.
func (s *BundleSigner) Sign(bundle *PolicyBundle) (*SignedBundle, error) {
	if bundle.ContentHash == "" {
		return nil, fmt.Errorf("bundle must have a content hash before signing")
	}

	sig, err := s.sign([]byte(bundle.ContentHash))
	if err != nil {
		return nil, fmt.Errorf("failed to sign bundle %s: %w", bundle.ID, err)
	}

	return &SignedBundle{
		Bundle:       bundle,
		Signature:    sig,
		SignerKeyID:  s.keyID,
		SignedAt:     time.Now(),
		SignatureAlg: "ed25519",
	}, nil
}

// VerifyBundle verifies a signed bundle's integrity.
// verify is a function that takes data and signature, returns true if valid.
func VerifyBundle(signed *SignedBundle, verify func(data []byte, sig string) (bool, error)) error {
	if signed.Bundle == nil {
		return fmt.Errorf("signed bundle has no bundle")
	}

	// Recompute content hash — must exclude ContentHash itself (same as loader)
	savedHash := signed.Bundle.ContentHash
	signed.Bundle.ContentHash = ""
	canonical, err := json.Marshal(signed.Bundle)
	signed.Bundle.ContentHash = savedHash // Restore immediately
	if err != nil {
		return fmt.Errorf("cannot recompute bundle hash: %w", err)
	}
	h := sha256.Sum256(canonical)
	recomputed := "sha256:" + hex.EncodeToString(h[:])

	if recomputed != signed.Bundle.ContentHash {
		return fmt.Errorf("bundle content hash mismatch: expected %s, got %s", signed.Bundle.ContentHash, recomputed)
	}

	// Verify signature
	valid, err := verify([]byte(signed.Bundle.ContentHash), signed.Signature)
	if err != nil {
		return fmt.Errorf("signature verification error: %w", err)
	}
	if !valid {
		return fmt.Errorf("invalid signature for bundle %s", signed.Bundle.ID)
	}

	return nil
}
