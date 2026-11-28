package controller

import (
	"errors"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func rbacReply(c *gin.Context, data interface{}, err error) {
	c.Header("Cache-Control", "no-store")
	if err == nil {
		common.ApiSuccess(c, data)
		return
	}
	if errors.Is(err, model.ErrRBACDenied) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}
	if errors.Is(err, model.ErrRBACConflict) {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"success": false, "message": "Role changed. Refresh and retry."})
		return
	}
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Role operation unavailable"})
}
func rbacDecode(c *gin.Context, data interface{}) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	if err := common.DecodeJson(c.Request.Body, data); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return false
	}
	return true
}
func RBACWorkspace(c *gin.Context) {
	tenant := 0
	if raw, exists := c.GetQuery("tenant_id"); exists {
		var err error
		tenant, err = strconv.Atoi(raw)
		if err != nil || tenant < 1 {
			c.AbortWithStatus(400)
			return
		}
	}
	data, err := model.GetRBACWorkspace(c.GetInt("id"), tenant)
	rbacReply(c, data, err)
}
func SaveRBACPolicy(c *gin.Context) {
	var input struct {
		Family   string                          `json:"family"`
		Grants   map[string]common.ResourceGrant `json:"grants"`
		Delegate bool                            `json:"delegate"`
		Version  int                             `json:"version"`
	}
	if !rbacDecode(c, &input) {
		return
	}
	row, err := model.SaveRBACPolicy(c.GetInt("id"), input.Family, input.Grants, input.Delegate, input.Version)
	rbacReply(c, row, err)
}
func SaveScopedRole(c *gin.Context) {
	var input model.ScopedRoleInput
	if !rbacDecode(c, &input) {
		return
	}
	row, err := model.SaveScopedRole(c.GetInt("id"), input)
	rbacReply(c, row, err)
}
func AssignScopedRole(c *gin.Context) {
	var input struct {
		UserId int `json:"user_id"`
		RoleId int `json:"role_id"`
	}
	if !rbacDecode(c, &input) {
		return
	}
	if input.UserId <= 0 || input.RoleId <= 0 {
		c.AbortWithStatus(400)
		return
	}
	err := model.AssignScopedRole(c.GetInt("id"), input.UserId, input.RoleId)
	if err == nil {
		_ = model.InvalidateUserCache(input.UserId)
	}
	rbacReply(c, nil, err)
}
func RevokeScopedRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.AbortWithStatus(400)
		return
	}
	err = model.RevokeScopedRole(c.GetInt("id"), id)
	if err == nil {
		_ = model.InvalidateUserCache(id)
	}
	rbacReply(c, nil, err)
}
