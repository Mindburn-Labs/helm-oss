package tenants

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/tiers"
)

// Provisioner handles tenant lifecycle operations.
type Provisioner interface {
	// Create creates a new tenant with default resources.
	Create(ctx context.Context, req CreateRequest) (*Tenant, string, error) // returns tenant and raw API key

	// GetByEmail retrieves a tenant by email.
	GetByEmail(ctx context.Context, email string) (*Tenant, error)
}

// PostgresProvisioner implements Provisioner with PostgreSQL.
type PostgresProvisioner struct {
	db *sql.DB
}

// NewPostgresProvisioner creates a new PostgreSQL-backed provisioner.
func NewPostgresProvisioner(db *sql.DB) *PostgresProvisioner {
	return &PostgresProvisioner{db: db}
}

const schema = `
CREATE TABLE IF NOT EXISTS tenants (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	email_verified BOOLEAN DEFAULT FALSE,
	tier_id TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL,
	suspended_at TIMESTAMP,
	deleted_at TIMESTAMP,
	metadata JSONB
);

CREATE TABLE IF NOT EXISTS tenant_budgets (
	tenant_id TEXT PRIMARY KEY REFERENCES tenants(id),
	daily_limit_cents BIGINT NOT NULL,
	monthly_limit_cents BIGINT NOT NULL,
	updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS api_keys (
	id TEXT PRIMARY KEY,
	tenant_id TEXT REFERENCES tenants(id),
	key_hash TEXT NOT NULL,
	name TEXT,
	created_at TIMESTAMP NOT NULL,
	revoked_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
`

// Init creates the necessary database tables.
func (p *PostgresProvisioner) Init(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, schema)
	return err
}

// Create creates a new tenant with all required resources.
func (p *PostgresProvisioner) Create(ctx context.Context, req CreateRequest) (*Tenant, string, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", fmt.Errorf("tenants: failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Create tenant
	tenant := &Tenant{
		ID:            uuid.New().String(),
		Email:         req.Email,
		EmailVerified: false,
		TierID:        tiers.TierFree,
		Status:        StatusActive,
		CreatedAt:     time.Now().UTC(),
		Metadata:      req.Metadata,
	}

	metaJSON, err := json.Marshal(tenant.Metadata)
	if err != nil {
		return nil, "", fmt.Errorf("tenants: failed to marshal metadata: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO tenants (id, email, email_verified, tier_id, status, created_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenant.ID, tenant.Email, tenant.EmailVerified, tenant.TierID, tenant.Status, tenant.CreatedAt, metaJSON)
	if err != nil {
		return nil, "", fmt.Errorf("tenants: failed to create tenant: %w", err)
	}

	// Create default budget based on free tier limits
	freeTier := tiers.Free
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tenant_budgets (tenant_id, daily_limit_cents, monthly_limit_cents, updated_at)
		VALUES ($1, $2, $3, NOW())
	`, tenant.ID, freeTier.Limits.DailyExecutions*10, freeTier.Limits.MonthlyTokens/100)
	if err != nil {
		return nil, "", fmt.Errorf("tenants: failed to create budget: %w", err)
	}

	// Generate API key
	rawKey, keyHash := generateAPIKey()
	apiKeyID := uuid.New().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO api_keys (id, tenant_id, key_hash, name, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, apiKeyID, tenant.ID, keyHash, "Default Key")
	if err != nil {
		return nil, "", fmt.Errorf("tenants: failed to create API key: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, "", fmt.Errorf("tenants: failed to commit: %w", err)
	}

	return tenant, rawKey, nil
}

// Get retrieves a tenant by ID.

// GetByEmail retrieves a tenant by email.
func (p *PostgresProvisioner) GetByEmail(ctx context.Context, email string) (*Tenant, error) {
	var tenant Tenant
	var metaJSON []byte
	err := p.db.QueryRowContext(ctx, `
		SELECT id, email, email_verified, tier_id, status, created_at, suspended_at, deleted_at, metadata
		FROM tenants WHERE email = $1
	`, email).Scan(
		&tenant.ID, &tenant.Email, &tenant.EmailVerified, &tenant.TierID,
		&tenant.Status, &tenant.CreatedAt, &tenant.SuspendedAt, &tenant.DeletedAt, &metaJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenants: not found")
		}
		return nil, fmt.Errorf("tenants: failed to get by email: %w", err)
	}
	if len(metaJSON) > 0 {
		if err := json.Unmarshal(metaJSON, &tenant.Metadata); err != nil {
			return nil, fmt.Errorf("tenants: failed to unmarshal metadata: %w", err)
		}
	}
	return &tenant, nil
}

// Suspend suspends a tenant.

// generateAPIKey creates a cryptographically secure API key.
func generateAPIKey() (raw, hash string) {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	raw = "helm_" + hex.EncodeToString(bytes)
	hashBytes := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(hashBytes[:])
	return raw, hash
}
