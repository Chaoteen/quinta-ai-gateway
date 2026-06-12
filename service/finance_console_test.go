package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func resetFinanceConsoleTables(t *testing.T) {
	t.Helper()
	cleanup := func() {
		model.DB.Exec("DELETE FROM revenue_share_records")
		model.DB.Exec("DELETE FROM voucher_redemptions")
		model.DB.Exec("DELETE FROM vouchers")
		model.DB.Exec("DELETE FROM voucher_batches")
		model.DB.Exec("DELETE FROM payment_orders")
		model.DB.Exec("DELETE FROM billing_records")
		model.DB.Exec("DELETE FROM user_subscriptions")
		model.DB.Exec("DELETE FROM distribution_channels")
		model.DB.Exec("DELETE FROM channels")
		model.DB.Exec("DELETE FROM users")
		model.DB.Exec("DELETE FROM tenants")
	}
	cleanup()
	t.Cleanup(cleanup)
}

func seedFinanceTenant(t *testing.T, id int, name string) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.Tenant{Id: id, Name: name, Status: 1}).Error)
}

func seedFinanceUser(t *testing.T, id int, tenantId int, quota int) {
	t.Helper()
	user := model.User{
		Id:       id,
		TenantId: tenantId,
		Username: fmt.Sprintf("finance-user-%d-%d", id, time.Now().UnixNano()),
		Role:     common.RoleCommonUser,
		RoleKey:  common.RoleKeyUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    quota,
		AffCode:  fmt.Sprintf("finance-aff-%d", id),
	}
	require.NoError(t, model.DB.Create(&user).Error)
}

func seedFinancePayment(t *testing.T, id int, tenantId int, userId int, amount float64, status string, provider string) {
	t.Helper()
	order := model.PaymentOrder{
		Id:           id,
		OrderNo:      fmt.Sprintf("FIN-PAY-%d-%d", id, time.Now().UnixNano()),
		TenantId:     tenantId,
		UserId:       userId,
		Provider:     provider,
		BusinessType: model.PaymentBusinessTokenRecharge,
		Amount:       amount,
		Currency:     "USD",
		Status:       status,
		Subject:      "finance test payment",
	}
	require.NoError(t, model.DB.Create(&order).Error)
}

func seedFinanceBilling(t *testing.T, id int, tenantId int, userId int, provider string, modelName string, amount int64, requests int64, tokens int64) {
	t.Helper()
	record := model.BillingRecord{
		Id:                id,
		TenantId:          tenantId,
		UserId:            userId,
		RequestId:         fmt.Sprintf("finance-req-%d", id),
		UsageRecordId:     100000 + id,
		ProviderName:      provider,
		ModelName:         modelName,
		BillingStatus:     model.BillingStatusSettled,
		RequestCount:      requests,
		TotalTokens:       tokens,
		QuotaCharged:      amount,
		FundingSource:     "quota",
		BillingPhase:      model.BillingPhaseUsageFact,
		PriceSnapshot:     "{}",
		UnitPriceSnapshot: "{}",
	}
	require.NoError(t, model.DB.Create(&record).Error)
}

func seedFinanceVoucherData(t *testing.T, tenantId int, userId int) {
	t.Helper()
	batch := model.VoucherBatch{
		Id:          9101 + tenantId,
		BatchNo:     fmt.Sprintf("FIN-VB-%d", tenantId),
		Name:        "finance voucher",
		VoucherType: model.VoucherTypeToken,
		Status:      model.VoucherBatchStatusActive,
		TenantId:    tenantId,
		CreatedBy:   userId,
	}
	require.NoError(t, model.DB.Create(&batch).Error)
	require.NoError(t, model.DB.Create(&model.Voucher{Id: 9201 + tenantId, BatchId: batch.Id, VoucherCode: fmt.Sprintf("FIN-V-%d-A", tenantId), VoucherType: model.VoucherTypeToken, QuotaAmount: 100, Status: model.VoucherStatusRedeemed}).Error)
	require.NoError(t, model.DB.Create(&model.Voucher{Id: 9301 + tenantId, BatchId: batch.Id, VoucherCode: fmt.Sprintf("FIN-V-%d-B", tenantId), VoucherType: model.VoucherTypeToken, QuotaAmount: 100, Status: model.VoucherStatusUnused}).Error)
	require.NoError(t, model.DB.Create(&model.VoucherRedemption{VoucherId: 9201 + tenantId, VoucherCode: fmt.Sprintf("FIN-V-%d-A", tenantId), UserId: userId, TenantId: tenantId, RedemptionType: model.VoucherTypeToken, RedemptionResult: model.VoucherRedemptionResultSuccess}).Error)
}

func seedFinanceRevenueShare(t *testing.T, id int, tenantId int, channelId int, gross float64, platform float64) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.RevenueShareRecord{
		Id:                    id,
		TenantId:              tenantId,
		SourceType:            model.RevenueShareSourceBilling,
		SourceId:              id,
		DistributionChannelId: channelId,
		GrossAmount:           gross,
		PlatformAmount:        platform,
		Currency:              FinanceConsoleCurrencyQuota,
		Status:                model.RevenueShareStatusCalculated,
	}).Error)
}

func financeActor(roleKey string, tenantId int, isRoot bool) FinanceConsoleActor {
	return FinanceConsoleActor{
		UserId: 1,
		Scope:  model.AccessScope{TenantId: tenantId, RoleKey: roleKey, IsRoot: isRoot},
	}
}

func TestFinanceConsoleSummaryAndRankings(t *testing.T) {
	resetFinanceConsoleTables(t)
	seedFinanceTenant(t, 81, "Tenant Alpha")
	seedFinanceTenant(t, 82, "Tenant Beta")
	seedFinanceUser(t, 8101, 81, 700)
	seedFinanceUser(t, 8201, 82, 300)
	require.NoError(t, model.DB.Create(&model.DistributionChannel{Id: 811, TenantId: 81, Name: "Alpha Channel", Code: "alpha-channel", Status: common.ChannelStatusEnabled}).Error)
	require.NoError(t, model.DB.Create(&model.DistributionChannel{Id: 821, TenantId: 82, Name: "Beta Channel", Code: "beta-channel", Status: common.ChannelStatusEnabled}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 801, TenantId: 81, Type: 1, Key: "finance-channel-key-a", Status: common.ChannelStatusEnabled}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 802, TenantId: 82, Type: 1, Key: "finance-channel-key-b", Status: common.ChannelStatusEnabled}).Error)

	seedFinancePayment(t, 81001, 81, 8101, 100, model.PaymentOrderStatusPaid, model.PaymentProviderMock)
	seedFinancePayment(t, 81002, 81, 8101, 50, model.PaymentOrderStatusPending, model.PaymentProviderBankTransfer)
	seedFinancePayment(t, 82001, 82, 8201, 200, model.PaymentOrderStatusPaid, model.PaymentProviderMock)
	seedFinanceBilling(t, 81011, 81, 8101, "openai", "gpt-4o", 300, 3, 1200)
	seedFinanceBilling(t, 82011, 82, 8201, "anthropic", "claude-sonnet", 700, 7, 2800)
	seedFinanceVoucherData(t, 81, 8101)
	seedFinanceRevenueShare(t, 81021, 81, 811, 300, 180)
	require.NoError(t, model.DB.Create(&model.UserSubscription{TenantId: 81, UserId: 8101, PlanId: 1, Status: model.SubscriptionLifecycleActive, EndTime: common.GetTimestamp() + 3600}).Error)

	summary, err := NewFinanceConsoleService().Summary(context.Background(), financeActor(common.RoleKeyFinance, 0, false), FinanceConsoleListInput{Days: 7})
	require.NoError(t, err)
	require.Equal(t, float64(300), summary.Revenue.TotalRechargeAmount)
	require.Equal(t, int64(3), summary.Revenue.PaymentOrderCount)
	require.Equal(t, float64(66.67), summary.Revenue.PaymentSuccessRate)
	require.Equal(t, int64(1000), summary.Consumption.TotalConsumptionAmount)
	require.Equal(t, int64(10), summary.Consumption.TotalRequests)
	require.Equal(t, int64(4000), summary.Consumption.TotalTokens)
	require.Equal(t, int64(2), summary.Activity.ActiveTenantCount)
	require.Equal(t, int64(2), summary.Activity.ActiveUserCount)
	require.Equal(t, int64(1), summary.Activity.ActiveSubscriptionCount)
	require.Equal(t, int64(2), summary.Activity.ActiveChannelCount)
	require.Equal(t, int64(2), summary.Voucher.TotalIssued)
	require.Equal(t, int64(1), summary.Voucher.TotalRedeemed)
	require.Equal(t, float64(50), summary.Voucher.RedemptionRate)
	require.Equal(t, float64(300), summary.RevenueShare.GrossAmount)
	require.Len(t, summary.Tenant.ConsumptionRanking, 2)
	require.Equal(t, "Tenant Beta", summary.Tenant.ConsumptionRanking[0].Name)

	topProviders, err := NewFinanceConsoleService().TopProviders(context.Background(), financeActor(common.RoleKeyFinance, 0, false), FinanceConsoleListInput{Page: 1, PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, int64(2), topProviders.Total)
	require.Len(t, topProviders.Items, 1)
	require.Equal(t, "anthropic", topProviders.Items[0].Name)
}

func TestFinanceConsoleTenantScope(t *testing.T) {
	resetFinanceConsoleTables(t)
	seedFinanceTenant(t, 83, "Tenant Scoped")
	seedFinanceTenant(t, 84, "Tenant Hidden")
	seedFinanceUser(t, 8301, 83, 100)
	seedFinanceUser(t, 8401, 84, 100)
	seedFinancePayment(t, 83001, 83, 8301, 120, model.PaymentOrderStatusPaid, model.PaymentProviderMock)
	seedFinancePayment(t, 84001, 84, 8401, 220, model.PaymentOrderStatusPaid, model.PaymentProviderMock)
	seedFinanceBilling(t, 83011, 83, 8301, "openai", "gpt-4o", 120, 1, 100)
	seedFinanceBilling(t, 84011, 84, 8401, "openai", "gpt-4o", 220, 1, 100)

	summary, err := NewFinanceConsoleService().Summary(context.Background(), financeActor(common.RoleKeyTenantAdmin, 83, false), FinanceConsoleListInput{})
	require.NoError(t, err)
	require.Equal(t, float64(120), summary.Revenue.TotalRechargeAmount)
	require.Equal(t, int64(120), summary.Consumption.TotalConsumptionAmount)
	require.Equal(t, int64(1), summary.Activity.ActiveTenantCount)

	topTenants, err := NewFinanceConsoleService().TopTenants(context.Background(), financeActor(common.RoleKeyTenantAdmin, 83, false), FinanceConsoleListInput{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), topTenants.Total)
	require.Equal(t, 83, topTenants.Items[0].TenantId)
}
