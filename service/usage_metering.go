package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/dto"
	"github.com/Chaoteen/quinta-ai-gateway/model"
)

const (
	UsageSourceUpstream  = "upstream"
	UsageSourceEstimated = "estimated"
	UsageSourceConverted = "converted"
	UsageSourceManual    = "manual"

	UsageSemanticOpenAI     = "openai"
	UsageSemanticAnthropic  = "anthropic"
	UsageSemanticGemini     = "gemini"
	UsageSemanticCompatible = "compatible"
	UsageSemanticUnknown    = "unknown"
)

var (
	ErrUsageMeteringInvalid = errors.New("usage metering input is invalid")
)

type UsageMeteringInput struct {
	RequestId             string
	ReservationId         string
	UserId                int
	UserSubscriptionId    int
	TenantId              int
	OrganizationId        int
	DepartmentId          int
	DistributionChannelId int
	ProviderName          string
	ChannelId             int
	ModelName             string
	UpstreamModelName     string
	UsageSource           string
	UsageSemantic         string
	RequestCount          int64
	InputTokens           int64
	OutputTokens          int64
	TotalTokens           int64
	TokenDelta            int64
	RequestDelta          int64
	Metadata              string
}

type UsageMeteringService interface {
	ValidateUsageFact(input UsageMeteringInput) error
	NormalizeUsage(input UsageMeteringInput, usage *dto.Usage) (UsageMeteringInput, error)
	CommitUsageFact(ctx context.Context, input UsageMeteringInput) (model.QuotaUsageRecord, error)
}

type FoundationUsageMeteringService struct{}

func NewFoundationUsageMeteringService() UsageMeteringService {
	return &FoundationUsageMeteringService{}
}

func (s *FoundationUsageMeteringService) ValidateUsageFact(input UsageMeteringInput) error {
	input = normalizeUsageMeteringScalars(input)
	if input.RequestId == "" {
		return errors.Join(ErrUsageMeteringInvalid, errors.New("request_id is required"))
	}
	if input.TenantId <= 0 {
		return errors.Join(ErrUsageMeteringInvalid, errors.New("tenant_id is required"))
	}
	if input.UserId <= 0 {
		return errors.Join(ErrUsageMeteringInvalid, errors.New("user_id is required"))
	}
	if input.InputTokens < 0 || input.OutputTokens < 0 || input.TotalTokens < 0 || input.TokenDelta < 0 || input.RequestDelta < 0 {
		return errors.Join(ErrUsageMeteringInvalid, errors.New("token fields cannot be negative"))
	}
	if input.RequestCount <= 0 {
		return errors.Join(ErrUsageMeteringInvalid, errors.New("request_count must be positive"))
	}
	if !validUsageSource(input.UsageSource) {
		return errors.Join(ErrUsageMeteringInvalid, errors.New("usage_source is invalid"))
	}
	if !validUsageSemantic(input.UsageSemantic) {
		return errors.Join(ErrUsageMeteringInvalid, errors.New("usage_semantic is invalid"))
	}
	return nil
}

func (s *FoundationUsageMeteringService) NormalizeUsage(input UsageMeteringInput, usage *dto.Usage) (UsageMeteringInput, error) {
	input = normalizeUsageMeteringScalars(input)
	if input.RequestCount == 0 {
		input.RequestCount = 1
	}
	if input.RequestDelta == 0 {
		input.RequestDelta = input.RequestCount
	}
	if input.UsageSource == "" {
		input.UsageSource = UsageSourceUpstream
	}
	if input.UsageSemantic == "" {
		input.UsageSemantic = UsageSemanticCompatible
	}
	if input.ProviderName == "" {
		input.ProviderName = providerNameForUsageSemantic(input.UsageSemantic)
	}

	if usage != nil {
		input = normalizeUsageTokens(input, usage)
	}
	if input.TotalTokens == 0 && (input.InputTokens > 0 || input.OutputTokens > 0) {
		input.TotalTokens = input.InputTokens + input.OutputTokens
	}
	if input.TokenDelta == 0 {
		input.TokenDelta = input.TotalTokens
	}
	if input.UpstreamModelName == "" {
		input.UpstreamModelName = input.ModelName
	}
	return input, s.ValidateUsageFact(input)
}

func (s *FoundationUsageMeteringService) CommitUsageFact(ctx context.Context, input UsageMeteringInput) (model.QuotaUsageRecord, error) {
	normalized, err := s.NormalizeUsage(input, nil)
	if err != nil {
		return model.QuotaUsageRecord{}, err
	}
	record := model.QuotaUsageRecord{
		TenantId:              normalized.TenantId,
		OrganizationId:        normalized.OrganizationId,
		DepartmentId:          normalized.DepartmentId,
		DistributionChannelId: normalized.DistributionChannelId,
		UserId:                normalized.UserId,
		UserSubscriptionId:    normalized.UserSubscriptionId,
		RequestId:             normalized.RequestId,
		ReservationId:         normalized.ReservationId,
		ProviderName:          normalized.ProviderName,
		ChannelId:             normalized.ChannelId,
		ModelName:             normalized.ModelName,
		UpstreamModelName:     normalized.UpstreamModelName,
		QuotaDimension:        model.QuotaDimensionToken,
		RequestCount:          normalized.RequestCount,
		InputTokens:           normalized.InputTokens,
		OutputTokens:          normalized.OutputTokens,
		TotalTokens:           normalized.TotalTokens,
		TokenDelta:            normalized.TokenDelta,
		RequestDelta:          normalized.RequestDelta,
		UsageSource:           normalized.UsageSource,
		UsageSemantic:         normalized.UsageSemantic,
		Status:                model.QuotaUsageStatusCommitted,
		Metadata:              normalized.Metadata,
	}
	if err := model.DB.WithContext(ctx).Create(&record).Error; err != nil {
		return model.QuotaUsageRecord{}, err
	}
	return record, nil
}

func normalizeUsageMeteringScalars(input UsageMeteringInput) UsageMeteringInput {
	input.RequestId = strings.TrimSpace(input.RequestId)
	input.ReservationId = strings.TrimSpace(input.ReservationId)
	input.ProviderName = strings.TrimSpace(input.ProviderName)
	input.ModelName = strings.TrimSpace(input.ModelName)
	input.UpstreamModelName = strings.TrimSpace(input.UpstreamModelName)
	input.UsageSource = strings.ToLower(strings.TrimSpace(input.UsageSource))
	input.UsageSemantic = strings.ToLower(strings.TrimSpace(input.UsageSemantic))
	input.Metadata = strings.TrimSpace(input.Metadata)
	return input
}

func normalizeUsageTokens(input UsageMeteringInput, usage *dto.Usage) UsageMeteringInput {
	switch input.UsageSemantic {
	case UsageSemanticOpenAI, UsageSemanticCompatible:
		input.InputTokens = firstPositiveInt64(input.InputTokens, int64(usage.InputTokens), int64(usage.PromptTokens))
		input.OutputTokens = firstPositiveInt64(input.OutputTokens, int64(usage.OutputTokens), int64(usage.CompletionTokens))
		input.TotalTokens = firstPositiveInt64(input.TotalTokens, int64(usage.TotalTokens), input.InputTokens+input.OutputTokens)
	case UsageSemanticAnthropic:
		input.InputTokens = firstPositiveInt64(input.InputTokens, int64(usage.InputTokens), int64(usage.PromptTokens))
		input.OutputTokens = firstPositiveInt64(input.OutputTokens, int64(usage.OutputTokens), int64(usage.CompletionTokens))
		input.TotalTokens = firstPositiveInt64(input.TotalTokens, int64(usage.TotalTokens), input.InputTokens+input.OutputTokens)
	case UsageSemanticGemini:
		input.InputTokens = firstPositiveInt64(input.InputTokens, int64(usage.PromptTokens))
		input.OutputTokens = firstPositiveInt64(input.OutputTokens, int64(usage.CompletionTokens))
		input.TotalTokens = firstPositiveInt64(input.TotalTokens, int64(usage.TotalTokens), input.InputTokens+input.OutputTokens)
	default:
		input.InputTokens = firstPositiveInt64(input.InputTokens, int64(usage.InputTokens), int64(usage.PromptTokens))
		input.OutputTokens = firstPositiveInt64(input.OutputTokens, int64(usage.OutputTokens), int64(usage.CompletionTokens))
		input.TotalTokens = firstPositiveInt64(input.TotalTokens, int64(usage.TotalTokens), input.InputTokens+input.OutputTokens)
	}
	return input
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func validUsageSource(source string) bool {
	switch source {
	case UsageSourceUpstream, UsageSourceEstimated, UsageSourceConverted, UsageSourceManual:
		return true
	default:
		return false
	}
}

func validUsageSemantic(semantic string) bool {
	switch semantic {
	case UsageSemanticOpenAI, UsageSemanticAnthropic, UsageSemanticGemini, UsageSemanticCompatible, UsageSemanticUnknown:
		return true
	default:
		return false
	}
}

func providerNameForUsageSemantic(semantic string) string {
	switch semantic {
	case UsageSemanticOpenAI:
		return "openai"
	case UsageSemanticAnthropic:
		return "claude"
	case UsageSemanticGemini:
		return "gemini"
	default:
		return "compatible"
	}
}
