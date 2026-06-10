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

func seedRoleAdoptionBillingRecord(t *testing.T, db *gorm.DB, id int, user roleAdoptionUser, quota int64) {
	t.Helper()
	require.NoError(t, db.Create(&model.BillingRecord{
		TenantId:       user.tenant,
		OrganizationId: user.organization,
		UserId:         user.id,
		RequestId:      fmt.Sprintf("bp-router-%d-%d", id, time.Now().UnixNano()),
		UsageRecordId:  id,
		ProviderName:   "openai",
		ModelName:      "gpt-4o",
		QuotaCharged:   quota,
		TotalTokens:    quota * 10,
		RequestCount:   1,
	}).Error)
}

func decodeBillingPortalSummaryConsumption(t *testing.T, body []byte) int64 {
	t.Helper()
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			TotalConsumptionAmount int64 `json:"total_consumption_amount"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(body, &resp))
	require.True(t, resp.Success, string(body))
	return resp.Data.TotalConsumptionAmount
}

func decodeBillingPortalTotal(t *testing.T, body []byte) int64 {
	t.Helper()
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(body, &resp))
	require.True(t, resp.Success, string(body))
	return resp.Data.Total
}

func TestBillingPortalRouteRBACScope(t *testing.T) {
	r := setupRoleAdoptionRouter(t)
	seedRoleAdoptionBillingRecord(t, model.DB, 9501, roleAdoptionUsers["user"], 100)
	seedRoleAdoptionBillingRecord(t, model.DB, 9502, roleAdoptionUsers["organization_user"], 200)
	seedRoleAdoptionBillingRecord(t, model.DB, 9503, roleAdoptionUsers["other_organization"], 300)
	seedRoleAdoptionBillingRecord(t, model.DB, 9504, roleAdoptionUsers["tenant2_user"], 400)

	userResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/billing/summary", "", roleAdoptionUsers["user"])
	require.Equal(t, int64(100), decodeBillingPortalSummaryConsumption(t, userResp.Body.Bytes()))

	orgResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/billing/summary", "", roleAdoptionUsers["organization_admin"])
	require.Equal(t, int64(200), decodeBillingPortalSummaryConsumption(t, orgResp.Body.Bytes()))

	tenantResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/billing/summary", "", roleAdoptionUsers["tenant_admin"])
	require.Equal(t, int64(600), decodeBillingPortalSummaryConsumption(t, tenantResp.Body.Bytes()))

	rootResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/billing/summary", "", roleAdoptionUsers["root"])
	require.Equal(t, int64(1000), decodeBillingPortalSummaryConsumption(t, rootResp.Body.Bytes()))
}

func TestBillingPortalRoutePagination(t *testing.T) {
	r := setupRoleAdoptionRouter(t)
	seedRoleAdoptionBillingRecord(t, model.DB, 9511, roleAdoptionUsers["user"], 100)
	seedRoleAdoptionBillingRecord(t, model.DB, 9512, roleAdoptionUsers["user"], 200)
	seedRoleAdoptionBillingRecord(t, model.DB, 9513, roleAdoptionUsers["tenant2_user"], 300)

	resp := performRoleAdoptionRequest(r, http.MethodGet, "/api/billing/records?p=1&page_size=1", "", roleAdoptionUsers["user"])
	require.Equal(t, int64(2), decodeBillingPortalTotal(t, resp.Body.Bytes()))
}
