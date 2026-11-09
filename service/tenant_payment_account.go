package service

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrTenantPaymentAccount      = errors.New("no unique active receiving account for this tenant, provider and currency")
	ErrTenantFundingPolicy       = errors.New("tenant funding policy is not configured")
	ErrTenantPaymentBinding      = errors.New("payment event does not match the order receiving account")
	ErrTenantSubscriptionPricing = errors.New("independent tenant subscriptions require tenant-owned product pricing")
	ErrBankReviewAuthority       = errors.New("bank approval requires an authorized different reviewer in the receiving tenant")
)

func independentTenantPayments() bool {
	return productionOutboundSecurity() || common.GetEnvOrDefaultBool("TENANT_INDEPENDENT_PAYMENTS", false)
}

// TenantIndependentCollectionEnabled also guards legacy global-merchant entrypoints.
func TenantIndependentCollectionEnabled() bool { return independentTenantPayments() }

func bindTenantPaymentAccount(ctx context.Context, tx *gorm.DB, order *model.PaymentOrder, accountId int) error {
	if accountId == 0 && !independentTenantPayments() {
		return nil
	}
	if accountId < 0 || order.TenantId <= 0 || order.Provider == model.PaymentProviderMock {
		return ErrTenantPaymentAccount
	}
	order.Currency = strings.ToUpper(strings.TrimSpace(order.Currency))
	if order.Currency == "" {
		order.Currency = "USD"
	}
	query := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND provider = ? AND currency = ? AND status = ?",
		order.TenantId, order.Provider, order.Currency, model.TenantPaymentAccountActive)
	if accountId > 0 {
		query = query.Where("id = ?", accountId)
	}
	var accounts []model.TenantPaymentAccount
	if err := query.Limit(2).Find(&accounts).Error; err != nil {
		return err
	}
	if len(accounts) != 1 {
		return ErrTenantPaymentAccount
	}
	account := accounts[0]
	order.PaymentAccountId = account.Id
	order.MerchantIdSnapshot = account.MerchantId
	order.AppIdSnapshot = account.AppId
	return nil
}

func quoteTenantFundingPolicy(ctx context.Context, tx *gorm.DB, order *model.PaymentOrder, amount decimal.Decimal) error {
	if !amount.Equal(amount.Round(2)) {
		return ErrPaymentInvalidAmount
	}
	var policy model.TenantFundingPolicy
	if err := tx.WithContext(ctx).Where("tenant_id = ? AND currency = ?", order.TenantId, order.Currency).
		Order("version DESC").First(&policy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTenantFundingPolicy
		}
		return err
	}
	rate, err := decimal.NewFromString(policy.QuotaUnitsPerCurrencyUnit)
	if err != nil || !rate.IsPositive() {
		return ErrTenantFundingPolicy
	}
	quota := amount.Mul(rate).Truncate(0)
	if quota.LessThan(decimal.NewFromInt(1)) || quota.GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return ErrPaymentInvalidAmount
	}
	snapshot, err := common.Marshal(policy)
	if err != nil {
		return err
	}
	order.FundingPolicyId = policy.Id
	order.FundingPolicySnapshot = string(snapshot)
	order.AmountMinor = amount.Mul(decimal.NewFromInt(100)).IntPart()
	order.QuotaGranted = quota.IntPart()
	order.BusinessId = 0
	return nil
}

func validateTenantPaymentEventTx(tx *gorm.DB, order *model.PaymentOrder, event ConfirmPaymentInput) error {
	if order.PaymentAccountId == 0 {
		if independentTenantPayments() {
			return ErrTenantPaymentBinding
		}
		return nil
	}
	var account model.TenantPaymentAccount
	if err := tx.Where("id = ? AND tenant_id = ?", order.PaymentAccountId, order.TenantId).First(&account).Error; err != nil {
		return ErrTenantPaymentBinding
	}
	// Disabled accounts stop new orders; verified receipts for historical orders
	// still use the original immutable identity, never another active account.
	if account.Provider != order.Provider || account.Currency != order.Currency || account.MerchantId != order.MerchantIdSnapshot || account.AppId != order.AppIdSnapshot ||
		event.TenantId != order.TenantId || event.PaymentAccountId != order.PaymentAccountId || event.MerchantId != order.MerchantIdSnapshot ||
		event.AppId != order.AppIdSnapshot || event.Currency != order.Currency || event.AmountMinor != order.AmountMinor || order.AmountMinor <= 0 {
		return ErrTenantPaymentBinding
	}
	return nil
}

func validateTenantBankReviewerTx(tx *gorm.DB, order *model.PaymentOrder, input ReviewBankTransferInput) error {
	if order.PaymentAccountId == 0 {
		if independentTenantPayments() {
			return ErrTenantPaymentBinding
		}
		return nil
	}
	var reviewer model.User
	if err := tx.Where("id = ?", input.ReviewerId).First(&reviewer).Error; err != nil {
		return ErrBankReviewAuthority
	}
	role := reviewer.NormalizedRoleKey()
	if reviewer.Status != common.UserStatusEnabled || reviewer.Id == order.UserId ||
		(role != common.RoleKeyRoot && (reviewer.TenantId != order.TenantId || (role != common.RoleKeyTenantAdmin && role != common.RoleKeyFinance))) {
		return ErrBankReviewAuthority
	}
	reference := strings.TrimSpace(input.BankTransactionReference)
	if input.Approved && (reference == "" || len(reference) > 128) {
		return errors.New("bank approval requires a statement transaction reference")
	}
	return nil
}
