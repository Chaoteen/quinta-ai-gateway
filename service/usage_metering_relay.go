package service

import (
	"errors"
	"fmt"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/dto"
	"github.com/Chaoteen/quinta-ai-gateway/logger"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	relaycommon "github.com/Chaoteen/quinta-ai-gateway/relay/common"
	"github.com/Chaoteen/quinta-ai-gateway/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var ErrRelayUsageMeteringSkipped = errors.New("relay usage metering skipped")

func TryCommitRelayUsageFactDryRun(c *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) {
	if _, err := CommitRelayUsageFactDryRun(c, relayInfo, usage); err != nil && !errors.Is(err, ErrRelayUsageMeteringSkipped) {
		logger.LogWarn(c.Request.Context(), "usage metering dry integration skipped: "+err.Error())
	}
}

func CommitRelayUsageFactDryRun(c *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) (model.QuotaUsageRecord, error) {
	if usage == nil {
		return model.QuotaUsageRecord{}, errors.Join(ErrRelayUsageMeteringSkipped, errors.New("usage is nil"))
	}
	input, err := BuildRelayUsageMeteringInput(c, relayInfo, usage)
	if err != nil {
		return model.QuotaUsageRecord{}, err
	}

	var existing model.QuotaUsageRecord
	err = model.DB.WithContext(c.Request.Context()).
		Where("request_id = ? AND status = ?", input.RequestId, model.QuotaUsageStatusCommitted).
		First(&existing).Error
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.QuotaUsageRecord{}, err
	}

	return NewFoundationUsageMeteringService().CommitUsageFact(c.Request.Context(), input)
}

func BuildRelayUsageMeteringInput(c *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) (UsageMeteringInput, error) {
	if relayInfo == nil {
		return UsageMeteringInput{}, errors.Join(ErrUsageMeteringInvalid, errors.New("relay_info is required"))
	}
	requestId := relayInfo.RequestId
	if requestId == "" && c != nil {
		requestId = c.GetString(common.RequestIdKey)
	}
	usageSemantic := relayUsageSemantic(relayInfo)
	usageSource, metadata := relayUsageSourceAndMetadata(relayInfo, usage)

	input := UsageMeteringInput{
		RequestId:             requestId,
		UserId:                relayInfo.UserId,
		UserSubscriptionId:    relayInfo.SubscriptionId,
		TenantId:              relayInfo.TenantId,
		OrganizationId:        relayInfo.OrganizationId,
		DepartmentId:          relayInfo.DepartmentId,
		DistributionChannelId: relayInfo.DistributionChannelId,
		ProviderName:          relayProviderName(relayInfo, usageSemantic),
		ChannelId:             relayInfo.ChannelId,
		ModelName:             relayInfo.OriginModelName,
		UpstreamModelName:     relayInfo.UpstreamModelName,
		UsageSource:           usageSource,
		UsageSemantic:         usageSemantic,
		RequestCount:          1,
		RequestDelta:          1,
		Metadata:              metadata,
	}
	return NewFoundationUsageMeteringService().NormalizeUsage(input, usage)
}

func relayUsageSemantic(relayInfo *relaycommon.RelayInfo) string {
	switch relayInfo.GetFinalRequestRelayFormat() {
	case types.RelayFormatClaude:
		return UsageSemanticAnthropic
	case types.RelayFormatGemini:
		return UsageSemanticGemini
	case types.RelayFormatOpenAI, types.RelayFormatOpenAIResponses:
		return UsageSemanticOpenAI
	}
	switch relayInfo.ChannelType {
	case constant.ChannelTypeAnthropic:
		return UsageSemanticAnthropic
	case constant.ChannelTypeGemini, constant.ChannelTypeVertexAi:
		return UsageSemanticGemini
	case constant.ChannelTypeOpenAI, constant.ChannelTypeAzure, constant.ChannelTypeOpenRouter,
		constant.ChannelTypeDeepSeek, constant.ChannelTypeAli, constant.ChannelTypeXai:
		return UsageSemanticOpenAI
	default:
		return UsageSemanticCompatible
	}
}

func relayProviderName(relayInfo *relaycommon.RelayInfo, usageSemantic string) string {
	switch relayInfo.ChannelType {
	case constant.ChannelTypeOpenAI:
		return "openai"
	case constant.ChannelTypeAzure:
		return "azure"
	case constant.ChannelTypeAnthropic:
		return "claude"
	case constant.ChannelTypeGemini:
		return "gemini"
	case constant.ChannelTypeVertexAi:
		return "vertex_ai"
	case constant.ChannelTypeDeepSeek:
		return "deepseek"
	case constant.ChannelTypeAli:
		return "qwen"
	case constant.ChannelTypeOpenRouter:
		return "openrouter"
	case constant.ChannelTypeXai:
		return "xai"
	default:
		return providerNameForUsageSemantic(usageSemantic)
	}
}

func relayUsageSourceAndMetadata(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) (string, string) {
	usageSource := UsageSourceUpstream
	metadata := map[string]any{
		"relay_format":               relayInfo.RelayFormat,
		"final_request_relay_format": relayInfo.GetFinalRequestRelayFormat(),
		"relay_mode":                 relayInfo.RelayMode,
	}
	if usage == nil {
		return usageSource, marshalUsageMeteringMetadata(metadata)
	}
	if validUsageSource(usage.UsageSource) {
		usageSource = usage.UsageSource
	} else if usage.UsageSource != "" {
		usageSource = UsageSourceConverted
		metadata["upstream_usage_source"] = usage.UsageSource
	}
	if usage.UsageSemantic != "" {
		metadata["upstream_usage_semantic"] = usage.UsageSemantic
	}
	return usageSource, marshalUsageMeteringMetadata(metadata)
}

func marshalUsageMeteringMetadata(metadata map[string]any) string {
	data, err := common.Marshal(metadata)
	if err != nil {
		return fmt.Sprintf(`{"marshal_error":%q}`, err.Error())
	}
	return string(data)
}
