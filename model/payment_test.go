package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaymentModelsAreMigrated(t *testing.T) {
	require.True(t, DB.Migrator().HasTable(&PaymentOrder{}))
	require.True(t, DB.Migrator().HasTable(&PaymentCallbackLog{}))
	require.True(t, DB.Migrator().HasTable(&BankTransferRecord{}))
	for _, column := range []string{
		"order_no",
		"tenant_id",
		"organization_id",
		"department_id",
		"distribution_channel_id",
		"user_id",
		"provider",
		"business_type",
		"business_id",
		"amount",
		"currency",
		"status",
		"fulfillment_status",
	} {
		require.True(t, DB.Migrator().HasColumn(&PaymentOrder{}, column), "missing payment order column %s", column)
	}
}

func TestPaymentModelNormalization(t *testing.T) {
	order := PaymentOrder{
		Provider:          "bank_transfer",
		BusinessType:      "subscription_purchase",
		Status:            "paid",
		Currency:          "usd",
		FulfillmentStatus: "success",
	}
	order.Normalize()
	require.Equal(t, PaymentProviderBankTransfer, order.Provider)
	require.Equal(t, PaymentBusinessSubscriptionPurchase, order.BusinessType)
	require.Equal(t, PaymentOrderStatusPaid, order.Status)
	require.Equal(t, "USD", order.Currency)
	require.Equal(t, PaymentFulfillmentSuccess, order.FulfillmentStatus)

	record := BankTransferRecord{ReviewStatus: "rejected"}
	record.Normalize()
	require.Equal(t, BankTransferReviewRejected, record.ReviewStatus)
}
