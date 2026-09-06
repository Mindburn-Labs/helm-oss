package approvalceremony

// quantum_posture: classical Ed25519 effect-close v3 signing and verification;
// no hybrid or post-quantum claim.

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/Mindburn-Labs/helm-ai-kernel/core/pkg/canonicalize"
	"github.com/Mindburn-Labs/helm-ai-kernel/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm-ai-kernel/core/pkg/crypto"
)

const (
	connectorEffectAcknowledgementSignatureDomainV3 = "HELM/ConnectorEffectAcknowledgementSignature/v3"
	effectCloseReceiptSignatureDomainV3             = "HELM/EffectCloseReceiptSignature/v3"
)

func ConnectorEffectAcknowledgementV3SigningPayload(acknowledgement contracts.ConnectorEffectAcknowledgementV3) ([]byte, error) {
	if err := acknowledgement.ValidateIntegrity(); err != nil {
		return nil, effectAcknowledgementRejected("acknowledgement integrity mismatch: " + err.Error())
	}
	payload, err := canonicalize.JCS(struct {
		Domain                 string `json:"domain"`
		ContractVersion        string `json:"contract_version"`
		AcknowledgementHash    string `json:"acknowledgement_hash"`
		ActivationRecordHash   string `json:"activation_record_hash"`
		AdapterCapabilityHash  string `json:"adapter_capability_hash"`
		DispositionReceiptHash string `json:"disposition_receipt_hash,omitempty"`
		IssuerID               string `json:"issuer_id"`
		SigningKeyRef          string `json:"signing_key_ref"`
		ConnectorID            string `json:"connector_id"`
		ConnectorVersion       string `json:"connector_version"`
		AdapterID              string `json:"adapter_id"`
		AdapterVersion         string `json:"adapter_version"`
		AdapterCapabilityRef   string `json:"adapter_capability_ref"`
		Algorithm              string `json:"algorithm"`
	}{
		Domain: connectorEffectAcknowledgementSignatureDomainV3, ContractVersion: acknowledgement.ContractVersion,
		AcknowledgementHash: acknowledgement.AcknowledgementHash, ActivationRecordHash: acknowledgement.ActivationRecordHash,
		AdapterCapabilityHash:  acknowledgement.AdapterCapabilityHash,
		DispositionReceiptHash: acknowledgement.DispositionReceiptHash,
		IssuerID:               acknowledgement.IssuerID, SigningKeyRef: acknowledgement.SigningKeyRef,
		ConnectorID: acknowledgement.ConnectorID, ConnectorVersion: acknowledgement.ConnectorVersion,
		AdapterID: acknowledgement.AdapterID, AdapterVersion: acknowledgement.AdapterVersion,
		AdapterCapabilityRef: acknowledgement.AdapterCapabilityRef,
		Algorithm:            acknowledgement.Algorithm,
	})
	if err != nil {
		return nil, effectAcknowledgementRejected("canonicalize signing payload: " + err.Error())
	}
	return payload, nil
}

func SignConnectorEffectAcknowledgementV3(
	acknowledgement contracts.ConnectorEffectAcknowledgementV3,
	signer crypto.Signer,
) (contracts.ConnectorEffectAcknowledgementEnvelopeV3, error) {
	if signer == nil {
		return contracts.ConnectorEffectAcknowledgementEnvelopeV3{}, effectAcknowledgementRejected("signer is not configured")
	}
	payload, err := ConnectorEffectAcknowledgementV3SigningPayload(acknowledgement)
	if err != nil {
		return contracts.ConnectorEffectAcknowledgementEnvelopeV3{}, err
	}
	signature, err := signer.Sign(payload)
	if err != nil {
		return contracts.ConnectorEffectAcknowledgementEnvelopeV3{}, effectAcknowledgementRejected("sign acknowledgement: " + err.Error())
	}
	envelope := contracts.ConnectorEffectAcknowledgementEnvelopeV3{Acknowledgement: acknowledgement, Signature: signature}
	if err := envelope.Validate(); err != nil {
		return contracts.ConnectorEffectAcknowledgementEnvelopeV3{}, effectAcknowledgementRejected("signer returned invalid signature encoding")
	}
	return envelope, nil
}

func (v *Ed25519EffectAcknowledgementVerifier) VerifyEnvelopeV3(envelope contracts.ConnectorEffectAcknowledgementEnvelopeV3) error {
	return v.verifyEnvelopeV3(envelope, true)
}

func (v *Ed25519EffectAcknowledgementVerifier) VerifyStoredEnvelopeV3(envelope contracts.ConnectorEffectAcknowledgementEnvelopeV3) error {
	return v.verifyEnvelopeV3(envelope, false)
}

func (v *Ed25519EffectAcknowledgementVerifier) verifyEnvelopeV3(envelope contracts.ConnectorEffectAcknowledgementEnvelopeV3, requireEnabled bool) error {
	if v == nil || len(v.keys) == 0 {
		return effectAcknowledgementRejected("verifier is not configured")
	}
	if err := envelope.Validate(); err != nil {
		return effectAcknowledgementRejected(err.Error())
	}
	a := envelope.Acknowledgement
	identity := effectAcknowledgementKeyIdentity(a.IssuerID, a.SigningKeyRef, a.ConnectorID, a.ConnectorVersion)
	key, ok := v.keys[identity]
	if !ok {
		return effectAcknowledgementRejected("signing key is not currently trusted for this connector release")
	}
	if requireEnabled && !key.Enabled {
		return effectAcknowledgementRejected("signing key is not currently trusted for this connector release")
	}
	if a.ObservedAt.Before(key.NotBefore) || !a.ObservedAt.Before(key.NotAfter) {
		return effectAcknowledgementRejected("acknowledgement was observed outside the pinned key lifetime")
	}
	payload, err := ConnectorEffectAcknowledgementV3SigningPayload(a)
	if err != nil {
		return err
	}
	raw, err := hex.DecodeString(envelope.Signature)
	if err != nil || len(raw) != ed25519.SignatureSize || !ed25519.Verify(key.PublicKey, payload, raw) {
		return effectAcknowledgementRejected("bad Ed25519 signature")
	}
	return nil
}

func EffectCloseReceiptV3SigningPayload(receipt contracts.EffectCloseReceiptV3, algorithm string) ([]byte, error) {
	if algorithm != GrantSignatureEd25519 {
		return nil, fmt.Errorf("%w: unsupported effect close receipt algorithm", ErrGrantSignatureRejected)
	}
	if err := receipt.ValidateIntegrity(); err != nil {
		return nil, fmt.Errorf("%w: effect close receipt integrity mismatch: %v", ErrGrantSignatureRejected, err)
	}
	payload, err := canonicalize.JCS(struct {
		Domain                 string `json:"domain"`
		ContractVersion        string `json:"contract_version"`
		ReceiptHash            string `json:"receipt_hash"`
		ActivationRecordHash   string `json:"activation_record_hash"`
		AdapterCapabilityHash  string `json:"adapter_capability_hash"`
		DispositionReceiptHash string `json:"disposition_receipt_hash,omitempty"`
		KernelTrustRootID      string `json:"kernel_trust_root_id"`
		SigningKeyRef          string `json:"signing_key_ref"`
		Algorithm              string `json:"algorithm"`
	}{
		Domain: effectCloseReceiptSignatureDomainV3, ContractVersion: receipt.ContractVersion,
		ReceiptHash: receipt.ReceiptHash, ActivationRecordHash: receipt.ActivationRecordHash,
		AdapterCapabilityHash:  receipt.AdapterCapabilityHash,
		DispositionReceiptHash: receipt.DispositionReceiptHash,
		KernelTrustRootID:      receipt.KernelTrustRootID, SigningKeyRef: receipt.SigningKeyRef, Algorithm: algorithm,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: canonicalize effect close receipt signing payload: %v", ErrGrantSignatureRejected, err)
	}
	return payload, nil
}

func SignEffectCloseReceiptV3(receipt contracts.EffectCloseReceiptV3, signer crypto.Signer) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("%w: signer is not configured", ErrGrantSignatureRejected)
	}
	payload, err := EffectCloseReceiptV3SigningPayload(receipt, GrantSignatureEd25519)
	if err != nil {
		return "", err
	}
	signature, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("%w: sign effect close receipt: %v", ErrGrantSignatureRejected, err)
	}
	if !validEd25519Signature(signature) {
		return "", fmt.Errorf("%w: signer returned invalid effect close receipt signature", ErrGrantSignatureRejected)
	}
	return signature, nil
}

func (v *Ed25519GrantSignatureVerifier) VerifyEffectCloseReceiptV3Signature(
	receipt contracts.EffectCloseReceiptV3,
	algorithm, signature string,
) error {
	if v == nil || len(v.publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: verifier is not configured", ErrGrantSignatureRejected)
	}
	if algorithm != GrantSignatureEd25519 || receipt.SigningKeyRef != v.signingKeyRef ||
		receipt.KernelTrustRootID != v.kernelTrustRootID {
		return fmt.Errorf("%w: effect close receipt trust-root metadata mismatch", ErrGrantSignatureRejected)
	}
	payload, err := EffectCloseReceiptV3SigningPayload(receipt, algorithm)
	if err != nil {
		return err
	}
	rawSignature, err := hex.DecodeString(signature)
	if err != nil || len(rawSignature) != ed25519.SignatureSize || hex.EncodeToString(rawSignature) != signature {
		return fmt.Errorf("%w: effect close receipt signature encoding is invalid", ErrGrantSignatureRejected)
	}
	if !ed25519.Verify(v.publicKey, payload, rawSignature) {
		return fmt.Errorf("%w: bad effect close receipt ed25519 signature", ErrGrantSignatureRejected)
	}
	return nil
}
