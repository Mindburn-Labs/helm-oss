package swarm_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts/swarm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSwarmDeployCommand_Contract verifies the SwarmDeployCommand JSON contract.
// Invariant: Fields must match the specified JSON tags for inter-service comms.
func TestSwarmDeployCommand_Contract(t *testing.T) {
	ts := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	cmd := swarm.SwarmDeployCommand{
		ID:            "job_uuid_1",
		Intent:        "deploy",
		Strategy:      "monolithic",
		TargetScope:   []string{"core/"},
		PhenotypeHash: "hash_abc",
		Timestamp:     ts,
	}

	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, "phenotype_hash")
	assert.Contains(t, jsonStr, "target_scope")
	assert.Contains(t, jsonStr, "timestamp")

	var decoded swarm.SwarmDeployCommand
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Time round-trip check (ensure zone/precision doesn't break equality check)
	assert.Equal(t, cmd.ID, decoded.ID)
	assert.Equal(t, cmd.Timestamp.Unix(), decoded.Timestamp.Unix())
}

// TestConstants verifies critical NATS subject constants.
// Invariant: Subject names must not change without version bump.
func TestConstants(t *testing.T) {
	assert.Equal(t, "TITAN_CMD.swarm.deploy", swarm.SubjectDeploy)
	assert.Equal(t, "TITAN_EVT.swarm.completed", swarm.SubjectResult)
}
