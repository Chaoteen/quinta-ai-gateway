package middleware

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTenantSuspensionSessionAndRecovery(t *testing.T) {
	for _, auth := range []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{"dashboard", UserAuth()}, {"mixed", TokenOrUserAuth()},
	} {
		t.Run(auth.name, func(t *testing.T) {
			r := setupAuthRBACTestRouter(t, auth.handler)
			tenant := model.Tenant{Id: 1, Name: "platform", Status: 1}
			if err := model.DB.Create(&tenant).Error; err != nil {
				t.Fatal(err)
			}
			for _, status := range []int{1, 2, 1} {
				if err := model.DB.Model(&tenant).Update("status", status).Error; err != nil {
					t.Fatal(err)
				}
				want := http.StatusOK
				if status != 1 {
					want = http.StatusForbidden
				}
				// A stale root role in the session must not bypass a user's tenant suspension.
				got := performAuthRBACTestRequest(r, 3, common.RoleRootUser)
				if got.Code != want {
					t.Fatalf("status %d: got %d want %d", status, got.Code, want)
				}
				if got := performAuthRBACTestRequest(r, 1, common.RoleRootUser); got.Code != http.StatusOK {
					t.Fatalf("root recovery blocked: %d", got.Code)
				}
			}
			if err := model.DB.Delete(&tenant).Error; err != nil {
				t.Fatal(err)
			}
			if got := performAuthRBACTestRequest(r, 3, common.RoleCommonUser); got.Code != http.StatusForbidden {
				t.Fatalf("deleted tenant accepted: %d", got.Code)
			}
		})
	}
}

func TestTenantSuspensionAPIKeys(t *testing.T) {
	for _, auth := range []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{"relay", TokenAuth()}, {"read-only", TokenAuthReadOnly()}, {"mixed-token", TokenOrUserAuth()},
	} {
		t.Run(auth.name, func(t *testing.T) {
			r := setupAuthRBACTestRouter(t, auth.handler)
			tenant := model.Tenant{Id: 1, Name: "platform", Status: 1}
			if err := model.DB.Create(&tenant).Error; err != nil {
				t.Fatal(err)
			}
			// Root-owned API keys must obey suspension too.
			token := model.Token{UserId: 1, TenantId: 1, Key: "tenanttestkey", Status: common.TokenStatusEnabled, UnlimitedQuota: true, ExpiredTime: -1}
			if err := model.DB.Create(&token).Error; err != nil {
				t.Fatal(err)
			}
			for _, status := range []int{1, 2, 1} {
				if err := model.DB.Model(&tenant).Update("status", status).Error; err != nil {
					t.Fatal(err)
				}
				req := httptest.NewRequest(http.MethodGet, "/protected", nil)
				req.Header.Set("Authorization", "Bearer sk-tenanttestkey")
				rec := httptest.NewRecorder()
				r.ServeHTTP(rec, req)
				want := http.StatusOK
				if status != 1 {
					want = http.StatusForbidden
				}
				if rec.Code != want {
					t.Fatalf("status %d: got %d want %d body=%s", status, rec.Code, want, rec.Body.String())
				}
			}
		})
	}
}

func TestTenantAccessMissingAndDatabaseFailure(t *testing.T) {
	r := setupAuthRBACTestRouter(t, UserAuth())
	// Legacy implicit platform tenant remains compatible; unknown enterprises fail closed.
	if got := performAuthRBACTestRequest(r, 3, common.RoleCommonUser); got.Code != http.StatusOK {
		t.Fatalf("legacy platform blocked: %d", got.Code)
	}
	if err := model.DB.Model(&model.User{}).Where("id = ?", 3).Update("tenant_id", 99).Error; err != nil {
		t.Fatal(err)
	}
	if got := performAuthRBACTestRequest(r, 3, common.RoleCommonUser); got.Code != http.StatusForbidden {
		t.Fatalf("unknown enterprise accepted: %d", got.Code)
	}
	if err := model.DB.Migrator().DropTable(&model.Tenant{}); err != nil {
		t.Fatal(err)
	}
	if got := performAuthRBACTestRequest(r, 3, common.RoleCommonUser); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("database failure accepted: %d", got.Code)
	}
}
