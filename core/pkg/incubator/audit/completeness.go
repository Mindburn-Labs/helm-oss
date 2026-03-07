package audit

import (
	"fmt"
	"strings"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/store"
)

// CompletenessVerifier checks that all required audit missions
// have corresponding evidence in the audit store's hash chain.
//
// This is the core mechanism for HELM self-verification:
//   - The AI audit agent emits a store.AuditEntry per mission
//   - The verifier queries the store for entries with EntryType=evidence
//   - It cross-references against the mission manifest
//   - The store's hash chain guarantees no evidence was tampered with
type CompletenessVerifier struct {
	manifest MissionManifest
	store    *store.AuditStore
}

// NewCompletenessVerifier creates a verifier for the given manifest and store.
func NewCompletenessVerifier(manifest MissionManifest, s *store.AuditStore) *CompletenessVerifier {
	return &CompletenessVerifier{manifest: manifest, store: s}
}

// Verify checks that all required missions have evidence entries in the store.
// Returns a CompletenessResult with detailed status.
func (v *CompletenessVerifier) Verify() (*CompletenessResult, error) {
	if v.store == nil {
		return nil, fmt.Errorf("audit: completeness verifier requires a configured store (fail-closed)")
	}

	// 1. Verify the hash chain integrity first — if the chain is broken,
	//    evidence may have been tampered with.
	if err := v.store.VerifyChain(); err != nil {
		return &CompletenessResult{
			AllMissionsRan:       false,
			MissionChainVerified: false,
			ChainHead:            v.store.GetChainHead(),
		}, fmt.Errorf("audit: chain verification failed: %w", err)
	}

	// 2. Query all evidence entries from the store.
	evidenceEntries := v.store.Query(store.QueryFilter{
		EntryType: store.EntryTypeEvidence,
	})

	// 3. Build a set of completed mission IDs from evidence subjects.
	completedMissions := make(map[string]bool)
	for _, entry := range evidenceEntries {
		// Evidence entries use subject format "mission:<id>"
		if strings.HasPrefix(entry.Subject, "mission:") {
			missionID := strings.TrimPrefix(entry.Subject, "mission:")
			completedMissions[missionID] = true
		}
	}

	// 4. Check required missions against completed set.
	var missing []string
	totalRequired := 0
	totalCompleted := 0

	for _, mission := range v.manifest.Missions {
		if !mission.Required {
			continue
		}
		totalRequired++
		if completedMissions[mission.ID] {
			totalCompleted++
		} else {
			missing = append(missing, mission.ID)
		}
	}

	result := &CompletenessResult{
		AllMissionsRan:       len(missing) == 0,
		MissionChainVerified: true,
		ChainHead:            v.store.GetChainHead(),
		MissingMissions:      missing,
		TotalRequired:        totalRequired,
		TotalCompleted:       totalCompleted,
	}

	if len(missing) > 0 {
		return result, fmt.Errorf("audit: %d required missions missing: %s",
			len(missing), strings.Join(missing, ", "))
	}

	return result, nil
}
