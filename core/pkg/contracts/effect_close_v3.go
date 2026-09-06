package contracts

// quantum_posture: classical Ed25519 effect-close contracts; no hybrid or
// post-quantum claim.

import "encoding/hex"

const (
	ConnectorEffectAcknowledgementSchemaV3   = "connector-effect-acknowledgement.v3"
	ConnectorEffectAcknowledgementContractV3 = "2026-09-06"
	EffectCloseReceiptSchemaV3               = "effect-close-receipt.v3"
	EffectCloseReceiptContractV3             = "2026-09-06"
)

// ConnectorEffectAcknowledgementV3 retains the frozen v2 fields and finality
// semantics and restores the exact disposition receipt commitment required
// when closing fenced work. The embedded fields remain flat on the wire; this
// is a v3 artifact, not a nested or independently valid v2 acknowledgement.
// Disposition evidence grants no authority to execute another effect.
type ConnectorEffectAcknowledgementV3 struct {
	ConnectorEffectAcknowledgementV2
	DispositionReceiptHash string `json:"disposition_receipt_hash,omitempty"`
}

type ConnectorEffectAcknowledgementEnvelopeV3 struct {
	Acknowledgement ConnectorEffectAcknowledgementV3 `json:"acknowledgement"`
	Signature       string                           `json:"signature"`
}

func (a ConnectorEffectAcknowledgementV3) Validate() error {
	if a.SchemaVersion != ConnectorEffectAcknowledgementSchemaV3 {
		return connectorEffectAcknowledgementInvalid("unsupported schema_version")
	}
	if a.ContractVersion != ConnectorEffectAcknowledgementContractV3 {
		return connectorEffectAcknowledgementInvalid("unsupported contract_version")
	}
	// Reuse shape validation only. Hashing and signing always use the complete
	// v3 artifact and v3 domain, never this local validation projection.
	shape := a.ConnectorEffectAcknowledgementV2
	shape.SchemaVersion = ConnectorEffectAcknowledgementSchemaV2
	shape.ContractVersion = ConnectorEffectAcknowledgementContractV2
	if err := shape.Validate(); err != nil {
		return err
	}
	if a.DispositionReceiptHash != "" {
		if !isApprovalGrantSHA256(a.DispositionReceiptHash) {
			return connectorEffectAcknowledgementInvalid("disposition_receipt_hash must be a lowercase sha256 reference")
		}
		if a.ReconciliationRef == "" {
			return connectorEffectAcknowledgementInvalid("disposition_receipt_hash requires reconciliation_ref")
		}
	}
	return nil
}

func (a ConnectorEffectAcknowledgementV3) Seal() (ConnectorEffectAcknowledgementV3, error) {
	if err := a.Validate(); err != nil {
		return ConnectorEffectAcknowledgementV3{}, err
	}
	a.AcknowledgementHash = ""
	hash, err := hashJCS(a)
	if err != nil {
		return ConnectorEffectAcknowledgementV3{}, connectorEffectAcknowledgementInvalid("seal: " + err.Error())
	}
	a.AcknowledgementHash = hash
	return a, nil
}

func (a ConnectorEffectAcknowledgementV3) ValidateIntegrity() error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.AcknowledgementHash == "" {
		return connectorEffectAcknowledgementInvalid("acknowledgement_hash is required")
	}
	sealed, err := a.Seal()
	if err != nil || sealed.AcknowledgementHash != a.AcknowledgementHash {
		return connectorEffectAcknowledgementInvalid("acknowledgement integrity mismatch")
	}
	return nil
}

func (e ConnectorEffectAcknowledgementEnvelopeV3) Validate() error {
	if err := e.Acknowledgement.ValidateIntegrity(); err != nil {
		return err
	}
	raw, err := hex.DecodeString(e.Signature)
	if err != nil || len(raw) != 64 || hex.EncodeToString(raw) != e.Signature {
		return connectorEffectAcknowledgementInvalid("signature must be canonical lowercase Ed25519 hex")
	}
	return nil
}

// EffectCloseReceiptV3 carries the same disposition receipt commitment as its
// acknowledgement while preserving v2 activation, capability and finality
// semantics. Validation does not verify the referenced disposition: the close
// owner must still verify its signatures, current fence and reservation head.
type EffectCloseReceiptV3 struct {
	EffectCloseReceiptV2
	DispositionReceiptHash string `json:"disposition_receipt_hash,omitempty"`
}

func (r EffectCloseReceiptV3) Validate() error {
	if r.SchemaVersion != EffectCloseReceiptSchemaV3 {
		return effectCloseReceiptInvalid("unsupported schema_version")
	}
	if r.ContractVersion != EffectCloseReceiptContractV3 {
		return effectCloseReceiptInvalid("unsupported contract_version")
	}
	shape := r.EffectCloseReceiptV2
	shape.SchemaVersion = EffectCloseReceiptSchemaV2
	shape.ContractVersion = EffectCloseReceiptContractV2
	if err := shape.Validate(); err != nil {
		return err
	}
	if r.DispositionReceiptHash != "" {
		if !isApprovalGrantSHA256(r.DispositionReceiptHash) {
			return effectCloseReceiptInvalid("disposition_receipt_hash must be a lowercase sha256 reference")
		}
		if r.ReconciliationRef == "" {
			return effectCloseReceiptInvalid("disposition_receipt_hash requires reconciliation_ref")
		}
	}
	return nil
}

func (r EffectCloseReceiptV3) Seal() (EffectCloseReceiptV3, error) {
	if err := r.Validate(); err != nil {
		return EffectCloseReceiptV3{}, err
	}
	r.ReceiptHash = ""
	hash, err := hashJCS(r)
	if err != nil {
		return EffectCloseReceiptV3{}, effectCloseReceiptInvalid("seal: " + err.Error())
	}
	r.ReceiptHash = hash
	return r, nil
}

func (r EffectCloseReceiptV3) ValidateIntegrity() error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.ReceiptHash == "" {
		return effectCloseReceiptInvalid("receipt_hash is required")
	}
	sealed, err := r.Seal()
	if err != nil || sealed.ReceiptHash != r.ReceiptHash {
		return effectCloseReceiptInvalid("receipt integrity mismatch")
	}
	return nil
}

func (r EffectCloseReceiptV3) ValidateAcknowledgement(a ConnectorEffectAcknowledgementV3) error {
	if err := r.ValidateIntegrity(); err != nil {
		return err
	}
	if err := a.ValidateIntegrity(); err != nil {
		return err
	}
	if r.DispositionReceiptHash != a.DispositionReceiptHash {
		return effectCloseReceiptInvalid("receipt does not match connector acknowledgement disposition")
	}
	return validateEffectCloseV2AcknowledgementBinding(r.EffectCloseReceiptV2, a.ConnectorEffectAcknowledgementV2)
}
