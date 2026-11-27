package router

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"net/http"
	"strings"
	"testing"
)

func TestResourceMatrixAndAssignmentBoundary(t *testing.T) {
	r := setupRootBoundaryRouter(t)
	for _, path := range []string{"/api/rbac/self", "/api/rbac/roles"} {
		rec := performRootBoundaryRequest(r, http.MethodGet, path, "", 1, common.RoleRootUser)
		if rec.Code != 200 || !decodeRootBoundaryResponse(t, rec).Success {
			t.Fatalf("root denied %s", path)
		}
	}
	rec := performRootBoundaryRequest(r, http.MethodGet, "/api/rbac/roles", "", 2, common.RoleAdminUser)
	if decodeRootBoundaryResponse(t, rec).Success {
		t.Fatal("tenant saw platform roles")
	}
	rec = performRootBoundaryRequest(r, http.MethodPost, "/api/rbac/assign", `{"user_id":2,"role_key":"enterprise_admin"}`, 1, common.RoleRootUser)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "VERIFICATION_REQUIRED") {
		t.Fatalf("assignment without step-up: %d", rec.Code)
	}
}
