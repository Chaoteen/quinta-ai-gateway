package model

import (
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
)

const (
	RevenueShareRuleScopeGlobal       = "global"
	RevenueShareRuleScopeProvider     = "provider"
	RevenueShareRuleScopeModel        = "model"
	RevenueShareRuleScopeSubscription = "subscription"
	RevenueShareRuleScopeAgent        = "agent"
	RevenueShareRuleScopeSkill        = "skill"
	RevenueShareRuleScopeVideo        = "video"

	RevenueShareSourceBilling      = "billing"
	RevenueShareSourcePayment      = "payment"
	RevenueShareSourceOrder        = "order"
	RevenueShareSourceSubscription = "subscription"
	RevenueShareSourceManual       = "manual"

	RevenueShareStatusPending    = "pending"
	RevenueShareStatusCalculated = "calculated"
	RevenueShareStatusLocked     = "locked"
	RevenueShareStatusSettled    = "settled"
	RevenueShareStatusCancelled  = "cancelled"
)

type RevenueShareRule struct {
	Id                         int     `json:"id"`
	TenantId                   int     `json:"tenant_id" gorm:"index;default:1"`
	DistributionChannelId      int     `json:"distribution_channel_id" gorm:"index;default:0"`
	RuleName                   string  `json:"rule_name" gorm:"type:varchar(128);not null;index"`
	RuleScope                  string  `json:"rule_scope" gorm:"type:varchar(32);not null;index"`
	ProviderName               string  `json:"provider_name" gorm:"type:varchar(64);index;default:''"`
	ModelName                  string  `json:"model_name" gorm:"type:varchar(255);index;default:''"`
	ProductType                string  `json:"product_type" gorm:"type:varchar(64);index;default:''"`
	PlatformShareRate          float64 `json:"platform_share_rate" gorm:"type:decimal(10,6);not null;default:100"`
	MasterDistributorShareRate float64 `json:"master_distributor_share_rate" gorm:"type:decimal(10,6);not null;default:0"`
	DistributorShareRate       float64 `json:"distributor_share_rate" gorm:"type:decimal(10,6);not null;default:0"`
	EffectiveFrom              int64   `json:"effective_from" gorm:"bigint;index;default:0"`
	EffectiveTo                int64   `json:"effective_to" gorm:"bigint;index;default:0"`
	Enabled                    bool    `json:"enabled" gorm:"index;default:true"`
	CreatedAt                  int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt                  int64   `json:"updated_at" gorm:"bigint"`
}

func (rule *RevenueShareRule) BeforeCreate(tx *gorm.DB) error {
	rule.normalize()
	now := common.GetTimestamp()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	return nil
}

func (rule *RevenueShareRule) BeforeUpdate(tx *gorm.DB) error {
	rule.normalize()
	rule.UpdatedAt = common.GetTimestamp()
	return nil
}

func (rule *RevenueShareRule) normalize() {
	rule.RuleName = strings.TrimSpace(rule.RuleName)
	rule.RuleScope = NormalizeRevenueShareRuleScope(rule.RuleScope)
	rule.ProviderName = strings.TrimSpace(rule.ProviderName)
	rule.ModelName = strings.TrimSpace(rule.ModelName)
	rule.ProductType = strings.TrimSpace(rule.ProductType)
	if rule.TenantId == 0 {
		rule.TenantId = 1
	}
}

type RevenueShareRecord struct {
	Id                      int     `json:"id"`
	TenantId                int     `json:"tenant_id" gorm:"index;default:1"`
	BillingRecordId         int     `json:"billing_record_id" gorm:"index;default:0"`
	SourceType              string  `json:"source_type" gorm:"type:varchar(32);index;not null"`
	SourceId                int     `json:"source_id" gorm:"index;default:0"`
	DistributionChannelId   int     `json:"distribution_channel_id" gorm:"index;default:0"`
	MasterDistributorId     int     `json:"master_distributor_id" gorm:"index;default:0"`
	DistributorId           int     `json:"distributor_id" gorm:"index;default:0"`
	GrossAmount             float64 `json:"gross_amount" gorm:"type:decimal(20,6);not null;default:0"`
	PlatformAmount          float64 `json:"platform_amount" gorm:"type:decimal(20,6);not null;default:0"`
	MasterDistributorAmount float64 `json:"master_distributor_amount" gorm:"type:decimal(20,6);not null;default:0"`
	DistributorAmount       float64 `json:"distributor_amount" gorm:"type:decimal(20,6);not null;default:0"`
	Currency                string  `json:"currency" gorm:"type:varchar(16);default:''"`
	ShareRuleId             int     `json:"share_rule_id" gorm:"index;default:0"`
	Status                  string  `json:"status" gorm:"type:varchar(32);index;default:'pending'"`
	CalculatedAt            int64   `json:"calculated_at" gorm:"bigint;index;default:0"`
	SettledAt               int64   `json:"settled_at" gorm:"bigint;index;default:0"`
	CreatedAt               int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt               int64   `json:"updated_at" gorm:"bigint"`
}

func (record *RevenueShareRecord) BeforeCreate(tx *gorm.DB) error {
	record.normalize()
	now := common.GetTimestamp()
	record.CreatedAt = now
	record.UpdatedAt = now
	return nil
}

func (record *RevenueShareRecord) BeforeUpdate(tx *gorm.DB) error {
	record.normalize()
	record.UpdatedAt = common.GetTimestamp()
	return nil
}

func (record *RevenueShareRecord) normalize() {
	record.SourceType = NormalizeRevenueShareSourceType(record.SourceType)
	record.Currency = strings.TrimSpace(record.Currency)
	record.Status = NormalizeRevenueShareStatus(record.Status)
	if record.TenantId == 0 {
		record.TenantId = 1
	}
}

func NormalizeRevenueShareRuleScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case RevenueShareRuleScopeProvider:
		return RevenueShareRuleScopeProvider
	case RevenueShareRuleScopeModel:
		return RevenueShareRuleScopeModel
	case RevenueShareRuleScopeSubscription:
		return RevenueShareRuleScopeSubscription
	case RevenueShareRuleScopeAgent:
		return RevenueShareRuleScopeAgent
	case RevenueShareRuleScopeSkill:
		return RevenueShareRuleScopeSkill
	case RevenueShareRuleScopeVideo:
		return RevenueShareRuleScopeVideo
	default:
		return RevenueShareRuleScopeGlobal
	}
}

func NormalizeRevenueShareSourceType(sourceType string) string {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case RevenueShareSourcePayment:
		return RevenueShareSourcePayment
	case RevenueShareSourceOrder:
		return RevenueShareSourceOrder
	case RevenueShareSourceSubscription:
		return RevenueShareSourceSubscription
	case RevenueShareSourceManual:
		return RevenueShareSourceManual
	default:
		return RevenueShareSourceBilling
	}
}

func NormalizeRevenueShareStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case RevenueShareStatusCalculated:
		return RevenueShareStatusCalculated
	case RevenueShareStatusLocked:
		return RevenueShareStatusLocked
	case RevenueShareStatusSettled:
		return RevenueShareStatusSettled
	case RevenueShareStatusCancelled:
		return RevenueShareStatusCancelled
	default:
		return RevenueShareStatusPending
	}
}
