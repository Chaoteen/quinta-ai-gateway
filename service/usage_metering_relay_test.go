package service

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/dto"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	relaycommon "github.com/Chaoteen/quinta-ai-gateway/relay/common"
	"github.com/Chaoteen/quinta-ai-gateway/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayUsageMeteringDryIntegrationOpenAINonStreamWritesCommittedFact(t *testing.T) {
	truncate(t)
	c := relayUsageMeteringContext("relay-metering-openai")
	info := relayUsageMeteringInfo("relay-metering-openai", constant.ChannelTypeOpenAI, types.RelayFormatOpenAI)

	record, err := CommitRelayUsageFactDryRun(c, info, &dto.Usage{
		PromptTokens:     120,
		CompletionTokens: 30,
		TotalTokens:      150,
	})
	require.NoError(t, err)
	require.Equal(t, model.QuotaUsageStatusCommitted, record.Status)
	require.Equal(t, "relay-metering-openai", record.RequestId)
	require.Equal(t, info.TenantId, record.TenantId)
	require.Equal(t, info.OrganizationId, record.OrganizationId)
	require.Equal(t, info.DepartmentId, record.DepartmentId)
	require.Equal(t, info.DistributionChannelId, record.DistributionChannelId)
	require.Equal(t, info.ChannelId, record.ChannelId)
	require.Equal(t, "openai", record.ProviderName)
	require.Equal(t, UsageSemanticOpenAI, record.UsageSemantic)
	require.Equal(t, UsageSourceUpstream, record.UsageSource)
	require.Equal(t, int64(120), record.InputTokens)
	require.Equal(t, int64(30), record.OutputTokens)
	require.Equal(t, int64(150), record.TotalTokens)
}

func TestRelayUsageMeteringDryIntegrationClaudeNonStreamWritesCommittedFact(t *testing.T) {
	truncate(t)
	c := relayUsageMeteringContext("relay-metering-claude")
	info := relayUsageMeteringInfo("relay-metering-claude", constant.ChannelTypeAnthropic, types.RelayFormatClaude)

	record, err := CommitRelayUsageFactDryRun(c, info, &dto.Usage{
		PromptTokens:     80,
		CompletionTokens: 20,
		UsageSource:      "anthropic",
	})
	require.NoError(t, err)
	require.Equal(t, model.QuotaUsageStatusCommitted, record.Status)
	require.Equal(t, "claude", record.ProviderName)
	require.Equal(t, UsageSemanticAnthropic, record.UsageSemantic)
	require.Equal(t, UsageSourceConverted, record.UsageSource)
	require.Equal(t, int64(80), record.InputTokens)
	require.Equal(t, int64(20), record.OutputTokens)
	require.Equal(t, int64(100), record.TotalTokens)
	require.Contains(t, record.Metadata, "anthropic")
}

func TestRelayUsageMeteringDryIntegrationGeminiNonStreamWritesCommittedFact(t *testing.T) {
	truncate(t)
	c := relayUsageMeteringContext("relay-metering-gemini")
	info := relayUsageMeteringInfo("relay-metering-gemini", constant.ChannelTypeGemini, types.RelayFormatGemini)

	record, err := CommitRelayUsageFactDryRun(c, info, &dto.Usage{
		PromptTokens:     200,
		CompletionTokens: 70,
		TotalTokens:      270,
	})
	require.NoError(t, err)
	require.Equal(t, model.QuotaUsageStatusCommitted, record.Status)
	require.Equal(t, "gemini", record.ProviderName)
	require.Equal(t, UsageSemanticGemini, record.UsageSemantic)
	require.Equal(t, info.DistributionChannelId, record.DistributionChannelId)
	require.Equal(t, int64(200), record.InputTokens)
	require.Equal(t, int64(70), record.OutputTokens)
	require.Equal(t, int64(270), record.TotalTokens)
}

func TestRelayUsageMeteringDryIntegrationIdempotentByRequestId(t *testing.T) {
	truncate(t)
	c := relayUsageMeteringContext("relay-metering-idempotent")
	info := relayUsageMeteringInfo("relay-metering-idempotent", constant.ChannelTypeOpenAI, types.RelayFormatOpenAI)
	usage := &dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}

	first, err := CommitRelayUsageFactDryRun(c, info, usage)
	require.NoError(t, err)
	second, err := CommitRelayUsageFactDryRun(c, info, usage)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)

	var count int64
	require.NoError(t, model.DB.Model(&model.QuotaUsageRecord{}).
		Where("request_id = ? AND status = ?", info.RequestId, model.QuotaUsageStatusCommitted).
		Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestRelayUsageMeteringDryIntegrationFailureDoesNotTouchBillingOrBalances(t *testing.T) {
	truncate(t)
	c := relayUsageMeteringContext("relay-metering-failure")
	billing := &relayUsageMeteringBillingSettler{}
	info := relayUsageMeteringInfo("relay-metering-failure", constant.ChannelTypeOpenAI, types.RelayFormatOpenAI)
	info.Billing = billing
	info.TenantId = 0

	user := model.User{
		Id:       7501,
		TenantId: 7,
		Username: "relay-metering-user-" + time.Now().Format("150405.000000"),
		Quota:    9000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(&user).Error)
	token := model.Token{
		Id:          7502,
		TenantId:    user.TenantId,
		UserId:      user.Id,
		Key:         "relay-metering-token-" + time.Now().Format("150405.000000"),
		Name:        "relay metering token",
		Status:      common.TokenStatusEnabled,
		RemainQuota: 8000,
	}
	require.NoError(t, model.DB.Create(&token).Error)

	TryCommitRelayUsageFactDryRun(c, info, &dto.Usage{PromptTokens: 10, CompletionTokens: 5})

	require.Zero(t, billing.settleCalls)
	require.Zero(t, billing.reserveCalls)
	require.Zero(t, billing.refundCalls)

	var count int64
	require.NoError(t, model.DB.Model(&model.QuotaUsageRecord{}).
		Where("request_id = ?", info.RequestId).
		Count(&count).Error)
	require.Zero(t, count)

	var reloadedUser model.User
	require.NoError(t, model.DB.Select("quota").Where("id = ?", user.Id).First(&reloadedUser).Error)
	require.Equal(t, user.Quota, reloadedUser.Quota)

	var reloadedToken model.Token
	require.NoError(t, model.DB.Select("remain_quota").Where("id = ?", token.Id).First(&reloadedToken).Error)
	require.Equal(t, token.RemainQuota, reloadedToken.RemainQuota)
}

func relayUsageMeteringContext(requestId string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req = req.WithContext(context.WithValue(req.Context(), common.RequestIdKey, requestId))
	c.Request = req
	c.Set(common.RequestIdKey, requestId)
	return c
}

func relayUsageMeteringInfo(requestId string, channelType int, relayFormat types.RelayFormat) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RequestId:               requestId,
		UserId:                  7101,
		TenantId:                7,
		OrganizationId:          8,
		DepartmentId:            9,
		DistributionChannelId:   10,
		OwnershipResolved:       true,
		SubscriptionId:          7201,
		OriginModelName:         "gpt-4o",
		RelayFormat:             relayFormat,
		FinalRequestRelayFormat: relayFormat,
		IsStream:                false,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       channelType,
			ChannelId:         7301,
			UpstreamModelName: "upstream-model",
		},
	}
}

type relayUsageMeteringBillingSettler struct {
	settleCalls  int
	reserveCalls int
	refundCalls  int
}

func (s *relayUsageMeteringBillingSettler) Settle(int) error {
	s.settleCalls++
	return nil
}

func (s *relayUsageMeteringBillingSettler) Refund(*gin.Context) {
	s.refundCalls++
}

func (s *relayUsageMeteringBillingSettler) NeedsRefund() bool {
	return false
}

func (s *relayUsageMeteringBillingSettler) GetPreConsumedQuota() int {
	return 0
}

func (s *relayUsageMeteringBillingSettler) Reserve(int) error {
	s.reserveCalls++
	return nil
}
