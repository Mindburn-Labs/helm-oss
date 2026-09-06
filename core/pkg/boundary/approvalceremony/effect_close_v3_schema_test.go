package approvalceremony

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestEffectCloseV3Schemas(t *testing.T) {
	root := effectCloseRepoRoot(t)
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020
	for _, version := range []string{"v2", "v3"} {
		for _, stem := range []string{"connector_effect_acknowledgement", "connector_effect_acknowledgement_envelope", "effect_close_receipt"} {
			name := stem + "_" + version + ".json"
			data, err := os.ReadFile(filepath.Join(root, "schemas", name))
			if err != nil {
				t.Fatal(err)
			}
			if err := compiler.AddResource("https://helm.mindburn.org/schemas/"+name, strings.NewReader(string(data))); err != nil {
				t.Fatal(err)
			}
		}
	}
	for stem, fixture := range map[string]string{
		"connector_effect_acknowledgement":          "acknowledgement.c14n.json",
		"connector_effect_acknowledgement_envelope": "acknowledgement_envelope.c14n.json",
		"effect_close_receipt":                      "receipt.c14n.json",
	} {
		t.Run(stem, func(t *testing.T) {
			schema, err := compiler.Compile("https://helm.mindburn.org/schemas/" + stem + "_v3.json")
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(root, "reference_packs", "effect-close-v3", fixture))
			if err != nil {
				t.Fatal(err)
			}
			load := func() map[string]any {
				var value map[string]any
				if err := json.Unmarshal(data, &value); err != nil {
					t.Fatal(err)
				}
				return value
			}
			if err := schema.Validate(load()); err != nil {
				t.Fatal(err)
			}
			unknown := load()
			unknown["unapproved_extension"] = true
			if err := schema.Validate(unknown); err == nil {
				t.Fatal("unknown extension accepted")
			}
			if strings.Contains(stem, "envelope") {
				return
			}
			for name, mutate := range map[string]func(map[string]any){
				"missing reconciliation": func(v map[string]any) { delete(v, "reconciliation_ref") },
				"malformed disposition":  func(v map[string]any) { v["disposition_receipt_hash"] = "bad" },
				"missing finality":       func(v map[string]any) { delete(v, "finality") },
			} {
				t.Run(name, func(t *testing.T) {
					value := load()
					mutate(value)
					if err := schema.Validate(value); err == nil {
						t.Fatal("invalid contract accepted")
					}
				})
			}
			// Reproduce why v2 cannot represent a fenced close: even when the
			// version fields match v2, its closed schema rejects the receipt pin.
			v2, err := compiler.Compile("https://helm.mindburn.org/schemas/" + stem + "_v2.json")
			if err != nil {
				t.Fatal(err)
			}
			legacy := load()
			legacy["schema_version"] = strings.ReplaceAll(legacy["schema_version"].(string), ".v3", ".v2")
			legacy["contract_version"] = "2026-09-04"
			if err := v2.Validate(legacy); err == nil {
				t.Fatal("published v2 silently accepts disposition extension")
			}
			delete(legacy, "disposition_receipt_hash")
			if err := v2.Validate(legacy); err != nil {
				t.Fatalf("v2 compatibility baseline: %v", err)
			}
		})
	}
}
