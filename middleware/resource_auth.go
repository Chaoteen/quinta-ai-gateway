package middleware

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ResourceAuth is an action gate, not an ownership filter. Non-platform grants
// must also pass controller/model ownership checks before querying or mutating.
func ResourceAuth(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHelper(c, common.RoleCommonUser, false)
		if c.IsAborted() {
			return
		}
		scope := model.AccessScopeFromContext(c)
		allowed := common.ResourceAllowed(scope.RoleKey, resource, action)
		if value, exists := c.Get("rbac_state"); exists {
			state, ok := value.(model.RBACState)
			allowed = ok && common.GrantAllowed(state.Grants[resource], action)
		}
		if !allowed || (!scope.IsRoot && scope.TenantId <= 0) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
			return
		}
		c.Next()
	}
}
