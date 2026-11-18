package controller

import (
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
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

func AdminCredentialResetAudits(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	before := 0
	if raw, ok := c.GetQuery("before"); ok {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid audit cursor"})
			return
		}
		before = n
	}
	rows, more, err := model.ListCredentialResetAudits(before)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "audit unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": rows, "has_more": more}})
}
