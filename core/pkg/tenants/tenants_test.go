package tenants_test

import (
	"context"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/tenants"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/tiers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvisioner implements Provisioner for testing
type MockProvisioner struct {
	tenants map[string]*tenants.Tenant
	keys    map[string]string // tenantID -> raw key
}

func NewMockProvisioner() *MockProvisioner {
	return &MockProvisioner{
		tenants: make(map[string]*tenants.Tenant),
		keys:    make(map[string]string),
	}
}

func (p *MockProvisioner) Create(ctx context.Context, req tenants.CreateRequest) (*tenants.Tenant, string, error) {
	tenant := &tenants.Tenant{
		ID:            "tenant-" + req.Email,
		Email:         req.Email,
		EmailVerified: false,
		TierID:        tiers.TierFree,
		Status:        tenants.StatusActive,
		CreatedAt:     time.Now().UTC(),
		Metadata:      req.Metadata,
	}
	rawKey := "helm_test_" + tenant.ID
	p.tenants[tenant.ID] = tenant
	p.keys[tenant.ID] = rawKey
	return tenant, rawKey, nil
}

func (p *MockProvisioner) Get(ctx context.Context, tenantID string) (*tenants.Tenant, error) {
	t, ok := p.tenants[tenantID]
	if !ok {
		return nil, assert.AnError
	}
	return t, nil
}

func (p *MockProvisioner) GetByEmail(ctx context.Context, email string) (*tenants.Tenant, error) {
	for _, t := range p.tenants {
		if t.Email == email {
			return t, nil
		}
	}
	return nil, assert.AnError
}

func (p *MockProvisioner) Suspend(ctx context.Context, tenantID, reason string) error {
	t, ok := p.tenants[tenantID]
	if !ok {
		return assert.AnError
	}
	now := time.Now().UTC()
	t.Status = tenants.StatusSuspended
	t.SuspendedAt = &now
	return nil
}

func (p *MockProvisioner) Reactivate(ctx context.Context, tenantID string) error {
	t, ok := p.tenants[tenantID]
	if !ok {
		return assert.AnError
	}
	t.Status = tenants.StatusActive
	t.SuspendedAt = nil
	return nil
}

func (p *MockProvisioner) Delete(ctx context.Context, tenantID string) error {
	t, ok := p.tenants[tenantID]
	if !ok {
		return assert.AnError
	}
	now := time.Now().UTC()
	t.Status = tenants.StatusDeleted
	t.DeletedAt = &now
	return nil
}

func (p *MockProvisioner) VerifyEmail(ctx context.Context, tenantID string) error {
	t, ok := p.tenants[tenantID]
	if !ok {
		return assert.AnError
	}
	t.EmailVerified = true
	return nil
}

func (p *MockProvisioner) Export(ctx context.Context, tenantID string) (*tenants.DataExport, error) {
	t, ok := p.tenants[tenantID]
	if !ok {
		return nil, assert.AnError
	}
	return &tenants.DataExport{
		Tenant:     t,
		ExportedAt: time.Now().UTC(),
	}, nil
}

func TestProvisioner_Create(t *testing.T) {
	prov := NewMockProvisioner()
	ctx := context.Background()

	tenant, apiKey, err := prov.Create(ctx, tenants.CreateRequest{
		Email: "test@example.com",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tenant.ID)
	assert.Equal(t, "test@example.com", tenant.Email)
	assert.False(t, tenant.EmailVerified)
	assert.Equal(t, tiers.TierFree, tenant.TierID)
	assert.Equal(t, tenants.StatusActive, tenant.Status)
	assert.NotEmpty(t, apiKey)
}

func TestProvisioner_Lifecycle(t *testing.T) {
	prov := NewMockProvisioner()
	ctx := context.Background()

	// Create
	tenant, _, _ := prov.Create(ctx, tenants.CreateRequest{Email: "lifecycle@test.com"})
	assert.True(t, tenant.IsActive())

	// Suspend
	err := prov.Suspend(ctx, tenant.ID, "testing")
	require.NoError(t, err)
	tenant, _ = prov.Get(ctx, tenant.ID)
	assert.Equal(t, tenants.StatusSuspended, tenant.Status)
	assert.NotNil(t, tenant.SuspendedAt)

	// Reactivate
	err = prov.Reactivate(ctx, tenant.ID)
	require.NoError(t, err)
	tenant, _ = prov.Get(ctx, tenant.ID)
	assert.Equal(t, tenants.StatusActive, tenant.Status)

	// Delete
	err = prov.Delete(ctx, tenant.ID)
	require.NoError(t, err)
	tenant, _ = prov.Get(ctx, tenant.ID)
	assert.Equal(t, tenants.StatusDeleted, tenant.Status)
	assert.NotNil(t, tenant.DeletedAt)
}

func TestProvisioner_VerifyEmail(t *testing.T) {
	prov := NewMockProvisioner()
	ctx := context.Background()

	tenant, _, _ := prov.Create(ctx, tenants.CreateRequest{Email: "verify@test.com"})
	assert.False(t, tenant.EmailVerified)

	err := prov.VerifyEmail(ctx, tenant.ID)
	require.NoError(t, err)

	tenant, _ = prov.Get(ctx, tenant.ID)
	assert.True(t, tenant.EmailVerified)
}

func TestProvisioner_Export(t *testing.T) {
	prov := NewMockProvisioner()
	ctx := context.Background()

	tenant, _, _ := prov.Create(ctx, tenants.CreateRequest{Email: "export@test.com"})

	export, err := prov.Export(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, tenant.ID, export.Tenant.ID)
	assert.NotZero(t, export.ExportedAt)
}

func TestProvisioner_GetByEmail(t *testing.T) {
	prov := NewMockProvisioner()
	ctx := context.Background()

	_, _, _ = prov.Create(ctx, tenants.CreateRequest{Email: "find@test.com"})

	found, err := prov.GetByEmail(ctx, "find@test.com")
	require.NoError(t, err)
	assert.Equal(t, "find@test.com", found.Email)

	_, err = prov.GetByEmail(ctx, "notfound@test.com")
	assert.Error(t, err)
}
