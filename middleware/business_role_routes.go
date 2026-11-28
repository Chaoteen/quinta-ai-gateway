package middleware

import (
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
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
	case path == "/pg/chat/completions":
		resource = "model_calls"
		action = "use"
	case path == "/api/rbac/workspace":
		return c.Request.Method == "GET"
	case strings.HasPrefix(path, "/api/rbac/custom-roles"), strings.HasPrefix(path, "/api/rbac/assignments"):
		value, exists := c.Get("rbac_state")
		state, ok := value.(model.RBACState)
		return exists && ok && state.CanDelegate
	case path == "/api/rbac/self", path == "/api/user/self", path == "/api/user/self/groups", path == "/api/user/models", path == "/api/models":
		return true
	case strings.HasPrefix(path, "/api/user/2fa/"), strings.HasPrefix(path, "/api/user/passkey"), path == "/api/verify", path == "/api/user/setting", path == "/api/user/self/chat-preferences":
		return true
	case path == "/api/user/self/chat-key":
		resource = "keys"
		action = "create"
	case path == "/api/user/self/assistant/context":
		resource = "keys"
		action = "use"
	case path == "/api/token/:id/key", path == "/api/token/batch/keys":
		resource = "keys"
		action = "use"
	case path == "/api/token/batch":
		resource = "keys"
		action = "delete"
	case path == "/api/invoices/profiles/:id/disable":
		resource = "invoice_requests"
		action = "operate"
	case path == "/api/user/topup/info", path == "/api/user/topup/self", path == "/api/user/aff", path == "/api/user/amount", path == "/api/user/stripe/amount", path == "/api/user/pay", path == "/api/user/stripe/pay", path == "/api/user/creem/pay", path == "/api/user/waffo/amount", path == "/api/user/waffo/pay":
		resource = "payments"
	case path == "/api/user/topup":
		if c.Request.Method != "POST" {
			return false
		}
		resource = "vouchers"
		action = "use"
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
	if value, exists := c.Get("rbac_state"); exists {
		state, ok := value.(model.RBACState)
		if !ok {
			return false
		}
		grant, exists := state.Grants[resource]
		if path == "/api/user/self/chat-key" && !common.GrantAllowed(state.Grants["keys"], "use") {
			return false
		}
		return exists && common.GrantAllowed(grant, action)
	}
	return common.ResourceAllowed(role, resource, action)
}
