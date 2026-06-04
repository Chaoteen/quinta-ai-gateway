package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBillingRecordDefaultsAndTrim(t *testing.T) {
	truncateTables(t)

	record := BillingRecord{
		TenantId:          2,
		RequestId:         " billing-request-1 ",
		ReservationId:     " billing-reservation-1 ",
		UsageRecordId:     1001,
		UserId:            2001,
		ProviderName:      " openai ",
		ChannelId:         3001,
		ModelName:         " gpt-4o ",
		InputTokens:       10,
		OutputTokens:      5,
		TotalTokens:       15,
		RequestCount:      1,
		QuotaCharged:      15,
		UnitPriceSnapshot: " {} ",
		PriceSnapshot:     " {} ",
		Metadata:          " {} ",
	}
	require.NoError(t, DB.Create(&record).Error)
	require.Equal(t, "billing-request-1", record.RequestId)
	require.Equal(t, "billing-reservation-1", record.ReservationId)
	require.Equal(t, "openai", record.ProviderName)
	require.Equal(t, "gpt-4o", record.ModelName)
	require.Equal(t, BillingStatusPending, record.BillingStatus)
	require.Equal(t, BillingPhaseUsageFact, record.BillingPhase)
	require.NotZero(t, record.CreatedAt)
	require.NotZero(t, record.UpdatedAt)
}

func TestBillingRecordIsMigrated(t *testing.T) {
	require.True(t, DB.Migrator().HasTable(&BillingRecord{}))
	for _, column := range []string{
		"tenant_id",
		"organization_id",
		"department_id",
		"distribution_channel_id",
		"request_id",
		"reservation_id",
		"usage_record_id",
		"user_id",
		"token_id",
		"user_subscription_id",
		"provider_name",
		"channel_id",
		"model_name",
		"funding_source",
		"billing_status",
		"billing_phase",
		"currency",
		"input_tokens",
		"output_tokens",
		"total_tokens",
		"request_count",
		"quota_charged",
		"unit_price_snapshot",
		"price_snapshot",
		"settled_delta",
		"metadata",
	} {
		require.True(t, DB.Migrator().HasColumn(&BillingRecord{}, column), "missing billing record column %s", column)
	}
}
