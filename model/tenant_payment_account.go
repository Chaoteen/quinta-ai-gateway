package model

import (
	"errors"
	"regexp"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const TenantPaymentAccountActive = "ACTIVE"
const TenantPaymentAccountDisabled = "DISABLED"

// One receiving identity belongs to exactly one tenant. Credentials live in a
// secret store, not this table. Bank MerchantId is an opaque receiving-account
// identifier, never an account number intended for a public JSON response.
type TenantPaymentAccount struct {
	Id                  int    `json:"id"`
	TenantId            int    `json:"tenant_id" gorm:"not null;index"`
	Provider            string `json:"provider" gorm:"type:varchar(32);not null;uniqueIndex:uq_payment_receiver"`
	MerchantId          string `json:"-" gorm:"type:varchar(128);not null;uniqueIndex:uq_payment_receiver"`
	AppId               string `json:"-" gorm:"type:varchar(128)"`
	CredentialReference string `json:"-" gorm:"type:varchar(512)"`
	Currency            string `json:"currency" gorm:"type:varchar(8);not null"`
	Status              string `json:"status" gorm:"type:varchar(32);not null"`
	CreatedAt           int64  `json:"created_at" gorm:"type:bigint"`
}

func (a *TenantPaymentAccount) BeforeCreate(tx *gorm.DB) error {
	a.Provider = strings.ToUpper(strings.TrimSpace(a.Provider))
	a.MerchantId = strings.TrimSpace(a.MerchantId)
	a.AppId = strings.TrimSpace(a.AppId)
	a.Currency = strings.ToUpper(strings.TrimSpace(a.Currency))
	if a.Status == "" {
		a.Status = TenantPaymentAccountActive
	}
	if a.TenantId <= 0 || a.MerchantId == "" || len(a.MerchantId) > 128 || len(a.AppId) > 128 || len(a.CredentialReference) > 512 || (a.Currency != "USD" && a.Currency != "CNY") {
		return errors.New("invalid tenant payment account")
	}
	if a.Provider != PaymentProviderBankTransfer && a.Provider != PaymentProviderAlipay && a.Provider != PaymentProviderWechatPay {
		return errors.New("unsupported tenant payment account provider")
	}
	if a.Provider != PaymentProviderBankTransfer && (a.AppId == "" || a.CredentialReference == "" || a.Currency != "CNY") {
		return errors.New("online payment account requires application, credential reference and CNY")
	}
	if a.CredentialReference != "" && !strings.HasPrefix(a.CredentialReference, "secret://") && !strings.HasPrefix(a.CredentialReference, "arn:aws:secretsmanager:") {
		return errors.New("credentials must be an opaque secret-store reference")
	}
	if a.Status != TenantPaymentAccountActive && a.Status != TenantPaymentAccountDisabled {
		return errors.New("invalid tenant payment account status")
	}
	a.CreatedAt = common.GetTimestamp()
	return nil
}

func (*TenantPaymentAccount) BeforeUpdate(tx *gorm.DB) error {
	// Historical orders must retain their receiving identity. Configure another
	// account through a reviewed migration; only activation state is mutable.
	if len(tx.Statement.Selects) != 1 || tx.Statement.Selects[0] != "status" {
		return errors.New("payment receiving identity is immutable; only explicit status updates are allowed")
	}
	values, ok := tx.Statement.Dest.(map[string]interface{})
	if !ok || (values["status"] != TenantPaymentAccountActive && values["status"] != TenantPaymentAccountDisabled) {
		return errors.New("invalid tenant payment account status")
	}
	return nil
}

func (*TenantPaymentAccount) BeforeDelete(tx *gorm.DB) error {
	return errors.New("payment receiving identity is immutable")
}

// Pricing policy is separate from the receiving account. Quota is an internal
// entitlement unit, not cash. Policies are versioned append-only records.
type TenantFundingPolicy struct {
	Id                        int    `json:"id"`
	TenantId                  int    `json:"tenant_id" gorm:"not null;uniqueIndex:uq_tenant_funding_policy"`
	Currency                  string `json:"currency" gorm:"type:varchar(8);not null;uniqueIndex:uq_tenant_funding_policy"`
	Version                   int    `json:"version" gorm:"not null;uniqueIndex:uq_tenant_funding_policy"`
	QuotaUnitsPerCurrencyUnit string `json:"quota_units_per_currency_unit" gorm:"type:varchar(64);not null"`
	CreatedAt                 int64  `json:"created_at" gorm:"type:bigint"`
}

var tenantFundingRatePattern = regexp.MustCompile(`^[0-9]{1,10}(\.[0-9]{1,6})?$`)

func (p *TenantFundingPolicy) BeforeCreate(tx *gorm.DB) error {
	p.Currency = strings.ToUpper(strings.TrimSpace(p.Currency))
	if !tenantFundingRatePattern.MatchString(p.QuotaUnitsPerCurrencyUnit) {
		return errors.New("invalid funding rate format")
	}
	rate, err := decimal.NewFromString(p.QuotaUnitsPerCurrencyUnit)
	if err != nil || !rate.IsPositive() || rate.GreaterThan(decimal.NewFromInt(1000000000)) ||
		p.TenantId <= 0 || p.Version <= 0 || (p.Currency != "USD" && p.Currency != "CNY") {
		return errors.New("invalid tenant funding policy")
	}
	p.QuotaUnitsPerCurrencyUnit = rate.String()
	p.CreatedAt = common.GetTimestamp()
	return nil
}

func (*TenantFundingPolicy) BeforeUpdate(tx *gorm.DB) error {
	return errors.New("funding policy is append-only")
}
func (*TenantFundingPolicy) BeforeDelete(tx *gorm.DB) error {
	return errors.New("funding policy is append-only")
}
