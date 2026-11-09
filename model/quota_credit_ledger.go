package model

import (
	"errors"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
)

const (
	QuotaCreditSourcePayment = "payment_order"
	QuotaCreditSourceVoucher = "voucher"
)

var ErrQuotaCreditLedgerImmutable = errors.New("quota credit ledger entries are append-only")

// QuotaCreditLedgerEntry records wallet credits in quota units, not currency or
// model-token counts. Source uniqueness is a second durable idempotency boundary.
// A complete accounting ledger must additionally cover consumption and reversals.
type QuotaCreditLedgerEntry struct {
	Id                    int    `json:"id"`
	TenantId              int    `json:"tenant_id" gorm:"index;not null"`
	OrganizationId        int    `json:"organization_id" gorm:"index;default:0"`
	DepartmentId          int    `json:"department_id" gorm:"index;default:0"`
	DistributionChannelId int    `json:"distribution_channel_id" gorm:"index;default:0"`
	UserId                int    `json:"user_id" gorm:"index;not null"`
	SourceType            string `json:"source_type" gorm:"type:varchar(32);not null;uniqueIndex:uq_quota_credit_source"`
	SourceId              int    `json:"source_id" gorm:"not null;uniqueIndex:uq_quota_credit_source"`
	QuotaDelta            int64  `json:"quota_delta" gorm:"type:bigint;not null"`
	BalanceAfter          int64  `json:"balance_after" gorm:"type:bigint;not null"`
	CreatedAt             int64  `json:"created_at" gorm:"type:bigint;not null"`
}

func (e *QuotaCreditLedgerEntry) BeforeCreate(tx *gorm.DB) error {
	if e.TenantId <= 0 || e.UserId <= 0 || e.SourceId <= 0 || e.QuotaDelta <= 0 ||
		(e.SourceType != QuotaCreditSourcePayment && e.SourceType != QuotaCreditSourceVoucher) {
		return errors.New("invalid quota credit ledger entry")
	}
	e.CreatedAt = common.GetTimestamp()
	return nil
}

func (*QuotaCreditLedgerEntry) BeforeUpdate(tx *gorm.DB) error {
	return ErrQuotaCreditLedgerImmutable
}

func (*QuotaCreditLedgerEntry) BeforeDelete(tx *gorm.DB) error {
	return ErrQuotaCreditLedgerImmutable
}
