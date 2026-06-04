package service

import (
	"errors"
	"fmt"

	"github.com/Chaoteen/quinta-ai-gateway/model"
)

var ErrBillingParityInvalidInput = errors.New("billing parity input is invalid")

type BillingCalculationSnapshot struct {
	RequestId             string
	ModelName             string
	ProviderName          string
	UsageRecordId         int
	TenantId              int
	OrganizationId        int
	DepartmentId          int
	DistributionChannelId int
	UserId                int
	UserSubscriptionId    int
	ChannelId             int
	InputTokens           int64
	OutputTokens          int64
	TotalTokens           int64
	CachedTokens          int64
	CacheCreationTokens   int64
	CacheCreation5mTokens int64
	CacheCreation1hTokens int64
	ImageTokens           int64
	AudioTokens           int64
	RequestCount          int64
	ModelRatio            float64
	CompletionRatio       float64
	CacheRatio            float64
	CacheCreationRatio    float64
	ImageRatio            float64
	AudioRatio            float64
	AudioCompletionRatio  float64
	GroupRatio            float64
	ModelPrice            float64
	UsePrice              bool
	QuotaCharged          int64
	Currency              string
	CalculationSource     string
	Metadata              string
}

type BillingCalculationComparison struct {
	Match         bool
	Delta         int64
	Reason        string
	ExpectedQuota int64
	ActualQuota   int64
}

func BuildBillingCalculationSnapshotFromUsage(usage model.QuotaUsageRecord, billing model.BillingRecord) (BillingCalculationSnapshot, error) {
	if err := validateBillingUsageFact(usage); err != nil {
		return BillingCalculationSnapshot{}, err
	}
	if billing.Id <= 0 {
		return BillingCalculationSnapshot{}, errors.Join(ErrBillingParityInvalidInput, errors.New("billing_record_id is required"))
	}
	if billing.UsageRecordId != usage.Id {
		return BillingCalculationSnapshot{}, errors.Join(ErrBillingParityInvalidInput, fmt.Errorf("billing usage_record_id %d does not match usage id %d", billing.UsageRecordId, usage.Id))
	}
	if billing.TenantId <= 0 {
		return BillingCalculationSnapshot{}, errors.Join(ErrBillingParityInvalidInput, errors.New("billing tenant_id is required"))
	}

	requestCount := billing.RequestCount
	if requestCount == 0 {
		requestCount = usage.RequestCount
	}

	return BillingCalculationSnapshot{
		RequestId:             usage.RequestId,
		ModelName:             usage.ModelName,
		ProviderName:          usage.ProviderName,
		UsageRecordId:         usage.Id,
		TenantId:              usage.TenantId,
		OrganizationId:        usage.OrganizationId,
		DepartmentId:          usage.DepartmentId,
		DistributionChannelId: usage.DistributionChannelId,
		UserId:                usage.UserId,
		UserSubscriptionId:    usage.UserSubscriptionId,
		ChannelId:             usage.ChannelId,
		InputTokens:           usage.InputTokens,
		OutputTokens:          usage.OutputTokens,
		TotalTokens:           usage.TotalTokens,
		RequestCount:          requestCount,
		QuotaCharged:          billing.QuotaCharged,
		Currency:              billing.Currency,
		CalculationSource:     "billing_record_shadow",
		Metadata:              billing.Metadata,
	}, nil
}

func CompareBillingCalculationSnapshot(expected BillingCalculationSnapshot, actual BillingCalculationSnapshot) BillingCalculationComparison {
	delta := actual.QuotaCharged - expected.QuotaCharged
	if delta == 0 {
		return BillingCalculationComparison{
			Match:         true,
			Delta:         0,
			Reason:        "quota_match",
			ExpectedQuota: expected.QuotaCharged,
			ActualQuota:   actual.QuotaCharged,
		}
	}
	return BillingCalculationComparison{
		Match:         false,
		Delta:         delta,
		Reason:        "quota_mismatch",
		ExpectedQuota: expected.QuotaCharged,
		ActualQuota:   actual.QuotaCharged,
	}
}
