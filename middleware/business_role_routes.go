package middleware

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/gin-gonic/gin"
	"strings"
)

func businessRole(role string) bool {
	return role == common.RoleKeySupplierAdmin || role == common.RoleKeyDistributorAdmin || role == common.RoleKeyEnterpriseAdmin
}

// Explicit route patterns prevent a new admin-class role from inheriting old
// numeric AdminAuth endpoints. Unknown routes fail closed for business admins.
func businessRouteGrant(c *gin.Context, role string) bool {
	path := strings.TrimSuffix(c.FullPath(), "/")
	action := "view"
	switch c.Request.Method {
	case "POST":
		action = "create"
	case "PUT", "PATCH":
		action = "update"
	case "DELETE":
		action = "delete"
	}
	resource := ""
	switch {
	case path == "/api/rbac/self", path == "/api/user/self", path == "/api/user/self/groups", path == "/api/user/models", path == "/api/models":
		return true
	case strings.HasPrefix(path, "/api/user/2fa/"), strings.HasPrefix(path, "/api/user/passkey"), path == "/api/verify", path == "/api/user/setting", path == "/api/user/self/chat-preferences", path == "/api/user/self/chat-key", path == "/api/user/self/assistant/context":
		return true
	case path == "/api/user/topup/info", path == "/api/user/topup/self", path == "/api/user/aff", path == "/api/user/amount", path == "/api/user/stripe/amount", path == "/api/user/pay", path == "/api/user/stripe/pay", path == "/api/user/creem/pay", path == "/api/user/waffo/amount", path == "/api/user/waffo/pay":
		resource = "payments"
	case path == "/api/user/topup":
		if c.Request.Method != "POST" {
			return false
		}
		resource = "payments"
	case strings.HasPrefix(path, "/api/token"):
		resource = "keys"
	case strings.HasPrefix(path, "/api/billing/"), strings.HasPrefix(path, "/api/payment/"):
		resource = "payments"
	case strings.HasPrefix(path, "/api/invoices/"):
		resource = "invoice_requests"
	case strings.HasPrefix(path, "/api/vouchers/"):
		resource = "vouchers"
		if c.Request.Method == "POST" {
			action = "use"
		}
	case strings.HasPrefix(path, "/api/subscription/") && !strings.HasPrefix(path, "/api/subscription/admin/"):
		resource = "subscriptions"
	case path == "/api/channel", path == "/api/channel/:id", path == "/api/channel/search", path == "/api/channel/models_enabled":
		resource = "supply_models"
	// Member list/detail/edit controllers enforce the tenant boundary. Credential
	// resets and role promotion endpoints remain outside this delegated surface.
	case path == "/api/user", path == "/api/user/search", path == "/api/user/:id":
		resource = "members"
	default:
		return false
	}
	return common.ResourceAllowed(role, resource, action)
}
