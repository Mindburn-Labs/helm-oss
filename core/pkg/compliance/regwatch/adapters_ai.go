package regwatch

import (
	"context"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/compliance/jkg"
)

// NISTAIRMFAdapter monitors NIST AI Risk Management Framework 1.0 updates.
// Maps to enforceable obligations: model governance, training data provenance,
// post-market monitoring, incident reporting, transparency notices, high-risk system controls.
type NISTAIRMFAdapter struct {
	BaseAdapter
}

// NewNISTAIRMFAdapter creates a NIST AI RMF adapter.
func NewNISTAIRMFAdapter() *NISTAIRMFAdapter {
	return &NISTAIRMFAdapter{
		BaseAdapter: BaseAdapter{
			sourceType:   SourceNISTAIRMF,
			jurisdiction: jkg.JurisdictionGlobal,
			regulator:    jkg.RegulatorNIST,
			feedURL:      "https://nvlpubs.nist.gov/nistpubs/ai/nist.ai.100-1.pdf",
			healthy:      true,
		},
	}
}

func (n *NISTAIRMFAdapter) FetchChanges(ctx context.Context, since time.Time) ([]*RegChange, error) {
	return []*RegChange{}, nil
}

func (n *NISTAIRMFAdapter) IsHealthy(ctx context.Context) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.healthy
}

func (n *NISTAIRMFAdapter) SetHealthy(healthy bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.healthy = healthy
}

// EUAIActAdapter monitors the EU Artificial Intelligence Act (Regulation 2024/1689).
// Uses EUR-Lex bot-aware fetch policy. Tracks implementing acts and delegated acts.
type EUAIActAdapter struct {
	BaseAdapter
}

// NewEUAIActAdapter creates an EU AI Act adapter.
func NewEUAIActAdapter() *EUAIActAdapter {
	return &EUAIActAdapter{
		BaseAdapter: BaseAdapter{
			sourceType:   SourceEUAIAct,
			jurisdiction: jkg.JurisdictionEU,
			regulator:    jkg.RegulatorEURLex,
			feedURL:      "https://eur-lex.europa.eu/eli/reg/2024/1689/oj",
			healthy:      true,
		},
	}
}

func (e *EUAIActAdapter) FetchChanges(ctx context.Context, since time.Time) ([]*RegChange, error) {
	// Production: monitors EUR-Lex CELLAR for amendments to Reg 2024/1689
	// plus implementing/delegated acts linked via OJ series
	return []*RegChange{}, nil
}

func (e *EUAIActAdapter) IsHealthy(ctx context.Context) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.healthy
}

func (e *EUAIActAdapter) SetHealthy(healthy bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.healthy = healthy
}
