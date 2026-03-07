package tiers_test

import (
	"testing"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/tiers"
	"github.com/stretchr/testify/assert"
)

func TestTiers_Get(t *testing.T) {
	tests := []struct {
		id       tiers.TierID
		expected string
	}{
		{tiers.TierFree, "Free"},
		{tiers.TierPro, "Pro"},
		{tiers.TierEnterprise, "Enterprise"},
	}

	for _, tt := range tests {
		tier := tiers.Get(tt.id)
		assert.NotNil(t, tier)
		assert.Equal(t, tt.expected, tier.Name)
	}
}

func TestTiers_GetUnknown(t *testing.T) {
	tier := tiers.Get("unknown-tier")
	assert.Nil(t, tier)
}

func TestTiers_FreeLimits(t *testing.T) {
	tier := tiers.Free
	assert.Equal(t, int64(100), tier.Limits.DailyExecutions)
	assert.Equal(t, int64(100_000), tier.Limits.MonthlyTokens)
	assert.Equal(t, int64(1), tier.Limits.StorageGB)
}

func TestTiers_ProLimits(t *testing.T) {
	tier := tiers.Pro
	assert.Equal(t, int64(10_000), tier.Limits.DailyExecutions)
	assert.Equal(t, int64(10_000_000), tier.Limits.MonthlyTokens)
	assert.Equal(t, int64(9900), tier.PricePerMonth)
}

func TestTiers_EnterpriseUnlimited(t *testing.T) {
	tier := tiers.Enterprise
	assert.True(t, tiers.IsUnlimited(tier.Limits.DailyExecutions))
	assert.True(t, tiers.IsUnlimited(tier.Limits.MonthlyTokens))
	assert.True(t, tiers.IsUnlimited(tier.Limits.StorageGB))
}

func TestTiers_HasFeature(t *testing.T) {
	// Free tier
	assert.True(t, tiers.Free.HasFeature("basic_governance"))
	assert.False(t, tiers.Free.HasFeature("hsm"))

	// Pro tier
	assert.True(t, tiers.Pro.HasFeature("advanced_receipts"))
	assert.False(t, tiers.Pro.HasFeature("hsm"))

	// Enterprise has "all"
	assert.True(t, tiers.Enterprise.HasFeature("hsm"))
	assert.True(t, tiers.Enterprise.HasFeature("any_feature")) // "all" matches anything
}

func TestTiers_AllTiers(t *testing.T) {
	assert.Len(t, tiers.AllTiers, 3)
	assert.Contains(t, tiers.AllTiers, tiers.TierFree)
	assert.Contains(t, tiers.AllTiers, tiers.TierPro)
	assert.Contains(t, tiers.AllTiers, tiers.TierEnterprise)
}
