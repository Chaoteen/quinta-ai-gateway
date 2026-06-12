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

func setupFinanceConsoleControllerTestDB(t *testing.T) *gorm.DB {
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
		&model.Tenant{},
		&model.User{},
		&model.Channel{},
		&model.DistributionChannel{},
		&model.PaymentOrder{},
		&model.BillingRecord{},
		&model.UserSubscription{},
		&model.VoucherBatch{},
		&model.Voucher{},
		&model.VoucherRedemption{},
		&model.RevenueShareRecord{},
	))
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
	})
	return db
}

func newFinanceConsoleControllerContext(method string, target string, userId int, tenantId int, roleKey string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, nil)
	ctx.Set("id", userId)
	common.SetContextKey(ctx, constant.ContextKeyTenantId, tenantId)
	common.SetContextKey(ctx, constant.ContextKeyUserRoleKey, roleKey)
	return ctx, recorder
}

func TestFinanceConsoleControllerSummary(t *testing.T) {
	db := setupFinanceConsoleControllerTestDB(t)
	require.NoError(t, db.Create(&model.Tenant{Id: 91, Name: "controller tenant", Status: 1}).Error)
	require.NoError(t, db.Create(&model.User{Id: 9101, TenantId: 91, Username: "finance-controller-user", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.PaymentOrder{
		Id:           91001,
		OrderNo:      "FIN-CTRL-PAID",
		TenantId:     91,
		UserId:       9101,
		Provider:     model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessTokenRecharge,
		Amount:       90,
		Status:       model.PaymentOrderStatusPaid,
		Subject:      "controller payment",
	}).Error)
	require.NoError(t, db.Create(&model.BillingRecord{Id: 91011, TenantId: 91, UserId: 9101, RequestId: "finance-controller-req", UsageRecordId: 91011, ProviderName: "openai", ModelName: "gpt-4o", BillingStatus: model.BillingStatusSettled, QuotaCharged: 45, RequestCount: 2, TotalTokens: 300}).Error)

	ctx, recorder := newFinanceConsoleControllerContext(http.MethodGet, "/api/admin/finance/summary?days=7", 9101, 91, common.RoleKeyTenantAdmin)
	GetFinanceSummary(ctx)

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
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success, recorder.Body.String())
	require.Equal(t, float64(90), resp.Data.Revenue.TotalRechargeAmount)
	require.Equal(t, int64(45), resp.Data.Consumption.TotalConsumptionAmount)
}
