package approvalceremony

// quantum_posture: classical Ed25519 effect-close v3 reference vectors only;
// these fixtures do not establish runtime or provider proof.

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-ai-kernel/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm-ai-kernel/core/pkg/crypto"
)

func TestEffectCloseV3ReferencePackMatchesGoImplementation(t *testing.T) {
	files := buildEffectCloseV3ReferencePack(t)
	root := filepath.Join("..", "..", "..", "..", "reference_packs", "effect-close-v3")
	if os.Getenv("UPDATE_EFFECT_CLOSE_V3_VECTORS") == "1" {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("create effect close v3 reference pack: %v", err)
		}
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(root, name), content, 0o644); err != nil {
				t.Fatalf("write %s: %v", name, err)
			}
		}
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s differs from source-owned Go fixture; run UPDATE_EFFECT_CLOSE_V3_VECTORS=1 go test ./pkg/boundary/approvalceremony -run TestEffectCloseV3ReferencePackMatchesGoImplementation", name)
		}
	}
}

func buildEffectCloseV3ReferencePack(t *testing.T) map[string][]byte {
	t.Helper()
	observedAt := time.Date(2026, 9, 6, 12, 0, 0, 123456000, time.UTC)
	deadline := observedAt.Add(2 * time.Hour)
	acknowledgement, err := (contracts.ConnectorEffectAcknowledgementV3{ConnectorEffectAcknowledgementV2: contracts.ConnectorEffectAcknowledgementV2{
		SchemaVersion:     contracts.ConnectorEffectAcknowledgementSchemaV3,
		ContractVersion:   contracts.ConnectorEffectAcknowledgementContractV3,
		AcknowledgementID: "effect-ack-vector-v3", AdmissionID: "dispatch-admission-vector-v3",
		AttemptID: "attempt-vector-v3", TenantID: "tenant-a", WorkspaceID: "workspace-a",
		Audience: "packs.lifecycle", ConnectorID: "github", ConnectorVersion: "1.2.3",
		ConnectorAction: "github.create_issue", ConnectorExecutionRef: "github-request-vector-v3",
		AdapterID: "adapter.github-app", AdapterVersion: "2026.09.06",
		AdapterCapabilityRef: "capability:github.issue.create", AdapterCapabilityHash: effectCloseVectorSHA("adapter-capability-v3"),
		ProofSessionRef: "proof-session-vector-v3", IntentRef: "intent-vector-v3",
		ActivationRecordRef: "activation-record-vector-v3", ActivationRecordHash: effectCloseVectorSHA("activation-record-v3"),
		IdempotencyKeyHash: effectCloseVectorSHA("idempotency-v3"), EffectHash: effectCloseVectorSHA("effect-v3"),
		Outcome: contracts.ConnectorEffectOutcomeApplied, ResponseHash: effectCloseVectorSHA("response-v3"),
		EffectRef: "github-issue-77", ReconciliationRef: "reconciliation-vector-v3",
		Finality: &contracts.EffectCloseFinalityV2{
			ExpectedFinalityPredicateRef:  "predicate:github.issue.created",
			ExpectedFinalityPredicateHash: effectCloseVectorSHA("predicate-v3"),
			ObservedExternalFacts: []contracts.EffectCloseObservedExternalFactV2{
				{Ref: "fact:delivery", Hash: effectCloseVectorSHA("fact-delivery-v3")},
				{Ref: "fact:webhook", Hash: effectCloseVectorSHA("fact-webhook-v3")},
			},
			ReconciliationDeadline: deadline, ResolutionRef: "resolution-vector-v3",
			ResolutionState:            contracts.EffectCloseResolutionStateCompensated,
			ConditionalCompensationRef: "compensation-vector-v3",
		},
		IssuerID: "publisher-a", SigningKeyRef: "kms://helm/connector-ack/key-a",
		Algorithm: contracts.ConnectorEffectAcknowledgementAlgorithm, ObservedAt: observedAt,
	}, DispositionReceiptHash: effectCloseVectorSHA("disposition-receipt-v3")}).Seal()
	if err != nil {
		t.Fatal(err)
	}
	ackSigner := crypto.NewEd25519SignerFromKey(
		ed25519.NewKeyFromSeed(bytes.Repeat([]byte{73}, ed25519.SeedSize)), "effect-ack-vector-v3",
	)
	ackEnvelope, err := SignConnectorEffectAcknowledgementV3(acknowledgement, ackSigner)
	if err != nil {
		t.Fatal(err)
	}
	ackJSON := effectCloseCanonical(t, acknowledgement)
	ackEnvelopeJSON := effectCloseCanonical(t, ackEnvelope)
	ackPayload, err := ConnectorEffectAcknowledgementV3SigningPayload(acknowledgement)
	if err != nil {
		t.Fatal(err)
	}

	receipt, err := (contracts.EffectCloseReceiptV3{EffectCloseReceiptV2: contracts.EffectCloseReceiptV2{
		SchemaVersion: contracts.EffectCloseReceiptSchemaV3, ContractVersion: contracts.EffectCloseReceiptContractV3,
		CloseID: "effect-close-vector-v3", State: contracts.EffectCloseReceiptStateClosed,
		AdmissionID: acknowledgement.AdmissionID, AttemptID: acknowledgement.AttemptID,
		TenantID: acknowledgement.TenantID, WorkspaceID: acknowledgement.WorkspaceID, Audience: acknowledgement.Audience,
		ConnectorID: acknowledgement.ConnectorID, ConnectorVersion: acknowledgement.ConnectorVersion,
		ConnectorAction: acknowledgement.ConnectorAction, AdapterID: acknowledgement.AdapterID, AdapterVersion: acknowledgement.AdapterVersion,
		AdapterCapabilityRef: acknowledgement.AdapterCapabilityRef, AdapterCapabilityHash: acknowledgement.AdapterCapabilityHash,
		PriorState: contracts.EffectClosePriorStateUncertain, ReservationSequence: 3,
		ReservationHeadHash: effectCloseVectorSHA("reservation-head-v3"), AcknowledgementHash: acknowledgement.AcknowledgementHash,
		ActivationRecordRef: acknowledgement.ActivationRecordRef, ActivationRecordHash: acknowledgement.ActivationRecordHash,
		IdempotencyKeyHash: acknowledgement.IdempotencyKeyHash, EffectHash: acknowledgement.EffectHash,
		Outcome: acknowledgement.Outcome, ResponseHash: acknowledgement.ResponseHash,
		ConnectorExecutionRef: acknowledgement.ConnectorExecutionRef, ProofSessionRef: acknowledgement.ProofSessionRef,
		IntentRef: acknowledgement.IntentRef, EffectRef: acknowledgement.EffectRef,
		ReconciliationRef: acknowledgement.ReconciliationRef, Finality: acknowledgement.Finality,
		EvidencePackRef: "evidence-pack-vector-v3", EvidencePackHash: effectCloseVectorSHA("evidence-pack-v3"),
		KernelTrustRootID: "kernel-root-a", SigningKeyRef: "kms://helm/approval/key-a",
		ClosedBy: "spiffe://helm/data-plane-a", ClosedAt: observedAt.Add(time.Second),
	}, DispositionReceiptHash: acknowledgement.DispositionReceiptHash}).Seal()
	if err != nil {
		t.Fatal(err)
	}
	kernelSigner := crypto.NewEd25519SignerFromKey(
		ed25519.NewKeyFromSeed(bytes.Repeat([]byte{74}, ed25519.SeedSize)), "effect-close-vector-v3",
	)
	receiptSignature, err := SignEffectCloseReceiptV3(receipt, kernelSigner)
	if err != nil {
		t.Fatal(err)
	}
	receiptJSON := effectCloseCanonical(t, receipt)
	receiptPayload, err := EffectCloseReceiptV3SigningPayload(receipt, GrantSignatureEd25519)
	if err != nil {
		t.Fatal(err)
	}

	index := effectCloseVectorIndex{
		Comment:         "quantum_posture: classical Ed25519 effect acknowledgement and close receipt only; no hybrid or post-quantum claim.",
		SchemaVersion:   "effect-close-vectors.v3",
		ContractVersion: contracts.EffectCloseReceiptContractV3,
		QuantumPosture:  "classical_ed25519_only",
		Acknowledgement: effectCloseSignedVector{
			Artifact:       effectCloseVectorFile{Canonical: "acknowledgement.c14n.json", SHA256: effectCloseVectorHash(ackJSON)},
			Envelope:       effectCloseVectorFile{Canonical: "acknowledgement_envelope.c14n.json", SHA256: effectCloseVectorHash(ackEnvelopeJSON)},
			SigningPayload: effectCloseVectorFile{Canonical: "acknowledgement_signing_payload.c14n.json", SHA256: effectCloseVectorHash(ackPayload)},
			PublicKey:      "ed25519:" + ackSigner.PublicKey(), Signature: "ed25519:" + ackEnvelope.Signature,
			KeyNotBefore: observedAt.Add(-time.Hour).Format(time.RFC3339Nano), KeyNotAfter: observedAt.Add(time.Hour).Format(time.RFC3339Nano),
		},
		Receipt: effectCloseSignedVector{
			Artifact:       effectCloseVectorFile{Canonical: "receipt.c14n.json", SHA256: effectCloseVectorHash(receiptJSON)},
			SigningPayload: effectCloseVectorFile{Canonical: "receipt_signing_payload.c14n.json", SHA256: effectCloseVectorHash(receiptPayload)},
			PublicKey:      "ed25519:" + kernelSigner.PublicKey(), Signature: "ed25519:" + receiptSignature,
			KeyNotBefore: observedAt.Add(-time.Hour).Format(time.RFC3339Nano), KeyNotAfter: observedAt.Add(time.Hour).Format(time.RFC3339Nano),
		},
		NegativeVectors: []effectCloseNegativeVector{
			{ID: "acknowledgement_disposition_hash_tamper", Mutation: "set_acknowledgement_disposition_hash_to_tampered", ExpectedError: "acknowledgement_hash_mismatch"},
			{ID: "receipt_disposition_substitution", Mutation: "set_receipt_disposition_hash_to_other_and_reseal", ExpectedError: "acknowledgement_binding_rejected"},
			{ID: "receipt_disposition_removed", Mutation: "remove_receipt_disposition_hash_and_reseal", ExpectedError: "acknowledgement_binding_rejected"},
			{ID: "disposition_without_reconciliation", Mutation: "remove_acknowledgement_reconciliation_ref_and_reseal", ExpectedError: "acknowledgement_contract_rejected"},
			{ID: "acknowledgement_activation_hash_tamper", Mutation: "set_acknowledgement_activation_record_hash_to_tampered", ExpectedError: "acknowledgement_hash_mismatch"},
			{ID: "acknowledgement_observed_fact_unsorted", Mutation: "reverse_acknowledgement_observed_external_facts_and_reseal", ExpectedError: "acknowledgement_contract_rejected"},
			{ID: "acknowledgement_compensation_removed", Mutation: "remove_acknowledgement_conditional_compensation_ref_and_reseal", ExpectedError: "acknowledgement_contract_rejected"},
			{ID: "receipt_activation_hash_substitution", Mutation: "set_receipt_activation_record_hash_to_other_and_reseal", ExpectedError: "acknowledgement_binding_rejected"},
			{ID: "receipt_signature_tamper", Mutation: "flip_receipt_signature_last_bit", ExpectedError: "receipt_signature_rejected"},
			{ID: "receipt_deadline_removal", Mutation: "remove_receipt_finality_deadline_and_reseal", ExpectedError: "receipt_contract_rejected"},
			{ID: "receipt_resolution_failure_without_outcome_flip", Mutation: "set_receipt_resolution_state_to_finalized_failure_and_reseal", ExpectedError: "receipt_contract_rejected"},
		},
	}
	indexJSON, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestJSON, err := json.MarshalIndent(map[string]any{
		"source_repository": "Mindburn-Labs/helm-ai-kernel",
		"source_path":       "reference_packs/effect-close-v3",
		"pinning_authority": "reference_packs/effect-close-v3/vectors.json",
		"verifier":          "reference_packs/effect-close-v3/verify_vectors.py",
		"immutable_payloads": []map[string]string{
			{"file": "acknowledgement.c14n.json", "sha256": strings.TrimPrefix(effectCloseVectorHash(ackJSON), "sha256:")},
			{"file": "acknowledgement_envelope.c14n.json", "sha256": strings.TrimPrefix(effectCloseVectorHash(ackEnvelopeJSON), "sha256:")},
			{"file": "acknowledgement_signing_payload.c14n.json", "sha256": strings.TrimPrefix(effectCloseVectorHash(ackPayload), "sha256:")},
			{"file": "receipt.c14n.json", "sha256": strings.TrimPrefix(effectCloseVectorHash(receiptJSON), "sha256:")},
			{"file": "receipt_signing_payload.c14n.json", "sha256": strings.TrimPrefix(effectCloseVectorHash(receiptPayload), "sha256:")},
		},
		"$comment": "quantum_posture: classical_ed25519_only. These `*.c14n.json` files are byte-exact canonical (RFC 8785) connector effect-close acknowledgement v3 and receipt v3 signing payloads whose SHA-256 digests are hash-pinned in vectors.json and covered by a valid classical Ed25519 signature. They cannot host an inline annotation without invalidating that signature or digest, so this manifest carries the posture note. This is not a post-quantum claim and does not alter any payload bytes.",
		"purpose":  "Local immutable-payload posture record for the effect-close v3 reference pack. vectors.json remains the verified hash-pinning authority.",
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{
		"SOURCE-MANIFEST.json":                      append(manifestJSON, '\n'),
		"acknowledgement.c14n.json":                 append(ackJSON, '\n'),
		"acknowledgement_envelope.c14n.json":        append(ackEnvelopeJSON, '\n'),
		"acknowledgement_signing_payload.c14n.json": append(ackPayload, '\n'),
		"receipt.c14n.json":                         append(receiptJSON, '\n'),
		"receipt_signing_payload.c14n.json":         append(receiptPayload, '\n'),
		"vectors.json":                              append(indexJSON, '\n'),
	}
}
