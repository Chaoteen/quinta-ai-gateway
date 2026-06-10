package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVoucherControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	oldDB := model.DB
	oldLogDB := model.LOG_DB
	model.DB = db
	model.LOG_DB = db
	common.UsingSQLite = true
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.VoucherBatch{},
		&model.Voucher{},
		&model.VoucherRedemption{},
	))
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
	})
	return db
}

func newVoucherControllerContext(method string, target string, body string, userId int, tenantId int, roleKey string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", userId)
	common.SetContextKey(ctx, constant.ContextKeyTenantId, tenantId)
	common.SetContextKey(ctx, constant.ContextKeyUserRoleKey, roleKey)
	return ctx, recorder
}

func TestVoucherControllerCreateBatchAndGenerate(t *testing.T) {
	db := setupVoucherControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{Id: 9701, TenantId: 71, Username: "voucher-admin", Status: common.UserStatusEnabled}).Error)

	ctx, recorder := newVoucherControllerContext(http.MethodPost, "/api/admin/vouchers/batches", `{"name":"Launch","voucher_type":"TOKEN"}`, 9701, 71, common.RoleKeyTenantAdmin)
	AdminCreateVoucherBatch(ctx)

	var batchResp struct {
		Success bool               `json:"success"`
		Data    model.VoucherBatch `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &batchResp))
	require.True(t, batchResp.Success, recorder.Body.String())
	require.Equal(t, model.VoucherBatchStatusActive, batchResp.Data.Status)

	genCtx, genRecorder := newVoucherControllerContext(http.MethodPost, "/api/admin/voucher-batches/1/generate", `{"quantity":2,"quota_amount":300}`, 9701, 71, common.RoleKeyTenantAdmin)
	genCtx.Params = gin.Params{{Key: "id", Value: "1"}}
	AdminGenerateVouchers(genCtx)

	var genResp struct {
		Success bool            `json:"success"`
		Data    []model.Voucher `json:"data"`
	}
	require.NoError(t, common.Unmarshal(genRecorder.Body.Bytes(), &genResp))
	require.True(t, genResp.Success, genRecorder.Body.String())
	require.Len(t, genResp.Data, 2)
}

func TestVoucherControllerHistoryOnlyOwnRows(t *testing.T) {
	db := setupVoucherControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{Id: 9702, TenantId: 72, Username: "voucher-user", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.VoucherRedemption{VoucherId: 1, VoucherCode: "OWN", UserId: 9702, TenantId: 72, RedemptionType: model.VoucherTypeToken, RedemptionResult: model.VoucherRedemptionResultSuccess}).Error)
	require.NoError(t, db.Create(&model.VoucherRedemption{VoucherId: 2, VoucherCode: "OTHER", UserId: 9703, TenantId: 72, RedemptionType: model.VoucherTypeToken, RedemptionResult: model.VoucherRedemptionResultSuccess}).Error)

	ctx, recorder := newVoucherControllerContext(http.MethodGet, "/api/vouchers/history?p=1&page_size=10", "", 9702, 72, common.RoleKeyUser)
	ListVoucherHistory(ctx)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success, recorder.Body.String())
	require.Equal(t, int64(1), resp.Data.Total)
}
