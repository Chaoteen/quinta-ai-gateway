package controller

import (
	"net/http"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/Chaoteen/quinta-ai-gateway/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSecurityUnsignedCreemEventsNeverAcceptedInTestMode(t *testing.T) {
	old := setting.CreemTestMode
	t.Cleanup(func() { setting.CreemTestMode = old })
	setting.CreemTestMode = true
	require.False(t, verifyCreemSignature("fixture", "", ""))
	require.False(t, verifyCreemSignature("fixture", "forged", ""))
	signature := generateCreemSignature("fixture", "synthetic-test-secret")
	require.True(t, verifyCreemSignature("fixture", signature, "synthetic-test-secret"))
	require.False(t, verifyCreemSignature("changed", signature, "synthetic-test-secret"))
}

func TestSecurityPasskeyResetRejectsPeerHigherAndSelf(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupInvoiceControllerTestDB(t)
	root := model.User{Id: 40, Username: "root-fixture", AffCode: "root-fixture-code", Role: common.RoleRootUser, RoleKey: common.RoleKeyRoot, TenantId: 1}
	admin := model.User{Id: 41, Username: "admin-fixture", AffCode: "admin-fixture-code", Role: common.RoleAdminUser, RoleKey: common.RoleKeyTenantAdmin, TenantId: 1}
	require.NoError(t, db.Create(&root).Error)
	require.NoError(t, db.Create(&admin).Error)
	for _, item := range []struct {
		actor  int
		role   int
		target string
	}{
		{42, common.RoleAdminUser, "40"},
		{42, common.RoleAdminUser, "41"},
		{40, common.RoleRootUser, "40"},
	} {
		c, w := newInvoiceControllerContext(http.MethodDelete, "/fixture", "", item.actor, 1, common.RoleKeyTenantAdmin)
		c.Set("role", item.role)
		c.Params = gin.Params{{Key: "id", Value: item.target}}
		AdminResetPasskey(c)
		require.Contains(t, w.Body.String(), "无权重置")
	}
}
