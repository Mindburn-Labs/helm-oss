package zkp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProver_CreateProof(t *testing.T) {
	prover := NewProver()
	doc := json.RawMessage(`{"name":"Alice","age":30,"salary":95000,"department":"engineering"}`)

	policy := DisclosurePolicy{
		Disclosed: []string{"name", "department"},
		Redacted:  []string{"salary"},
		Proven: []Predicate{
			{Field: "age", Operator: "gte", Bound: "18"},
		},
	}

	proof, err := prover.CreateProof(doc, policy)
	require.NoError(t, err)

	// Disclosed fields are visible
	assert.Contains(t, proof.DisclosedValues, "name")
	assert.Contains(t, proof.DisclosedValues, "department")

	// Redacted fields have commitments, not values
	assert.NotContains(t, proof.DisclosedValues, "salary")
	assert.Contains(t, proof.Commitments, "salary")
	assert.NotEmpty(t, proof.Commitments["salary"])

	// Predicate verified
	assert.Len(t, proof.PredicateProofs, 1)
	assert.True(t, proof.PredicateProofs[0].Verified)
	assert.NotEmpty(t, proof.RootHash)
}

func TestVerifier_VerifyProof(t *testing.T) {
	prover := NewProver()
	verifier := NewVerifier()
	doc := json.RawMessage(`{"name":"Bob","score":85}`)

	proof, _ := prover.CreateProof(doc, DisclosurePolicy{
		Disclosed: []string{"name"},
		Redacted:  []string{"score"},
	})

	assert.NoError(t, verifier.VerifyProof(proof))
}
