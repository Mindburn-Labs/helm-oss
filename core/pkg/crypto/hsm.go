package crypto

import (
	"crypto/ed25519"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/crypto/pqc"
)

// SoftHSM simulates a Hardware Security Module using file-backed keys.
type SoftHSM struct {
	keyDir string
	mu     sync.RWMutex
	keys   map[string]ed25519.PrivateKey
}

func NewSoftHSM(keyDir string) (*SoftHSM, error) {
	if err := os.MkdirAll(keyDir,0o700); err != nil {
		return nil, fmt.Errorf("failed to create key dir: %w", err)
	}
	return &SoftHSM{
		keyDir: keyDir,
		keys:   make(map[string]ed25519.PrivateKey),
	}, nil
}

// GetPQCSigner returns a fully initialized hybrid PQC signer.
func (h *SoftHSM) GetPQCSigner(keyLabel string) (*pqc.PQCSigner, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 1. Check if we already have a PQC signer in memory?
	// The current `keys` map stores `ed25519.PrivateKey`.
	// We need to upgrade the cache or just rehydrate every time (hsm is slow anyway).
	// Let's rehydrate for now to avoid breaking `keys` map type change yet, or I should update struct.
	// Actually, let's just read from disk.

	// Paths
	edKeyPath := filepath.Join(h.keyDir, keyLabel+".key")
	mlkemKeyPath := filepath.Join(h.keyDir, keyLabel+".mlkem")

	// 2. Generate if missing (Ed25519 is mandatory)
	if _, err := os.Stat(edKeyPath); os.IsNotExist(err) {
		// Generate NEW PQC Signer
		signer, err := pqc.NewPQCSigner(nil) // Default config
		if err != nil {
			return nil, fmt.Errorf("failed to generate PQC signer: %w", err)
		}

		// Save Ed25519
		kp := signer.KeyPairFromSigner(pqc.AlgorithmEd25519)
		if err := os.WriteFile(edKeyPath, kp.PrivateKey,0o600); err != nil {
			return nil, fmt.Errorf("failed to save ed25519 key: %w", err)
		}

		// Save ML-KEM
		mlKP := signer.KeyPairFromSigner(pqc.AlgorithmMLKEM768)
		if mlKP != nil {
			if err := os.WriteFile(mlkemKeyPath, mlKP.PrivateKey,0o600); err != nil {
				return nil, fmt.Errorf("failed to save mlkem key: %w", err)
			}
		}

		return signer, nil
	}

	// 3. Load Existing
	edBytes, err := os.ReadFile(edKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ed25519 key: %w", err)
	}
	// Historical format: we persisted the Ed25519 seed (32 bytes). Accept and expand it.
	if len(edBytes) == ed25519.SeedSize {
		edBytes = ed25519.NewKeyFromSeed(edBytes)
	}
	if len(edBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid ed25519 key size")
	}

	// Try load ML-KEM
	var mlkemSeed []byte
	if _, err := os.Stat(mlkemKeyPath); err == nil {
		mlkemSeed, err = os.ReadFile(mlkemKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read mlkem key: %w", err)
		}
	}

	return pqc.NewPQCSignerFromKeys(ed25519.PrivateKey(edBytes), mlkemSeed, keyLabel)
}

// GetSigner returns a legacy signer interface (wraps PQC signer).
func (h *SoftHSM) GetSigner(keyLabel string) (Signer, error) {
	pqcSigner, err := h.GetPQCSigner(keyLabel)
	if err != nil {
		return nil, err
	}
	// We need an adapter or just return Ed25519Signer for now if PQC isn't needed by legacy consumers.
	// But wait, we want consistency.
	// If `Signer` interface is strictly Ed25519 string-returning, `PQCSigner` doesn't implement it directly yet.
	// I'll return the Ed25519 component as a legacy signer for now.
	// This preserves `main.go` compatibility.
	// `PostgresVaultStorage` will call `GetPQCSigner` explicitly.

	// Extract private key and make a standard signer
	kp := pqcSigner.KeyPairFromSigner(pqc.AlgorithmEd25519)
	return NewEd25519SignerFromKey(ed25519.PrivateKey(kp.PrivateKey), keyLabel), nil
}
