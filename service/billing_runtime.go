package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"gorm.io/gorm"
)

var (
	ErrBillingRuntimeInvalidUsage  = errors.New("billing runtime usage fact is invalid")
	ErrBillingRecordNotFound       = errors.New("billing record not found")
	ErrBillingUsageRecordNotFound  = errors.New("billing usage record not found")
	ErrBillingUsageRecordAmbiguous = errors.New("billing usage record is ambiguous")
)

type BillingRuntimeService interface {
	CreateBillingRecordFromUsage(ctx context.Context, usage model.QuotaUsageRecord) (model.BillingRecord, error)
	CreateShadowBillingFromUsageRecordId(ctx context.Context, usageRecordId int) (model.BillingRecord, error)
	CreateShadowBillingFromRequestId(ctx context.Context, requestId string) (model.BillingRecord, error)
	EnsureBillingRecordForUsage(ctx context.Context, usage model.QuotaUsageRecord) (model.BillingRecord, error)
	CalculateCharge(ctx context.Context, usage model.QuotaUsageRecord) (BillingCharge, error)
	GetBillingRecordByRequestId(ctx context.Context, requestId string) (model.BillingRecord, error)
	GetBillingRecordByUsageRecordId(ctx context.Context, usageRecordId int) (model.BillingRecord, error)
}

type BillingCharge struct {
	QuotaCharged      int64
	Currency          string
	UnitPriceSnapshot string
	PriceSnapshot     string
	Metadata          string
}

type FoundationBillingRuntimeService struct{}

func NewFoundationBillingRuntimeService() BillingRuntimeService {
	return &FoundationBillingRuntimeService{}
}

func (s *FoundationBillingRuntimeService) CreateBillingRecordFromUsage(ctx context.Context, usage model.QuotaUsageRecord) (model.BillingRecord, error) {
	return s.EnsureBillingRecordForUsage(ctx, usage)
}

func (s *FoundationBillingRuntimeService) CreateShadowBillingFromUsageRecordId(ctx context.Context, usageRecordId int) (model.BillingRecord, error) {
	if usageRecordId <= 0 {
		return model.BillingRecord{}, errors.Join(ErrBillingUsageRecordNotFound, errors.New("usage_record_id is required"))
	}
	var usage model.QuotaUsageRecord
	if err := model.DB.WithContext(ctx).Where("id = ?", usageRecordId).First(&usage).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.BillingRecord{}, ErrBillingUsageRecordNotFound
		}
		return model.BillingRecord{}, err
	}
	return s.EnsureBillingRecordForUsage(ctx, usage)
}

func (s *FoundationBillingRuntimeService) CreateShadowBillingFromRequestId(ctx context.Context, requestId string) (model.BillingRecord, error) {
	requestId = strings.TrimSpace(requestId)
	if requestId == "" {
		return model.BillingRecord{}, errors.Join(ErrBillingRuntimeInvalidUsage, errors.New("request_id is required"))
	}

	var usages []model.QuotaUsageRecord
	if err := model.DB.WithContext(ctx).
		Where("request_id = ? AND status = ?", requestId, model.QuotaUsageStatusCommitted).
		Order("id asc").
		Find(&usages).Error; err != nil {
		return model.BillingRecord{}, err
	}
	switch len(usages) {
	case 0:
		return model.BillingRecord{}, ErrBillingUsageRecordNotFound
	case 1:
		return s.EnsureBillingRecordForUsage(ctx, usages[0])
	default:
		return model.BillingRecord{}, errors.Join(ErrBillingUsageRecordAmbiguous, errors.New("multiple committed usage records for request_id"))
	}
}

func (s *FoundationBillingRuntimeService) EnsureBillingRecordForUsage(ctx context.Context, usage model.QuotaUsageRecord) (model.BillingRecord, error) {
	if err := validateBillingUsageFact(usage); err != nil {
		return model.BillingRecord{}, err
	}

	var record model.BillingRecord
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.BillingRecord
		err := tx.Where("usage_record_id = ?", usage.Id).First(&existing).Error
		if err == nil {
			record = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		charge, err := s.CalculateCharge(ctx, usage)
		if err != nil {
			return err
		}
		record = model.BillingRecord{
			TenantId:              usage.TenantId,
			OrganizationId:        usage.OrganizationId,
			DepartmentId:          usage.DepartmentId,
			DistributionChannelId: usage.DistributionChannelId,
			RequestId:             usage.RequestId,
			ReservationId:         usage.ReservationId,
			UsageRecordId:         usage.Id,
			UserId:                usage.UserId,
			UserSubscriptionId:    usage.UserSubscriptionId,
			ProviderName:          usage.ProviderName,
			ChannelId:             usage.ChannelId,
			ModelName:             usage.ModelName,
			BillingStatus:         model.BillingStatusPending,
			BillingPhase:          model.BillingPhaseUsageFact,
			Currency:              charge.Currency,
			InputTokens:           usage.InputTokens,
			OutputTokens:          usage.OutputTokens,
			TotalTokens:           usage.TotalTokens,
			RequestCount:          usage.RequestCount,
			QuotaCharged:          charge.QuotaCharged,
			UnitPriceSnapshot:     charge.UnitPriceSnapshot,
			PriceSnapshot:         charge.PriceSnapshot,
			SettledDelta:          0,
			Metadata:              charge.Metadata,
		}
		if err := tx.Create(&record).Error; err != nil {
			var dup model.BillingRecord
			if findErr := tx.Where("usage_record_id = ?", usage.Id).First(&dup).Error; findErr == nil {
				record = dup
				return nil
			}
			return err
		}
		return nil
	})
	if err != nil {
		return model.BillingRecord{}, err
	}
	return record, nil
}

func (s *FoundationBillingRuntimeService) CalculateCharge(ctx context.Context, usage model.QuotaUsageRecord) (BillingCharge, error) {
	_ = ctx
	if err := validateBillingUsageFact(usage); err != nil {
		return BillingCharge{}, err
	}
	metadata := map[string]any{
		"mode":            "shadow",
		"usage_record_id": usage.Id,
		"usage_source":    usage.UsageSource,
		"usage_semantic":  usage.UsageSemantic,
	}
	metadataBytes, err := common.Marshal(metadata)
	if err != nil {
		return BillingCharge{}, err
	}
	quotaCharged := usage.TokenDelta
	if quotaCharged == 0 {
		quotaCharged = usage.TotalTokens
	}
	return BillingCharge{
		QuotaCharged:      quotaCharged,
		Currency:          "QUOTA",
		UnitPriceSnapshot: `{"mode":"shadow"}`,
		PriceSnapshot:     `{"source":"quota_usage_record"}`,
		Metadata:          string(metadataBytes),
	}, nil
}

func (s *FoundationBillingRuntimeService) GetBillingRecordByRequestId(ctx context.Context, requestId string) (model.BillingRecord, error) {
	requestId = strings.TrimSpace(requestId)
	if requestId == "" {
		return model.BillingRecord{}, errors.Join(ErrBillingRecordNotFound, errors.New("request_id is required"))
	}
	var record model.BillingRecord
	if err := model.DB.WithContext(ctx).Where("request_id = ?", requestId).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.BillingRecord{}, ErrBillingRecordNotFound
		}
		return model.BillingRecord{}, err
	}
	return record, nil
}

func (s *FoundationBillingRuntimeService) GetBillingRecordByUsageRecordId(ctx context.Context, usageRecordId int) (model.BillingRecord, error) {
	if usageRecordId <= 0 {
		return model.BillingRecord{}, errors.Join(ErrBillingRecordNotFound, errors.New("usage_record_id is required"))
	}
	var record model.BillingRecord
	if err := model.DB.WithContext(ctx).Where("usage_record_id = ?", usageRecordId).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.BillingRecord{}, ErrBillingRecordNotFound
		}
		return model.BillingRecord{}, err
	}
	return record, nil
}

func validateBillingUsageFact(usage model.QuotaUsageRecord) error {
	if usage.Id <= 0 {
		return errors.Join(ErrBillingRuntimeInvalidUsage, errors.New("usage_record_id is required"))
	}
	if strings.TrimSpace(usage.RequestId) == "" {
		return errors.Join(ErrBillingRuntimeInvalidUsage, errors.New("request_id is required"))
	}
	if usage.TenantId <= 0 {
		return errors.Join(ErrBillingRuntimeInvalidUsage, errors.New("tenant_id is required"))
	}
	if usage.UserId <= 0 {
		return errors.Join(ErrBillingRuntimeInvalidUsage, errors.New("user_id is required"))
	}
	if usage.Status != model.QuotaUsageStatusCommitted {
		return errors.Join(ErrBillingRuntimeInvalidUsage, errors.New("usage record must be committed"))
	}
	return nil
}
