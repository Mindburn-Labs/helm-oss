package audit

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// AuditAttestation is a cryptographically signed statement that an audit
// report with a specific hash was produced at a given git SHA.
//
// This prevents post-hoc tampering — anyone can verify the attestation
// against the report without access to the signing key.
type AuditAttestation struct {
	Kind       string    `json:"kind"`
	Version    string    `json:"version"`
	ReportHash string    `json:"report_hash"`
	MerkleRoot string    `json:"merkle_root"`
	GitSHA     string    `json:"git_sha"`
	SignerID   string    `json:"signer_id"`
	Signature  string    `json:"signature"`
	Timestamp  time.Time `json:"timestamp"`
	ReportPath string    `json:"report_path,omitempty"`
}

// NewAttestation creates an unsigned attestation from report data.
func NewAttestation(reportHash, merkleRoot, gitSHA, signerID string) *AuditAttestation {
	return &AuditAttestation{
		Kind:       "AUDIT_ATTESTATION",
		Version:    "1.0.0",
		ReportHash: reportHash,
		MerkleRoot: merkleRoot,
		GitSHA:     gitSHA,
		SignerID:   signerID,
		Timestamp:  time.Now().UTC(),
	}
}

// SignablePayload returns the canonical bytes that get signed.
func (a *AuditAttestation) SignablePayload() []byte {
	payload := fmt.Sprintf("%s:%s:%s:%s",
		a.ReportHash, a.MerkleRoot, a.GitSHA, a.Timestamp.Format(time.RFC3339))
	return []byte(payload)
}

// Sign signs the attestation with an Ed25519 private key.
func (a *AuditAttestation) Sign(privateKey ed25519.PrivateKey) error {
	if len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("attestation: invalid private key size")
	}
	sig := ed25519.Sign(privateKey, a.SignablePayload())
	a.Signature = hex.EncodeToString(sig)
	return nil
}

// VerifyAttestation verifies that the attestation signature is valid
// and that the report hash matches the given report data.
func VerifyAttestation(attestation *AuditAttestation, publicKey ed25519.PublicKey, reportData []byte) error {
	if attestation == nil {
		return fmt.Errorf("attestation: nil attestation (fail-closed)")
	}

	// 1. Verify report hash matches
	h := sha256.Sum256(reportData)
	computedHash := "sha256:" + hex.EncodeToString(h[:])
	if computedHash != attestation.ReportHash {
		return fmt.Errorf("attestation: report hash mismatch (computed %s, attested %s)",
			truncate(computedHash, 24), truncate(attestation.ReportHash, 24))
	}

	// 2. Verify signature
	if attestation.Signature == "" || attestation.Signature == "unsigned" {
		return fmt.Errorf("attestation: unsigned attestation")
	}

	sigBytes, err := hex.DecodeString(attestation.Signature)
	if err != nil {
		return fmt.Errorf("attestation: invalid signature encoding: %w", err)
	}

	if !ed25519.Verify(publicKey, attestation.SignablePayload(), sigBytes) {
		return fmt.Errorf("attestation: signature verification failed")
	}

	return nil
}

// MarshalJSON serializes the attestation.
func (a *AuditAttestation) MarshalJSON() ([]byte, error) {
	type alias AuditAttestation
	return json.Marshal((*alias)(a))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
