package middleware

import (
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

func setupAuthRBACTestRouter(t *testing.T, auth gin.HandlerFunc) *gin.Engine {
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

	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	seedAuthRBACTestUser(t, db, 1, "root", common.RoleRootUser)
	seedAuthRBACTestUser(t, db, 2, "tenant-admin", common.RoleAdminUser)
	seedAuthRBACTestUser(t, db, 3, "user", common.RoleCommonUser)

	r := gin.New()
	store := cookie.NewStore([]byte("auth-rbac-test-secret"))
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
	r.GET("/protected", auth, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	return r
}

func seedAuthRBACTestUser(t *testing.T, db *gorm.DB, id int, username string, role int) {
	t.Helper()
	user := model.User{
		Id:          id,
		TenantId:    1,
		Username:    username,
		Password:    "password123",
		DisplayName: username,
		Role:        role,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     fmt.Sprintf("auth-rbac-%d", id),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to seed user %d: %v", id, err)
	}
}

func performAuthRBACTestRequest(r *gin.Engine, userID int, role int) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("New-Api-User", strconv.Itoa(userID))
	req.Header.Set("X-Test-User-ID", strconv.Itoa(userID))
	req.Header.Set("X-Test-Role", strconv.Itoa(role))
	r.ServeHTTP(recorder, req)
	return recorder
}

func authRBACTestSuccess(t *testing.T, recorder *httptest.ResponseRecorder) bool {
	t.Helper()
	var response struct {
		Success bool `json:"success"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response %q: %v", recorder.Body.String(), err)
	}
	return response.Success
}

func TestRoleAuthRootOnlyAllowsRoot(t *testing.T) {
	r := setupAuthRBACTestRouter(t, RoleAuth(common.RoleKeyRoot))

	if !authRBACTestSuccess(t, performAuthRBACTestRequest(r, 1, common.RoleRootUser)) {
		t.Fatal("root should pass RoleAuth(root)")
	}
	if authRBACTestSuccess(t, performAuthRBACTestRequest(r, 2, common.RoleAdminUser)) {
		t.Fatal("tenant admin should not pass RoleAuth(root)")
	}
	if authRBACTestSuccess(t, performAuthRBACTestRequest(r, 3, common.RoleCommonUser)) {
		t.Fatal("user should not pass RoleAuth(root)")
	}
}

func TestRoleAuthTenantAdmin(t *testing.T) {
	r := setupAuthRBACTestRouter(t, RoleAuth(common.RoleKeyTenantAdmin))

	if !authRBACTestSuccess(t, performAuthRBACTestRequest(r, 2, common.RoleAdminUser)) {
		t.Fatal("tenant admin should pass RoleAuth(tenant_admin)")
	}
	if authRBACTestSuccess(t, performAuthRBACTestRequest(r, 3, common.RoleCommonUser)) {
		t.Fatal("user should not pass RoleAuth(tenant_admin)")
	}
}

func TestLegacyAuthMiddlewareBehaviorUnchanged(t *testing.T) {
	tests := []struct {
		name    string
		auth    gin.HandlerFunc
		userID  int
		role    int
		allowed bool
	}{
		{name: "RootAuth allows root", auth: RootAuth(), userID: 1, role: common.RoleRootUser, allowed: true},
		{name: "RootAuth rejects admin", auth: RootAuth(), userID: 2, role: common.RoleAdminUser, allowed: false},
		{name: "AdminAuth allows root", auth: AdminAuth(), userID: 1, role: common.RoleRootUser, allowed: true},
		{name: "AdminAuth allows admin", auth: AdminAuth(), userID: 2, role: common.RoleAdminUser, allowed: true},
		{name: "AdminAuth rejects user", auth: AdminAuth(), userID: 3, role: common.RoleCommonUser, allowed: false},
		{name: "UserAuth allows root", auth: UserAuth(), userID: 1, role: common.RoleRootUser, allowed: true},
		{name: "UserAuth allows admin", auth: UserAuth(), userID: 2, role: common.RoleAdminUser, allowed: true},
		{name: "UserAuth allows user", auth: UserAuth(), userID: 3, role: common.RoleCommonUser, allowed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupAuthRBACTestRouter(t, tt.auth)
			got := authRBACTestSuccess(t, performAuthRBACTestRequest(r, tt.userID, tt.role))
			if got != tt.allowed {
				t.Fatalf("allowed = %v, want %v", got, tt.allowed)
			}
		})
	}
}
