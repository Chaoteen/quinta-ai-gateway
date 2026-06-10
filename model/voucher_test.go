package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVoucherModelsAreMigrated(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&VoucherBatch{}, &Voucher{}, &VoucherRedemption{}))
	require.True(t, DB.Migrator().HasTable(&VoucherBatch{}))
	require.True(t, DB.Migrator().HasTable(&Voucher{}))
	require.True(t, DB.Migrator().HasTable(&VoucherRedemption{}))

	for _, column := range []string{"batch_no", "voucher_type", "tenant_id", "organization_id", "distribution_channel_id"} {
		require.True(t, DB.Migrator().HasColumn(&VoucherBatch{}, column), "missing voucher_batches.%s", column)
	}
	for _, column := range []string{"batch_id", "voucher_code", "quota_amount", "subscription_plan_id", "status"} {
		require.True(t, DB.Migrator().HasColumn(&Voucher{}, column), "missing vouchers.%s", column)
	}
	for _, column := range []string{"voucher_id", "voucher_code", "user_id", "tenant_id", "redemption_type", "redemption_result"} {
		require.True(t, DB.Migrator().HasColumn(&VoucherRedemption{}, column), "missing voucher_redemptions.%s", column)
	}
}

func TestVoucherDefaultsAndNormalization(t *testing.T) {
	batch := VoucherBatch{
		BatchNo:     " batch-1 ",
		Name:        " Spring ",
		VoucherType: "subscription",
		Status:      "active",
	}
	batch.Normalize()
	require.Equal(t, "BATCH-1", batch.BatchNo)
	require.Equal(t, "Spring", batch.Name)
	require.Equal(t, VoucherTypeSubscription, batch.VoucherType)
	require.Equal(t, VoucherBatchStatusActive, batch.Status)
	require.Equal(t, 1, batch.TenantId)

	voucher := Voucher{VoucherCode: " ab-cd ", VoucherType: "token", Status: "redeemed"}
	voucher.Normalize()
	require.Equal(t, "AB-CD", voucher.VoucherCode)
	require.Equal(t, VoucherTypeToken, voucher.VoucherType)
	require.Equal(t, VoucherStatusRedeemed, voucher.Status)
}
