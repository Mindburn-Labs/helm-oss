package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SoftHSM provides file-backed Ed25519 key management.
// This is a software implementation suitable for development and testing.
// For production deployments requiring hardware-grade key protection,
// use the PKCS#11 provider in crypto/hsm.
type SoftHSM struct {
	keyDir string
	mu     sync.RWMutex
	keys   map[string]ed25519.PrivateKey
}

func NewSoftHSM(keyDir string) (*SoftHSM, error) {
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create key dir: %w", err)
	}
	return &SoftHSM{
		keyDir: keyDir,
		keys:   make(map[string]ed25519.PrivateKey),
	}, nil
}

// GetSigner returns an Ed25519 signer for the given key label.
// If no key exists at the label path, a new Ed25519 key pair is generated
// and persisted to disk.
func (h *SoftHSM) GetSigner(keyLabel string) (Signer, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check in-memory cache first
	if key, ok := h.keys[keyLabel]; ok {
		return NewEd25519SignerFromKey(key, keyLabel), nil
	}

	keyPath := filepath.Join(h.keyDir, keyLabel+".key")

	// Load existing key
	if _, err := os.Stat(keyPath); err == nil {
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read key %s: %w", keyLabel, err)
		}

		// Handle both seed (32 bytes) and full private key (64 bytes)
		var privKey ed25519.PrivateKey
		switch len(keyBytes) {
		case ed25519.SeedSize:
			privKey = ed25519.NewKeyFromSeed(keyBytes)
		case ed25519.PrivateKeySize:
			privKey = keyBytes
		default:
			return nil, fmt.Errorf("invalid key size for %s: %d", keyLabel, len(keyBytes))
		}

		h.keys[keyLabel] = privKey
		return NewEd25519SignerFromKey(privKey, keyLabel), nil
	}

	// Generate new Ed25519 key pair
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}

	// Persist seed (32 bytes) for compact storage
	if err := os.WriteFile(keyPath, privKey.Seed(), 0o600); err != nil {
		return nil, fmt.Errorf("failed to save key %s: %w", keyLabel, err)
	}

	h.keys[keyLabel] = privKey
	return NewEd25519SignerFromKey(privKey, keyLabel), nil
}
