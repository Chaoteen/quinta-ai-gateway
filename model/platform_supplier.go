package model

import (
	"errors"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
)

// Procurement records are platform-owned; never expose these through tenant APIs.
type PlatformSupplier struct {
	Id                 int    `json:"id"`
	Provider           string `json:"provider" gorm:"size:32;index"`
	Name               string `json:"name" gorm:"size:128"`
	LegalEntity        string `json:"legal_entity" gorm:"size:200"`
	Contact            string `json:"contact" gorm:"size:200"`
	Mode               string `json:"mode" gorm:"size:24"`
	Status             string `json:"status" gorm:"size:24"`
	Currency           string `json:"currency" gorm:"size:3"`
	AgreementRef       string `json:"agreement_ref" gorm:"size:200"`
	AgreementExpiresAt int64  `json:"agreement_expires_at"`
	ReviewedAt         int64  `json:"reviewed_at"`
	ReviewedBy         int    `json:"reviewed_by"`
	Version            int    `json:"version"`
	CreatedAt          int64  `json:"created_at"`
	UpdatedAt          int64  `json:"updated_at"`
}

type SupplierAccount struct {
	Endpoint   string `json:"endpoint" gorm:"size:512"`
	Protocol   int    `json:"protocol"`
	Id         int    `json:"id"`
	SupplierId int    `json:"supplier_id" gorm:"index"`
	ChannelId  int    `json:"channel_id" gorm:"uniqueIndex"`
	Name       string `json:"name" gorm:"size:128"`
	AccountRef string `json:"account_ref" gorm:"size:128"`
	KeyRef     string `json:"key_ref" gorm:"size:128"`
	Region     string `json:"region" gorm:"size:64"`
	Status     string `json:"status" gorm:"size:24"`
	CreatedAt  int64  `json:"created_at"`
}

type SupplierPrice struct {
	Unit             string `json:"unit" gorm:"size:32"`
	UnitPriceMicros  int64  `json:"unit_price_micros"`
	Id               int    `json:"id"`
	AccountId        int    `json:"account_id" gorm:"uniqueIndex:idx_supplier_price,priority:1"`
	Model            string `json:"model" gorm:"size:128;uniqueIndex:idx_supplier_price,priority:2"`
	EffectiveAt      int64  `json:"effective_at" gorm:"uniqueIndex:idx_supplier_price,priority:3"`
	InputMicros      int64  `json:"input_micros"`
	OutputMicros     int64  `json:"output_micros"`
	CacheReadMicros  int64  `json:"cache_read_micros"`
	CacheWriteMicros int64  `json:"cache_write_micros"`
	CreatedAt        int64  `json:"created_at"`
}

type SupplierStatement struct {
	Id               int    `json:"id"`
	AccountId        int    `json:"account_id" gorm:"uniqueIndex:idx_supplier_statement,priority:1"`
	ExternalRef      string `json:"external_ref" gorm:"size:128;uniqueIndex:idx_supplier_statement,priority:2"`
	StartAt          int64  `json:"start_at"`
	EndAt            int64  `json:"end_at"`
	Currency         string `json:"currency" gorm:"size:3"`
	AmountMinor      int64  `json:"amount_minor"`
	PaidMinor        int64  `json:"paid_minor"`
	InvoiceRef       string `json:"invoice_ref" gorm:"size:128"`
	PlatformInput    int64  `json:"platform_input"`
	PlatformOutput   int64  `json:"platform_output"`
	PlatformRequests int64  `json:"platform_requests"`
	SnapshotAt       int64  `json:"snapshot_at"`
	ReviewNote       string `json:"review_note" gorm:"size:500"`
	ReviewedBy       int    `json:"reviewed_by"`
	ReviewedAt       int64  `json:"reviewed_at"`
	Status           string `json:"status" gorm:"size:24"`
	CreatedAt        int64  `json:"created_at"`
}

type SupplierPayment struct {
	Id          int    `json:"id"`
	StatementId int    `json:"statement_id" gorm:"index"`
	Reference   string `json:"reference" gorm:"size:128;uniqueIndex"`
	AmountMinor int64  `json:"amount_minor"`
	CreatedBy   int    `json:"created_by"`
	CreatedAt   int64  `json:"created_at"`
}

type SupplierAudit struct {
	Id         int    `json:"id"`
	ActorId    int    `json:"actor_id"`
	Action     string `json:"action" gorm:"size:64"`
	ResourceId int    `json:"resource_id"`
	CreatedAt  int64  `json:"created_at"`
}

// CheckSupplierChannelAccess is a live check, so pausing a supplier cannot be
// bypassed by an already cached channel. Existing unbound channels are unchanged.
func CheckSupplierChannelAccess(channel *Channel) error {
	if channel.SupplierAccountId == 0 {
		return nil
	}
	var account SupplierAccount
	if err := DB.Where("id = ? AND channel_id = ?", channel.SupplierAccountId, channel.Id).First(&account).Error; err != nil {
		return err
	}
	var supplier PlatformSupplier
	if err := DB.First(&supplier, account.SupplierId).Error; err != nil {
		return err
	}
	if channel.Type < 1 || channel.Type >= len(constant.ChannelBaseURLs) {
		return errors.New("invalid supplier channel protocol")
	}
	endpoint := constant.ChannelBaseURLs[channel.Type]
	if channel.BaseURL != nil && *channel.BaseURL != "" {
		endpoint = *channel.BaseURL
	}
	if endpoint != account.Endpoint || channel.Type != account.Protocol {
		return errors.New("supplier channel configuration changed; review the binding")
	}
	if account.Status != "active" || supplier.Status != "active" || supplier.ReviewedBy == 0 || supplier.AgreementExpiresAt <= common.GetTimestamp() {
		return errors.New("supplier resource unavailable")
	}
	return nil
}

// SupplierRevision preserves the contract scope approved at each change.
type SupplierRevision struct {
	Id                 int    `json:"id"`
	SupplierId         int    `json:"supplier_id" gorm:"uniqueIndex:idx_supplier_revision,priority:1"`
	Version            int    `json:"version" gorm:"uniqueIndex:idx_supplier_revision,priority:2"`
	Name               string `json:"name" gorm:"size:128"`
	LegalEntity        string `json:"legal_entity" gorm:"size:200"`
	Mode               string `json:"mode" gorm:"size:24"`
	Currency           string `json:"currency" gorm:"size:3"`
	Status             string `json:"status" gorm:"size:24"`
	AgreementRef       string `json:"agreement_ref" gorm:"size:200"`
	AgreementExpiresAt int64  `json:"agreement_expires_at"`
	ActorId            int    `json:"actor_id"`
	CreatedAt          int64  `json:"created_at"`
}
