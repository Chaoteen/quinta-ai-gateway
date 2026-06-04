package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRevenueShareModelsMigrated(t *testing.T) {
	require.True(t, DB.Migrator().HasTable(&RevenueShareRule{}))
	require.True(t, DB.Migrator().HasTable(&RevenueShareRecord{}))

	for _, column := range []string{
		"tenant_id",
		"distribution_channel_id",
		"rule_scope",
		"platform_share_rate",
		"master_distributor_share_rate",
		"distributor_share_rate",
	} {
		require.True(t, DB.Migrator().HasColumn(&RevenueShareRule{}, column), "missing revenue share rule column %s", column)
	}

	for _, column := range []string{
		"tenant_id",
		"billing_record_id",
		"source_type",
		"gross_amount",
		"platform_amount",
		"master_distributor_amount",
		"distributor_amount",
		"share_rule_id",
		"status",
	} {
		require.True(t, DB.Migrator().HasColumn(&RevenueShareRecord{}, column), "missing revenue share record column %s", column)
	}

}

func TestRevenueShareDefaultsAndNormalize(t *testing.T) {
	truncateTables(t)

	rule := RevenueShareRule{
		RuleName:                   " Global Share ",
		RuleScope:                  "unknown",
		PlatformShareRate:          100,
		MasterDistributorShareRate: 0,
		DistributorShareRate:       0,
	}
	require.NoError(t, DB.Create(&rule).Error)
	require.Equal(t, 1, rule.TenantId)
	require.Equal(t, RevenueShareRuleScopeGlobal, rule.RuleScope)
	require.Equal(t, "Global Share", rule.RuleName)

	record := RevenueShareRecord{
		TenantId:        2,
		SourceType:      "unknown",
		GrossAmount:     100,
		PlatformAmount:  100,
		Currency:        "QUOTA",
		Status:          "unknown",
		BillingRecordId: 1,
	}
	require.NoError(t, DB.Create(&record).Error)
	require.Equal(t, RevenueShareSourceBilling, record.SourceType)
	require.Equal(t, RevenueShareStatusPending, record.Status)
	require.NotZero(t, record.CreatedAt)
	require.NotZero(t, record.UpdatedAt)
}
