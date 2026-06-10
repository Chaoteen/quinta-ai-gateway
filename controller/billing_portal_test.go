package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBillingPortalControllerTestDB(t *testing.T) *gorm.DB {
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
		&model.PaymentOrder{},
		&model.QuotaUsageRecord{},
		&model.BillingRecord{},
		&model.UserSubscription{},
	))
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
	})
	return db
}

func newBillingPortalControllerContext(method string, target string, userId int, tenantId int, organizationId int, roleKey string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, nil)
	ctx.Set("id", userId)
	common.SetContextKey(ctx, constant.ContextKeyTenantId, tenantId)
	common.SetContextKey(ctx, constant.ContextKeyOrganizationId, organizationId)
	common.SetContextKey(ctx, constant.ContextKeyUserRoleKey, roleKey)
	return ctx, recorder
}

func TestBillingPortalControllerSummary(t *testing.T) {
	db := setupBillingPortalControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{Id: 9401, TenantId: 51, Username: "bp-controller", Quota: 500, Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.BillingRecord{
		TenantId:      51,
		UserId:        9401,
		RequestId:     "bp-controller-summary",
		UsageRecordId: 940101,
		ProviderName:  "openai",
		ModelName:     "gpt-4o",
		QuotaCharged:  90,
		TotalTokens:   450,
		RequestCount:  2,
	}).Error)

	ctx, recorder := newBillingPortalControllerContext(http.MethodGet, "/api/billing/summary", 9401, 51, 0, common.RoleKeyUser)
	GetBillingPortalSummary(ctx)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			BalanceQuota           int64 `json:"balance_quota"`
			TotalConsumptionAmount int64 `json:"total_consumption_amount"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success, recorder.Body.String())
	require.Equal(t, int64(500), resp.Data.BalanceQuota)
	require.Equal(t, int64(90), resp.Data.TotalConsumptionAmount)
}

func TestBillingPortalControllerPaymentsOnlyOwnRows(t *testing.T) {
	db := setupBillingPortalControllerTestDB(t)
	require.NoError(t, db.Create(&model.PaymentOrder{
		OrderNo:      "BP-CTRL-OWN",
		TenantId:     52,
		UserId:       9402,
		Provider:     model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessTokenRecharge,
		BusinessId:   100,
		Amount:       1,
		Currency:     "USD",
		Status:       model.PaymentOrderStatusPaid,
	}).Error)
	require.NoError(t, db.Create(&model.PaymentOrder{
		OrderNo:      "BP-CTRL-OTHER",
		TenantId:     52,
		UserId:       9403,
		Provider:     model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessTokenRecharge,
		BusinessId:   100,
		Amount:       1,
		Currency:     "USD",
		Status:       model.PaymentOrderStatusPaid,
	}).Error)

	ctx, recorder := newBillingPortalControllerContext(http.MethodGet, "/api/billing/payments?p=1&page_size=10", 9402, 52, 0, common.RoleKeyUser)
	GetBillingPortalPayments(ctx)

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
