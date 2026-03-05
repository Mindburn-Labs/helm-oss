//go:build !go1.24

package pqc

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Legacy Fallback for Go < 1.24 (No ML-KEM Support)

const (
	AlgorithmMLKEM768 = "ML-KEM-768"
	AlgorithmEd25519  = "Ed25519"
	AlgorithmHybrid   = "Hybrid-ML-KEM+Ed25519"

	MLKEMPublicKeySize    = 1184
	MLKEMPrivateKeySize   = 64
	MLKEMCiphertextSize   = 1088
	MLKEMSharedSecretSize = 32

	Ed25519PublicKeySize  = ed25519.PublicKeySize
	Ed25519PrivateKeySize = ed25519.PrivateKeySize
	Ed25519SignatureSize  = ed25519.SignatureSize
)

// KeyPair stub
type KeyPair struct {
	PublicKey  []byte    `json:"public_key"`
	PrivateKey []byte    `json:"private_key"`
	Algorithm  string    `json:"algorithm"`
	KeyID      string    `json:"key_id"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
}

// Signature stub
type Signature struct {
	Value     []byte    `json:"value"`
	Algorithm string    `json:"algorithm"`
	KeyID     string    `json:"key_id"`
	Timestamp time.Time `json:"timestamp"`
}

// EncapsulatedKey stub
type EncapsulatedKey struct {
	Ciphertext   []byte `json:"ciphertext"`
	SharedSecret []byte `json:"shared_secret"`
	Algorithm    string `json:"algorithm"`
}

// PQCSigner stub for legacy environment
type PQCSigner struct {
	mu          sync.RWMutex
	ed25519Pub  ed25519.PublicKey
	ed25519Priv ed25519.PrivateKey
	enablePQC   bool
	keyID       string
	createdAt   time.Time
	expiresAt   time.Time
}

type PQCConfig struct {
	KeyID     string
	EnablePQC bool
	KeyExpiry time.Duration
}

func DefaultPQCConfig() *PQCConfig {
	return &PQCConfig{
		KeyID:     generateKeyID(),
		EnablePQC: false, // Disabled in legacy
		KeyExpiry: 365 * 24 * time.Hour,
	}
}

func NewPQCSigner(config *PQCConfig) (*PQCSigner, error) {
	if config == nil {
		config = DefaultPQCConfig()
	}

	now := time.Now()
	signer := &PQCSigner{
		enablePQC: false, // Force disable PQC in legacy
		keyID:     config.KeyID,
		createdAt: now,
		expiresAt: now.Add(config.KeyExpiry),
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ed25519 keygen failed: %w", err)
	}
	signer.ed25519Pub = pub
	signer.ed25519Priv = priv

	return signer, nil
}

// NewPQCSignerFromKeys creates a signer from existing keys (Legacy: Ed25519 only).
func NewPQCSignerFromKeys(ed25519Priv ed25519.PrivateKey, mlkemSeed []byte, keyID string) (*PQCSigner, error) {
	if len(ed25519Priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid ed25519 private key size")
	}

	signer := &PQCSigner{
		enablePQC:   false,
		keyID:       keyID,
		createdAt:   time.Now(),
		ed25519Priv: ed25519Priv,
		ed25519Pub:  ed25519Priv.Public().(ed25519.PublicKey),
	}
	// In legacy mode, we ignore mlkemSeed as we don't support PQC
	return signer, nil
}

func (s *PQCSigner) Sign(data []byte) (*Signature, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sig := ed25519.Sign(s.ed25519Priv, data)
	return &Signature{
		Value:     sig,
		Algorithm: AlgorithmEd25519,
		KeyID:     s.keyID,
		Timestamp: time.Now(),
	}, nil
}

func (s *PQCSigner) SignWithContext(data []byte, context string) (*Signature, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := sha256.New()
	h.Write([]byte(context))
	h.Write(data)
	message := h.Sum(nil)
	sig := ed25519.Sign(s.ed25519Priv, message)
	return &Signature{
		Value:     sig,
		Algorithm: AlgorithmEd25519,
		KeyID:     s.keyID,
		Timestamp: time.Now(),
	}, nil
}

func (s *PQCSigner) Verify(data []byte, sig *Signature) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ed25519.Verify(s.ed25519Pub, data, sig.Value), nil
}

func (s *PQCSigner) VerifyWithContext(data []byte, sig *Signature, context string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := sha256.New()
	h.Write([]byte(context))
	h.Write(data)
	message := h.Sum(nil)
	return ed25519.Verify(s.ed25519Pub, message, sig.Value), nil
}

func (s *PQCSigner) Encapsulate(recipientPubKey []byte) (*EncapsulatedKey, error) {
	return nil, fmt.Errorf("PQC not supported in Go < 1.24")
}

func (s *PQCSigner) EncapsulateToSelf() (*EncapsulatedKey, error) {
	return nil, fmt.Errorf("PQC not supported in Go < 1.24")
}

func (s *PQCSigner) Decapsulate(ciphertext []byte) ([]byte, error) {
	return nil, fmt.Errorf("PQC not supported in Go < 1.24")
}

func (s *PQCSigner) PublicKeys() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]string{
		"ed25519": hex.EncodeToString(s.ed25519Pub),
	}
}

func (s *PQCSigner) Ed25519PublicKey() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return []byte(s.ed25519Pub)
}

func (s *PQCSigner) MLKEMPublicKey() []byte {
	return nil
}

func (s *PQCSigner) KeyID() string {
	return s.keyID
}

func (s *PQCSigner) IsPQCEnabled() bool {
	return false
}

func (s *PQCSigner) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().After(s.expiresAt)
}

func (s *PQCSigner) CreatedAt() time.Time {
	return s.createdAt
}

func (s *PQCSigner) ExpiresAt() time.Time {
	return s.expiresAt
}

func generateKeyID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use time-based ID rather than panicking in a governance system
		h := sha256.Sum256([]byte(fmt.Sprintf("fallback-%d", time.Now().UnixNano())))
		return hex.EncodeToString(h[:])[:16]
	}
	return hex.EncodeToString(b)[:16]
}

func (s *PQCSigner) KeyPairFromSigner(algorithm string) *KeyPair {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if algorithm == AlgorithmEd25519 {
		return &KeyPair{
			PublicKey:  []byte(s.ed25519Pub),
			PrivateKey: s.ed25519Priv.Seed(),
			Algorithm:  AlgorithmEd25519,
			KeyID:      s.keyID,
			CreatedAt:  s.createdAt,
			ExpiresAt:  s.expiresAt,
		}
	}
	return nil
}
