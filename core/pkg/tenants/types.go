// Package tenants provides tenant lifecycle management including
// provisioning, suspension, deletion, and data export.
package tenants

import (
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/tiers"
)

// Status represents the current status of a tenant.
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
)

// Tenant represents a HELM tenant.
type Tenant struct {
	ID            string         `json:"id"`
	Email         string         `json:"email"`
	EmailVerified bool           `json:"email_verified"`
	TierID        tiers.TierID   `json:"tier_id"`
	Status        Status         `json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	SuspendedAt   *time.Time     `json:"suspended_at,omitempty"`
	DeletedAt     *time.Time     `json:"deleted_at,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// IsActive returns true if the tenant is active.
func (t *Tenant) IsActive() bool {
	return t.Status == StatusActive
}

// CreateRequest contains the data needed to create a new tenant.
type CreateRequest struct {
	Email    string         `json:"email"`
	Password string         `json:"password,omitempty"` // omitempty for OAuth
	Metadata map[string]any `json:"metadata,omitempty"`
}

// APIKey represents an API key for a tenant.
type APIKey struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	Name      string     `json:"name"`
	KeyPrefix string     `json:"key_prefix"` // First 8 chars for identification
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// DataExport contains all data associated with a tenant for GDPR export.
type DataExport struct {
	Tenant        *Tenant   `json:"tenant"`
	ExportedAt    time.Time `json:"exported_at"`
	UsageEvents   []any     `json:"usage_events,omitempty"`
	Receipts      []any     `json:"receipts,omitempty"`
	BudgetHistory []any     `json:"budget_history,omitempty"`
	APIKeys       []APIKey  `json:"api_keys,omitempty"`
}
