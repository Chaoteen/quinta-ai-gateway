package model

import (
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
)

const (
	VoucherTypeToken        = "TOKEN"
	VoucherTypeSubscription = "SUBSCRIPTION"
)

const (
	VoucherBatchStatusDraft    = "DRAFT"
	VoucherBatchStatusActive   = "ACTIVE"
	VoucherBatchStatusDisabled = "DISABLED"
	VoucherBatchStatusFinished = "FINISHED"
)

const (
	VoucherStatusUnused   = "UNUSED"
	VoucherStatusRedeemed = "REDEEMED"
	VoucherStatusExpired  = "EXPIRED"
	VoucherStatusDisabled = "DISABLED"
)

const (
	VoucherRedemptionResultSuccess = "SUCCESS"
	VoucherRedemptionResultIgnored = "IGNORED"
	VoucherRedemptionResultFailed  = "FAILED"
)

type VoucherBatch struct {
	Id                    int    `json:"id"`
	BatchNo               string `json:"batch_no" gorm:"unique;type:varchar(64);index"`
	Name                  string `json:"name" gorm:"type:varchar(128);index;not null"`
	Description           string `json:"description" gorm:"type:text"`
	VoucherType           string `json:"voucher_type" gorm:"type:varchar(32);index;not null"`
	Quantity              int    `json:"quantity" gorm:"default:0"`
	Status                string `json:"status" gorm:"type:varchar(32);index;default:'DRAFT'"`
	TenantId              int    `json:"tenant_id" gorm:"index;default:1"`
	OrganizationId        int    `json:"organization_id" gorm:"index;default:0"`
	DepartmentId          int    `json:"department_id" gorm:"index;default:0"`
	DistributionChannelId int    `json:"distribution_channel_id" gorm:"index;default:0"`
	CreatedBy             int    `json:"created_by" gorm:"index;default:0"`
	CreatedAt             int64  `json:"created_at" gorm:"type:bigint"`
	UpdatedAt             int64  `json:"updated_at" gorm:"type:bigint"`
}

func (b *VoucherBatch) BeforeCreate(tx *gorm.DB) error {
	b.Normalize()
	now := common.GetTimestamp()
	b.CreatedAt = now
	b.UpdatedAt = now
	return nil
}

func (b *VoucherBatch) BeforeUpdate(tx *gorm.DB) error {
	b.Normalize()
	b.UpdatedAt = common.GetTimestamp()
	return nil
}

func (b *VoucherBatch) Normalize() {
	b.BatchNo = strings.ToUpper(strings.TrimSpace(b.BatchNo))
	b.Name = strings.TrimSpace(b.Name)
	b.Description = strings.TrimSpace(b.Description)
	b.VoucherType = NormalizeVoucherType(b.VoucherType)
	b.Status = NormalizeVoucherBatchStatus(b.Status)
	if b.TenantId == 0 {
		b.TenantId = 1
	}
}

type Voucher struct {
	Id                 int    `json:"id"`
	BatchId            int    `json:"batch_id" gorm:"index;not null"`
	VoucherCode        string `json:"voucher_code" gorm:"unique;type:varchar(64);index;not null"`
	VoucherType        string `json:"voucher_type" gorm:"type:varchar(32);index;not null"`
	QuotaAmount        int64  `json:"quota_amount" gorm:"default:0"`
	SubscriptionPlanId int    `json:"subscription_plan_id" gorm:"index;default:0"`
	Status             string `json:"status" gorm:"type:varchar(32);index;default:'UNUSED'"`
	ActivatedBy        int    `json:"activated_by" gorm:"index;default:0"`
	ActivatedAt        int64  `json:"activated_at" gorm:"type:bigint;default:0"`
	ExpiredAt          int64  `json:"expired_at" gorm:"type:bigint;default:0;index"`
	CreatedAt          int64  `json:"created_at" gorm:"type:bigint"`
	UpdatedAt          int64  `json:"updated_at" gorm:"type:bigint"`
}

func (v *Voucher) BeforeCreate(tx *gorm.DB) error {
	v.Normalize()
	now := common.GetTimestamp()
	v.CreatedAt = now
	v.UpdatedAt = now
	return nil
}

func (v *Voucher) BeforeUpdate(tx *gorm.DB) error {
	v.Normalize()
	v.UpdatedAt = common.GetTimestamp()
	return nil
}

func (v *Voucher) Normalize() {
	v.VoucherCode = NormalizeVoucherCode(v.VoucherCode)
	v.VoucherType = NormalizeVoucherType(v.VoucherType)
	v.Status = NormalizeVoucherStatus(v.Status)
}

type VoucherRedemption struct {
	Id                    int    `json:"id"`
	VoucherId             int    `json:"voucher_id" gorm:"index"`
	VoucherCode           string `json:"voucher_code" gorm:"type:varchar(64);index"`
	UserId                int    `json:"user_id" gorm:"index"`
	TenantId              int    `json:"tenant_id" gorm:"index;default:1"`
	OrganizationId        int    `json:"organization_id" gorm:"index;default:0"`
	DepartmentId          int    `json:"department_id" gorm:"index;default:0"`
	DistributionChannelId int    `json:"distribution_channel_id" gorm:"index;default:0"`
	RedemptionType        string `json:"redemption_type" gorm:"type:varchar(32);index"`
	RedemptionResult      string `json:"redemption_result" gorm:"type:varchar(32);index"`
	CreatedAt             int64  `json:"created_at" gorm:"type:bigint"`
}

func (r *VoucherRedemption) BeforeCreate(tx *gorm.DB) error {
	r.VoucherCode = NormalizeVoucherCode(r.VoucherCode)
	r.RedemptionType = NormalizeVoucherType(r.RedemptionType)
	r.RedemptionResult = NormalizeVoucherRedemptionResult(r.RedemptionResult)
	if r.TenantId == 0 {
		r.TenantId = 1
	}
	r.CreatedAt = common.GetTimestamp()
	return nil
}

func NormalizeVoucherCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func NormalizeVoucherType(voucherType string) string {
	switch strings.ToUpper(strings.TrimSpace(voucherType)) {
	case VoucherTypeSubscription:
		return VoucherTypeSubscription
	default:
		return VoucherTypeToken
	}
}

func NormalizeVoucherBatchStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case VoucherBatchStatusActive:
		return VoucherBatchStatusActive
	case VoucherBatchStatusDisabled:
		return VoucherBatchStatusDisabled
	case VoucherBatchStatusFinished:
		return VoucherBatchStatusFinished
	default:
		return VoucherBatchStatusDraft
	}
}

func NormalizeVoucherStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case VoucherStatusRedeemed:
		return VoucherStatusRedeemed
	case VoucherStatusExpired:
		return VoucherStatusExpired
	case VoucherStatusDisabled:
		return VoucherStatusDisabled
	default:
		return VoucherStatusUnused
	}
}

func NormalizeVoucherRedemptionResult(result string) string {
	switch strings.ToUpper(strings.TrimSpace(result)) {
	case VoucherRedemptionResultIgnored:
		return VoucherRedemptionResultIgnored
	case VoucherRedemptionResultFailed:
		return VoucherRedemptionResultFailed
	default:
		return VoucherRedemptionResultSuccess
	}
}
