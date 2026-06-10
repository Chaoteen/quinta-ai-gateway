package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPaymentControllerTestDB(t *testing.T) *gorm.DB {
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
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.PaymentOrder{}, &model.PaymentCallbackLog{}, &model.BankTransferRecord{}))
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
	})
	return db
}

func seedPaymentControllerOrder(t *testing.T, db *gorm.DB, id int, userId int, tenantId int) {
	t.Helper()
	require.NoError(t, db.Create(&model.PaymentOrder{
		Id:                id,
		OrderNo:           fmt.Sprintf("PAY-CTRL-%d-%s", id, time.Now().Format("150405.000000000")),
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

func newPaymentControllerContext(method string, target string, userId int, tenantId int, roleKey string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, nil)
	ctx.Set("id", userId)
	common.SetContextKey(ctx, constant.ContextKeyTenantId, tenantId)
	common.SetContextKey(ctx, constant.ContextKeyUserRoleKey, roleKey)
	return ctx, recorder
}

func decodePaymentControllerPageTotal(t *testing.T, recorder *httptest.ResponseRecorder) int {
	t.Helper()
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success, recorder.Body.String())
	return resp.Data.Total
}

func TestListUserPaymentOrdersOnlyReturnsOwnOrders(t *testing.T) {
	db := setupPaymentControllerTestDB(t)
	seedPaymentControllerOrder(t, db, 1, 1001, 1)
	seedPaymentControllerOrder(t, db, 2, 1002, 1)

	ctx, recorder := newPaymentControllerContext(http.MethodGet, "/api/payment/orders?p=1&page_size=10", 1001, 1, common.RoleKeyUser)
	ListUserPaymentOrders(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, decodePaymentControllerPageTotal(t, recorder))
}

func TestAdminListPaymentOrdersHonorsTenantScope(t *testing.T) {
	db := setupPaymentControllerTestDB(t)
	seedPaymentControllerOrder(t, db, 1, 1001, 1)
	seedPaymentControllerOrder(t, db, 2, 2001, 2)

	tenantCtx, tenantRecorder := newPaymentControllerContext(http.MethodGet, "/api/admin/payment/orders?p=1&page_size=10", 9001, 1, common.RoleKeyTenantAdmin)
	AdminListPaymentOrders(tenantCtx)
	require.Equal(t, 1, decodePaymentControllerPageTotal(t, tenantRecorder))

	financeCtx, financeRecorder := newPaymentControllerContext(http.MethodGet, "/api/admin/payment/orders?p=1&page_size=10", 9002, 2, common.RoleKeyFinance)
	AdminListPaymentOrders(financeCtx)
	require.Equal(t, 1, decodePaymentControllerPageTotal(t, financeRecorder))

	rootCtx, rootRecorder := newPaymentControllerContext(http.MethodGet, "/api/admin/payment/orders?p=1&page_size=10", 1, 0, common.RoleKeyRoot)
	AdminListPaymentOrders(rootCtx)
	require.Equal(t, 2, decodePaymentControllerPageTotal(t, rootRecorder))
}
