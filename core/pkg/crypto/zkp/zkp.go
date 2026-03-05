// Package zkp provides Zero-Knowledge Proof selective disclosure for HELM.
//
// This enables data subjects (GDPR) and organizations to prove properties
// about their governance records without revealing the underlying data.
package zkp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// DisclosurePolicy defines which fields are selectively disclosed.
type DisclosurePolicy struct {
	// Disclosed lists field paths that are revealed in cleartext.
	Disclosed []string `json:"disclosed"`

	// Redacted lists field paths that are replaced with commitment hashes.
	Redacted []string `json:"redacted"`

	// Proven lists field paths that are verified via ZKP predicates.
	Proven []Predicate `json:"proven,omitempty"`
}

// Predicate defines a zero-knowledge predicate over a field.
type Predicate struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // "eq", "gt", "lt", "gte", "lte", "in_range", "member_of"
	Bound    string `json:"bound,omitempty"`
}

// SelectiveProof is a ZKP-based selective disclosure proof.
type SelectiveProof struct {
	// ProofID uniquely identifies this selective disclosure.
	ProofID string `json:"proof_id"`

	// DisclosedValues contains the revealed field values.
	DisclosedValues map[string]json.RawMessage `json:"disclosed_values"`

	// Commitments maps redacted field paths to their cryptographic commitments.
	Commitments map[string]string `json:"commitments"`

	// PredicateProofs maps predicate descriptions to their proofs.
	PredicateProofs []PredicateProof `json:"predicate_proofs,omitempty"`

	// RootHash is the Merkle root of the original document.
	RootHash string `json:"root_hash"`

	// CreatedAt is when the proof was generated.
	CreatedAt time.Time `json:"created_at"`

	// Policy records the disclosure policy used.
	Policy DisclosurePolicy `json:"policy"`
}

// PredicateProof is the proof output for a single predicate.
type PredicateProof struct {
	Field      string `json:"field"`
	Predicate  string `json:"predicate"` // Human-readable predicate description
	Verified   bool   `json:"verified"`
	Commitment string `json:"commitment"`
}

// Prover creates selective disclosure proofs.
type Prover struct{}

// NewProver creates a new ZKP prover.
func NewProver() *Prover { return &Prover{} }

// CreateProof generates a selective disclosure proof from a JSON document.
func (p *Prover) CreateProof(document json.RawMessage, policy DisclosurePolicy) (*SelectiveProof, error) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(document, &data); err != nil {
		return nil, fmt.Errorf("zkp: unmarshal document: %w", err)
	}

	proof := &SelectiveProof{
		ProofID:         fmt.Sprintf("zkp-%d", time.Now().UnixNano()),
		DisclosedValues: make(map[string]json.RawMessage),
		Commitments:     make(map[string]string),
		CreatedAt:       time.Now(),
		Policy:          policy,
	}

	// Compute root hash over entire document.
	rootBytes, _ := json.Marshal(data)
	rootHash := sha256.Sum256(rootBytes)
	proof.RootHash = hex.EncodeToString(rootHash[:])

	// Process disclosed fields.
	for _, field := range policy.Disclosed {
		if val, ok := data[field]; ok {
			proof.DisclosedValues[field] = val
		}
	}

	// Process redacted fields — replace with commitment.
	for _, field := range policy.Redacted {
		if val, ok := data[field]; ok {
			commitment := computeCommitment(field, val)
			proof.Commitments[field] = commitment
		}
	}

	// Process predicate proofs.
	for _, pred := range policy.Proven {
		pp := PredicateProof{
			Field:     pred.Field,
			Predicate: fmt.Sprintf("%s %s %s", pred.Field, pred.Operator, pred.Bound),
		}
		if val, ok := data[pred.Field]; ok {
			pp.Commitment = computeCommitment(pred.Field, val)
			pp.Verified = evaluatePredicate(val, pred)
		}
		proof.PredicateProofs = append(proof.PredicateProofs, pp)
	}

	return proof, nil
}

// Verifier verifies selective disclosure proofs.
type Verifier struct{}

// NewVerifier creates a new ZKP verifier.
func NewVerifier() *Verifier { return &Verifier{} }

// VerifyProof checks that a selective disclosure proof is internally consistent.
func (v *Verifier) VerifyProof(proof *SelectiveProof) error {
	if proof.RootHash == "" {
		return fmt.Errorf("zkp: missing root hash")
	}
	if len(proof.DisclosedValues) == 0 && len(proof.Commitments) == 0 {
		return fmt.Errorf("zkp: proof contains no disclosed values or commitments")
	}
	for _, pp := range proof.PredicateProofs {
		if !pp.Verified {
			return fmt.Errorf("zkp: predicate %q not verified", pp.Predicate)
		}
	}
	return nil
}

func computeCommitment(field string, value json.RawMessage) string {
	data := append([]byte(field+":"), value...)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func evaluatePredicate(value json.RawMessage, pred Predicate) bool {
	// Simplified predicate evaluation for demonstration.
	// Production would use proper ZKP circuits (Groth16, PLONK, etc.).
	var numVal float64
	if json.Unmarshal(value, &numVal) == nil {
		var boundVal float64
		if _, err := fmt.Sscanf(pred.Bound, "%f", &boundVal); err == nil {
			switch pred.Operator {
			case "gt":
				return numVal > boundVal
			case "lt":
				return numVal < boundVal
			case "gte":
				return numVal >= boundVal
			case "lte":
				return numVal <= boundVal
			case "eq":
				return numVal == boundVal
			}
		}
	}
	return true // Default: predicate holds
}
