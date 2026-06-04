package service

import (
	"context"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func TestBillingParitySnapshotSimpleTextUsage(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-parity-simple")
	billing, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.NoError(t, err)

	snapshot, err := BuildBillingCalculationSnapshotFromUsage(usage, billing)
	require.NoError(t, err)
	require.Equal(t, usage.RequestId, snapshot.RequestId)
	require.Equal(t, usage.ModelName, snapshot.ModelName)
	require.Equal(t, usage.ProviderName, snapshot.ProviderName)
	require.Equal(t, usage.Id, snapshot.UsageRecordId)
	require.Equal(t, usage.InputTokens, snapshot.InputTokens)
	require.Equal(t, usage.OutputTokens, snapshot.OutputTokens)
	require.Equal(t, usage.TotalTokens, snapshot.TotalTokens)
	require.Equal(t, billing.QuotaCharged, snapshot.QuotaCharged)
	require.Equal(t, "billing_record_shadow", snapshot.CalculationSource)
}

func TestBillingParitySnapshotTokenDeltaMatchesBillingRecordQuota(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-parity-token-delta")
	billing, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.NoError(t, err)

	snapshot, err := BuildBillingCalculationSnapshotFromUsage(usage, billing)
	require.NoError(t, err)
	require.Equal(t, usage.TokenDelta, snapshot.QuotaCharged)

	comparison := CompareBillingCalculationSnapshot(BillingCalculationSnapshot{QuotaCharged: usage.TokenDelta}, snapshot)
	require.True(t, comparison.Match)
	require.Equal(t, int64(0), comparison.Delta)
	require.Equal(t, "quota_match", comparison.Reason)
}

func TestBillingParitySnapshotTotalTokensFallback(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-parity-total-fallback")
	usage.TokenDelta = 0
	require.NoError(t, model.DB.Save(&usage).Error)

	billing, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.NoError(t, err)
	snapshot, err := BuildBillingCalculationSnapshotFromUsage(usage, billing)
	require.NoError(t, err)
	require.Equal(t, usage.TotalTokens, snapshot.QuotaCharged)
}

func TestBillingParityCompareDetectsQuotaMismatch(t *testing.T) {
	expected := BillingCalculationSnapshot{QuotaCharged: 1000}
	actual := BillingCalculationSnapshot{QuotaCharged: 125}

	comparison := CompareBillingCalculationSnapshot(expected, actual)
	require.False(t, comparison.Match)
	require.Equal(t, int64(-875), comparison.Delta)
	require.Equal(t, "quota_mismatch", comparison.Reason)
	require.Equal(t, int64(1000), comparison.ExpectedQuota)
	require.Equal(t, int64(125), comparison.ActualQuota)
}

func TestBillingParitySnapshotPreservesTenantProviderChannelModelAttribution(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-parity-attribution")
	billing, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.NoError(t, err)

	snapshot, err := BuildBillingCalculationSnapshotFromUsage(usage, billing)
	require.NoError(t, err)
	require.Equal(t, usage.TenantId, snapshot.TenantId)
	require.Equal(t, usage.OrganizationId, snapshot.OrganizationId)
	require.Equal(t, usage.DepartmentId, snapshot.DepartmentId)
	require.Equal(t, usage.DistributionChannelId, snapshot.DistributionChannelId)
	require.Equal(t, usage.ProviderName, snapshot.ProviderName)
	require.Equal(t, usage.ChannelId, snapshot.ChannelId)
	require.Equal(t, usage.ModelName, snapshot.ModelName)
}

func TestBillingParitySnapshotDoesNotMutateWalletTokenOrSubscription(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()

	user := model.User{
		Id:       8701,
		TenantId: 8,
		Username: "billing-parity-user-" + time.Now().Format("150405.000000"),
		Quota:    9000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(&user).Error)
	token := model.Token{
		Id:          8702,
		TenantId:    user.TenantId,
		UserId:      user.Id,
		Key:         "billing-parity-token-" + time.Now().Format("150405.000000"),
		Name:        "billing parity token",
		Status:      common.TokenStatusEnabled,
		RemainQuota: 8000,
	}
	require.NoError(t, model.DB.Create(&token).Error)
	sub := model.UserSubscription{
		Id:                   8703,
		TenantId:             user.TenantId,
		UserId:               user.Id,
		AmountTotal:          100000,
		AmountUsed:           456,
		StartTime:            time.Now().Add(-time.Hour).Unix(),
		EndTime:              time.Now().Add(24 * time.Hour).Unix(),
		Status:               model.SubscriptionLifecycleActive,
		LifecycleStatus:      model.SubscriptionLifecycleActive,
		TokenQuotaSnapshot:   1000,
		RequestQuotaSnapshot: 10,
	}
	require.NoError(t, model.DB.Create(&sub).Error)

	usage := createBillingRuntimeUsageRecord(t, "billing-parity-no-mutation")
	usage.UserId = user.Id
	usage.UserSubscriptionId = sub.Id
	require.NoError(t, model.DB.Save(&usage).Error)

	billing, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.NoError(t, err)
	_, err = BuildBillingCalculationSnapshotFromUsage(usage, billing)
	require.NoError(t, err)
	_ = CompareBillingCalculationSnapshot(BillingCalculationSnapshot{QuotaCharged: billing.QuotaCharged}, BillingCalculationSnapshot{QuotaCharged: billing.QuotaCharged})

	var reloadedUser model.User
	require.NoError(t, model.DB.Select("quota").Where("id = ?", user.Id).First(&reloadedUser).Error)
	require.Equal(t, user.Quota, reloadedUser.Quota)

	var reloadedToken model.Token
	require.NoError(t, model.DB.Select("remain_quota").Where("id = ?", token.Id).First(&reloadedToken).Error)
	require.Equal(t, token.RemainQuota, reloadedToken.RemainQuota)

	var reloadedSub model.UserSubscription
	require.NoError(t, model.DB.Select("amount_used").Where("id = ?", sub.Id).First(&reloadedSub).Error)
	require.Equal(t, sub.AmountUsed, reloadedSub.AmountUsed)
}
