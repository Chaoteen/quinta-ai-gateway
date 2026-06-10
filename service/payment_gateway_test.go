package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func seedPaymentUser(t *testing.T, id int, tenantId int, quota int) model.User {
	t.Helper()
	user := model.User{
		Id:       id,
		TenantId: tenantId,
		Username: fmt.Sprintf("payment-user-%d-%d", id, time.Now().UnixNano()),
		Password: "password123",
		Role:     common.RoleCommonUser,
		RoleKey:  common.RoleKeyUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    quota,
	}
	require.NoError(t, model.DB.Create(&user).Error)
	return user
}

func TestPaymentOrderCreateSuccess(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9101, 21, 0)

	order, err := NewPaymentGatewayService().CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId:       user.Id,
		Provider:     model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessTokenRecharge,
		BusinessId:   1000,
		Amount:       10,
		Currency:     "USD",
		Subject:      "Token recharge",
	})

	require.NoError(t, err)
	require.NotZero(t, order.Id)
	require.NotEmpty(t, order.OrderNo)
	require.Equal(t, user.TenantId, order.TenantId)
	require.Equal(t, model.PaymentOrderStatusPending, order.Status)
	require.Equal(t, model.PaymentFulfillmentPending, order.FulfillmentStatus)
}

func TestPaymentOrderRejectsNonPositiveAmount(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9102, 21, 0)

	_, err := NewPaymentGatewayService().CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId:       user.Id,
		Provider:     model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessTokenRecharge,
		BusinessId:   1000,
		Amount:       0,
	})

	require.ErrorIs(t, err, ErrPaymentInvalidAmount)
}

func TestConfirmPaymentIsIdempotentAndDoesNotDoubleFulfill(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9103, 21, 0)
	svc := NewPaymentGatewayService()
	order, err := svc.CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId:       user.Id,
		Provider:     model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessTokenRecharge,
		BusinessId:   1234,
		Amount:       12.34,
	})
	require.NoError(t, err)

	_, err = svc.ConfirmPayment(context.Background(), ConfirmPaymentInput{
		OrderNo:        order.OrderNo,
		Provider:       model.PaymentProviderMock,
		EventType:      "mock.paid",
		RawPayload:     `{"paid":true}`,
		SignatureValid: true,
	})
	require.NoError(t, err)
	_, err = svc.ConfirmPayment(context.Background(), ConfirmPaymentInput{
		OrderNo:        order.OrderNo,
		Provider:       model.PaymentProviderMock,
		EventType:      "mock.paid.retry",
		RawPayload:     `{"retry":true}`,
		SignatureValid: true,
	})
	require.NoError(t, err)

	var reloaded model.User
	require.NoError(t, model.DB.Select("quota").Where("id = ?", user.Id).First(&reloaded).Error)
	require.Equal(t, 1234, reloaded.Quota)

	var logCount int64
	require.NoError(t, model.DB.Model(&model.PaymentCallbackLog{}).Where("payment_order_id = ?", order.Id).Count(&logCount).Error)
	require.Equal(t, int64(2), logCount)
}

func TestConfirmPaymentRejectsTerminalStatuses(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9104, 21, 0)
	for _, status := range []string{model.PaymentOrderStatusFailed, model.PaymentOrderStatusCanceled, model.PaymentOrderStatusExpired} {
		order, err := NewPaymentGatewayService().CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
			UserId:       user.Id,
			Provider:     model.PaymentProviderMock,
			BusinessType: model.PaymentBusinessTokenRecharge,
			BusinessId:   100,
			Amount:       1,
			Subject:      status,
		})
		require.NoError(t, err)
		require.NoError(t, model.DB.Model(&model.PaymentOrder{}).Where("id = ?", order.Id).Update("status", status).Error)

		_, err = NewPaymentGatewayService().ConfirmPayment(context.Background(), ConfirmPaymentInput{
			OrderNo:        order.OrderNo,
			Provider:       model.PaymentProviderMock,
			SignatureValid: true,
		})
		require.True(t, errors.Is(err, ErrPaymentOrderStatusInvalid), "status %s err %v", status, err)
	}

	var reloaded model.User
	require.NoError(t, model.DB.Select("quota").Where("id = ?", user.Id).First(&reloaded).Error)
	require.Equal(t, 0, reloaded.Quota)
}

func TestBankTransferApprovalTriggersConfirmOnce(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9105, 21, 0)
	svc := NewPaymentGatewayService()
	order, err := svc.CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId:       user.Id,
		Provider:     model.PaymentProviderBankTransfer,
		BusinessType: model.PaymentBusinessTokenRecharge,
		BusinessId:   2222,
		Amount:       22.22,
	})
	require.NoError(t, err)
	record, err := svc.CreateBankTransferRecord(context.Background(), CreateBankTransferRecordInput{
		PaymentOrderId:  order.Id,
		UserId:          user.Id,
		BankAccountName: "Alice",
		TransferAmount:  22.22,
		ProofUrl:        "https://example.test/proof.png",
	})
	require.NoError(t, err)

	_, err = svc.ReviewBankTransfer(context.Background(), record.Id, ReviewBankTransferInput{
		ReviewerId: 9901,
		Approved:   true,
		ReviewNote: "matched",
	})
	require.NoError(t, err)
	_, err = svc.ReviewBankTransfer(context.Background(), record.Id, ReviewBankTransferInput{
		ReviewerId: 9901,
		Approved:   true,
		ReviewNote: "duplicate",
	})
	require.ErrorIs(t, err, ErrBankTransferStatusInvalid)

	var reloaded model.User
	require.NoError(t, model.DB.Select("quota").Where("id = ?", user.Id).First(&reloaded).Error)
	require.Equal(t, 2222, reloaded.Quota)

	var paidOrder model.PaymentOrder
	require.NoError(t, model.DB.Where("id = ?", order.Id).First(&paidOrder).Error)
	require.Equal(t, model.PaymentOrderStatusPaid, paidOrder.Status)
}

func TestPaymentSubscriptionPurchaseFulfillment(t *testing.T) {
	truncate(t)
	user := seedPaymentUser(t, 9106, 21, 0)
	plan := model.SubscriptionPlan{
		Code:          "payment-plan",
		Name:          "Payment Plan",
		Title:         "Payment Plan",
		PriceAmount:   9.99,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TokenQuota:    3000,
	}
	require.NoError(t, model.DB.Create(&plan).Error)

	svc := NewPaymentGatewayService()
	order, err := svc.CreatePaymentOrder(context.Background(), CreatePaymentOrderInput{
		UserId:       user.Id,
		Provider:     model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessSubscriptionPurchase,
		BusinessId:   plan.Id,
		Amount:       9.99,
	})
	require.NoError(t, err)
	_, err = svc.ConfirmPayment(context.Background(), ConfirmPaymentInput{
		OrderNo:        order.OrderNo,
		Provider:       model.PaymentProviderMock,
		SignatureValid: true,
	})
	require.NoError(t, err)

	var count int64
	require.NoError(t, model.DB.Model(&model.UserSubscription{}).Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).Count(&count).Error)
	require.Equal(t, int64(1), count)
}
