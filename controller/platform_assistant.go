package controller

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"unicode/utf8"
)

// PlatformAssistantContext retrieves only public documentation and optional web
// summaries. Model generation still goes through the normal authenticated relay.
func PlatformAssistantContext(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var request struct {
		Query string `json:"query"`
		Web   bool   `json:"web"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || strings.TrimSpace(request.Query) == "" || utf8.RuneCountInString(request.Query) > 4000 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "问题不能为空且不能超过4000字"})
		return
	}
	result, err := service.BuildAssistantContext(c.Request.Context(), request.Query, request.Web)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "联网检索未完成，请稍后重试或关闭联网检索；请勿在搜索问题中包含密钥。"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
