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

func seedBillingPortalUser(t *testing.T, id int, tenantId int, organizationId int, quota int) model.User {
	t.Helper()
	user := model.User{
		Id:             id,
		TenantId:       tenantId,
		OrganizationId: organizationId,
		Username:       fmt.Sprintf("billing-portal-user-%d-%d", id, time.Now().UnixNano()),
		Password:       "password123",
		Role:           common.RoleCommonUser,
		RoleKey:        common.RoleKeyUser,
		Status:         common.UserStatusEnabled,
		Group:          "default",
		Quota:          quota,
		AffCode:        fmt.Sprintf("bp-aff-%d", id),
	}
	require.NoError(t, model.DB.Create(&user).Error)
	return user
}

func TestBillingPortalSummaryStatistics(t *testing.T) {
	truncate(t)
	now := common.GetTimestamp()
	user := seedBillingPortalUser(t, 9301, 31, 101, 7000)
	other := seedBillingPortalUser(t, 9302, 31, 101, 9000)
	require.NoError(t, model.DB.Create(&model.PaymentOrder{
		OrderNo:        "BP-SUM-1",
		TenantId:       user.TenantId,
		OrganizationId: user.OrganizationId,
		UserId:         user.Id,
		Provider:       model.PaymentProviderMock,
		BusinessType:   model.PaymentBusinessTokenRecharge,
		BusinessId:     1000,
		Amount:         12.5,
		Currency:       "USD",
		Status:         model.PaymentOrderStatusPaid,
		PaidAt:         now,
	}).Error)
	require.NoError(t, model.DB.Create(&model.BillingRecord{
		TenantId:       user.TenantId,
		OrganizationId: user.OrganizationId,
		UserId:         user.Id,
		RequestId:      "bp-summary-usage-1",
		UsageRecordId:  930101,
		ProviderName:   "openai",
		ModelName:      "gpt-4o",
		QuotaCharged:   300,
		TotalTokens:    1200,
		RequestCount:   3,
		CreatedAt:      now,
	}).Error)
	require.NoError(t, model.DB.Create(&model.BillingRecord{
		TenantId:       other.TenantId,
		OrganizationId: other.OrganizationId,
		UserId:         other.Id,
		RequestId:      "bp-summary-usage-2",
		UsageRecordId:  930102,
		ProviderName:   "claude",
		ModelName:      "claude-3",
		QuotaCharged:   999,
		TotalTokens:    9999,
		RequestCount:   9,
		CreatedAt:      now,
	}).Error)

	summary, err := NewBillingPortalService().Summary(context.Background(), BillingPortalActor{
		UserId: user.Id,
		Scope:  model.AccessScope{TenantId: user.TenantId, RoleKey: common.RoleKeyUser},
	})

	require.NoError(t, err)
	require.Equal(t, int64(7000), summary.BalanceQuota)
	require.Equal(t, 12.5, summary.TotalRechargeAmount)
	require.Equal(t, int64(300), summary.TotalConsumptionAmount)
	require.Equal(t, int64(1200), summary.TotalTokens)
	require.Equal(t, int64(3), summary.TotalRequests)
	require.Len(t, summary.ModelConsumptionRanking, 1)
	require.Equal(t, "gpt-4o", summary.ModelConsumptionRanking[0].Name)
}

func TestBillingPortalPaymentHistoryPaginationAndScope(t *testing.T) {
	truncate(t)
	now := common.GetTimestamp()
	user := seedBillingPortalUser(t, 9303, 32, 0, 0)
	other := seedBillingPortalUser(t, 9304, 33, 0, 0)
	for i := 0; i < 3; i++ {
		require.NoError(t, model.DB.Create(&model.PaymentOrder{
			OrderNo:      fmt.Sprintf("BP-PAGE-%d", i),
			TenantId:     user.TenantId,
			UserId:       user.Id,
			Provider:     model.PaymentProviderMock,
			BusinessType: model.PaymentBusinessTokenRecharge,
			BusinessId:   100,
			Amount:       float64(i + 1),
			Currency:     "USD",
			Status:       model.PaymentOrderStatusPaid,
			PaidAt:       now,
		}).Error)
	}
	require.NoError(t, model.DB.Create(&model.PaymentOrder{
		OrderNo:      "BP-PAGE-OTHER",
		TenantId:     other.TenantId,
		UserId:       other.Id,
		Provider:     model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessTokenRecharge,
		BusinessId:   100,
		Amount:       100,
		Currency:     "USD",
		Status:       model.PaymentOrderStatusPaid,
		PaidAt:       now,
	}).Error)

	page, err := NewBillingPortalService().PaymentHistory(context.Background(), BillingPortalActor{
		UserId: user.Id,
		Scope:  model.AccessScope{TenantId: user.TenantId, RoleKey: common.RoleKeyUser},
	}, BillingPortalListInput{Page: 1, PageSize: 2, Status: model.PaymentOrderStatusPaid})

	require.NoError(t, err)
	require.Equal(t, int64(3), page.Total)
	require.Len(t, page.Items, 2)
}

func TestBillingPortalTenantAndOrganizationScopes(t *testing.T) {
	truncate(t)
	now := common.GetTimestamp()
	userA := seedBillingPortalUser(t, 9305, 41, 501, 0)
	userB := seedBillingPortalUser(t, 9306, 41, 502, 0)
	userC := seedBillingPortalUser(t, 9307, 42, 501, 0)
	for _, user := range []model.User{userA, userB, userC} {
		require.NoError(t, model.DB.Create(&model.BillingRecord{
			TenantId:       user.TenantId,
			OrganizationId: user.OrganizationId,
			UserId:         user.Id,
			RequestId:      fmt.Sprintf("bp-scope-%d", user.Id),
			UsageRecordId:  user.Id,
			ProviderName:   "openai",
			ModelName:      "gpt-4o",
			QuotaCharged:   100,
			TotalTokens:    1000,
			RequestCount:   1,
			CreatedAt:      now,
		}).Error)
	}

	tenantSummary, err := NewBillingPortalService().Summary(context.Background(), BillingPortalActor{
		UserId: 1,
		Scope:  model.AccessScope{TenantId: 41, RoleKey: common.RoleKeyTenantAdmin},
	})
	require.NoError(t, err)
	require.Equal(t, int64(200), tenantSummary.TotalConsumptionAmount)

	orgSummary, err := NewBillingPortalService().Summary(context.Background(), BillingPortalActor{
		UserId: 1,
		Scope:  model.AccessScope{TenantId: 41, OrganizationId: 501, RoleKey: common.RoleKeyOrganizationAdmin},
	})
	require.NoError(t, err)
	require.Equal(t, int64(100), orgSummary.TotalConsumptionAmount)

	rootSummary, err := NewBillingPortalService().Summary(context.Background(), BillingPortalActor{
		UserId: 1,
		Scope:  model.AccessScope{IsRoot: true, RoleKey: common.RoleKeyRoot},
	})
	require.NoError(t, err)
	require.Equal(t, int64(300), rootSummary.TotalConsumptionAmount)
}
