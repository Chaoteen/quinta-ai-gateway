package model

import (
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
)

const (
	BillingStatusPending  = "pending"
	BillingStatusSettled  = "settled"
	BillingStatusFailed   = "failed"
	BillingStatusRefunded = "refunded"

	BillingPhaseUsageFact = "usage_fact"
)

// BillingRecord is the durable Billing Runtime foundation fact. It records
// what should be charged from a usage fact, but does not mutate balances.
type BillingRecord struct {
	Id                    int    `json:"id"`
	TenantId              int    `json:"tenant_id" gorm:"index;default:1"`
	OrganizationId        int    `json:"organization_id" gorm:"index;default:0"`
	DepartmentId          int    `json:"department_id" gorm:"index;default:0"`
	DistributionChannelId int    `json:"distribution_channel_id" gorm:"index;default:0"`
	RequestId             string `json:"request_id" gorm:"type:varchar(128);index"`
	ReservationId         string `json:"reservation_id" gorm:"type:varchar(128);index"`
	UsageRecordId         int    `json:"usage_record_id" gorm:"uniqueIndex"`
	UserId                int    `json:"user_id" gorm:"index"`
	TokenId               int    `json:"token_id" gorm:"index;default:0"`
	UserSubscriptionId    int    `json:"user_subscription_id" gorm:"index;default:0"`
	ProviderName          string `json:"provider_name" gorm:"type:varchar(64);index"`
	ChannelId             int    `json:"channel_id" gorm:"index;default:0"`
	ModelName             string `json:"model_name" gorm:"type:varchar(255);index"`
	FundingSource         string `json:"funding_source" gorm:"type:varchar(32);index;default:''"`
	BillingStatus         string `json:"billing_status" gorm:"type:varchar(32);index;default:'pending'"`
	BillingPhase          string `json:"billing_phase" gorm:"type:varchar(32);index;default:'usage_fact'"`
	Currency              string `json:"currency" gorm:"type:varchar(8);default:''"`
	InputTokens           int64  `json:"input_tokens" gorm:"type:bigint;not null;default:0"`
	OutputTokens          int64  `json:"output_tokens" gorm:"type:bigint;not null;default:0"`
	TotalTokens           int64  `json:"total_tokens" gorm:"type:bigint;not null;default:0"`
	RequestCount          int64  `json:"request_count" gorm:"type:bigint;not null;default:0"`
	QuotaCharged          int64  `json:"quota_charged" gorm:"type:bigint;not null;default:0"`
	UnitPriceSnapshot     string `json:"unit_price_snapshot" gorm:"type:text"`
	PriceSnapshot         string `json:"price_snapshot" gorm:"type:text"`
	SettledDelta          int64  `json:"settled_delta" gorm:"type:bigint;not null;default:0"`
	Metadata              string `json:"metadata" gorm:"type:text"`
	CreatedAt             int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt             int64  `json:"updated_at" gorm:"bigint"`
}

func (r *BillingRecord) BeforeCreate(tx *gorm.DB) error {
	r.normalize()
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *BillingRecord) BeforeUpdate(tx *gorm.DB) error {
	r.normalize()
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

func (r *BillingRecord) normalize() {
	r.RequestId = strings.TrimSpace(r.RequestId)
	r.ReservationId = strings.TrimSpace(r.ReservationId)
	r.ProviderName = strings.TrimSpace(r.ProviderName)
	r.ModelName = strings.TrimSpace(r.ModelName)
	r.FundingSource = strings.TrimSpace(r.FundingSource)
	r.BillingStatus = strings.TrimSpace(r.BillingStatus)
	r.BillingPhase = strings.TrimSpace(r.BillingPhase)
	r.Currency = strings.TrimSpace(r.Currency)
	r.UnitPriceSnapshot = strings.TrimSpace(r.UnitPriceSnapshot)
	r.PriceSnapshot = strings.TrimSpace(r.PriceSnapshot)
	r.Metadata = strings.TrimSpace(r.Metadata)
	if r.BillingStatus == "" {
		r.BillingStatus = BillingStatusPending
	}
	if r.BillingPhase == "" {
		r.BillingPhase = BillingPhaseUsageFact
	}
}
