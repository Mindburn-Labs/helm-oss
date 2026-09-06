package approvalceremony

// quantum_posture: classical Ed25519 v3 signature and recovery-key tests only;
// no hybrid or post-quantum claim.

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-ai-kernel/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm-ai-kernel/core/pkg/crypto"
)

func TestEffectCloseV3SignaturesBindDispositionAndSeparateDomains(t *testing.T) {
	files := buildEffectCloseV3ReferencePack(t)
	var acknowledgement contracts.ConnectorEffectAcknowledgementV3
	var receipt contracts.EffectCloseReceiptV3
	if err := json.Unmarshal(files["acknowledgement.c14n.json"], &acknowledgement); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(files["receipt.c14n.json"], &receipt); err != nil {
		t.Fatal(err)
	}
	connectorSigner := crypto.NewEd25519SignerFromKey(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{73}, ed25519.SeedSize)), "effect-ack-v3-test")
	envelope, err := SignConnectorEffectAcknowledgementV3(acknowledgement, connectorSigner)
	if err != nil {
		t.Fatal(err)
	}
	key := TrustedEffectAcknowledgementKey{
		IssuerID: acknowledgement.IssuerID, SigningKeyRef: acknowledgement.SigningKeyRef,
		ConnectorID: acknowledgement.ConnectorID, ConnectorVersion: acknowledgement.ConnectorVersion,
		PublicKey: connectorSigner.PublicKeyBytes(), Enabled: true,
		NotBefore: acknowledgement.ObservedAt.Add(-time.Hour), NotAfter: acknowledgement.ObservedAt.Add(time.Hour),
	}
	verifier, err := NewEd25519EffectAcknowledgementVerifier([]TrustedEffectAcknowledgementKey{key})
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyEnvelopeV3(envelope); err != nil {
		t.Fatal(err)
	}
	payload, err := ConnectorEffectAcknowledgementV3SigningPayload(acknowledgement)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["disposition_receipt_hash"] != acknowledgement.DispositionReceiptHash {
		t.Fatal("signing payload omitted exact disposition hash")
	}
	wrongDomain := envelope
	wrongDomain.Signature, err = connectorSigner.Sign(bytes.ReplaceAll(payload, []byte("Signature/v3"), []byte("Signature/v2")))
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyEnvelopeV3(wrongDomain); !errors.Is(err, ErrEffectAcknowledgementRejected) {
		t.Fatalf("cross-domain signature accepted: %v", err)
	}
	tampered := envelope
	tampered.Acknowledgement.DispositionReceiptHash = effectCloseVectorSHA("other-disposition")
	tampered.Acknowledgement, err = tampered.Acknowledgement.Seal()
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyEnvelopeV3(tampered); !errors.Is(err, ErrEffectAcknowledgementRejected) {
		t.Fatalf("resealed disposition accepted: %v", err)
	}

	for name, mutate := range map[string]func(*TrustedEffectAcknowledgementKey){
		"disabled":        func(k *TrustedEffectAcknowledgementKey) { k.Enabled = false },
		"other release":   func(k *TrustedEffectAcknowledgementKey) { k.ConnectorVersion = "other" },
		"before lifetime": func(k *TrustedEffectAcknowledgementKey) { k.NotBefore = acknowledgement.ObservedAt.Add(time.Second) },
		"at lifetime end": func(k *TrustedEffectAcknowledgementKey) { k.NotAfter = acknowledgement.ObservedAt },
	} {
		t.Run(name, func(t *testing.T) {
			changed := key
			mutate(&changed)
			v, err := NewEd25519EffectAcknowledgementVerifier([]TrustedEffectAcknowledgementKey{changed})
			if err != nil {
				t.Fatal(err)
			}
			if err := v.VerifyEnvelopeV3(envelope); !errors.Is(err, ErrEffectAcknowledgementRejected) {
				t.Fatalf("new acknowledgement accepted: %v", err)
			}
			err = v.VerifyStoredEnvelopeV3(envelope)
			if name == "disabled" {
				if err != nil {
					t.Fatalf("historical recovery failed after disabling key: %v", err)
				}
			} else if !errors.Is(err, ErrEffectAcknowledgementRejected) {
				t.Fatalf("historical verification ignored trust/lifetime: %v", err)
			}
			if err := v.VerifyStoredEnvelopeV3(tampered); !errors.Is(err, ErrEffectAcknowledgementRejected) {
				t.Fatalf("tampered historical evidence accepted: %v", err)
			}
		})
	}

	kernelSigner := crypto.NewEd25519SignerFromKey(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{74}, ed25519.SeedSize)), "effect-close-v3-test")
	signature, err := SignEffectCloseReceiptV3(receipt, kernelSigner)
	if err != nil {
		t.Fatal(err)
	}
	kernelVerifier, err := NewEd25519GrantSignatureVerifier(kernelSigner.PublicKeyBytes(), receipt.SigningKeyRef, receipt.KernelTrustRootID)
	if err != nil {
		t.Fatal(err)
	}
	if err := kernelVerifier.VerifyEffectCloseReceiptV3Signature(receipt, GrantSignatureEd25519, signature); err != nil {
		t.Fatal(err)
	}
	receiptPayload, err := EffectCloseReceiptV3SigningPayload(receipt, GrantSignatureEd25519)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(receiptPayload, []byte(receipt.DispositionReceiptHash)) {
		t.Fatal("receipt signing payload omitted disposition")
	}
	wrongSignature, err := kernelSigner.Sign(bytes.ReplaceAll(receiptPayload, []byte("Signature/v3"), []byte("Signature/v2")))
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{wrongSignature, strings.ToUpper(signature), "00", envelope.Signature} {
		if err := kernelVerifier.VerifyEffectCloseReceiptV3Signature(receipt, GrantSignatureEd25519, invalid); !errors.Is(err, ErrGrantSignatureRejected) {
			t.Fatalf("invalid receipt signature accepted: %v", err)
		}
	}
	tamperedReceipt := receipt
	tamperedReceipt.DispositionReceiptHash = effectCloseVectorSHA("other-disposition")
	tamperedReceipt, err = tamperedReceipt.Seal()
	if err != nil {
		t.Fatal(err)
	}
	if err := kernelVerifier.VerifyEffectCloseReceiptV3Signature(tamperedReceipt, GrantSignatureEd25519, signature); !errors.Is(err, ErrGrantSignatureRejected) {
		t.Fatalf("resealed receipt substitution accepted: %v", err)
	}
	wrongRoot, err := NewEd25519GrantSignatureVerifier(kernelSigner.PublicKeyBytes(), receipt.SigningKeyRef, "other-root")
	if err != nil {
		t.Fatal(err)
	}
	if err := wrongRoot.VerifyEffectCloseReceiptV3Signature(receipt, GrantSignatureEd25519, signature); !errors.Is(err, ErrGrantSignatureRejected) {
		t.Fatalf("wrong trust root accepted: %v", err)
	}
	if _, err := SignConnectorEffectAcknowledgementV3(acknowledgement, nil); err == nil {
		t.Fatal("nil connector signer accepted")
	}
	if _, err := SignEffectCloseReceiptV3(receipt, nil); err == nil {
		t.Fatal("nil Kernel signer accepted")
	}
}
