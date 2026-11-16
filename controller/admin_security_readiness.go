package controller

import (
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

func AdminSecurityReadiness(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	result, err := model.GetAdminSecurityReadiness()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "管理员认证状态暂时不可用，请稍后重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
