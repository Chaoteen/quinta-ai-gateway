package router

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type roleAdoptionUser struct {
	id           int
	role         int
	roleKey      string
	tenant       int
	organization int
}

var roleAdoptionUsers = map[string]roleAdoptionUser{
	"root":                 {id: 1, role: common.RoleRootUser, roleKey: common.RoleKeyRoot, tenant: 1},
	"tenant_admin":         {id: 2, role: common.RoleAdminUser, roleKey: common.RoleKeyTenantAdmin, tenant: 1},
	"finance":              {id: 3, role: common.RoleCommonUser, roleKey: common.RoleKeyFinance, tenant: 1},
	"ops":                  {id: 4, role: common.RoleCommonUser, roleKey: common.RoleKeyOps, tenant: 1},
	"auditor":              {id: 5, role: common.RoleCommonUser, roleKey: common.RoleKeyAuditor, tenant: 1},
	"user":                 {id: 6, role: common.RoleCommonUser, roleKey: common.RoleKeyUser, tenant: 1},
	"tenant2_user":         {id: 7, role: common.RoleCommonUser, roleKey: common.RoleKeyUser, tenant: 2},
	"organization_admin":   {id: 8, role: common.RoleCommonUser, roleKey: common.RoleKeyOrganizationAdmin, tenant: 1, organization: 10},
	"organization_admin_0": {id: 9, role: common.RoleCommonUser, roleKey: common.RoleKeyOrganizationAdmin, tenant: 1},
	"organization_user":    {id: 10, role: common.RoleCommonUser, roleKey: common.RoleKeyUser, tenant: 1, organization: 10},
	"other_organization":   {id: 11, role: common.RoleCommonUser, roleKey: common.RoleKeyUser, tenant: 1, organization: 20},
	"tenant2_organization": {id: 12, role: common.RoleCommonUser, roleKey: common.RoleKeyUser, tenant: 2, organization: 10},
}

func setupRoleAdoptionRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	previousSQLitePath := common.SQLitePath
	previousIsMasterNode := common.IsMasterNode
	common.IsMasterNode = true

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.SQLitePath = dsn
	t.Cleanup(func() {
		common.SQLitePath = previousSQLitePath
		common.IsMasterNode = previousIsMasterNode
	})
	if err := model.InitDB(); err != nil {
		t.Fatalf("failed to initialize sqlite db: %v", err)
	}
	db := model.DB
	model.LOG_DB = db
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	for name, user := range roleAdoptionUsers {
		seedRoleAdoptionUser(t, db, user.id, name, user.role, user.roleKey, user.tenant, user.organization)
	}
	seedRoleAdoptionRedemption(t, db)
	seedRoleAdoptionChannels(t, db)
	seedRoleAdoptionSubscriptions(t, db)
	seedRoleAdoptionCatalog(t, db)

	r := gin.New()
	store := cookie.NewStore([]byte("role-adoption-test-secret"))
	r.Use(sessions.Sessions("session", store))
	r.Use(func(c *gin.Context) {
		userIDHeader := c.GetHeader("X-Test-User-ID")
		roleHeader := c.GetHeader("X-Test-Role")
		if userIDHeader == "" || roleHeader == "" {
			c.Next()
			return
		}
		userID, err := strconv.Atoi(userIDHeader)
		if err != nil {
			t.Fatalf("invalid test user id: %v", err)
		}
		role, err := strconv.Atoi(roleHeader)
		if err != nil {
			t.Fatalf("invalid test role: %v", err)
		}
		session := sessions.Default(c)
		session.Set("id", userID)
		session.Set("username", fmt.Sprintf("test-user-%d", userID))
		session.Set("role", role)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		if err := session.Save(); err != nil {
			t.Fatalf("failed to save test session: %v", err)
		}
		c.Next()
	})
	SetApiRouter(r)
	return r
}

func seedRoleAdoptionUser(t *testing.T, db *gorm.DB, id int, username string, role int, roleKey string, tenantId int, organizationId int) {
	t.Helper()
	user := model.User{
		Id:             id,
		TenantId:       tenantId,
		OrganizationId: organizationId,
		Username:       username,
		Password:       "password123",
		DisplayName:    username,
		Role:           role,
		RoleKey:        roleKey,
		Status:         common.UserStatusEnabled,
		Group:          "default",
		AffCode:        fmt.Sprintf("role-adoption-%d", id),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to seed user %d: %v", id, err)
	}
}

func seedRoleAdoptionRedemption(t *testing.T, db *gorm.DB) {
	t.Helper()
	redemption := model.Redemption{
		Id:       1,
		TenantId: 1,
		Key:      "roleadoptionredemption000000001",
		Status:   common.RedemptionCodeStatusEnabled,
		Name:     "role adoption",
		Quota:    100,
	}
	if err := db.Create(&redemption).Error; err != nil {
		t.Fatalf("failed to seed redemption: %v", err)
	}
}

func seedRoleAdoptionChannels(t *testing.T, db *gorm.DB) {
	t.Helper()
	tag := "phase2"
	channels := []model.Channel{
		{Id: 1, TenantId: 1, Name: "tenant-1-channel", Type: 1, Key: "tenant-1-key", Status: common.ChannelStatusEnabled, Models: "gpt-4o,gpt-4o-mini", Group: "default", Tag: &tag},
		{Id: 2, TenantId: 2, Name: "tenant-2-channel", Type: 1, Key: "tenant-2-key", Status: common.ChannelStatusEnabled, Models: "gpt-4.1", Group: "default", Tag: &tag},
	}
	for _, channel := range channels {
		requireCreateRoleAdoptionRecord(t, db.Create(&channel).Error)
	}
	abilities := []model.Ability{
		{TenantId: 1, ChannelId: 1, Model: "gpt-4o", Group: "default", Enabled: true},
		{TenantId: 2, ChannelId: 2, Model: "gpt-4.1", Group: "default", Enabled: true},
	}
	for _, ability := range abilities {
		requireCreateRoleAdoptionRecord(t, db.Create(&ability).Error)
	}
}

func seedRoleAdoptionSubscriptions(t *testing.T, db *gorm.DB) {
	t.Helper()
	plan := model.SubscriptionPlan{Id: 1, Title: "role adoption plan", Enabled: true, TotalAmount: 1000}
	requireCreateRoleAdoptionRecord(t, db.Create(&plan).Error)
	subscriptions := []model.UserSubscription{
		{Id: 1, TenantId: 1, UserId: roleAdoptionUsers["user"].id, PlanId: 1, Status: "active", AmountTotal: 1000, EndTime: common.GetTimestamp() + 3600},
		{Id: 2, TenantId: 2, UserId: roleAdoptionUsers["tenant2_user"].id, PlanId: 1, Status: "active", AmountTotal: 1000, EndTime: common.GetTimestamp() + 3600},
	}
	for _, subscription := range subscriptions {
		requireCreateRoleAdoptionRecord(t, db.Create(&subscription).Error)
	}
}

func seedRoleAdoptionCatalog(t *testing.T, db *gorm.DB) {
	t.Helper()
	vendor := model.Vendor{Id: 1, Name: "role-adoption-vendor", Status: 1}
	requireCreateRoleAdoptionRecord(t, db.Create(&vendor).Error)
	modelMeta := model.Model{Id: 1, ModelName: "gpt-4o", VendorID: 1, Status: 1, SyncOfficial: 1}
	requireCreateRoleAdoptionRecord(t, db.Create(&modelMeta).Error)
}

func requireCreateRoleAdoptionRecord(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("failed to seed role adoption record: %v", err)
	}
}

func performRoleAdoptionRequest(r *gin.Engine, method string, target string, body string, user roleAdoptionUser) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("New-Api-User", strconv.Itoa(user.id))
	req.Header.Set("X-Test-User-ID", strconv.Itoa(user.id))
	req.Header.Set("X-Test-Role", strconv.Itoa(user.role))
	r.ServeHTTP(recorder, req)
	return recorder
}

func decodeRoleAdoptionSuccess(t *testing.T, recorder *httptest.ResponseRecorder) bool {
	t.Helper()
	var response struct {
		Success bool `json:"success"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response body %q: %v", recorder.Body.String(), err)
	}
	return response.Success
}

func decodeRoleAdoptionUserListIDs(t *testing.T, recorder *httptest.ResponseRecorder) []int {
	t.Helper()
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Items []struct {
				Id int `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response body %q: %v", recorder.Body.String(), err)
	}
	if !response.Success {
		t.Fatalf("expected user list response success, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	ids := make([]int, 0, len(response.Data.Items))
	for _, item := range response.Data.Items {
		ids = append(ids, item.Id)
	}
	return ids
}

func decodeRoleAdoptionUserDetailID(t *testing.T, recorder *httptest.ResponseRecorder) int {
	t.Helper()
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Id int `json:"id"`
		} `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response body %q: %v", recorder.Body.String(), err)
	}
	if !response.Success {
		t.Fatalf("expected user detail response success, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	return response.Data.Id
}

func requireRoleAdoptionIDPresence(t *testing.T, ids []int, id int, want bool) {
	t.Helper()
	for _, current := range ids {
		if current == id {
			if !want {
				t.Fatalf("expected id %d to be absent from %v", id, ids)
			}
			return
		}
	}
	if want {
		t.Fatalf("expected id %d to be present in %v", id, ids)
	}
}

func assertRoleAdoptionAllowed(t *testing.T, r *gin.Engine, path string, user roleAdoptionUser) {
	t.Helper()
	recorder := performRoleAdoptionRequest(r, http.MethodGet, path, "", user)
	if !decodeRoleAdoptionSuccess(t, recorder) {
		t.Fatalf("expected role %s to access %s, status=%d body=%s", user.roleKey, path, recorder.Code, recorder.Body.String())
	}
}

func assertRoleAdoptionRejected(t *testing.T, r *gin.Engine, path string, user roleAdoptionUser) {
	t.Helper()
	recorder := performRoleAdoptionRequest(r, http.MethodGet, path, "", user)
	if decodeRoleAdoptionSuccess(t, recorder) {
		t.Fatalf("expected role %s to be rejected from %s, status=%d body=%s", user.roleKey, path, recorder.Code, recorder.Body.String())
	}
}

func assertRoleAdoptionMethodRejected(t *testing.T, r *gin.Engine, method string, path string, body string, user roleAdoptionUser) {
	t.Helper()
	recorder := performRoleAdoptionRequest(r, method, path, body, user)
	if decodeRoleAdoptionSuccess(t, recorder) {
		t.Fatalf("expected role %s to be rejected from %s %s, status=%d body=%s", user.roleKey, method, path, recorder.Code, recorder.Body.String())
	}
}

func TestRoleAuthReadRoutesAllowExpectedRoles(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	financeReadPaths := []string{
		"/api/log/",
		"/api/log/stat?start_timestamp=0&end_timestamp=1",
		"/api/user/topup",
		"/api/redemption/",
		"/api/redemption/search",
		"/api/redemption/1",
	}
	opsReadPaths := []string{
		"/api/task/",
		"/api/mj/",
	}

	for _, path := range financeReadPaths {
		t.Run("tenant_admin finance read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["tenant_admin"])
		})
		t.Run("finance finance read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["finance"])
		})
		t.Run("auditor finance read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["auditor"])
		})
		t.Run("root finance read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["root"])
		})
		t.Run("ops rejected finance read "+path, func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, path, roleAdoptionUsers["ops"])
		})
		t.Run("user rejected finance read "+path, func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, path, roleAdoptionUsers["user"])
		})
	}

	for _, path := range opsReadPaths {
		t.Run("tenant_admin ops read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["tenant_admin"])
		})
		t.Run("ops ops read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["ops"])
		})
		t.Run("auditor ops read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["auditor"])
		})
		t.Run("root ops read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["root"])
		})
		t.Run("finance rejected ops read "+path, func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, path, roleAdoptionUsers["finance"])
		})
		t.Run("user rejected ops read "+path, func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, path, roleAdoptionUsers["user"])
		})
	}
}

func TestRoleAuthPhase2ReadRoutesAllowExpectedRoles(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	channelReadPaths := []string{
		"/api/channel/",
		"/api/channel/search",
		"/api/channel/1",
		"/api/channel/models_enabled",
		"/api/channel/tag/models?tag=phase2",
	}
	subscriptionReadPath := "/api/subscription/admin/users/6/subscriptions"

	for _, path := range channelReadPaths {
		t.Run("tenant_admin channel read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["tenant_admin"])
		})
		t.Run("ops channel read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["ops"])
		})
		t.Run("auditor channel read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["auditor"])
		})
		t.Run("root channel read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["root"])
		})
		t.Run("finance rejected channel read "+path, func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, path, roleAdoptionUsers["finance"])
		})
		t.Run("user rejected channel read "+path, func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, path, roleAdoptionUsers["user"])
		})
	}

	for _, roleName := range []string{"tenant_admin", "finance", "auditor", "root"} {
		t.Run(roleName+" subscription read", func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, subscriptionReadPath, roleAdoptionUsers[roleName])
		})
	}
	for _, roleName := range []string{"ops", "user"} {
		t.Run(roleName+" rejected subscription read", func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, subscriptionReadPath, roleAdoptionUsers[roleName])
		})
	}
}

func TestRoleAuthPhase2TenantBoundaries(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	for _, roleName := range []string{"tenant_admin", "ops", "auditor"} {
		t.Run(roleName+" rejects tenant 2 channel", func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, "/api/channel/2", roleAdoptionUsers[roleName])
		})
	}
	assertRoleAdoptionAllowed(t, r, "/api/channel/2", roleAdoptionUsers["root"])

	for _, roleName := range []string{"tenant_admin", "finance", "auditor"} {
		t.Run(roleName+" rejects tenant 2 subscriptions", func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, "/api/subscription/admin/users/7/subscriptions", roleAdoptionUsers[roleName])
		})
	}
	assertRoleAdoptionAllowed(t, r, "/api/subscription/admin/users/7/subscriptions", roleAdoptionUsers["root"])
}

func TestRoleAuthPhase2DeferredRoutesRemainAdminOnly(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	deferredPaths := []string{
		"/api/channel/models",
		"/api/models/",
		"/api/group/",
		"/api/prefill_group/",
	}
	for _, path := range deferredPaths {
		for _, roleName := range []string{"finance", "ops", "auditor", "user"} {
			t.Run(roleName+" rejected deferred "+path, func(t *testing.T) {
				assertRoleAdoptionRejected(t, r, path, roleAdoptionUsers[roleName])
			})
		}
	}
}

func TestRoleAuthPhase3ReadRoutesAllowExpectedRoles(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	catalogReadPaths := []string{
		"/api/vendors/",
		"/api/vendors/search",
		"/api/vendors/1",
		"/api/models/missing",
	}
	billingReadPath := "/api/subscription/admin/plans"

	for _, path := range catalogReadPaths {
		t.Run("tenant_admin catalog read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["tenant_admin"])
		})
		t.Run("ops catalog read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["ops"])
		})
		t.Run("auditor catalog read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["auditor"])
		})
		t.Run("root catalog read "+path, func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, path, roleAdoptionUsers["root"])
		})
		t.Run("finance rejected catalog read "+path, func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, path, roleAdoptionUsers["finance"])
		})
		t.Run("user rejected catalog read "+path, func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, path, roleAdoptionUsers["user"])
		})
	}

	for _, roleName := range []string{"tenant_admin", "finance", "auditor", "root"} {
		t.Run(roleName+" billing read", func(t *testing.T) {
			assertRoleAdoptionAllowed(t, r, billingReadPath, roleAdoptionUsers[roleName])
		})
	}
	for _, roleName := range []string{"ops", "user"} {
		t.Run(roleName+" rejected billing read", func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, billingReadPath, roleAdoptionUsers[roleName])
		})
	}
}

func TestRoleAuthPhase3DeferredRoutesRemainAdminOnly(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	deferredPaths := []string{
		"/api/models/",
		"/api/models/search",
		"/api/models/1",
		"/api/models/sync_upstream/preview",
		"/api/group/",
		"/api/prefill_group/",
	}
	for _, path := range deferredPaths {
		for _, roleName := range []string{"finance", "ops", "auditor", "user"} {
			t.Run(roleName+" rejected deferred "+path, func(t *testing.T) {
				assertRoleAdoptionRejected(t, r, path, roleAdoptionUsers[roleName])
			})
		}
	}
}

func TestRoleAuthPhase3WriteRoutesRemainRootOnly(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	writeRoutes := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/api/models/", body: `{"model_name":"phase3-model"}`},
		{method: http.MethodPost, path: "/api/vendors/", body: `{"name":"phase3-vendor"}`},
		{method: http.MethodPost, path: "/api/subscription/admin/plans", body: `{"plan":{"title":"phase3 plan"}}`},
	}

	for _, route := range writeRoutes {
		for _, roleName := range []string{"tenant_admin", "finance", "ops", "auditor", "user"} {
			t.Run(roleName+" rejected write "+route.path, func(t *testing.T) {
				assertRoleAdoptionMethodRejected(t, r, route.method, route.path, route.body, roleAdoptionUsers[roleName])
			})
		}
	}
}

func TestOrganizationAdminUserReadRoutes(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	orgAdmin := roleAdoptionUsers["organization_admin"]
	orgAdminNoOrg := roleAdoptionUsers["organization_admin_0"]
	tenantAdmin := roleAdoptionUsers["tenant_admin"]
	root := roleAdoptionUsers["root"]

	recorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/user/", "", orgAdmin)
	ids := decodeRoleAdoptionUserListIDs(t, recorder)
	requireRoleAdoptionIDPresence(t, ids, roleAdoptionUsers["organization_admin"].id, true)
	requireRoleAdoptionIDPresence(t, ids, roleAdoptionUsers["organization_user"].id, true)
	requireRoleAdoptionIDPresence(t, ids, roleAdoptionUsers["other_organization"].id, false)
	requireRoleAdoptionIDPresence(t, ids, roleAdoptionUsers["tenant2_organization"].id, false)

	recorder = performRoleAdoptionRequest(r, http.MethodGet, "/api/user/search?keyword=organization", "", orgAdmin)
	ids = decodeRoleAdoptionUserListIDs(t, recorder)
	requireRoleAdoptionIDPresence(t, ids, roleAdoptionUsers["organization_user"].id, true)
	requireRoleAdoptionIDPresence(t, ids, roleAdoptionUsers["other_organization"].id, false)
	requireRoleAdoptionIDPresence(t, ids, roleAdoptionUsers["tenant2_organization"].id, false)

	recorder = performRoleAdoptionRequest(r, http.MethodGet, "/api/user/10", "", orgAdmin)
	if got := decodeRoleAdoptionUserDetailID(t, recorder); got != roleAdoptionUsers["organization_user"].id {
		t.Fatalf("organization_admin detail id = %d, want %d", got, roleAdoptionUsers["organization_user"].id)
	}
	assertRoleAdoptionRejected(t, r, "/api/user/11", orgAdmin)
	assertRoleAdoptionRejected(t, r, "/api/user/12", orgAdmin)
	assertRoleAdoptionRejected(t, r, "/api/user/", orgAdminNoOrg)
	assertRoleAdoptionRejected(t, r, "/api/user/search?keyword=organization", orgAdminNoOrg)
	assertRoleAdoptionRejected(t, r, "/api/user/10", orgAdminNoOrg)

	recorder = performRoleAdoptionRequest(r, http.MethodGet, "/api/user/", "", tenantAdmin)
	ids = decodeRoleAdoptionUserListIDs(t, recorder)
	requireRoleAdoptionIDPresence(t, ids, roleAdoptionUsers["organization_user"].id, true)
	requireRoleAdoptionIDPresence(t, ids, roleAdoptionUsers["other_organization"].id, true)
	requireRoleAdoptionIDPresence(t, ids, roleAdoptionUsers["tenant2_organization"].id, false)
	assertRoleAdoptionAllowed(t, r, "/api/user/11", tenantAdmin)
	assertRoleAdoptionRejected(t, r, "/api/user/12", tenantAdmin)

	assertRoleAdoptionAllowed(t, r, "/api/user/12", root)
	assertRoleAdoptionRejected(t, r, "/api/user/", roleAdoptionUsers["user"])
}

func TestOrganizationAdminCannotAccessUserWriteRoutes(t *testing.T) {
	r := setupRoleAdoptionRouter(t)
	orgAdmin := roleAdoptionUsers["organization_admin"]

	writeRoutes := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/api/user/", body: `{"username":"org-created","password":"password123","role":1}`},
		{method: http.MethodPut, path: "/api/user/", body: `{"id":10,"username":"organization_user","password":"password123","role":1}`},
		{method: http.MethodPost, path: "/api/user/manage", body: `{"id":10,"action":"disable"}`},
		{method: http.MethodDelete, path: "/api/user/10", body: ""},
		{method: http.MethodDelete, path: "/api/user/10/reset_passkey", body: ""},
		{method: http.MethodDelete, path: "/api/user/10/2fa", body: ""},
		{method: http.MethodDelete, path: "/api/user/10/bindings/email", body: ""},
		{method: http.MethodDelete, path: "/api/user/10/oauth/bindings/1", body: ""},
	}

	for _, route := range writeRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			assertRoleAdoptionMethodRejected(t, r, route.method, route.path, route.body, orgAdmin)
		})
	}
}

func TestRootAuthRoutesStillRejectNonRootRoles(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	for _, roleName := range []string{"tenant_admin", "finance", "ops", "auditor"} {
		t.Run(roleName, func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, "/api/status/test", roleAdoptionUsers[roleName])
		})
	}
	assertRoleAdoptionAllowed(t, r, "/api/status/test", roleAdoptionUsers["root"])
}
