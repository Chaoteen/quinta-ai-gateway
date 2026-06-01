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
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type roleAdoptionUser struct {
	id      int
	role    int
	roleKey string
}

var roleAdoptionUsers = map[string]roleAdoptionUser{
	"root":         {id: 1, role: common.RoleRootUser, roleKey: common.RoleKeyRoot},
	"tenant_admin": {id: 2, role: common.RoleAdminUser, roleKey: common.RoleKeyTenantAdmin},
	"finance":      {id: 3, role: common.RoleCommonUser, roleKey: common.RoleKeyFinance},
	"ops":          {id: 4, role: common.RoleCommonUser, roleKey: common.RoleKeyOps},
	"auditor":      {id: 5, role: common.RoleCommonUser, roleKey: common.RoleKeyAuditor},
	"user":         {id: 6, role: common.RoleCommonUser, roleKey: common.RoleKeyUser},
}

func setupRoleAdoptionRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	if err := db.AutoMigrate(
		&model.Tenant{},
		&model.User{},
		&model.Log{},
		&model.TopUp{},
		&model.Redemption{},
		&model.Task{},
		&model.Midjourney{},
	); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	for name, user := range roleAdoptionUsers {
		seedRoleAdoptionUser(t, db, user.id, name, user.role, user.roleKey)
	}
	seedRoleAdoptionRedemption(t, db)

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

func seedRoleAdoptionUser(t *testing.T, db *gorm.DB, id int, username string, role int, roleKey string) {
	t.Helper()
	user := model.User{
		Id:          id,
		TenantId:    1,
		Username:    username,
		Password:    "password123",
		DisplayName: username,
		Role:        role,
		RoleKey:     roleKey,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     fmt.Sprintf("role-adoption-%d", id),
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

func TestRootAuthRoutesStillRejectNonRootRoles(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	for _, roleName := range []string{"tenant_admin", "finance", "ops", "auditor"} {
		t.Run(roleName, func(t *testing.T) {
			assertRoleAdoptionRejected(t, r, "/api/status/test", roleAdoptionUsers[roleName])
		})
	}
	assertRoleAdoptionAllowed(t, r, "/api/status/test", roleAdoptionUsers["root"])
}
