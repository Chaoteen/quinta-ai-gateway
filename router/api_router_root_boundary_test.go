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

type rootBoundaryResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func setupRootBoundaryRouter(t *testing.T) *gin.Engine {
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
		&model.TwoFA{},
		&model.Log{},
		&model.QuotaData{},
		&model.Channel{},
		&model.Ability{},
		&model.Model{},
		&model.Vendor{},
	); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	seedRootBoundaryUser(t, db, 1, "root", common.RoleRootUser, 1)
	seedRootBoundaryUser(t, db, 2, "tenant-admin", common.RoleAdminUser, 2)

	r := gin.New()
	store := cookie.NewStore([]byte("root-boundary-test-secret"))
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

func seedRootBoundaryUser(t *testing.T, db *gorm.DB, id int, username string, role int, tenantID int) {
	t.Helper()
	user := model.User{
		Id:          id,
		TenantId:    tenantID,
		Username:    username,
		Password:    "password123",
		DisplayName: username,
		Role:        role,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     fmt.Sprintf("aff-%d", id),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to seed user %d: %v", id, err)
	}
}

func performRootBoundaryRequest(r *gin.Engine, method string, target string, body string, userID int, role int) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("New-Api-User", strconv.Itoa(userID))
	req.Header.Set("X-Test-User-ID", strconv.Itoa(userID))
	req.Header.Set("X-Test-Role", strconv.Itoa(role))
	r.ServeHTTP(recorder, req)
	return recorder
}

func decodeRootBoundaryResponse(t *testing.T, recorder *httptest.ResponseRecorder) rootBoundaryResponse {
	t.Helper()
	var response rootBoundaryResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response body %q: %v", recorder.Body.String(), err)
	}
	return response
}

func assertRootBoundaryAllowsRoot(t *testing.T, r *gin.Engine, method string, target string, body string) {
	t.Helper()
	recorder := performRootBoundaryRequest(r, method, target, body, 1, common.RoleRootUser)
	if recorder.Code == http.StatusUnauthorized || recorder.Code == http.StatusForbidden {
		t.Fatalf("root request was auth-blocked with status %d body %s", recorder.Code, recorder.Body.String())
	}
	response := decodeRootBoundaryResponse(t, recorder)
	if !response.Success {
		t.Fatalf("root request did not reach a successful handler, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func assertRootBoundaryRejectsAdmin(t *testing.T, r *gin.Engine, method string, target string, body string) {
	t.Helper()
	recorder := performRootBoundaryRequest(r, method, target, body, 2, common.RoleAdminUser)
	response := decodeRootBoundaryResponse(t, recorder)
	if response.Success {
		t.Fatalf("tenant admin unexpectedly accessed root-only route, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRootBoundaryRoutesRequireRoot(t *testing.T) {
	r := setupRootBoundaryRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "2fa stats", method: http.MethodGet, path: "/api/user/2fa/stats"},
		{name: "channel fix", method: http.MethodPost, path: "/api/channel/fix"},
		{name: "delete logs", method: http.MethodDelete, path: "/api/log/?target_timestamp=1"},
		{name: "quota data", method: http.MethodGet, path: "/api/data/?start_timestamp=0&end_timestamp=1"},
		{name: "create model", method: http.MethodPost, path: "/api/models/", body: `{"model_name":"root-boundary-model","status":1,"sync_official":1}`},
		{name: "create vendor", method: http.MethodPost, path: "/api/vendors/", body: `{"name":"root-boundary-vendor","status":1}`},
		{name: "deployments settings", method: http.MethodGet, path: "/api/deployments/settings"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/root", func(t *testing.T) {
			assertRootBoundaryAllowsRoot(t, r, tt.method, tt.path, tt.body)
		})
		t.Run(tt.name+"/admin", func(t *testing.T) {
			assertRootBoundaryRejectsAdmin(t, r, tt.method, tt.path, tt.body)
		})
	}
}

func TestTenantScopedAdminRouteStillAllowsAdmin(t *testing.T) {
	r := setupRootBoundaryRouter(t)

	recorder := performRootBoundaryRequest(r, http.MethodGet, "/api/log/stat?start_timestamp=0&end_timestamp=1", "", 2, common.RoleAdminUser)
	response := decodeRootBoundaryResponse(t, recorder)
	if !response.Success {
		t.Fatalf("tenant scoped admin route was not accessible to admin, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
