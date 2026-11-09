package service

import (
	"context"
	"math"
	"sync"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func securityPaymentOrder(t *testing.T, user model.User, provider string) *model.PaymentOrder {
	t.Helper()
	order, err := NewPaymentGatewayService().CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId: user.Id, Provider: provider, BusinessType: model.PaymentBusinessTokenRecharge,
		BusinessId: 999999999, Amount: 1, Currency: "USD",
	})
	require.NoError(t, err)
	return order
}

func TestSecurityPaymentUsesServerQuote(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9801, 71, 0)
	order := securityPaymentOrder(t, user, model.PaymentProviderMock)
	require.Zero(t, order.BusinessId)
	require.Equal(t, int64(common.QuotaPerUnit), order.QuotaGranted)
	require.Equal(t, paymentPricingVersion, order.PricingVersion)
	_, err := NewPaymentGatewayService().FulfillPaymentOrder(context.Background(), order.Id)
	require.ErrorIs(t, err, ErrPaymentOrderStatusInvalid)
	_, err = NewPaymentGatewayService().ConfirmPayment(context.Background(), ConfirmPaymentInput{
		OrderNo: order.OrderNo, Provider: model.PaymentProviderMock, SignatureValid: false,
	})
	require.ErrorIs(t, err, ErrPaymentInvalidSignature)
	var reloaded model.User
	require.NoError(t, model.DB.First(&reloaded, user.Id).Error)
	require.Zero(t, reloaded.Quota)
}

func TestSecurityPaymentRejectsUnsafeInputs(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9802, 71, 0)
	svc := NewPaymentGatewayService()
	for _, amount := range []float64{0, -1, math.NaN(), math.Inf(1), 1000000, 1000001, 0.0000001} {
		_, err := svc.CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
			UserId: user.Id, Provider: model.PaymentProviderMock, BusinessType: model.PaymentBusinessTokenRecharge, Amount: amount,
		})
		require.ErrorIs(t, err, ErrPaymentInvalidAmount)
	}
	_, err := svc.CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId: user.Id, Provider: "unknown", BusinessType: model.PaymentBusinessTokenRecharge, Amount: 1,
	})
	require.ErrorIs(t, err, ErrPaymentInvalidProvider)
	_, err = svc.CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId: user.Id, TenantId: 999, Provider: model.PaymentProviderMock, BusinessType: model.PaymentBusinessTokenRecharge, Amount: 1,
	})
	require.ErrorIs(t, err, ErrPaymentOwnership)
	_, err = svc.CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId: user.Id, Provider: model.PaymentProviderMock, BusinessType: model.PaymentBusinessTokenRecharge, Amount: 1, Currency: "CNY",
	})
	require.ErrorIs(t, err, ErrPaymentInvalidAmount)
	t.Setenv("ENABLE_MOCK_PAYMENTS", "false")
	_, err = svc.CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId: user.Id, Provider: model.PaymentProviderMock, BusinessType: model.PaymentBusinessTokenRecharge, Amount: 1,
	})
	require.ErrorIs(t, err, ErrPaymentInvalidProvider)
}

func TestSecurityPaymentSnapshotSurvivesPlanChanges(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9803, 71, 0)
	plan := seedVoucherPlan(t, "security-price")
	require.NoError(t, model.DB.Model(&plan).Update("price_amount", 10).Error)
	svc := NewPaymentGatewayService()
	input := CreatePaymentOrderInput{UserId: user.Id, Provider: model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessSubscriptionPurchase, BusinessId: plan.Id, Amount: 1, Currency: "USD"}
	_, err := svc.CreatePaymentOrder(context.Background(), input)
	require.ErrorIs(t, err, ErrPaymentInvalidAmount)
	input.Amount = 10
	order, err := svc.CreatePaymentOrder(context.Background(), input)
	require.NoError(t, err)
	require.NoError(t, model.DB.Model(&plan).Updates(map[string]interface{}{"token_quota": 99999, "price_amount": 100}).Error)
	_, err = svc.ConfirmPayment(context.Background(), ConfirmPaymentInput{OrderNo: order.OrderNo, Provider: model.PaymentProviderMock, SignatureValid: true})
	require.NoError(t, err)
	var sub model.UserSubscription
	require.NoError(t, model.DB.Where("user_id = ?", user.Id).First(&sub).Error)
	require.Equal(t, int64(1000), sub.TokenQuotaSnapshot)
}

func TestSecurityPaymentConcurrentConfirmation(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9804, 71, 0)
	order := securityPaymentOrder(t, user, model.PaymentProviderMock)
	var wg sync.WaitGroup
	errs := make(chan error, 24)
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := NewPaymentGatewayService().ConfirmPayment(context.Background(), ConfirmPaymentInput{
				OrderNo: order.OrderNo, Provider: model.PaymentProviderMock, SignatureValid: true,
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var reloaded model.User
	require.NoError(t, model.DB.First(&reloaded, user.Id).Error)
	require.Equal(t, int(order.QuotaGranted), reloaded.Quota)
	var ledger []model.QuotaCreditLedgerEntry
	require.NoError(t, model.DB.Where("source_type = ? AND source_id = ?", model.QuotaCreditSourcePayment, order.Id).Find(&ledger).Error)
	require.Len(t, ledger, 1)
	require.Equal(t, order.QuotaGranted, ledger[0].QuotaDelta)
	require.ErrorIs(t, model.DB.Model(&ledger[0]).Update("quota_delta", 1).Error, model.ErrQuotaCreditLedgerImmutable)
	require.ErrorIs(t, model.DB.Delete(&ledger[0]).Error, model.ErrQuotaCreditLedgerImmutable)
}

func TestSecurityBankApprovalRollsBackWhenFulfillmentFails(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9805, 71, 0)
	svc := NewPaymentGatewayService()
	order := securityPaymentOrder(t, user, model.PaymentProviderBankTransfer)
	record, err := svc.CreateBankTransferRecord(context.Background(), CreateBankTransferRecordInput{
		PaymentOrderId: order.Id, UserId: user.Id, BankAccountName: "test", TransferAmount: 1,
	})
	require.NoError(t, err)
	// Simulate an old unpaid order with no trustworthy quote.
	require.NoError(t, model.DB.Model(order).Update("pricing_version", "").Error)
	_, err = svc.ReviewBankTransfer(context.Background(), record.Id, ReviewBankTransferInput{ReviewerId: 123, Approved: true})
	require.ErrorIs(t, err, ErrPaymentUntrustedQuote)
	require.NoError(t, model.DB.First(record, record.Id).Error)
	require.Equal(t, model.BankTransferReviewPending, record.ReviewStatus)
	require.NoError(t, model.DB.First(order, order.Id).Error)
	require.Equal(t, model.PaymentOrderStatusPending, order.Status)
}

func securityVoucher(t *testing.T, user model.User) model.Voucher {
	t.Helper()
	svc := NewVoucherService()
	batch, err := svc.CreateVoucherBatch(context.Background(), voucherTenantActor(user), CreateVoucherBatchInput{Name: "security", VoucherType: model.VoucherTypeToken})
	require.NoError(t, err)
	vouchers, err := svc.GenerateVouchers(context.Background(), voucherTenantActor(user), GenerateVouchersInput{BatchId: batch.Id, Quantity: 1, QuotaAmount: 100})
	require.NoError(t, err)
	return vouchers[0]
}

func TestSecurityVoucherRejectsCrossTenantRedemption(t *testing.T) {
	truncate(t)
	a := seedVoucherUser(t, 9806, 71, 0, 0)
	b := seedVoucherUser(t, 9807, 72, 0, 0)
	voucher := securityVoucher(t, a)
	_, err := NewVoucherService().RedeemVoucher(context.Background(), b.Id, voucher.VoucherCode)
	require.ErrorIs(t, err, ErrVoucherOutOfScope)
	require.NoError(t, model.DB.First(&voucher, voucher.Id).Error)
	require.Equal(t, model.VoucherStatusUnused, voucher.Status)
}

func TestSecurityVoucherConcurrentRedemption(t *testing.T) {
	truncate(t)
	user := seedVoucherUser(t, 9808, 71, 0, 0)
	voucher := securityVoucher(t, user)
	var wg sync.WaitGroup
	errs := make(chan error, 24)
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := NewVoucherService().RedeemVoucher(context.Background(), user.Id, voucher.VoucherCode)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var reloaded model.User
	require.NoError(t, model.DB.First(&reloaded, user.Id).Error)
	require.Equal(t, 100, reloaded.Quota)
	var count int64
	require.NoError(t, model.DB.Model(&model.VoucherRedemption{}).Where("voucher_id = ?", voucher.Id).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestSecurityVoucherRollsBackOnBalanceOverflow(t *testing.T) {
	truncate(t)
	user := seedVoucherUser(t, 9809, 71, 0, math.MaxInt)
	voucher := securityVoucher(t, user)
	_, err := NewVoucherService().RedeemVoucher(context.Background(), user.Id, voucher.VoucherCode)
	require.Error(t, err)
	require.NoError(t, model.DB.First(&voucher, voucher.Id).Error)
	require.Equal(t, model.VoucherStatusUnused, voucher.Status)
}
