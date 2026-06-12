package model

import (
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
)

const (
	InvoiceProfileTypeCompany  = "COMPANY"
	InvoiceProfileTypePersonal = "PERSONAL"

	InvoiceProfileStatusActive   = "ACTIVE"
	InvoiceProfileStatusDisabled = "DISABLED"
)

const (
	InvoiceTypeVATNormal  = "VAT_NORMAL"
	InvoiceTypeVATSpecial = "VAT_SPECIAL"

	InvoiceStatusPending  = "PENDING"
	InvoiceStatusApproved = "APPROVED"
	InvoiceStatusRejected = "REJECTED"
	InvoiceStatusIssued   = "ISSUED"
	InvoiceStatusCanceled = "CANCELED"

	InvoiceSourcePaymentOrder  = "PAYMENT_ORDER"
	InvoiceSourceBillingPeriod = "BILLING_PERIOD"
	InvoiceSourceManual        = "MANUAL"
)

const (
	InvoiceFileTypePDF   = "PDF"
	InvoiceFileTypeImage = "IMAGE"
	InvoiceFileTypeOFD   = "OFD"
	InvoiceFileTypeOther = "OTHER"
)

type InvoiceProfile struct {
	Id               int    `json:"id"`
	TenantId         int    `json:"tenant_id" gorm:"index;default:1"`
	OrganizationId   int    `json:"organization_id" gorm:"index;default:0"`
	DepartmentId     int    `json:"department_id" gorm:"index;default:0"`
	UserId           int    `json:"user_id" gorm:"index"`
	ProfileType      string `json:"profile_type" gorm:"type:varchar(32);index;not null"`
	Title            string `json:"title" gorm:"type:varchar(255);not null"`
	TaxNo            string `json:"tax_no" gorm:"type:varchar(128);default:''"`
	BankName         string `json:"bank_name" gorm:"type:varchar(255);default:''"`
	BankAccount      string `json:"bank_account" gorm:"type:varchar(128);default:''"`
	CompanyAddress   string `json:"company_address" gorm:"type:varchar(255);default:''"`
	CompanyPhone     string `json:"company_phone" gorm:"type:varchar(64);default:''"`
	RecipientName    string `json:"recipient_name" gorm:"type:varchar(128);default:''"`
	RecipientPhone   string `json:"recipient_phone" gorm:"type:varchar(64);default:''"`
	RecipientEmail   string `json:"recipient_email" gorm:"type:varchar(128);default:''"`
	RecipientAddress string `json:"recipient_address" gorm:"type:varchar(255);default:''"`
	IsDefault        bool   `json:"is_default" gorm:"index;default:false"`
	Status           string `json:"status" gorm:"type:varchar(32);index;default:'ACTIVE'"`
	CreatedAt        int64  `json:"created_at" gorm:"type:bigint"`
	UpdatedAt        int64  `json:"updated_at" gorm:"type:bigint"`
}

func (p *InvoiceProfile) BeforeCreate(tx *gorm.DB) error {
	p.Normalize()
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *InvoiceProfile) BeforeUpdate(tx *gorm.DB) error {
	p.Normalize()
	p.UpdatedAt = common.GetTimestamp()
	return nil
}

func (p *InvoiceProfile) Normalize() {
	p.ProfileType = NormalizeInvoiceProfileType(p.ProfileType)
	p.Title = strings.TrimSpace(p.Title)
	p.TaxNo = strings.TrimSpace(p.TaxNo)
	p.BankName = strings.TrimSpace(p.BankName)
	p.BankAccount = strings.TrimSpace(p.BankAccount)
	p.CompanyAddress = strings.TrimSpace(p.CompanyAddress)
	p.CompanyPhone = strings.TrimSpace(p.CompanyPhone)
	p.RecipientName = strings.TrimSpace(p.RecipientName)
	p.RecipientPhone = strings.TrimSpace(p.RecipientPhone)
	p.RecipientEmail = strings.TrimSpace(p.RecipientEmail)
	p.RecipientAddress = strings.TrimSpace(p.RecipientAddress)
	p.Status = NormalizeInvoiceProfileStatus(p.Status)
	if p.TenantId == 0 {
		p.TenantId = 1
	}
}

type InvoiceApplication struct {
	Id                    int     `json:"id"`
	ApplicationNo         string  `json:"application_no" gorm:"unique;type:varchar(64);index"`
	TenantId              int     `json:"tenant_id" gorm:"index;default:1"`
	OrganizationId        int     `json:"organization_id" gorm:"index;default:0"`
	DepartmentId          int     `json:"department_id" gorm:"index;default:0"`
	DistributionChannelId int     `json:"distribution_channel_id" gorm:"index;default:0"`
	UserId                int     `json:"user_id" gorm:"index"`
	InvoiceProfileId      int     `json:"invoice_profile_id" gorm:"index"`
	Amount                float64 `json:"amount" gorm:"type:decimal(12,6);not null;default:0"`
	Currency              string  `json:"currency" gorm:"type:varchar(8);not null;default:'USD'"`
	InvoiceType           string  `json:"invoice_type" gorm:"type:varchar(32);index;not null"`
	Status                string  `json:"status" gorm:"type:varchar(32);index;not null;default:'PENDING'"`
	SourceType            string  `json:"source_type" gorm:"type:varchar(32);index;not null"`
	SourceId              int     `json:"source_id" gorm:"index;default:0"`
	ReviewerId            int     `json:"reviewer_id" gorm:"index;default:0"`
	ReviewedAt            int64   `json:"reviewed_at" gorm:"type:bigint;default:0"`
	ReviewNote            string  `json:"review_note" gorm:"type:text"`
	InvoiceNo             string  `json:"invoice_no" gorm:"type:varchar(128);index;default:''"`
	InvoiceDate           int64   `json:"invoice_date" gorm:"type:bigint;default:0"`
	IssuedAt              int64   `json:"issued_at" gorm:"type:bigint;default:0"`
	CreatedAt             int64   `json:"created_at" gorm:"type:bigint"`
	UpdatedAt             int64   `json:"updated_at" gorm:"type:bigint"`
}

func (a *InvoiceApplication) BeforeCreate(tx *gorm.DB) error {
	a.Normalize()
	now := common.GetTimestamp()
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

func (a *InvoiceApplication) BeforeUpdate(tx *gorm.DB) error {
	a.Normalize()
	a.UpdatedAt = common.GetTimestamp()
	return nil
}

func (a *InvoiceApplication) Normalize() {
	a.ApplicationNo = strings.TrimSpace(a.ApplicationNo)
	a.Currency = strings.ToUpper(strings.TrimSpace(a.Currency))
	if a.Currency == "" {
		a.Currency = "USD"
	}
	a.InvoiceType = NormalizeInvoiceType(a.InvoiceType)
	a.Status = NormalizeInvoiceStatus(a.Status)
	a.SourceType = NormalizeInvoiceSourceType(a.SourceType)
	a.ReviewNote = strings.TrimSpace(a.ReviewNote)
	a.InvoiceNo = strings.TrimSpace(a.InvoiceNo)
	if a.TenantId == 0 {
		a.TenantId = 1
	}
}

type InvoiceFile struct {
	Id                   int    `json:"id"`
	InvoiceApplicationId int    `json:"invoice_application_id" gorm:"index"`
	FileName             string `json:"file_name" gorm:"type:varchar(255);not null"`
	FileUrl              string `json:"file_url" gorm:"type:text;not null"`
	FileType             string `json:"file_type" gorm:"type:varchar(32);index;default:'PDF'"`
	UploadedBy           int    `json:"uploaded_by" gorm:"index;default:0"`
	CreatedAt            int64  `json:"created_at" gorm:"type:bigint"`
}

func (f *InvoiceFile) BeforeCreate(tx *gorm.DB) error {
	f.FileName = strings.TrimSpace(f.FileName)
	f.FileUrl = strings.TrimSpace(f.FileUrl)
	f.FileType = NormalizeInvoiceFileType(f.FileType)
	f.CreatedAt = common.GetTimestamp()
	return nil
}

func NormalizeInvoiceProfileType(profileType string) string {
	switch strings.ToUpper(strings.TrimSpace(profileType)) {
	case InvoiceProfileTypePersonal:
		return InvoiceProfileTypePersonal
	default:
		return InvoiceProfileTypeCompany
	}
}

func NormalizeInvoiceProfileStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case InvoiceProfileStatusDisabled:
		return InvoiceProfileStatusDisabled
	default:
		return InvoiceProfileStatusActive
	}
}

func NormalizeInvoiceType(invoiceType string) string {
	switch strings.ToUpper(strings.TrimSpace(invoiceType)) {
	case InvoiceTypeVATSpecial:
		return InvoiceTypeVATSpecial
	default:
		return InvoiceTypeVATNormal
	}
}

func NormalizeInvoiceStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case InvoiceStatusApproved:
		return InvoiceStatusApproved
	case InvoiceStatusRejected:
		return InvoiceStatusRejected
	case InvoiceStatusIssued:
		return InvoiceStatusIssued
	case InvoiceStatusCanceled:
		return InvoiceStatusCanceled
	default:
		return InvoiceStatusPending
	}
}

func NormalizeInvoiceSourceType(sourceType string) string {
	switch strings.ToUpper(strings.TrimSpace(sourceType)) {
	case InvoiceSourceBillingPeriod:
		return InvoiceSourceBillingPeriod
	case InvoiceSourceManual:
		return InvoiceSourceManual
	default:
		return InvoiceSourcePaymentOrder
	}
}

func NormalizeInvoiceFileType(fileType string) string {
	switch strings.ToUpper(strings.TrimSpace(fileType)) {
	case InvoiceFileTypeImage:
		return InvoiceFileTypeImage
	case InvoiceFileTypeOFD:
		return InvoiceFileTypeOFD
	case InvoiceFileTypeOther:
		return InvoiceFileTypeOther
	default:
		return InvoiceFileTypePDF
	}
}
