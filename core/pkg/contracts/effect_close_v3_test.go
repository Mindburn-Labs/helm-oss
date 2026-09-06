package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func effectCloseV3ContractFixture(t *testing.T) (ConnectorEffectAcknowledgementV3, EffectCloseReceiptV3) {
	t.Helper()
	var acknowledgement ConnectorEffectAcknowledgementV3
	var receipt EffectCloseReceiptV3
	for name, value := range map[string]any{
		"acknowledgement.c14n.json": &acknowledgement,
		"receipt.c14n.json":         &receipt,
	} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "reference_packs", "effect-close-v3", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, value); err != nil {
			t.Fatal(err)
		}
	}
	return acknowledgement, receipt
}

func TestEffectCloseV3BindsExactDispositionAndFrozenV2Fields(t *testing.T) {
	acknowledgement, receipt := effectCloseV3ContractFixture(t)
	if err := receipt.ValidateAcknowledgement(acknowledgement); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*EffectCloseReceiptV3){
		"missing disposition":         func(r *EffectCloseReceiptV3) { r.DispositionReceiptHash = "" },
		"other disposition":           func(r *EffectCloseReceiptV3) { r.DispositionReceiptHash = effectCloseTestSHA("other") },
		"activation":                  func(r *EffectCloseReceiptV3) { r.ActivationRecordHash = effectCloseTestSHA("other") },
		"capability":                  func(r *EffectCloseReceiptV3) { r.AdapterCapabilityHash = effectCloseTestSHA("other") },
		"reservation acknowledgement": func(r *EffectCloseReceiptV3) { r.AcknowledgementHash = effectCloseTestSHA("other") },
		"reconciliation":              func(r *EffectCloseReceiptV3) { r.ReconciliationRef = "other" },
		"finality":                    func(r *EffectCloseReceiptV3) { r.Finality.ResolutionRef = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			_, changed := effectCloseV3ContractFixture(t)
			mutate(&changed)
			sealed, err := changed.Seal()
			if err != nil {
				t.Fatal(err)
			}
			if err := sealed.ValidateAcknowledgement(acknowledgement); err == nil {
				t.Fatal("resealed receipt substitution accepted")
			}
		})
	}
	changed := acknowledgement
	changed.DispositionReceiptHash = effectCloseTestSHA("other")
	if err := changed.ValidateIntegrity(); err == nil {
		t.Fatal("disposition is outside acknowledgement hash")
	}
	changedReceipt := receipt
	changedReceipt.DispositionReceiptHash = effectCloseTestSHA("other")
	if err := changedReceipt.ValidateIntegrity(); err == nil {
		t.Fatal("disposition is outside receipt hash")
	}
}

func TestEffectCloseV3RejectsInvalidDispositionAndFinality(t *testing.T) {
	for name, mutate := range map[string]func(*ConnectorEffectAcknowledgementV3){
		"malformed disposition": func(a *ConnectorEffectAcknowledgementV3) { a.DispositionReceiptHash = "not-a-hash" },
		"uppercase disposition": func(a *ConnectorEffectAcknowledgementV3) {
			a.DispositionReceiptHash = "sha256:" + strings.Repeat("A", 64)
		},
		"missing reconciliation": func(a *ConnectorEffectAcknowledgementV3) { a.ReconciliationRef = "" },
		"missing finality":       func(a *ConnectorEffectAcknowledgementV3) { a.Finality = nil },
		"missing facts":          func(a *ConnectorEffectAcknowledgementV3) { a.Finality.ObservedExternalFacts = nil },
		"unsorted facts": func(a *ConnectorEffectAcknowledgementV3) {
			a.Finality.ObservedExternalFacts[0], a.Finality.ObservedExternalFacts[1] = a.Finality.ObservedExternalFacts[1], a.Finality.ObservedExternalFacts[0]
		},
		"missing compensation": func(a *ConnectorEffectAcknowledgementV3) { a.Finality.ConditionalCompensationRef = "" },
		"v2 schema":            func(a *ConnectorEffectAcknowledgementV3) { a.SchemaVersion = ConnectorEffectAcknowledgementSchemaV2 },
		"v2 contract": func(a *ConnectorEffectAcknowledgementV3) {
			a.ContractVersion = ConnectorEffectAcknowledgementContractV2
		},
	} {
		t.Run(name, func(t *testing.T) {
			acknowledgement, _ := effectCloseV3ContractFixture(t)
			mutate(&acknowledgement)
			if _, err := acknowledgement.Seal(); err == nil {
				t.Fatal("invalid acknowledgement accepted")
			}
		})
	}
	for _, value := range []string{"not-a-hash", "sha256:" + strings.Repeat("A", 64)} {
		_, receipt := effectCloseV3ContractFixture(t)
		receipt.DispositionReceiptHash = value
		if _, err := receipt.Seal(); err == nil {
			t.Fatal("invalid receipt disposition accepted")
		}
	}
	_, receipt := effectCloseV3ContractFixture(t)
	receipt.PriorState = EffectClosePriorStateStarted
	receipt.ReconciliationRef = ""
	if _, err := receipt.Seal(); err == nil {
		t.Fatal("receipt disposition without reconciliation accepted")
	}
}

func TestEffectCloseV3AllowsNoDispositionWithoutInventingFenceAuthority(t *testing.T) {
	acknowledgement, receipt := effectCloseV3ContractFixture(t)
	acknowledgement.DispositionReceiptHash = ""
	acknowledgement.ReconciliationRef = ""
	var err error
	acknowledgement, err = acknowledgement.Seal()
	if err != nil {
		t.Fatal(err)
	}
	receipt.DispositionReceiptHash = ""
	receipt.ReconciliationRef = ""
	receipt.PriorState = EffectClosePriorStateStarted
	receipt.AcknowledgementHash = acknowledgement.AcknowledgementHash
	receipt, err = receipt.Seal()
	if err != nil {
		t.Fatal(err)
	}
	if err := receipt.ValidateAcknowledgement(acknowledgement); err != nil {
		t.Fatal(err)
	}
	// Contract validity alone cannot decide whether the current scope is fenced.
	// Runtime must reject this shape when a disposition is required.
	if err := acknowledgement.ConnectorEffectAcknowledgementV2.Validate(); err == nil {
		t.Fatal("v3 acknowledgement accepted by frozen v2 validator")
	}
	if err := receipt.EffectCloseReceiptV2.Validate(); err == nil {
		t.Fatal("v3 receipt accepted by frozen v2 validator")
	}
}
