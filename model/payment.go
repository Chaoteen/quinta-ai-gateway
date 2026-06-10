package model

import (
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
)

const (
	PaymentProviderMock         = "MOCK"
	PaymentProviderWechatPay    = "WECHAT_PAY"
	PaymentProviderAlipay       = "ALIPAY"
	PaymentProviderBankTransfer = "BANK_TRANSFER"
)

const (
	PaymentBusinessTokenRecharge        = "TOKEN_RECHARGE"
	PaymentBusinessSubscriptionPurchase = "SUBSCRIPTION_PURCHASE"
	PaymentBusinessSubscriptionRenewal  = "SUBSCRIPTION_RENEWAL"
)

const (
	PaymentOrderStatusPending  = "PENDING"
	PaymentOrderStatusPaid     = "PAID"
	PaymentOrderStatusFailed   = "FAILED"
	PaymentOrderStatusCanceled = "CANCELED"
	PaymentOrderStatusExpired  = "EXPIRED"
	PaymentOrderStatusRefunded = "REFUNDED"
)

const (
	PaymentCallbackProcessSuccess = "SUCCESS"
	PaymentCallbackProcessIgnored = "IGNORED"
	PaymentCallbackProcessFailed  = "FAILED"

	PaymentFulfillmentPending = "PENDING"
	PaymentFulfillmentSuccess = "SUCCESS"
	PaymentFulfillmentFailed  = "FAILED"
)

const (
	BankTransferReviewPending  = "PENDING"
	BankTransferReviewApproved = "APPROVED"
	BankTransferReviewRejected = "REJECTED"
)

type PaymentOrder struct {
	Id                    int     `json:"id"`
	OrderNo               string  `json:"order_no" gorm:"unique;type:varchar(64);index"`
	TenantId              int     `json:"tenant_id" gorm:"index;default:1"`
	OrganizationId        int     `json:"organization_id" gorm:"index;default:0"`
	DepartmentId          int     `json:"department_id" gorm:"index;default:0"`
	DistributionChannelId int     `json:"distribution_channel_id" gorm:"index;default:0"`
	UserId                int     `json:"user_id" gorm:"index"`
	Provider              string  `json:"provider" gorm:"type:varchar(32);index;not null"`
	BusinessType          string  `json:"business_type" gorm:"type:varchar(64);index;not null"`
	BusinessId            int     `json:"business_id" gorm:"index;default:0"`
	Amount                float64 `json:"amount" gorm:"type:decimal(12,6);not null;default:0"`
	Currency              string  `json:"currency" gorm:"type:varchar(8);not null;default:'USD'"`
	Status                string  `json:"status" gorm:"type:varchar(32);index;not null;default:'PENDING'"`
	Subject               string  `json:"subject" gorm:"type:varchar(255);default:''"`
	Description           string  `json:"description" gorm:"type:text"`
	PaidAt                int64   `json:"paid_at" gorm:"type:bigint;default:0"`
	ExpiredAt             int64   `json:"expired_at" gorm:"type:bigint;default:0;index"`
	FulfillmentStatus     string  `json:"fulfillment_status" gorm:"type:varchar(32);index;default:'PENDING'"`
	FulfillmentMessage    string  `json:"fulfillment_message" gorm:"type:text"`
	FulfilledAt           int64   `json:"fulfilled_at" gorm:"type:bigint;default:0"`
	CreatedAt             int64   `json:"created_at" gorm:"type:bigint"`
	UpdatedAt             int64   `json:"updated_at" gorm:"type:bigint"`
}

func (o *PaymentOrder) BeforeCreate(tx *gorm.DB) error {
	o.Normalize()
	now := common.GetTimestamp()
	o.CreatedAt = now
	o.UpdatedAt = now
	return nil
}

func (o *PaymentOrder) BeforeUpdate(tx *gorm.DB) error {
	o.Normalize()
	o.UpdatedAt = common.GetTimestamp()
	return nil
}

func (o *PaymentOrder) Normalize() {
	o.OrderNo = strings.TrimSpace(o.OrderNo)
	o.Provider = NormalizePaymentProvider(o.Provider)
	o.BusinessType = NormalizePaymentBusinessType(o.BusinessType)
	o.Status = NormalizePaymentOrderStatus(o.Status)
	o.Currency = strings.ToUpper(strings.TrimSpace(o.Currency))
	if o.Currency == "" {
		o.Currency = "USD"
	}
	o.Subject = strings.TrimSpace(o.Subject)
	o.Description = strings.TrimSpace(o.Description)
	o.FulfillmentStatus = NormalizePaymentFulfillmentStatus(o.FulfillmentStatus)
	o.FulfillmentMessage = strings.TrimSpace(o.FulfillmentMessage)
	if o.TenantId == 0 {
		o.TenantId = 1
	}
}

type PaymentCallbackLog struct {
	Id             int    `json:"id"`
	PaymentOrderId int    `json:"payment_order_id" gorm:"index"`
	OrderNo        string `json:"order_no" gorm:"type:varchar(64);index"`
	Provider       string `json:"provider" gorm:"type:varchar(32);index"`
	EventType      string `json:"event_type" gorm:"type:varchar(64);index"`
	RawPayload     string `json:"raw_payload" gorm:"type:text"`
	SignatureValid bool   `json:"signature_valid" gorm:"default:false"`
	ProcessStatus  string `json:"process_status" gorm:"type:varchar(32);index"`
	ProcessMessage string `json:"process_message" gorm:"type:text"`
	CreatedAt      int64  `json:"created_at" gorm:"type:bigint"`
}

func (l *PaymentCallbackLog) BeforeCreate(tx *gorm.DB) error {
	l.OrderNo = strings.TrimSpace(l.OrderNo)
	l.Provider = NormalizePaymentProvider(l.Provider)
	l.EventType = strings.TrimSpace(l.EventType)
	l.ProcessStatus = strings.TrimSpace(l.ProcessStatus)
	l.ProcessMessage = strings.TrimSpace(l.ProcessMessage)
	l.CreatedAt = common.GetTimestamp()
	return nil
}

type BankTransferRecord struct {
	Id                  int     `json:"id"`
	PaymentOrderId      int     `json:"payment_order_id" gorm:"index"`
	TenantId            int     `json:"tenant_id" gorm:"index;default:1"`
	UserId              int     `json:"user_id" gorm:"index"`
	BankAccountName     string  `json:"bank_account_name" gorm:"type:varchar(128);not null"`
	BankAccountNoMasked string  `json:"bank_account_no_masked" gorm:"type:varchar(64);default:''"`
	TransferAmount      float64 `json:"transfer_amount" gorm:"type:decimal(12,6);not null;default:0"`
	TransferTime        int64   `json:"transfer_time" gorm:"type:bigint;default:0"`
	ProofUrl            string  `json:"proof_url" gorm:"type:text"`
	ReviewStatus        string  `json:"review_status" gorm:"type:varchar(32);index;default:'PENDING'"`
	ReviewedBy          int     `json:"reviewed_by" gorm:"index;default:0"`
	ReviewedAt          int64   `json:"reviewed_at" gorm:"type:bigint;default:0"`
	ReviewNote          string  `json:"review_note" gorm:"type:text"`
	CreatedAt           int64   `json:"created_at" gorm:"type:bigint"`
	UpdatedAt           int64   `json:"updated_at" gorm:"type:bigint"`
}

func (r *BankTransferRecord) BeforeCreate(tx *gorm.DB) error {
	r.Normalize()
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *BankTransferRecord) BeforeUpdate(tx *gorm.DB) error {
	r.Normalize()
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

func (r *BankTransferRecord) Normalize() {
	r.BankAccountName = strings.TrimSpace(r.BankAccountName)
	r.BankAccountNoMasked = strings.TrimSpace(r.BankAccountNoMasked)
	r.ProofUrl = strings.TrimSpace(r.ProofUrl)
	r.ReviewStatus = NormalizeBankTransferReviewStatus(r.ReviewStatus)
	r.ReviewNote = strings.TrimSpace(r.ReviewNote)
	if r.TenantId == 0 {
		r.TenantId = 1
	}
}

func NormalizePaymentProvider(provider string) string {
	switch strings.ToUpper(strings.TrimSpace(provider)) {
	case PaymentProviderWechatPay:
		return PaymentProviderWechatPay
	case PaymentProviderAlipay:
		return PaymentProviderAlipay
	case PaymentProviderBankTransfer:
		return PaymentProviderBankTransfer
	default:
		return PaymentProviderMock
	}
}

func NormalizePaymentBusinessType(businessType string) string {
	switch strings.ToUpper(strings.TrimSpace(businessType)) {
	case PaymentBusinessSubscriptionPurchase:
		return PaymentBusinessSubscriptionPurchase
	case PaymentBusinessSubscriptionRenewal:
		return PaymentBusinessSubscriptionRenewal
	default:
		return PaymentBusinessTokenRecharge
	}
}

func NormalizePaymentOrderStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case PaymentOrderStatusPaid:
		return PaymentOrderStatusPaid
	case PaymentOrderStatusFailed:
		return PaymentOrderStatusFailed
	case PaymentOrderStatusCanceled:
		return PaymentOrderStatusCanceled
	case PaymentOrderStatusExpired:
		return PaymentOrderStatusExpired
	case PaymentOrderStatusRefunded:
		return PaymentOrderStatusRefunded
	default:
		return PaymentOrderStatusPending
	}
}

func NormalizePaymentFulfillmentStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case PaymentFulfillmentSuccess:
		return PaymentFulfillmentSuccess
	case PaymentFulfillmentFailed:
		return PaymentFulfillmentFailed
	default:
		return PaymentFulfillmentPending
	}
}

func NormalizeBankTransferReviewStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case BankTransferReviewApproved:
		return BankTransferReviewApproved
	case BankTransferReviewRejected:
		return BankTransferReviewRejected
	default:
		return BankTransferReviewPending
	}
}
