package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func TestBillingRuntimeCreateBillingRecordFromUsagePreservesAttribution(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-runtime-request-1")

	record, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.NoError(t, err)
	require.NotZero(t, record.Id)
	require.Equal(t, model.BillingStatusPending, record.BillingStatus)
	require.Equal(t, model.BillingPhaseUsageFact, record.BillingPhase)
	require.Equal(t, usage.TenantId, record.TenantId)
	require.Equal(t, usage.OrganizationId, record.OrganizationId)
	require.Equal(t, usage.DepartmentId, record.DepartmentId)
	require.Equal(t, usage.DistributionChannelId, record.DistributionChannelId)
	require.Equal(t, usage.RequestId, record.RequestId)
	require.Equal(t, usage.ReservationId, record.ReservationId)
	require.Equal(t, usage.Id, record.UsageRecordId)
	require.Equal(t, usage.UserId, record.UserId)
	require.Equal(t, usage.UserSubscriptionId, record.UserSubscriptionId)
	require.Equal(t, usage.ProviderName, record.ProviderName)
	require.Equal(t, usage.ChannelId, record.ChannelId)
	require.Equal(t, usage.ModelName, record.ModelName)
	require.Equal(t, usage.InputTokens, record.InputTokens)
	require.Equal(t, usage.OutputTokens, record.OutputTokens)
	require.Equal(t, usage.TotalTokens, record.TotalTokens)
	require.Equal(t, usage.RequestCount, record.RequestCount)
	require.Equal(t, usage.TokenDelta, record.QuotaCharged)
	require.Equal(t, int64(0), record.SettledDelta)
	require.Equal(t, "QUOTA", record.Currency)
}

func TestBillingRuntimeQueriesByRequestIdAndUsageRecordId(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-runtime-request-2")

	record, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.NoError(t, err)

	byRequest, err := service.GetBillingRecordByRequestId(ctx, usage.RequestId)
	require.NoError(t, err)
	require.Equal(t, record.Id, byRequest.Id)

	byUsage, err := service.GetBillingRecordByUsageRecordId(ctx, usage.Id)
	require.NoError(t, err)
	require.Equal(t, record.Id, byUsage.Id)
}

func TestBillingRuntimeCreateBillingRecordIsIdempotentByUsageRecordId(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-runtime-request-3")

	first, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.NoError(t, err)
	second, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)

	var count int64
	require.NoError(t, model.DB.Model(&model.BillingRecord{}).
		Where("usage_record_id = ?", usage.Id).
		Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestBillingRuntimeRejectsInvalidUsageFacts(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-runtime-request-4")

	usage.TenantId = 0
	_, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrBillingRuntimeInvalidUsage))

	usage = createBillingRuntimeUsageRecord(t, "billing-runtime-request-5")
	usage.Status = model.QuotaUsageStatusReserved
	_, err = service.CreateBillingRecordFromUsage(ctx, usage)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrBillingRuntimeInvalidUsage))
}

func TestBillingRuntimeDoesNotMutateWalletTokenOrSubscription(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()

	user := model.User{
		Id:       8501,
		TenantId: 8,
		Username: "billing-runtime-user-" + time.Now().Format("150405.000000"),
		Quota:    9000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(&user).Error)
	token := model.Token{
		Id:          8502,
		TenantId:    user.TenantId,
		UserId:      user.Id,
		Key:         "billing-runtime-token-" + time.Now().Format("150405.000000"),
		Name:        "billing runtime token",
		Status:      common.TokenStatusEnabled,
		RemainQuota: 8000,
	}
	require.NoError(t, model.DB.Create(&token).Error)
	sub := model.UserSubscription{
		Id:                   8503,
		TenantId:             user.TenantId,
		UserId:               user.Id,
		AmountTotal:          100000,
		AmountUsed:           123,
		StartTime:            time.Now().Add(-time.Hour).Unix(),
		EndTime:              time.Now().Add(24 * time.Hour).Unix(),
		Status:               model.SubscriptionLifecycleActive,
		LifecycleStatus:      model.SubscriptionLifecycleActive,
		TokenQuotaSnapshot:   1000,
		RequestQuotaSnapshot: 10,
	}
	require.NoError(t, model.DB.Create(&sub).Error)

	usage := createBillingRuntimeUsageRecord(t, "billing-runtime-request-6")
	usage.UserId = user.Id
	usage.UserSubscriptionId = sub.Id
	require.NoError(t, model.DB.Save(&usage).Error)

	_, err := service.CreateBillingRecordFromUsage(ctx, usage)
	require.NoError(t, err)

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

func TestBillingRuntimeShadowGenerationFromUsageRecordId(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-shadow-usage-id")

	record, err := service.CreateShadowBillingFromUsageRecordId(ctx, usage.Id)
	require.NoError(t, err)
	require.Equal(t, usage.Id, record.UsageRecordId)
	require.Equal(t, usage.RequestId, record.RequestId)
}

func TestBillingRuntimeShadowGenerationFromRequestId(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-shadow-request-id")

	record, err := service.CreateShadowBillingFromRequestId(ctx, usage.RequestId)
	require.NoError(t, err)
	require.Equal(t, usage.Id, record.UsageRecordId)
	require.Equal(t, usage.RequestId, record.RequestId)
}

func TestBillingRuntimeShadowGenerationIsIdempotent(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-shadow-idempotent")

	first, err := service.CreateShadowBillingFromUsageRecordId(ctx, usage.Id)
	require.NoError(t, err)
	second, err := service.EnsureBillingRecordForUsage(ctx, usage)
	require.NoError(t, err)
	third, err := service.CreateShadowBillingFromRequestId(ctx, usage.RequestId)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)
	require.Equal(t, first.Id, third.Id)

	var count int64
	require.NoError(t, model.DB.Model(&model.BillingRecord{}).
		Where("usage_record_id = ?", usage.Id).
		Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestBillingRuntimeShadowGenerationRejectsMissingUsageRecord(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()

	_, err := service.CreateShadowBillingFromUsageRecordId(ctx, 999999)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrBillingUsageRecordNotFound))
}

func TestBillingRuntimeShadowGenerationRejectsNonCommittedUsageRecord(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-shadow-non-committed")
	require.NoError(t, model.DB.Model(&model.QuotaUsageRecord{}).
		Where("id = ?", usage.Id).
		Update("status", model.QuotaUsageStatusReserved).Error)

	_, err := service.CreateShadowBillingFromUsageRecordId(ctx, usage.Id)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrBillingRuntimeInvalidUsage))
}

func TestBillingRuntimeShadowGenerationRejectsEmptyRequestId(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()

	_, err := service.CreateShadowBillingFromRequestId(ctx, " ")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrBillingRuntimeInvalidUsage))
}

func TestBillingRuntimeShadowGenerationRejectsAmbiguousRequestId(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	requestId := "billing-shadow-ambiguous"
	createBillingRuntimeUsageRecord(t, requestId)
	createBillingRuntimeUsageRecord(t, requestId)

	_, err := service.CreateShadowBillingFromRequestId(ctx, requestId)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrBillingUsageRecordAmbiguous))

	var count int64
	require.NoError(t, model.DB.Model(&model.BillingRecord{}).
		Where("request_id = ?", requestId).
		Count(&count).Error)
	require.Zero(t, count)
}

func TestBillingRuntimeShadowGenerationRejectsMissingTenant(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()
	usage := createBillingRuntimeUsageRecord(t, "billing-shadow-missing-tenant")
	require.NoError(t, model.DB.Model(&model.QuotaUsageRecord{}).
		Where("id = ?", usage.Id).
		Update("tenant_id", 0).Error)

	_, err := service.CreateShadowBillingFromUsageRecordId(ctx, usage.Id)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrBillingRuntimeInvalidUsage))
}

func TestBillingRuntimeShadowGenerationDoesNotMutateWalletTokenOrSubscription(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationBillingRuntimeService()

	user := model.User{
		Id:       8601,
		TenantId: 8,
		Username: "billing-shadow-user-" + time.Now().Format("150405.000000"),
		Quota:    9000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(&user).Error)
	token := model.Token{
		Id:          8602,
		TenantId:    user.TenantId,
		UserId:      user.Id,
		Key:         "billing-shadow-token-" + time.Now().Format("150405.000000"),
		Name:        "billing shadow token",
		Status:      common.TokenStatusEnabled,
		RemainQuota: 8000,
	}
	require.NoError(t, model.DB.Create(&token).Error)
	sub := model.UserSubscription{
		Id:                   8603,
		TenantId:             user.TenantId,
		UserId:               user.Id,
		AmountTotal:          100000,
		AmountUsed:           321,
		StartTime:            time.Now().Add(-time.Hour).Unix(),
		EndTime:              time.Now().Add(24 * time.Hour).Unix(),
		Status:               model.SubscriptionLifecycleActive,
		LifecycleStatus:      model.SubscriptionLifecycleActive,
		TokenQuotaSnapshot:   1000,
		RequestQuotaSnapshot: 10,
	}
	require.NoError(t, model.DB.Create(&sub).Error)

	usage := createBillingRuntimeUsageRecord(t, "billing-shadow-no-mutation")
	usage.UserId = user.Id
	usage.UserSubscriptionId = sub.Id
	require.NoError(t, model.DB.Save(&usage).Error)

	_, err := service.CreateShadowBillingFromUsageRecordId(ctx, usage.Id)
	require.NoError(t, err)

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

func createBillingRuntimeUsageRecord(t *testing.T, requestId string) model.QuotaUsageRecord {
	t.Helper()
	usage := model.QuotaUsageRecord{
		TenantId:              11,
		OrganizationId:        12,
		DepartmentId:          13,
		DistributionChannelId: 14,
		UserId:                8101,
		UserSubscriptionId:    8201,
		RequestId:             requestId,
		ReservationId:         requestId + "-reservation",
		ProviderName:          "openai",
		ChannelId:             8301,
		ModelName:             "gpt-4o",
		UpstreamModelName:     "gpt-4o-2024-08-06",
		QuotaDimension:        model.QuotaDimensionToken,
		RequestCount:          1,
		InputTokens:           100,
		OutputTokens:          25,
		TotalTokens:           125,
		TokenDelta:            125,
		RequestDelta:          1,
		UsageSource:           UsageSourceUpstream,
		UsageSemantic:         UsageSemanticOpenAI,
		Status:                model.QuotaUsageStatusCommitted,
	}
	require.NoError(t, model.DB.Create(&usage).Error)
	return usage
}
