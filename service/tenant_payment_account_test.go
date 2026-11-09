package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func tenantBankFixture(t *testing.T, tenant int, userId int) (model.User, model.User, model.TenantPaymentAccount) {
	t.Helper()
	user := seedPaymentUser(t, userId, tenant, 0)
	reviewer := seedPaymentUser(t, userId+1, tenant, 0)
	require.NoError(t, model.DB.Model(&reviewer).Updates(map[string]interface{}{"role": common.RoleAdminUser, "role_key": common.RoleKeyTenantAdmin}).Error)
	account := model.TenantPaymentAccount{TenantId: tenant, Provider: model.PaymentProviderBankTransfer,
		MerchantId: fmt.Sprintf("bank-receiver-%d", tenant), Currency: "CNY"}
	require.NoError(t, model.DB.Create(&account).Error)
	require.NoError(t, model.DB.Create(&model.TenantFundingPolicy{TenantId: tenant, Currency: "CNY", Version: 1, QuotaUnitsPerCurrencyUnit: "70000"}).Error)
	return user, reviewer, account
}

func tenantBankOrder(t *testing.T, user model.User, accountId int) *model.PaymentOrder {
	t.Helper()
	order, err := NewPaymentGatewayService().CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId: user.Id, PaymentAccountId: accountId, Provider: model.PaymentProviderBankTransfer,
		BusinessType: model.PaymentBusinessTokenRecharge, Currency: "CNY", Amount: 10,
	})
	require.NoError(t, err)
	return order
}

func TestSecurityTenantBankAccountCannotBeBorrowed(t *testing.T) {
	truncate(t)
	t.Setenv("TENANT_INDEPENDENT_PAYMENTS", "true")
	a, _, account := tenantBankFixture(t, 81, 11001)
	b := seedPaymentUser(t, 11003, 82, 0)
	_, err := NewPaymentGatewayService().CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId: b.Id, PaymentAccountId: account.Id, Provider: model.PaymentProviderBankTransfer,
		BusinessType: model.PaymentBusinessTokenRecharge, Currency: "CNY", Amount: 10,
	})
	require.ErrorIs(t, err, ErrTenantPaymentAccount)
	order := tenantBankOrder(t, a, account.Id)
	require.Equal(t, int64(1000), order.AmountMinor)
	require.Equal(t, int64(700000), order.QuotaGranted)
	require.Equal(t, account.MerchantId, order.MerchantIdSnapshot)
	// A later policy version must not alter an existing order.
	require.NoError(t, model.DB.Create(&model.TenantFundingPolicy{TenantId: a.TenantId, Currency: "CNY", Version: 2, QuotaUnitsPerCurrencyUnit: "1"}).Error)
	require.NoError(t, model.DB.First(order, order.Id).Error)
	require.Equal(t, int64(700000), order.QuotaGranted)
}

func TestSecurityTenantBankReviewAndReceiptAreScoped(t *testing.T) {
	truncate(t)
	t.Setenv("TENANT_INDEPENDENT_PAYMENTS", "true")
	user, reviewer, account := tenantBankFixture(t, 83, 11011)
	_, otherReviewer, _ := tenantBankFixture(t, 84, 11013)
	svc := NewPaymentGatewayService()
	order := tenantBankOrder(t, user, account.Id)
	record, err := svc.CreateBankTransferRecord(context.Background(), CreateBankTransferRecordInput{
		PaymentOrderId: order.Id, UserId: user.Id, BankAccountName: "payer", TransferAmount: 10,
	})
	require.NoError(t, err)
	_, err = svc.ReviewBankTransfer(context.Background(), record.Id, ReviewBankTransferInput{ReviewerId: otherReviewer.Id, Approved: true, BankTransactionReference: "statement-001"})
	require.ErrorIs(t, err, ErrBankReviewAuthority)
	_, err = svc.ReviewBankTransfer(context.Background(), record.Id, ReviewBankTransferInput{ReviewerId: otherReviewer.Id, Approved: false})
	require.ErrorIs(t, err, ErrBankReviewAuthority)
	_, err = svc.ReviewBankTransfer(context.Background(), record.Id, ReviewBankTransferInput{ReviewerId: user.Id, Approved: true, BankTransactionReference: "statement-001"})
	require.ErrorIs(t, err, ErrBankReviewAuthority)
	_, err = svc.ReviewBankTransfer(context.Background(), record.Id, ReviewBankTransferInput{ReviewerId: reviewer.Id, Approved: true})
	require.Error(t, err)
	_, err = svc.ReviewBankTransfer(context.Background(), record.Id, ReviewBankTransferInput{ReviewerId: reviewer.Id, Approved: true, BankTransactionReference: "statement-001"})
	require.NoError(t, err)
	require.NoError(t, model.DB.First(&user, user.Id).Error)
	require.Equal(t, 700000, user.Quota)
	second := tenantBankOrder(t, user, account.Id)
	secondRecord, err := svc.CreateBankTransferRecord(context.Background(), CreateBankTransferRecordInput{
		PaymentOrderId: second.Id, UserId: user.Id, BankAccountName: "payer", TransferAmount: 10,
	})
	require.NoError(t, err)
	_, err = svc.ReviewBankTransfer(context.Background(), secondRecord.Id, ReviewBankTransferInput{ReviewerId: reviewer.Id, Approved: true, BankTransactionReference: "statement-001"})
	require.Error(t, err)
	require.NoError(t, model.DB.First(second, second.Id).Error)
	require.Equal(t, model.PaymentOrderStatusPending, second.Status)
	require.NoError(t, model.DB.First(&user, user.Id).Error)
	require.Equal(t, 700000, user.Quota)
}

func TestSecurityTenantPaymentEventBindsReceiverAndAmount(t *testing.T) {
	truncate(t)
	t.Setenv("TENANT_INDEPENDENT_PAYMENTS", "true")
	user, _, account := tenantBankFixture(t, 85, 11021)
	order := tenantBankOrder(t, user, account.Id)
	event := ConfirmPaymentInput{TenantId: user.TenantId, PaymentAccountId: account.Id,
		MerchantId: account.MerchantId, Currency: "CNY", AmountMinor: 1000}
	require.NoError(t, validateTenantPaymentEventTx(model.DB, order, event))
	for _, mutate := range []func(*ConfirmPaymentInput){
		func(e *ConfirmPaymentInput) { e.TenantId++ },
		func(e *ConfirmPaymentInput) { e.PaymentAccountId++ },
		func(e *ConfirmPaymentInput) { e.MerchantId = "other-receiver" },
		func(e *ConfirmPaymentInput) { e.Currency = "USD" },
		func(e *ConfirmPaymentInput) { e.AmountMinor++ },
	} {
		bad := event
		mutate(&bad)
		require.ErrorIs(t, validateTenantPaymentEventTx(model.DB, order, bad), ErrTenantPaymentBinding)
	}
}

func TestSecurityIndependentPaymentNeverFallsBackToGlobal(t *testing.T) {
	truncate(t)
	t.Setenv("TENANT_INDEPENDENT_PAYMENTS", "true")
	user := seedPaymentUser(t, 11031, 86, 0)
	_, err := NewPaymentGatewayService().CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId: user.Id, Provider: model.PaymentProviderBankTransfer,
		BusinessType: model.PaymentBusinessTokenRecharge, Currency: "CNY", Amount: 10,
	})
	require.ErrorIs(t, err, ErrTenantPaymentAccount)
}

func TestSecurityTenantPaymentConfigurationIsImmutableAndScoped(t *testing.T) {
	truncate(t)
	_, _, account := tenantBankFixture(t, 87, 11041)
	duplicate := model.TenantPaymentAccount{TenantId: 88, Provider: account.Provider, MerchantId: account.MerchantId, Currency: account.Currency}
	require.Error(t, model.DB.Create(&duplicate).Error)
	require.Error(t, model.DB.Model(&account).Update("tenant_id", 88).Error)
	require.Error(t, model.DB.Model(&account).Update("merchant_id", "replacement").Error)
	require.NoError(t, model.DB.Model(&account).Select("status").Updates(map[string]interface{}{"status": model.TenantPaymentAccountDisabled}).Error)
	policy := model.TenantFundingPolicy{TenantId: 87, Currency: "CNY", Version: 2, QuotaUnitsPerCurrencyUnit: "1e1000000000"}
	require.Error(t, model.DB.Create(&policy).Error)
	policy.QuotaUnitsPerCurrencyUnit = "100"
	require.NoError(t, model.DB.Create(&policy).Error)
	require.Error(t, model.DB.Model(&policy).Update("quota_units_per_currency_unit", "200").Error)
}

func TestSecurityOnlineProvidersRemainClosedBeforeAdapters(t *testing.T) {
	truncate(t)
	t.Setenv("TENANT_INDEPENDENT_PAYMENTS", "true")
	user, _, _ := tenantBankFixture(t, 89, 11051)
	for _, provider := range []string{model.PaymentProviderAlipay, model.PaymentProviderWechatPay} {
		account := model.TenantPaymentAccount{TenantId: user.TenantId, Provider: provider, MerchantId: "tenant-online-receiver",
			AppId: "tenant-app", CredentialReference: "secret://tenant/89/payment", Currency: "CNY"}
		require.NoError(t, model.DB.Create(&account).Error)
		_, err := NewPaymentGatewayService().CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
			UserId: user.Id, PaymentAccountId: account.Id, Provider: provider,
			BusinessType: model.PaymentBusinessTokenRecharge, Currency: "CNY", Amount: 10,
		})
		require.ErrorIs(t, err, ErrPaymentInvalidProvider)
	}
}
