package middleware

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestBusinessRoleRouteActionIsolation(t *testing.T) {
	for _, tc := range []struct {
		role, method, path string
		allow              bool
	}{
		{common.RoleKeyEnterpriseAdmin, "GET", "/api/user", true},
		{common.RoleKeyEnterpriseAdmin, "GET", "/api/channel", false},
		{common.RoleKeyEnterpriseAdmin, "POST", "/api/admin/invoices/applications/:id/issue", false},
		{common.RoleKeyEnterpriseAdmin, "POST", "/api/invoices/applications", true},
		{common.RoleKeySupplierAdmin, "POST", "/api/channel", true},
		{common.RoleKeySupplierAdmin, "DELETE", "/api/channel/:id", false},
		{common.RoleKeyDistributorAdmin, "PUT", "/api/option", false},
		{common.RoleKeyDistributorAdmin, "POST", "/api/user/manage", false},
		{common.RoleKeyEnterpriseAdmin, "DELETE", "/api/payment/orders/:id", false},
	} {
		r := gin.New()
		r.Handle(tc.method, tc.path, func(c *gin.Context) {
			if businessRouteGrant(c, tc.role) != tc.allow {
				t.Errorf("%s %s %s", tc.role, tc.method, tc.path)
			}
		})
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(tc.method, tc.path, nil))
	}
}
