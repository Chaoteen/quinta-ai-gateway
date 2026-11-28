package router

import (
	"bytes"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/middleware"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func rbacRequest(t *testing.T, r *gin.Engine, user, role, epoch int, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	data, err := common.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", strconv.Itoa(user))
	req.Header.Set("X-Test-Role", strconv.Itoa(role))
	req.Header.Set("X-Test-Auth-Epoch", strconv.Itoa(epoch))
	req.Header.Set("X-Test-Verified", "1")
	req.Header.Set("New-Api-User", strconv.Itoa(user))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestRBACHTTPDelegationAndRevocation(t *testing.T) {
	r := setupRootBoundaryRouter(t)
	require.NoError(t, model.DB.AutoMigrate(&model.RBACPolicy{}, &model.ScopedRole{}, &model.RBACBinding{}, &model.RBACAudit{}))
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", 2).Update("role_key", common.RoleKeyEnterpriseAdmin).Error)
	seedRootBoundaryUser(t, model.DB, 3, "scoped-member", 1, 2)
	seedRootBoundaryUser(t, model.DB, 4, "foreign-member", 1, 1)
	family := common.RoleKeyEnterpriseAdmin
	caps := common.ResourcePermissions(family)
	response := rbacRequest(t, r, 1, 100, 0, "PUT", "/api/rbac/policy", map[string]interface{}{"family": family, "delegate": true, "version": 0, "grants": caps})
	require.Equal(t, 200, response.Code, response.Body.String())
	grants := map[string]common.ResourceGrant{"invoice_requests": caps["invoice_requests"], "keys": {Scope: "tenant", View: true, Create: true}}
	response = rbacRequest(t, r, 2, 10, 0, "POST", "/api/rbac/custom-roles", model.ScopedRoleInput{TenantId: 2, Family: family, Name: "Finance", Active: true, Grants: grants})
	require.Equal(t, 200, response.Code, response.Body.String())
	var created struct {
		Data model.ScopedRole `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &created))
	require.Positive(t, created.Data.Id)
	body := map[string]int{"user_id": 4, "role_id": created.Data.Id}
	response = rbacRequest(t, r, 2, 10, 0, "POST", "/api/rbac/assignments", body)
	require.Equal(t, 403, response.Code)
	body["user_id"] = 3
	response = rbacRequest(t, r, 2, 10, 0, "POST", "/api/rbac/assignments", body)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = rbacRequest(t, r, 3, 10, 0, "GET", "/api/rbac/self", nil)
	require.Equal(t, 401, response.Code, "old session revoked")
	response = rbacRequest(t, r, 3, 10, 1, "GET", "/api/rbac/self", nil)
	require.Equal(t, 200, response.Code)
	response = rbacRequest(t, r, 3, 10, 1, "GET", "/api/user/", nil)
	require.Equal(t, 403, response.Code, "no member visibility")
	response = rbacRequest(t, r, 3, 10, 1, "POST", "/api/user/self/chat-key", map[string]string{"preset_id": "fixture"})
	require.Equal(t, 403, response.Code, "shortcut cannot reveal a key without use permission")
	// This role has key creation, but batch DELETE requires its own delete grant.
	response = rbacRequest(t, r, 3, 10, 1, "POST", "/api/token/batch", map[string]interface{}{"ids": []int{1}})
	require.Equal(t, 403, response.Code, "POST does not automatically mean create")
	response = rbacRequest(t, r, 3, 10, 1, "POST", "/api/rbac/custom-roles", model.ScopedRoleInput{TenantId: 2, Family: family, Name: "Escalate", Active: true, Grants: caps})
	require.Equal(t, 403, response.Code)
	caps["invoice_requests"] = common.ResourceGrant{Scope: "tenant"}
	caps["keys"] = common.ResourceGrant{Scope: "tenant"}
	response = rbacRequest(t, r, 1, 100, 0, "PUT", "/api/rbac/policy", map[string]interface{}{"family": family, "delegate": true, "version": 1, "grants": caps})
	require.Equal(t, 200, response.Code)
	response = rbacRequest(t, r, 3, 10, 1, "GET", "/api/invoices/profiles", nil)
	require.Equal(t, 403, response.Code, "policy revocation applied without signing in again")
	response = rbacRequest(t, r, 2, 10, 0, "GET", "/api/rbac/workspace?tenant_id=1", nil)
	require.Equal(t, 403, response.Code)
	response = rbacRequest(t, r, 3, 10, 1, "POST", "/api/user/self/chat-key", map[string]string{"preset_id": "fixture"})
	require.Equal(t, 403, response.Code, "profile shortcut respects key creation policy")
	require.NoError(t, model.DB.AutoMigrate(&model.Token{}))
	const key = "rbacfixturekey00000000000000000000000000"
	require.NoError(t, model.DB.Create(&model.Token{UserId: 3, TenantId: 2, Key: key, Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true}).Error)
	reached := false
	r.POST("/rbac-inference-fixture", middleware.TokenAuth(), func(c *gin.Context) { reached = true; c.Status(204) })
	req := httptest.NewRequest("POST", "/rbac-inference-fixture", nil)
	req.Header.Set("Authorization", "Bearer sk-"+key)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code, rec.Body.String())
	require.True(t, strings.Contains(rec.Body.String(), "Model usage is not allowed"))
	require.False(t, reached)
	r.GET("/rbac-video-fixture", middleware.TokenOrUserAuth(), func(c *gin.Context) { c.Status(204) })
	response = rbacRequest(t, r, 3, 10, 0, "GET", "/rbac-video-fixture", nil)
	require.Equal(t, 401, response.Code, "video session also checks revoked epoch")
	response = rbacRequest(t, r, 3, 10, 1, "GET", "/rbac-video-fixture", nil)
	require.Equal(t, 403, response.Code, "video session respects model use withdrawal")

}
