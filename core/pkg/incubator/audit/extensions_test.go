package audit_test

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/incubator/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ProofGraph Tests ---

func TestProofGraph_AddNode(t *testing.T) {
	g := audit.NewProofGraph()
	node, err := g.AddNode(audit.ProofNodeManifest, map[string]string{"files": "81"})
	require.NoError(t, err)
	assert.NotEmpty(t, node.Hash)
	assert.Equal(t, audit.ProofNodeManifest, node.Kind)
	assert.Equal(t, 1, g.Size())
}

func TestProofGraph_DAGEdges(t *testing.T) {
	g := audit.NewProofGraph()
	manifest, _ := g.AddNode(audit.ProofNodeManifest, map[string]string{"files": "100"})
	mech, _ := g.AddNode(audit.ProofNodeMechanical, map[string]string{"sections": "25"}, manifest.ID)
	ai, _ := g.AddNode(audit.ProofNodeAIAudit, map[string]string{"missions": "7"}, manifest.ID)
	merge, err := g.AddNode(audit.ProofNodeMerge, map[string]string{"verdict": "PASS"}, mech.ID, ai.ID)

	require.NoError(t, err)
	assert.Equal(t, 4, g.Size())
	assert.Len(t, merge.ParentHashes, 2)

	// Heads should only be the merge node (leaf)
	heads := g.Heads()
	assert.Len(t, heads, 1)
	assert.Equal(t, merge.ID, heads[0])
}

func TestProofGraph_Verify(t *testing.T) {
	g := audit.NewProofGraph()
	m, _ := g.AddNode(audit.ProofNodeManifest, map[string]string{"hash": "abc"})
	_, _ = g.AddNode(audit.ProofNodeMechanical, nil, m.ID)

	err := g.Verify()
	assert.NoError(t, err)
}

func TestProofGraph_InvalidParent(t *testing.T) {
	g := audit.NewProofGraph()
	_, err := g.AddNode(audit.ProofNodeMerge, nil, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProofGraph_Export(t *testing.T) {
	g := audit.NewProofGraph()
	_, _ = g.AddNode(audit.ProofNodeManifest, map[string]string{"test": "true"})

	data, err := g.Export()
	require.NoError(t, err)
	assert.Contains(t, string(data), "MANIFEST")
	assert.Contains(t, string(data), "\"count\": 1")
}

// --- Attestation Tests ---

func TestAttestation_SignAndVerify(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	reportData := []byte(`{"verdict":"PASS","hash":"test"}`)
	h := sha256.Sum256(reportData)
	reportHash := "sha256:" + hex.EncodeToString(h[:])

	att := audit.NewAttestation(reportHash, "merkle-root-123", "abc123", "test-signer")
	err = att.Sign(priv)
	require.NoError(t, err)
	assert.NotEqual(t, "unsigned", att.Signature)

	err = audit.VerifyAttestation(att, pub, reportData)
	assert.NoError(t, err)
}

func TestAttestation_TamperedReport(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)

	originalData := []byte(`{"verdict":"PASS"}`)
	h := sha256.Sum256(originalData)

	att := audit.NewAttestation("sha256:"+hex.EncodeToString(h[:]), "", "abc", "signer")
	_ = att.Sign(priv)

	// Tamper with report
	tamperedData := []byte(`{"verdict":"FAIL"}`)
	err := audit.VerifyAttestation(att, pub, tamperedData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hash mismatch")
}

func TestAttestation_Unsigned(t *testing.T) {
	att := audit.NewAttestation("hash", "root", "sha", "signer")
	err := audit.VerifyAttestation(att, nil, nil)
	assert.Error(t, err)
}

func TestAttestation_Nil(t *testing.T) {
	err := audit.VerifyAttestation(nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

// --- Remediation Tests ---

func TestFindingTranslator_Architecture(t *testing.T) {
	translator := audit.NewFindingTranslator()
	findings := []audit.Finding{
		{File: "app/Page.tsx", Category: audit.RemediationArchitecture, Severity: "medium", Verdict: "FAIL", Title: "Unnecessary 'use client' on presentational component"},
	}
	mutations := translator.Translate(findings)
	assert.Len(t, mutations, 1)
	assert.Contains(t, mutations[0].Description, "use client")
	assert.False(t, mutations[0].AutoApply)
}

func TestFindingTranslator_PassFindingsIgnored(t *testing.T) {
	translator := audit.NewFindingTranslator()
	findings := []audit.Finding{
		{File: "ok.tsx", Category: audit.RemediationSecurity, Verdict: "PASS", Title: "Clean"},
	}
	mutations := translator.Translate(findings)
	assert.Len(t, mutations, 0)
}

func TestFindingTranslator_SecurityLowConfidence(t *testing.T) {
	translator := audit.NewFindingTranslator()
	findings := []audit.Finding{
		{File: "danger.tsx", Category: audit.RemediationSecurity, Verdict: "FAIL", Title: "dangerouslySetInnerHTML without sanitization"},
	}
	mutations := translator.Translate(findings)
	assert.Len(t, mutations, 1)
	assert.Less(t, mutations[0].Confidence, 0.5)
}

// --- Anomaly Detector Tests ---

func TestAnomalyDetector_BurstDetection(t *testing.T) {
	config := audit.AnomalyConfig{
		BurstThreshold:   5,
		BurstWindow:      1 * time.Minute,
		FailureThreshold: 100,
	}
	d := audit.NewAnomalyDetector(config)

	var detected []audit.Anomaly
	d.OnAnomaly(func(a audit.Anomaly) { detected = append(detected, a) })

	for i := 0; i < 6; i++ {
		_ = d.Record(nil, audit.EventAccess, "read", "/data", nil)
	}

	assert.GreaterOrEqual(t, len(detected), 1)
	assert.Equal(t, audit.AnomalyBurstActivity, detected[0].Type)
}

func TestAnomalyDetector_RepeatedFailure(t *testing.T) {
	config := audit.DefaultAnomalyConfig()
	config.FailureThreshold = 3
	d := audit.NewAnomalyDetector(config)

	var detected []audit.Anomaly
	d.OnAnomaly(func(a audit.Anomaly) { detected = append(detected, a) })

	for i := 0; i < 4; i++ {
		_ = d.Record(nil, audit.EventDeny, "write", "/secret",
			map[string]interface{}{"actor": "attacker@evil.com"})
	}

	found := false
	for _, a := range detected {
		if a.Type == audit.AnomalyRepeatedFailure {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected REPEATED_FAILURE anomaly")
}

func TestAnomalyDetector_ChainDrift(t *testing.T) {
	d := audit.NewAnomalyDetector(audit.DefaultAnomalyConfig())

	var detected []audit.Anomaly
	d.OnAnomaly(func(a audit.Anomaly) { detected = append(detected, a) })

	d.CheckChainDrift("sha256:abc123def456")
	d.CheckChainDrift("genesis") // Reset!

	assert.Len(t, detected, 1)
	assert.Equal(t, audit.AnomalyChainDrift, detected[0].Type)
	assert.Equal(t, "critical", detected[0].Severity)
}
