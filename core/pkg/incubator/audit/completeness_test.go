package audit_test

import (
	"testing"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/incubator/audit"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletenessVerifier_AllPresent(t *testing.T) {
	s := store.NewAuditStore()
	manifest := audit.MissionManifest{
		Missions: []audit.MissionSpec{
			{ID: "architecture_coherence", Required: true},
			{ID: "security_posture", Required: true},
			{ID: "doc_code_drift", Required: true},
		},
	}

	// Simulate AI agent completing all missions
	for _, m := range manifest.Missions {
		_, err := s.Append(store.EntryTypeEvidence, "mission:"+m.ID, "completed", map[string]string{
			"severity": "low", "finding_count": "2",
		}, nil)
		require.NoError(t, err)
	}

	v := audit.NewCompletenessVerifier(manifest, s)
	result, err := v.Verify()
	require.NoError(t, err)
	assert.True(t, result.AllMissionsRan)
	assert.True(t, result.MissionChainVerified)
	assert.Equal(t, 3, result.TotalRequired)
	assert.Equal(t, 3, result.TotalCompleted)
	assert.Empty(t, result.MissingMissions)
	assert.NotEmpty(t, result.ChainHead)
}

func TestCompletenessVerifier_OneMissing(t *testing.T) {
	s := store.NewAuditStore()
	manifest := audit.MissionManifest{
		Missions: []audit.MissionSpec{
			{ID: "architecture_coherence", Required: true},
			{ID: "security_posture", Required: true},
			{ID: "doc_code_drift", Required: true},
		},
	}

	// Only complete 2 of 3 missions
	_, _ = s.Append(store.EntryTypeEvidence, "mission:architecture_coherence", "completed", nil, nil)
	_, _ = s.Append(store.EntryTypeEvidence, "mission:security_posture", "completed", nil, nil)
	// doc_code_drift is missing!

	v := audit.NewCompletenessVerifier(manifest, s)
	result, err := v.Verify()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "doc_code_drift")
	assert.False(t, result.AllMissionsRan)
	assert.Equal(t, 2, result.TotalCompleted)
	assert.Equal(t, []string{"doc_code_drift"}, result.MissingMissions)
}

func TestCompletenessVerifier_OptionalMissing(t *testing.T) {
	s := store.NewAuditStore()
	manifest := audit.MissionManifest{
		Missions: []audit.MissionSpec{
			{ID: "architecture_coherence", Required: true},
			{ID: "experimental_check", Required: false}, // Optional
		},
	}

	_, _ = s.Append(store.EntryTypeEvidence, "mission:architecture_coherence", "completed", nil, nil)
	// experimental_check not completed — but it's optional

	v := audit.NewCompletenessVerifier(manifest, s)
	result, err := v.Verify()
	require.NoError(t, err)
	assert.True(t, result.AllMissionsRan)
	assert.Equal(t, 1, result.TotalRequired)
	assert.Equal(t, 1, result.TotalCompleted)
}

func TestCompletenessVerifier_Duplicate(t *testing.T) {
	s := store.NewAuditStore()
	manifest := audit.MissionManifest{
		Missions: []audit.MissionSpec{
			{ID: "security_posture", Required: true},
		},
	}

	// Same mission evidence appended twice (idempotent)
	_, _ = s.Append(store.EntryTypeEvidence, "mission:security_posture", "completed", nil, nil)
	_, _ = s.Append(store.EntryTypeEvidence, "mission:security_posture", "completed", nil, nil)

	v := audit.NewCompletenessVerifier(manifest, s)
	result, err := v.Verify()
	require.NoError(t, err)
	assert.True(t, result.AllMissionsRan)
}

func TestCompletenessVerifier_EmptyManifest(t *testing.T) {
	s := store.NewAuditStore()
	manifest := audit.MissionManifest{Missions: nil}

	v := audit.NewCompletenessVerifier(manifest, s)
	result, err := v.Verify()
	require.NoError(t, err)
	assert.True(t, result.AllMissionsRan) // Vacuously true
	assert.Equal(t, 0, result.TotalRequired)
}

func TestCompletenessVerifier_NilStore(t *testing.T) {
	manifest := audit.MissionManifest{
		Missions: []audit.MissionSpec{
			{ID: "test", Required: true},
		},
	}

	v := audit.NewCompletenessVerifier(manifest, nil)
	_, err := v.Verify()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fail-closed")
}

func TestCompletenessVerifier_ChainIntegrity(t *testing.T) {
	s := store.NewAuditStore()
	manifest := audit.MissionManifest{
		Missions: []audit.MissionSpec{
			{ID: "m1", Required: true},
			{ID: "m2", Required: true},
		},
	}

	_, _ = s.Append(store.EntryTypeEvidence, "mission:m1", "completed", nil, nil)
	_, _ = s.Append(store.EntryTypeEvidence, "mission:m2", "completed", nil, nil)

	// Verify chain is intact
	v := audit.NewCompletenessVerifier(manifest, s)
	result, err := v.Verify()
	require.NoError(t, err)
	assert.True(t, result.MissionChainVerified)
	assert.NotEmpty(t, result.ChainHead)
}
