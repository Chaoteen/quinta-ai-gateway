package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func TestRevenueShareRuleCreateAndRateValidation(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	scope := rootRevenueShareScope()

	rule, err := CreateRevenueShareRule(ctx, scope, RevenueShareRuleInput{
		TenantId:                   1,
		RuleName:                   "global",
		RuleScope:                  model.RevenueShareRuleScopeGlobal,
		PlatformShareRate:          70,
		MasterDistributorShareRate: 20,
		DistributorShareRate:       10,
		Enabled:                    true,
	})
	require.NoError(t, err)
	require.NotZero(t, rule.Id)
	require.Equal(t, 1, rule.TenantId)

	_, err = CreateRevenueShareRule(ctx, scope, RevenueShareRuleInput{
		TenantId:                   1,
		RuleName:                   "invalid",
		RuleScope:                  model.RevenueShareRuleScopeGlobal,
		PlatformShareRate:          70,
		MasterDistributorShareRate: 20,
		DistributorShareRate:       9,
		Enabled:                    true,
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRevenueShareInvalidInput))
}

func TestRevenueShareRuleMatchingPriority(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	scope := rootRevenueShareScope()

	createRevenueShareRuleForTest(t, ctx, scope, RevenueShareRuleInput{
		TenantId:                   1,
		RuleName:                   "global",
		RuleScope:                  model.RevenueShareRuleScopeGlobal,
		PlatformShareRate:          100,
		MasterDistributorShareRate: 0,
		DistributorShareRate:       0,
		Enabled:                    true,
	})
	provider := createRevenueShareRuleForTest(t, ctx, scope, RevenueShareRuleInput{
		TenantId:                   1,
		RuleName:                   "provider",
		RuleScope:                  model.RevenueShareRuleScopeProvider,
		ProviderName:               "openai",
		PlatformShareRate:          60,
		MasterDistributorShareRate: 30,
		DistributorShareRate:       10,
		Enabled:                    true,
	})
	modelRule := createRevenueShareRuleForTest(t, ctx, scope, RevenueShareRuleInput{
		TenantId:                   1,
		RuleName:                   "model",
		RuleScope:                  model.RevenueShareRuleScopeModel,
		ModelName:                  "gpt-4o",
		PlatformShareRate:          50,
		MasterDistributorShareRate: 30,
		DistributorShareRate:       20,
		Enabled:                    true,
	})

	providerResult, err := CalculateRevenueShare(ctx, RevenueShareCalculationInput{
		TenantId:     1,
		GrossAmount:  100,
		ProviderName: "openai",
		ModelName:    "gpt-3.5",
	})
	require.NoError(t, err)
	require.Equal(t, provider.Id, providerResult.MatchedRuleId)
	require.Equal(t, float64(60), providerResult.PlatformAmount)

	modelResult, err := CalculateRevenueShare(ctx, RevenueShareCalculationInput{
		TenantId:     1,
		GrossAmount:  100,
		ProviderName: "openai",
		ModelName:    "gpt-4o",
	})
	require.NoError(t, err)
	require.Equal(t, modelRule.Id, modelResult.MatchedRuleId)
	require.Equal(t, float64(50), modelResult.PlatformAmount)
}

func TestRevenueShareNoRuleDefaultsPlatformHundredPercent(t *testing.T) {
	truncate(t)

	result, err := CalculateRevenueShare(context.Background(), RevenueShareCalculationInput{
		TenantId:     1,
		GrossAmount:  123.45,
		ProviderName: "unknown",
		ModelName:    "unknown-model",
	})
	require.NoError(t, err)
	require.Zero(t, result.MatchedRuleId)
	require.Equal(t, 123.45, result.PlatformAmount)
	require.Zero(t, result.MasterDistributorAmount)
	require.Zero(t, result.DistributorAmount)
}

func TestRevenueShareBillingRecordIdempotency(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	scope := rootRevenueShareScope()
	billing := createRevenueShareBillingRecord(t, model.BillingRecord{
		TenantId:              1,
		DistributionChannelId: 0,
		RequestId:             "revenue-idempotent",
		UsageRecordId:         101,
		UserId:                11,
		ProviderName:          "openai",
		ModelName:             "gpt-4o",
		Currency:              "QUOTA",
		QuotaCharged:          1000,
	})

	first, err := CreateRevenueShareRecordFromBillingRecord(ctx, scope, billing.Id)
	require.NoError(t, err)
	second, err := CreateRevenueShareRecordFromBillingRecord(ctx, scope, billing.Id)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)

	var count int64
	require.NoError(t, model.DB.Model(&model.RevenueShareRecord{}).Where("billing_record_id = ?", billing.Id).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestRevenueShareTenantScopeIsolation(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	tenantAdmin := model.AccessScope{TenantId: 1, RoleKey: common.RoleKeyTenantAdmin}
	otherBilling := createRevenueShareBillingRecord(t, model.BillingRecord{
		TenantId:      2,
		RequestId:     "tenant-two-billing",
		UsageRecordId: 202,
		UserId:        22,
		QuotaCharged:  100,
	})

	_, err := CreateRevenueShareRecordFromBillingRecord(ctx, tenantAdmin, otherBilling.Id)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRevenueShareTenantScopeDenied))
}

func TestRevenueShareChannelHierarchy(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	master := createRevenueShareDistributionChannel(t, model.DistributionChannel{
		TenantId: 1,
		Name:     "master",
		Code:     "master-revenue",
	})
	distributor := createRevenueShareDistributionChannel(t, model.DistributionChannel{
		TenantId: 1,
		Name:     "distributor",
		Code:     "distributor-revenue",
		ParentId: master.Id,
	})

	result, err := CalculateRevenueShare(ctx, RevenueShareCalculationInput{
		TenantId:              1,
		GrossAmount:           100,
		DistributionChannelId: distributor.Id,
	})
	require.NoError(t, err)
	require.Equal(t, master.Id, result.MasterDistributorId)
	require.Equal(t, distributor.Id, result.DistributorId)

	child := createRevenueShareDistributionChannel(t, model.DistributionChannel{
		TenantId: 1,
		Name:     "child",
		Code:     "child-revenue",
		ParentId: distributor.Id,
	})
	_, err = CalculateRevenueShare(ctx, RevenueShareCalculationInput{
		TenantId:              1,
		GrossAmount:           100,
		DistributionChannelId: child.Id,
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRevenueShareChannelLevelInvalid))
}

func createRevenueShareRuleForTest(t *testing.T, ctx context.Context, scope model.AccessScope, input RevenueShareRuleInput) model.RevenueShareRule {
	t.Helper()
	rule, err := CreateRevenueShareRule(ctx, scope, input)
	require.NoError(t, err)
	return rule
}

func createRevenueShareBillingRecord(t *testing.T, billing model.BillingRecord) model.BillingRecord {
	t.Helper()
	require.NoError(t, model.DB.Create(&billing).Error)
	return billing
}

func createRevenueShareDistributionChannel(t *testing.T, channel model.DistributionChannel) model.DistributionChannel {
	t.Helper()
	require.NoError(t, model.DB.Create(&channel).Error)
	return channel
}

func rootRevenueShareScope() model.AccessScope {
	return model.AccessScope{TenantId: 1, RoleKey: common.RoleKeyRoot, IsRoot: true}
}
