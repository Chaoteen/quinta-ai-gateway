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

func setupInvoiceControllerTestDB(t *testing.T) *gorm.DB {
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
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.PaymentOrder{}, &model.InvoiceProfile{}, &model.InvoiceApplication{}, &model.InvoiceFile{}))
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
	})
	return db
}

func newInvoiceControllerContext(method string, target string, body string, userId int, tenantId int, roleKey string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", userId)
	common.SetContextKey(ctx, constant.ContextKeyTenantId, tenantId)
	common.SetContextKey(ctx, constant.ContextKeyUserRoleKey, roleKey)
	return ctx, recorder
}

func TestInvoiceControllerCreateProfileAndApplication(t *testing.T) {
	db := setupInvoiceControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{Id: 12001, TenantId: 121, Username: "invoice-controller-user", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.PaymentOrder{Id: 12101, OrderNo: "INV-CTRL-PAY", TenantId: 121, UserId: 12001, Provider: model.PaymentProviderMock, BusinessType: model.PaymentBusinessTokenRecharge, Amount: 60, Currency: "USD", Status: model.PaymentOrderStatusPaid, Subject: "paid"}).Error)

	ctx, recorder := newInvoiceControllerContext(http.MethodPost, "/api/invoices/profiles", `{"profile_type":"COMPANY","title":"Company","tax_no":"TAX","is_default":true}`, 12001, 121, common.RoleKeyUser)
	CreateInvoiceProfile(ctx)
	var profileResp struct {
		Success bool                 `json:"success"`
		Data    model.InvoiceProfile `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &profileResp))
	require.True(t, profileResp.Success, recorder.Body.String())
	require.True(t, profileResp.Data.IsDefault)

	appCtx, appRecorder := newInvoiceControllerContext(http.MethodPost, "/api/invoices/applications", `{"invoice_profile_id":1,"amount":60,"invoice_type":"VAT_NORMAL","source_type":"PAYMENT_ORDER","source_id":12101}`, 12001, 121, common.RoleKeyUser)
	CreateInvoiceApplication(appCtx)
	var appResp struct {
		Success bool                     `json:"success"`
		Data    model.InvoiceApplication `json:"data"`
	}
	require.NoError(t, common.Unmarshal(appRecorder.Body.Bytes(), &appResp))
	require.True(t, appResp.Success, appRecorder.Body.String())
	require.Equal(t, model.InvoiceStatusPending, appResp.Data.Status)
}
