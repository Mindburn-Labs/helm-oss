package bundles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleLoader_LoadFromFile(t *testing.T) {
	// Create a temp YAML bundle
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "gdpr.policy.yaml")

	yaml := `id: gdpr-basic
name: GDPR Basic Profile
version: "1.0.0"
description: Basic GDPR compliance rules
author: HELM Core
kind: compliance
regime: EU/GDPR
region: EU
rules:
  - id: pii-require-consent
    effect: deny
    condition: "effect.data_class == 'PII' && !context.consent_given"
    engine: cel
    priority: 10
    reason_code: POLICY_VIOLATION
    tags: [gdpr, privacy]
  - id: cross-border-check
    effect: escalate
    condition: "effect.cross_border && !context.adequacy_decision"
    engine: cel
    priority: 5
    reason_code: JURISDICTION_VIOLATION
`
	err := os.WriteFile(bundlePath, []byte(yaml), 0644)
	require.NoError(t, err)

	loader := NewBundleLoader()
	bundle, err := loader.LoadFromFile(bundlePath)
	require.NoError(t, err)

	assert.Equal(t, "gdpr-basic", bundle.ID)
	assert.Equal(t, "GDPR Basic Profile", bundle.Name)
	assert.Equal(t, BundleCompliance, bundle.Kind)
	assert.Equal(t, "EU/GDPR", bundle.Regime)
	assert.Len(t, bundle.Rules, 2)
	assert.Equal(t, "pii-require-consent", bundle.Rules[0].ID)
	assert.Equal(t, 10, bundle.Rules[0].Priority)
	assert.NotEmpty(t, bundle.ContentHash)
	assert.Contains(t, bundle.ContentHash, "sha256:")
}

func TestBundleLoader_LoadFromDir(t *testing.T) {
	dir := t.TempDir()

	// Two valid bundles
	yaml1 := `id: bundle-1
name: Bundle One
kind: compliance
rules: []
`
	yaml2 := `id: bundle-2
name: Bundle Two
kind: jurisdiction
regime: UK/FCA
rules:
  - id: fca-rule-1
    effect: deny
    condition: "true"
    engine: cel
`
	// A non-bundle file (should be ignored)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b1.policy.yaml"), []byte(yaml1), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b2.policy.yaml"), []byte(yaml2), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Not a bundle"), 0644))

	loader := NewBundleLoader()
	loaded, err := loader.LoadFromDir(dir)
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Len(t, loader.All(), 2)
}

func TestBundleLoader_Validation(t *testing.T) {
	dir := t.TempDir()

	// Missing ID
	invalidYAML := `name: No ID Bundle
kind: compliance
rules: []
`
	path := filepath.Join(dir, "bad.policy.yaml")
	require.NoError(t, os.WriteFile(path, []byte(invalidYAML), 0644))

	loader := NewBundleLoader()
	_, err := loader.LoadFromFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field 'id'")
}

func TestBundleLoader_ByKind(t *testing.T) {
	loader := NewBundleLoader()

	dir := t.TempDir()
	comp := `id: comp-1
name: Compliance
kind: compliance
rules: []
`
	jur := `id: jur-1
name: Jurisdiction
kind: jurisdiction
rules: []
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "c.policy.yaml"), []byte(comp), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "j.policy.yaml"), []byte(jur), 0644))

	_, err := loader.LoadFromDir(dir)
	require.NoError(t, err)

	assert.Len(t, loader.ByKind(BundleCompliance), 1)
	assert.Len(t, loader.ByKind(BundleJurisdiction), 1)
	assert.Len(t, loader.ByKind(BundleIndustry), 0)
}

func TestBundleSigner_SignAndVerify(t *testing.T) {
	dir := t.TempDir()
	yaml := `id: signed-bundle
name: Signed Bundle Test
kind: compliance
rules:
  - id: rule-1
    effect: deny
    condition: "true"
    engine: cel
`
	path := filepath.Join(dir, "signed.policy.yaml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0644))

	loader := NewBundleLoader()
	bundle, err := loader.LoadFromFile(path)
	require.NoError(t, err)

	// Mock signer
	signer := NewBundleSigner(func(data []byte) (string, error) {
		return "mock-sig-" + string(data[:8]), nil
	}, "test-key-001")

	signed, err := signer.Sign(bundle)
	require.NoError(t, err)
	assert.Contains(t, signed.Signature, "mock-sig-")
	assert.Equal(t, "test-key-001", signed.SignerKeyID)
	assert.Equal(t, "ed25519", signed.SignatureAlg)

	// Verify
	err = VerifyBundle(signed, func(data []byte, sig string) (bool, error) {
		return sig == "mock-sig-"+string(data[:8]), nil
	})
	assert.NoError(t, err)
}
