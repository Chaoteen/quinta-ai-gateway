package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/dto"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func TestUsageMeteringNormalizeOpenAIUsage(t *testing.T) {
	service := NewFoundationUsageMeteringService()
	input := baseUsageMeteringInput(6101)
	input.UsageSemantic = UsageSemanticOpenAI
	input = clearUsageMeteringTokens(input)
	input.ProviderName = ""

	normalized, err := service.NormalizeUsage(input, &dto.Usage{
		PromptTokens:     120,
		CompletionTokens: 30,
		TotalTokens:      150,
	})
	require.NoError(t, err)
	require.Equal(t, UsageSourceUpstream, normalized.UsageSource)
	require.Equal(t, UsageSemanticOpenAI, normalized.UsageSemantic)
	require.Equal(t, "openai", normalized.ProviderName)
	require.Equal(t, int64(120), normalized.InputTokens)
	require.Equal(t, int64(30), normalized.OutputTokens)
	require.Equal(t, int64(150), normalized.TotalTokens)
	require.Equal(t, int64(150), normalized.TokenDelta)
	require.Equal(t, int64(1), normalized.RequestCount)
	require.Equal(t, int64(1), normalized.RequestDelta)
}

func TestUsageMeteringNormalizeClaudeUsage(t *testing.T) {
	service := NewFoundationUsageMeteringService()
	input := baseUsageMeteringInput(6102)
	input.UsageSemantic = UsageSemanticAnthropic
	input = clearUsageMeteringTokens(input)
	input.ProviderName = ""

	normalized, err := service.NormalizeUsage(input, &dto.Usage{
		PromptTokens:     80,
		CompletionTokens: 20,
	})
	require.NoError(t, err)
	require.Equal(t, UsageSemanticAnthropic, normalized.UsageSemantic)
	require.Equal(t, "claude", normalized.ProviderName)
	require.Equal(t, int64(80), normalized.InputTokens)
	require.Equal(t, int64(20), normalized.OutputTokens)
	require.Equal(t, int64(100), normalized.TotalTokens)
}

func TestUsageMeteringNormalizeGeminiUsage(t *testing.T) {
	service := NewFoundationUsageMeteringService()
	input := baseUsageMeteringInput(6103)
	input.UsageSemantic = UsageSemanticGemini
	input = clearUsageMeteringTokens(input)
	input.ProviderName = ""

	normalized, err := service.NormalizeUsage(input, &dto.Usage{
		PromptTokens:     200,
		CompletionTokens: 70,
		TotalTokens:      300,
	})
	require.NoError(t, err)
	require.Equal(t, UsageSemanticGemini, normalized.UsageSemantic)
	require.Equal(t, "gemini", normalized.ProviderName)
	require.Equal(t, int64(200), normalized.InputTokens)
	require.Equal(t, int64(70), normalized.OutputTokens)
	require.Equal(t, int64(300), normalized.TotalTokens)
}

func TestUsageMeteringNormalizeEstimatedUsage(t *testing.T) {
	service := NewFoundationUsageMeteringService()
	input := baseUsageMeteringInput(6104)
	input.UsageSource = UsageSourceEstimated
	input.UsageSemantic = UsageSemanticUnknown
	input = clearUsageMeteringTokens(input)
	input.InputTokens = 44
	input.OutputTokens = 11

	normalized, err := service.NormalizeUsage(input, nil)
	require.NoError(t, err)
	require.Equal(t, UsageSourceEstimated, normalized.UsageSource)
	require.Equal(t, UsageSemanticUnknown, normalized.UsageSemantic)
	require.Equal(t, int64(44), normalized.InputTokens)
	require.Equal(t, int64(11), normalized.OutputTokens)
	require.Equal(t, int64(55), normalized.TotalTokens)
	require.Equal(t, int64(55), normalized.TokenDelta)
}

func TestUsageMeteringCommitUsageFactWritesCommittedRecord(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationUsageMeteringService()

	input := baseUsageMeteringInput(6201)
	input = clearUsageMeteringTokens(input)
	input.ChannelId = 88
	input.ProviderName = "openai"
	input.UsageSource = UsageSourceConverted
	input.UsageSemantic = UsageSemanticOpenAI
	input.InputTokens = 100
	input.OutputTokens = 25

	record, err := service.CommitUsageFact(ctx, input)
	require.NoError(t, err)
	require.NotZero(t, record.Id)
	require.Equal(t, model.QuotaUsageStatusCommitted, record.Status)
	require.Equal(t, input.RequestId, record.RequestId)
	require.Equal(t, input.ReservationId, record.ReservationId)
	require.Equal(t, input.TenantId, record.TenantId)
	require.Equal(t, input.OrganizationId, record.OrganizationId)
	require.Equal(t, input.DepartmentId, record.DepartmentId)
	require.Equal(t, input.DistributionChannelId, record.DistributionChannelId)
	require.Equal(t, input.UserId, record.UserId)
	require.Equal(t, input.UserSubscriptionId, record.UserSubscriptionId)
	require.Equal(t, 88, record.ChannelId)
	require.Equal(t, "openai", record.ProviderName)
	require.Equal(t, UsageSourceConverted, record.UsageSource)
	require.Equal(t, UsageSemanticOpenAI, record.UsageSemantic)
	require.Equal(t, int64(100), record.InputTokens)
	require.Equal(t, int64(25), record.OutputTokens)
	require.Equal(t, int64(125), record.TotalTokens)
	require.Equal(t, int64(1), record.RequestCount)
}

func TestUsageMeteringValidationRejectsInvalidInput(t *testing.T) {
	service := NewFoundationUsageMeteringService()

	t.Run("request_id missing", func(t *testing.T) {
		input := baseUsageMeteringInput(6301)
		input.RequestId = ""
		_, err := service.NormalizeUsage(input, nil)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrUsageMeteringInvalid))
	})

	t.Run("tenant ownership missing", func(t *testing.T) {
		input := baseUsageMeteringInput(6302)
		input.TenantId = 0
		_, err := service.NormalizeUsage(input, nil)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrUsageMeteringInvalid))
	})

	t.Run("negative token fields", func(t *testing.T) {
		input := baseUsageMeteringInput(6303)
		input.InputTokens = -1
		_, err := service.NormalizeUsage(input, nil)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrUsageMeteringInvalid))
	})

	t.Run("request_count zero normalizes to one", func(t *testing.T) {
		input := baseUsageMeteringInput(6304)
		input.RequestCount = 0
		normalized, err := service.NormalizeUsage(input, nil)
		require.NoError(t, err)
		require.Equal(t, int64(1), normalized.RequestCount)
	})

	t.Run("invalid usage source", func(t *testing.T) {
		input := baseUsageMeteringInput(6305)
		input.UsageSource = "billing"
		_, err := service.NormalizeUsage(input, nil)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrUsageMeteringInvalid))
	})

	t.Run("invalid usage semantic", func(t *testing.T) {
		input := baseUsageMeteringInput(6306)
		input.UsageSemantic = "provider-specific"
		_, err := service.NormalizeUsage(input, nil)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrUsageMeteringInvalid))
	})
}

func TestUsageMeteringCommitUsageFactDoesNotMutateBillingWalletTokenOrLegacySubscription(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	service := NewFoundationUsageMeteringService()

	user := model.User{
		Id:       6401,
		TenantId: 2,
		Username: "usage-metering-user-" + time.Now().Format("150405.000000"),
		Quota:    9000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(&user).Error)

	token := model.Token{
		Id:          6402,
		TenantId:    user.TenantId,
		UserId:      user.Id,
		Key:         "usage-metering-token-" + time.Now().Format("150405.000000"),
		Name:        "usage metering token",
		Status:      common.TokenStatusEnabled,
		RemainQuota: 8000,
	}
	require.NoError(t, model.DB.Create(&token).Error)

	sub := model.UserSubscription{
		Id:                   6403,
		TenantId:             user.TenantId,
		UserId:               user.Id,
		AmountTotal:          100000,
		AmountUsed:           66,
		StartTime:            time.Now().Add(-time.Hour).Unix(),
		EndTime:              time.Now().Add(24 * time.Hour).Unix(),
		Status:               model.SubscriptionLifecycleActive,
		LifecycleStatus:      model.SubscriptionLifecycleActive,
		TokenQuotaSnapshot:   1000,
		RequestQuotaSnapshot: 10,
	}
	require.NoError(t, model.DB.Create(&sub).Error)

	input := baseUsageMeteringInput(user.Id)
	input.TenantId = user.TenantId
	input.UserSubscriptionId = sub.Id
	input.InputTokens = 50
	input.OutputTokens = 20
	_, err := service.CommitUsageFact(ctx, input)
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

func baseUsageMeteringInput(id int) UsageMeteringInput {
	return UsageMeteringInput{
		RequestId:             "usage-request-" + time.Now().Format("150405.000000"),
		ReservationId:         "usage-reservation-" + time.Now().Format("150405.000000"),
		UserId:                id,
		UserSubscriptionId:    id + 10000,
		TenantId:              1,
		OrganizationId:        2,
		DepartmentId:          3,
		DistributionChannelId: 4,
		ProviderName:          "compatible",
		ChannelId:             99,
		ModelName:             "gpt-4o",
		UpstreamModelName:     "gpt-4o-2024-08-06",
		UsageSource:           UsageSourceUpstream,
		UsageSemantic:         UsageSemanticCompatible,
		RequestCount:          1,
		InputTokens:           10,
		OutputTokens:          5,
		TotalTokens:           15,
		TokenDelta:            15,
		RequestDelta:          1,
		Metadata:              `{"channel_name":"display only"}`,
	}
}

func clearUsageMeteringTokens(input UsageMeteringInput) UsageMeteringInput {
	input.InputTokens = 0
	input.OutputTokens = 0
	input.TotalTokens = 0
	input.TokenDelta = 0
	input.RequestDelta = 0
	return input
}
