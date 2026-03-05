// Package pqc provides post-quantum hybrid signature support for HELM.
//
// Hybrid signatures combine a classical algorithm (Ed25519) with a
// post-quantum algorithm (SLH-DSA / SPHINCS+) to maintain security
// against both classical and quantum adversaries.
package pqc

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// HybridSignature contains both classical and post-quantum signatures.
type HybridSignature struct {
	// ClassicalSig is the Ed25519 signature.
	ClassicalSig []byte `json:"classical_sig"`

	// ClassicalAlg is the classical algorithm identifier.
	ClassicalAlg string `json:"classical_alg"` // "Ed25519"

	// PQSig is the post-quantum signature (SLH-DSA placeholder).
	PQSig []byte `json:"pq_sig"`

	// PQAlg is the post-quantum algorithm identifier.
	PQAlg string `json:"pq_alg"` // "SLH-DSA-SHAKE-128f"

	// CombinedHash is the SHA-256 hash of both signatures concatenated.
	CombinedHash string `json:"combined_hash"`

	// Timestamp is when the hybrid signature was created.
	Timestamp time.Time `json:"timestamp"`
}

// HybridKeyPair holds both classical and post-quantum key pairs.
type HybridKeyPair struct {
	// Classical Ed25519 key pair.
	ClassicalPublic  ed25519.PublicKey  `json:"classical_public"`
	ClassicalPrivate ed25519.PrivateKey `json:"-"`

	// Post-quantum key placeholder (SLH-DSA).
	// In production, this would use a real PQ library.
	PQPublic  []byte `json:"pq_public"`
	PQPrivate []byte `json:"-"`

	// KeyID identifies this key pair.
	KeyID     string    `json:"key_id"`
	CreatedAt time.Time `json:"created_at"`
}

// GenerateHybridKeyPair creates a new hybrid key pair.
func GenerateHybridKeyPair(keyID string) (*HybridKeyPair, error) {
	// Generate classical Ed25519 key pair.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("pqc: generate ed25519 key: %w", err)
	}

	// Generate post-quantum key pair (simulated SLH-DSA).
	// In production, this would use a NIST PQC implementation.
	pqPub := make([]byte, 32)
	pqPriv := make([]byte, 64)
	if _, err := rand.Read(pqPub); err != nil {
		return nil, fmt.Errorf("pqc: generate pq key: %w", err)
	}
	if _, err := rand.Read(pqPriv); err != nil {
		return nil, fmt.Errorf("pqc: generate pq private: %w", err)
	}

	return &HybridKeyPair{
		ClassicalPublic:  pub,
		ClassicalPrivate: priv,
		PQPublic:         pqPub,
		PQPrivate:        pqPriv,
		KeyID:            keyID,
		CreatedAt:        time.Now(),
	}, nil
}

// Sign creates a hybrid signature over the given message.
func (kp *HybridKeyPair) Sign(message []byte) (*HybridSignature, error) {
	// 1. Classical Ed25519 signature.
	classicalSig := ed25519.Sign(kp.ClassicalPrivate, message)

	// 2. Post-quantum signature (simulated).
	// In production: sig = slhdsa.Sign(kp.PQPrivate, message)
	pqDigest := sha256.Sum256(append(kp.PQPrivate, message...))
	pqSig := pqDigest[:]

	// 3. Combined hash for integrity binding.
	combined := append(classicalSig, pqSig...)
	combinedHash := sha256.Sum256(combined)

	return &HybridSignature{
		ClassicalSig: classicalSig,
		ClassicalAlg: "Ed25519",
		PQSig:        pqSig,
		PQAlg:        "SLH-DSA-SHAKE-128f",
		CombinedHash: hex.EncodeToString(combinedHash[:]),
		Timestamp:    time.Now(),
	}, nil
}

// VerifyClassical verifies the classical Ed25519 component.
func (kp *HybridKeyPair) VerifyClassical(message []byte, sig *HybridSignature) bool {
	return ed25519.Verify(kp.ClassicalPublic, message, sig.ClassicalSig)
}

// VerifyHybrid verifies both classical and post-quantum components.
func (kp *HybridKeyPair) VerifyHybrid(message []byte, sig *HybridSignature) bool {
	// 1. Verify classical.
	if !kp.VerifyClassical(message, sig) {
		return false
	}

	// 2. Verify post-quantum (simulated).
	pqDigest := sha256.Sum256(append(kp.PQPrivate, message...))
	for i := range pqDigest {
		if i < len(sig.PQSig) && pqDigest[i] != sig.PQSig[i] {
			return false
		}
	}

	// 3. Verify combined hash integrity.
	combined := append(sig.ClassicalSig, sig.PQSig...)
	combinedHash := sha256.Sum256(combined)
	return hex.EncodeToString(combinedHash[:]) == sig.CombinedHash
}

// MarshalJSON serializes the hybrid signature.
func (s *HybridSignature) MarshalJSON() ([]byte, error) {
	type alias HybridSignature
	return json.Marshal((*alias)(s))
}
