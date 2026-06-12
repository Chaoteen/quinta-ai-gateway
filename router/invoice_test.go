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

func seedRoleAdoptionInvoiceApplication(t *testing.T, db *gorm.DB, id int, user roleAdoptionUser, amount float64) {
	t.Helper()
	profile := model.InvoiceProfile{
		Id:          id,
		TenantId:    user.tenant,
		UserId:      user.id,
		ProfileType: model.InvoiceProfileTypeCompany,
		Title:       fmt.Sprintf("Invoice Profile %d", id),
		TaxNo:       fmt.Sprintf("TAX-%d", id),
		Status:      model.InvoiceProfileStatusActive,
	}
	require.NoError(t, db.Create(&profile).Error)
	app := model.InvoiceApplication{
		Id:               id,
		ApplicationNo:    fmt.Sprintf("INV-ROUTE-%d-%d", id, time.Now().UnixNano()),
		TenantId:         user.tenant,
		UserId:           user.id,
		InvoiceProfileId: profile.Id,
		Amount:           amount,
		Currency:         "USD",
		InvoiceType:      model.InvoiceTypeVATNormal,
		Status:           model.InvoiceStatusPending,
		SourceType:       model.InvoiceSourcePaymentOrder,
		SourceId:         id,
	}
	require.NoError(t, db.Create(&app).Error)
}

func decodeInvoicePageTotal(t *testing.T, body []byte) int64 {
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

func TestInvoiceRouteRBAC(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	for _, roleName := range []string{"root", "finance", "tenant_admin"} {
		recorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/invoices/applications", "", roleAdoptionUsers[roleName])
		success, _ := decodeRoleAdoptionBasicResponse(t, recorder)
		require.True(t, success, roleName)
	}
	for _, roleName := range []string{"user", "ops", "auditor"} {
		recorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/invoices/applications", "", roleAdoptionUsers[roleName])
		success, _ := decodeRoleAdoptionBasicResponse(t, recorder)
		require.False(t, success, roleName)
	}
}

func TestInvoiceRouteOwnershipScope(t *testing.T) {
	r := setupRoleAdoptionRouter(t)
	seedRoleAdoptionInvoiceApplication(t, model.DB, 11901, roleAdoptionUsers["user"], 100)
	seedRoleAdoptionInvoiceApplication(t, model.DB, 11902, roleAdoptionUsers["tenant2_user"], 200)

	userResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/invoices/applications?p=1&page_size=10", "", roleAdoptionUsers["user"])
	require.Equal(t, int64(1), decodeInvoicePageTotal(t, userResp.Body.Bytes()))

	tenantResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/invoices/applications?p=1&page_size=10", "", roleAdoptionUsers["tenant_admin"])
	require.Equal(t, int64(1), decodeInvoicePageTotal(t, tenantResp.Body.Bytes()))

	financeResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/invoices/applications?p=1&page_size=10", "", roleAdoptionUsers["finance"])
	require.Equal(t, int64(2), decodeInvoicePageTotal(t, financeResp.Body.Bytes()))
}
