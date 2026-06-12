package router

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedRoleAdoptionFinancePayment(t *testing.T, db *gorm.DB, id int, user roleAdoptionUser, amount float64) {
	t.Helper()
	require.NoError(t, db.Create(&model.PaymentOrder{
		Id:           id,
		OrderNo:      fmt.Sprintf("FIN-ROUTE-%d-%d", id, time.Now().UnixNano()),
		TenantId:     user.tenant,
		UserId:       user.id,
		Provider:     model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessTokenRecharge,
		Amount:       amount,
		Status:       model.PaymentOrderStatusPaid,
		Subject:      "finance route payment",
	}).Error)
}

func seedRoleAdoptionFinanceBilling(t *testing.T, db *gorm.DB, id int, user roleAdoptionUser, amount int64) {
	t.Helper()
	require.NoError(t, db.Create(&model.BillingRecord{
		Id:            id,
		TenantId:      user.tenant,
		UserId:        user.id,
		RequestId:     fmt.Sprintf("finance-route-req-%d", id),
		UsageRecordId: 200000 + id,
		ProviderName:  "openai",
		ModelName:     "gpt-4o",
		BillingStatus: model.BillingStatusSettled,
		QuotaCharged:  amount,
		RequestCount:  1,
		TotalTokens:   100,
	}).Error)
}

func decodeFinanceSummaryTotals(t *testing.T, body []byte) (float64, int64) {
	t.Helper()
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Revenue struct {
				TotalRechargeAmount float64 `json:"total_recharge_amount"`
			} `json:"revenue"`
			Consumption struct {
				TotalConsumptionAmount int64 `json:"total_consumption_amount"`
			} `json:"consumption"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(body, &resp))
	require.True(t, resp.Success, string(body))
	return resp.Data.Revenue.TotalRechargeAmount, resp.Data.Consumption.TotalConsumptionAmount
}

func TestFinanceConsoleRouteRBAC(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	for _, roleName := range []string{"root", "finance", "tenant_admin"} {
		recorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/finance/summary", "", roleAdoptionUsers[roleName])
		success, _ := decodeRoleAdoptionBasicResponse(t, recorder)
		require.True(t, success, roleName)
	}
	for _, roleName := range []string{"user", "ops", "auditor"} {
		recorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/finance/summary", "", roleAdoptionUsers[roleName])
		success, _ := decodeRoleAdoptionBasicResponse(t, recorder)
		require.False(t, success, roleName)
	}
}

func TestFinanceConsoleRouteOwnershipScope(t *testing.T) {
	r := setupRoleAdoptionRouter(t)
	seedRoleAdoptionFinancePayment(t, model.DB, 9901, roleAdoptionUsers["user"], 100)
	seedRoleAdoptionFinancePayment(t, model.DB, 9902, roleAdoptionUsers["tenant2_user"], 300)
	seedRoleAdoptionFinanceBilling(t, model.DB, 9911, roleAdoptionUsers["user"], 100)
	seedRoleAdoptionFinanceBilling(t, model.DB, 9912, roleAdoptionUsers["tenant2_user"], 300)

	tenantResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/finance/summary", "", roleAdoptionUsers["tenant_admin"])
	tenantRevenue, tenantConsumption := decodeFinanceSummaryTotals(t, tenantResp.Body.Bytes())
	require.Equal(t, float64(100), tenantRevenue)
	require.Equal(t, int64(100), tenantConsumption)

	financeResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/finance/summary", "", roleAdoptionUsers["finance"])
	financeRevenue, financeConsumption := decodeFinanceSummaryTotals(t, financeResp.Body.Bytes())
	require.Equal(t, float64(400), financeRevenue)
	require.Equal(t, int64(400), financeConsumption)
}
