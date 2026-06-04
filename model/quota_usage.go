package model

import (
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
)

const (
	QuotaDimensionToken   = "token"
	QuotaDimensionRequest = "request"
	QuotaDimensionModel   = "model"
	QuotaDimensionReset   = "reset"

	QuotaUsageStatusReserved   = "reserved"
	QuotaUsageStatusCommitted  = "committed"
	QuotaUsageStatusRolledBack = "rolled_back"
	QuotaUsageStatusReset      = "reset"
)

// QuotaUsageRecord is an append-oriented foundation record for quota usage
// state transitions. It is not a billing, payment, or wallet deduction record.
type QuotaUsageRecord struct {
	Id                    int    `json:"id"`
	TenantId              int    `json:"tenant_id" gorm:"index;default:1"`
	OrganizationId        int    `json:"organization_id" gorm:"index;default:0"`
	DepartmentId          int    `json:"department_id" gorm:"index;default:0"`
	DistributionChannelId int    `json:"distribution_channel_id" gorm:"index;default:0"`
	UserId                int    `json:"user_id" gorm:"index"`
	UserSubscriptionId    int    `json:"user_subscription_id" gorm:"index"`
	RequestId             string `json:"request_id" gorm:"type:varchar(128);index"`
	ReservationId         string `json:"reservation_id" gorm:"type:varchar(128);index"`
	ModelName             string `json:"model_name" gorm:"type:varchar(255);index"`
	QuotaDimension        string `json:"quota_dimension" gorm:"type:varchar(32);index"`
	TokenDelta            int64  `json:"token_delta" gorm:"type:bigint;not null;default:0"`
	RequestDelta          int64  `json:"request_delta" gorm:"type:bigint;not null;default:0"`
	Status                string `json:"status" gorm:"type:varchar(32);index"`
	Metadata              string `json:"metadata" gorm:"type:text"`
	OccurredAt            int64  `json:"occurred_at" gorm:"bigint;index"`
	CreatedAt             int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt             int64  `json:"updated_at" gorm:"bigint"`
}

func (r *QuotaUsageRecord) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if r.OccurredAt == 0 {
		r.OccurredAt = now
	}
	r.QuotaDimension = strings.TrimSpace(r.QuotaDimension)
	r.Status = strings.TrimSpace(r.Status)
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *QuotaUsageRecord) BeforeUpdate(tx *gorm.DB) error {
	r.QuotaDimension = strings.TrimSpace(r.QuotaDimension)
	r.Status = strings.TrimSpace(r.Status)
	r.UpdatedAt = common.GetTimestamp()
	return nil
}
