package middleware

import (
	"net/http"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
)

// Only current platform root dashboard access may bypass suspension for recovery.
// API keys, including root-owned keys, never bypass tenant suspension.
func requireActiveTenant(c *gin.Context, platformRecovery bool) bool {
	if platformRecovery {
		return true
	}
	enabled, err := model.TenantAccessEnabled(common.GetContextKeyInt(c, constant.ContextKeyTenantId))
	status, message, code := http.StatusForbidden, "企业账户已停用或不可用，请联系平台管理员", "tenant_unavailable"
	if err != nil {
		common.SysLog("tenant authorization database error: " + err.Error())
		status, message, code = http.StatusServiceUnavailable, "暂时无法验证企业账户状态，请稍后重试", "tenant_check_unavailable"
	}
	if err == nil && enabled {
		return true
	}
	c.AbortWithStatusJSON(status, gin.H{
		"success": false, "message": message,
		"error": gin.H{"message": message, "type": "access_denied", "code": code},
	})
	return false
}
