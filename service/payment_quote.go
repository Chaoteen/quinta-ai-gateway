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
)

const paymentPricingVersion = "server-quote-v1"

// DECIMAL(12,6) leaves six integer digits; bound cents before any DB write.
const maxPaymentAmount = 999999.99

var (
	ErrPaymentUntrustedQuote   = errors.New("payment requires a server-owned quote; recreate unpaid legacy orders")
	ErrPaymentInvalidSignature = errors.New("payment confirmation is not verified")
	ErrPaymentOwnership        = errors.New("payment ownership does not match user")
)

func validPaymentTransferAmount(amount float64) bool {
	return !math.IsNaN(amount) && !math.IsInf(amount, 0) && amount > 0 && amount <= maxPaymentAmount
}

// quotePayment derives entitlements from server data, never from client BusinessId
// for top-ups. This foundation uses USD quota units; other recharge currencies
// require a versioned exchange-rate/product quote before they may be accepted.
func quotePayment(ctx context.Context, tx *gorm.DB, input CreatePaymentOrderInput, order *model.PaymentOrder) error {
	if math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) || input.Amount <= 0 || input.Amount > maxPaymentAmount {
		return ErrPaymentInvalidAmount
	}
	amount := decimal.NewFromFloat(input.Amount)
	if !amount.Equal(amount.Round(6)) {
		return ErrPaymentInvalidAmount
	}
	order.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if order.Currency == "" {
		order.Currency = "USD"
	}
	order.PricingVersion = paymentPricingVersion
	if order.PaymentAccountId > 0 && order.BusinessType != model.PaymentBusinessTokenRecharge {
		return ErrTenantSubscriptionPricing
	}
	if order.BusinessType == model.PaymentBusinessTokenRecharge {
		if order.PaymentAccountId > 0 {
			return quoteTenantFundingPolicy(ctx, tx, order, amount)
		}
		if order.Currency != "USD" || common.QuotaPerUnit <= 0 {
			return ErrPaymentInvalidAmount
		}
		quota := amount.Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Truncate(0)
		if quota.LessThan(decimal.NewFromInt(1)) || quota.GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
			return ErrPaymentInvalidAmount
		}
		order.QuotaGranted = quota.IntPart()
		order.BusinessId = 0
		return nil
	}
	planId := input.BusinessId
	if order.BusinessType == model.PaymentBusinessSubscriptionRenewal {
		var sub model.UserSubscription
		if err := tx.WithContext(ctx).Where("id = ? AND user_id = ? AND tenant_id = ?", input.BusinessId, input.UserId, order.TenantId).First(&sub).Error; err != nil {
			return err
		}
		planId = sub.PlanId
	}
	plan, err := getPaymentSubscriptionPlanTx(tx.WithContext(ctx), planId)
	if err != nil {
		return err
	}
	if !plan.Enabled || plan.Status != model.SubscriptionPlanStatusEnabled || plan.PriceAmount <= 0 || math.IsNaN(plan.PriceAmount) || math.IsInf(plan.PriceAmount, 0) {
		return ErrPaymentInvalidAmount
	}
	if !amount.Equal(decimal.NewFromFloat(plan.PriceAmount)) || order.Currency != strings.ToUpper(plan.Currency) {
		return ErrPaymentInvalidAmount
	}
	order.Amount = plan.PriceAmount
	snapshot, err := common.Marshal(plan)
	if err != nil {
		return err
	}
	order.PlanSnapshot = string(snapshot)
	return nil
}

// The balance bound is checked in the same write that grants quota.
func grantQuotedQuotaTx(tx *gorm.DB, order *model.PaymentOrder) error {
	return grantQuotaTx(tx, order.UserId, order.TenantId, order.QuotaGranted, model.QuotaCreditSourcePayment, order.Id)
}

func grantQuotaTx(tx *gorm.DB, userId int, tenantId int, quota int64, sourceType string, sourceId int) error {
	if quota <= 0 || quota > int64(math.MaxInt) {
		return ErrPaymentUntrustedQuote
	}
	result := tx.Model(&model.User{}).
		Where("id = ? AND tenant_id = ? AND quota <= ?", userId, tenantId, int64(math.MaxInt)-quota).
		Update("quota", gorm.Expr("quota + ?", quota))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrPaymentOwnership
	}
	var user model.User
	if err := tx.Select("quota", "tenant_id", "organization_id", "department_id", "distribution_channel_id").
		Where("id = ? AND tenant_id = ?", userId, tenantId).First(&user).Error; err != nil {
		return err
	}
	entry := model.QuotaCreditLedgerEntry{
		TenantId: tenantId, OrganizationId: user.OrganizationId, DepartmentId: user.DepartmentId,
		DistributionChannelId: user.DistributionChannelId, UserId: userId,
		SourceType: sourceType, SourceId: sourceId, QuotaDelta: quota, BalanceAfter: int64(user.Quota),
	}
	return tx.Create(&entry).Error
}
