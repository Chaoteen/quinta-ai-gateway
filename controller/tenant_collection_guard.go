package controller

import (
	"net/http"

	"github.com/Chaoteen/quinta-ai-gateway/service"
	"github.com/gin-gonic/gin"
)

// Only creation is blocked. Historical provider callbacks must remain available
// during a controlled migration so real payments are not silently abandoned.
func rejectLegacyPlatformCollection(c *gin.Context) bool {
	if !service.TenantIndependentCollectionEnabled() {
		return false
	}
	c.AbortWithStatusJSON(http.StatusConflict, gin.H{
		"success": false, "message": "Tenant-owned receiving accounts are required; the global payment route is unavailable.",
	})
	return true
}
