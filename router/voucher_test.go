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

func seedRoleAdoptionVoucherBatch(t *testing.T, db *gorm.DB, id int, user roleAdoptionUser, name string) model.VoucherBatch {
	t.Helper()
	batch := model.VoucherBatch{
		Id:             id,
		BatchNo:        fmt.Sprintf("VR-%d-%d", id, time.Now().UnixNano()),
		Name:           name,
		VoucherType:    model.VoucherTypeToken,
		Status:         model.VoucherBatchStatusActive,
		TenantId:       user.tenant,
		OrganizationId: user.organization,
		CreatedBy:      user.id,
	}
	require.NoError(t, db.Create(&batch).Error)
	return batch
}

func seedRoleAdoptionVoucherRedemption(t *testing.T, db *gorm.DB, id int, user roleAdoptionUser, code string) {
	t.Helper()
	require.NoError(t, db.Create(&model.VoucherRedemption{
		Id:               id,
		VoucherId:        id,
		VoucherCode:      code,
		UserId:           user.id,
		TenantId:         user.tenant,
		OrganizationId:   user.organization,
		RedemptionType:   model.VoucherTypeToken,
		RedemptionResult: model.VoucherRedemptionResultSuccess,
	}).Error)
}

func decodeVoucherTotal(t *testing.T, body []byte) int64 {
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

func TestVoucherRouteRBAC(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	userResp := performRoleAdoptionRequest(r, http.MethodPost, "/api/admin/vouchers/batches", `{"name":"Denied","voucher_type":"TOKEN"}`, roleAdoptionUsers["user"])
	success, _ := decodeRoleAdoptionBasicResponse(t, userResp)
	require.False(t, success)

	financeListResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/vouchers", "", roleAdoptionUsers["finance"])
	success, _ = decodeRoleAdoptionBasicResponse(t, financeListResp)
	require.False(t, success)

	financeRedemptionResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/vouchers/redemptions", "", roleAdoptionUsers["finance"])
	success, _ = decodeRoleAdoptionBasicResponse(t, financeRedemptionResp)
	require.True(t, success)

	adminResp := performRoleAdoptionRequest(r, http.MethodPost, "/api/admin/vouchers/batches", `{"name":"Allowed","voucher_type":"TOKEN"}`, roleAdoptionUsers["tenant_admin"])
	success, _ = decodeRoleAdoptionBasicResponse(t, adminResp)
	require.True(t, success)
}

func TestVoucherRouteOwnershipScope(t *testing.T) {
	r := setupRoleAdoptionRouter(t)
	seedRoleAdoptionVoucherBatch(t, model.DB, 9801, roleAdoptionUsers["user"], "tenant-one")
	seedRoleAdoptionVoucherBatch(t, model.DB, 9802, roleAdoptionUsers["tenant2_user"], "tenant-two")
	seedRoleAdoptionVoucherRedemption(t, model.DB, 9811, roleAdoptionUsers["user"], "VR-OWN")
	seedRoleAdoptionVoucherRedemption(t, model.DB, 9812, roleAdoptionUsers["tenant2_user"], "VR-OTHER")

	tenantResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/vouchers/batches?p=1&page_size=10", "", roleAdoptionUsers["tenant_admin"])
	require.Equal(t, int64(1), decodeVoucherTotal(t, tenantResp.Body.Bytes()))

	rootResp := performRoleAdoptionRequest(r, http.MethodGet, "/api/admin/vouchers/batches?p=1&page_size=10", "", roleAdoptionUsers["root"])
	require.Equal(t, int64(2), decodeVoucherTotal(t, rootResp.Body.Bytes()))

	userHistory := performRoleAdoptionRequest(r, http.MethodGet, "/api/vouchers/history?p=1&page_size=10", "", roleAdoptionUsers["user"])
	require.Equal(t, int64(1), decodeVoucherTotal(t, userHistory.Body.Bytes()))
}
