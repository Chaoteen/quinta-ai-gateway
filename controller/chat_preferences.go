package controller

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/Chaoteen/quinta-ai-gateway/setting"
	"github.com/gin-gonic/gin"
	"net/http"
)

func UpdateChatPreferences(c *gin.Context) {
	var req struct {
		Apps *[]string `json:"enabled_chat_apps"`
	}
	if c.ShouldBindJSON(&req) != nil || req.Apps == nil || len(*req.Apps) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid chat app selection"})
		return
	}
	allowed := map[string]bool{}
	for _, entry := range setting.Chats {
		for name := range entry {
			allowed[name] = true
		}
	}
	selected := make([]string, 0, len(*req.Apps))
	seen := map[string]bool{}
	for _, name := range *req.Apps {
		if !allowed[name] {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Unknown chat app"})
			return
		}
		if !seen[name] {
			selected = append(selected, name)
			seen[name] = true
		}
	}
	if err := model.UpdateChatPreferences(c.GetInt("id"), selected); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"enabled_chat_apps": selected})
}

func ConfigureChatKey(c *gin.Context) {
	var req struct {
		App     string `json:"app"`
		TokenId int    `json:"token_id"`
		Create  bool   `json:"create"`
	}
	if c.ShouldBindJSON(&req) != nil || req.App == "" || (!req.Create && req.TokenId <= 0) || (req.Create && req.TokenId != 0) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid chat key configuration"})
		return
	}
	allowed := false
	for _, entry := range setting.Chats {
		if _, ok := entry[req.App]; ok {
			allowed = true
		}
	}
	if !allowed {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Unknown chat app"})
		return
	}
	tokenId, preferences, err := model.ConfigureChatKey(c.GetInt("id"), req.App, req.TokenId, req.Create)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"token_id": tokenId, "enabled_chat_apps": preferences.EnabledChatApps, "chat_key_bindings": preferences.ChatKeyBindings})
}
