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

func seedRoleAdoptionPaymentOrder(t *testing.T, db *gorm.DB, id int, userId int, tenantId int) {
	t.Helper()
	require.NoError(t, db.Create(&model.PaymentOrder{
		Id:                id,
		OrderNo:           fmt.Sprintf("PAY-ROUTER-%d-%s", id, time.Now().Format("150405.000000000")),
		UserId:            userId,
		TenantId:          tenantId,
		Provider:          model.PaymentProviderMock,
		BusinessType:      model.PaymentBusinessTokenRecharge,
		BusinessId:        100,
		Amount:            1,
		Currency:          "USD",
		Status:            model.PaymentOrderStatusPending,
		FulfillmentStatus: model.PaymentFulfillmentPending,
	}).Error)
}

func decodePaymentRouterTotal(t *testing.T, recorderBody []byte) int {
	t.Helper()
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorderBody, &resp))
	require.True(t, resp.Success, string(recorderBody))
	return resp.Data.Total
}

func TestPaymentAdminRouteScopeByRole(t *testing.T) {
	r := setupRoleAdoptionRouter(t)
	seedRoleAdoptionPaymentOrder(t, model.DB, 301, roleAdoptionUsers["user"].id, 1)
	seedRoleAdoptionPaymentOrder(t, model.DB, 302, roleAdoptionUsers["tenant2_user"].id, 2)

	root := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/payment/orders?p=1&page_size=10", "", roleAdoptionUsers["root"])
	require.Equal(t, 2, decodePaymentRouterTotal(t, root.Body.Bytes()))

	tenantAdmin := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/payment/orders?p=1&page_size=10", "", roleAdoptionUsers["tenant_admin"])
	require.Equal(t, 1, decodePaymentRouterTotal(t, tenantAdmin.Body.Bytes()))

	finance := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/payment/orders?p=1&page_size=10", "", roleAdoptionUsers["finance"])
	require.Equal(t, 1, decodePaymentRouterTotal(t, finance.Body.Bytes()))
}

func TestPaymentAdminRouteDeniedForNonFinanceRoles(t *testing.T) {
	r := setupRoleAdoptionRouter(t)
	for _, roleName := range []string{"ops", "auditor", "organization_admin", "user"} {
		recorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/payment/orders?p=1&page_size=10", "", roleAdoptionUsers[roleName])
		success, _ := decodeRoleAdoptionBasicResponse(t, recorder)
		require.False(t, success, roleName)
	}
}
