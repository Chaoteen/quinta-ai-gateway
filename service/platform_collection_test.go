package service

import (
	"context"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func TestSecurityPlatformCollectsWithoutChangingCustomerOwnership(t *testing.T) {
	truncate(t)
	t.Setenv("PAYMENT_COLLECTION_MODE", "platform")
	t.Setenv("PLATFORM_TENANT_ID", "91")
	_, platformReviewer, account := tenantBankFixture(t, 91, 19001)
	require.NoError(t, model.DB.Model(&platformReviewer).Update("role_key", common.RoleKeyFinance).Error)
	customer, customerReviewer, customerAccount := tenantBankFixture(t, 92, 19003)
	svc := NewPaymentGatewayService()
	_, err := svc.CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId: customer.Id, PaymentAccountId: customerAccount.Id,
		Provider: model.PaymentProviderBankTransfer, BusinessType: model.PaymentBusinessTokenRecharge, Amount: 10, Currency: "CNY",
	})
	require.ErrorIs(t, err, ErrTenantPaymentAccount)
	order := tenantBankOrder(t, customer, account.Id)
	require.Equal(t, customer.TenantId, order.TenantId)
	require.Equal(t, account.TenantId, order.ReceivingTenantId)
	require.Equal(t, "PLATFORM", order.CollectionMode)
	record, err := svc.CreateBankTransferRecord(context.Background(), CreateBankTransferRecordInput{
		PaymentOrderId: order.Id, UserId: customer.Id, BankAccountName: "fixture payer", TransferAmount: 10,
	})
	require.NoError(t, err)
	_, err = svc.ReviewBankTransfer(context.Background(), record.Id, ReviewBankTransferInput{
		ReviewerId: customerReviewer.Id, Approved: true, BankTransactionReference: "platform-statement-1",
	})
	require.ErrorIs(t, err, ErrBankReviewAuthority)
	_, err = svc.ReviewBankTransfer(context.Background(), record.Id, ReviewBankTransferInput{
		ReviewerId: platformReviewer.Id, Approved: true, BankTransactionReference: "platform-statement-1",
	})
	require.NoError(t, err)
	require.NoError(t, model.DB.First(&customer, customer.Id).Error)
	require.Equal(t, 700000, customer.Quota)
	var credits int64
	require.NoError(t, model.DB.Model(&model.QuotaCreditLedgerEntry{}).Where("source_id = ?", order.Id).Count(&credits).Error)
	require.EqualValues(t, 1, credits)
}
