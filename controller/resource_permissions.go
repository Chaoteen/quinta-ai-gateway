package controller

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
)

func GetResourcePermissions(c *gin.Context) {
	state, err := model.EffectiveRBAC(c.GetInt("id"))
	rbacReply(c, state, err)
}

func GetRolePermissionMatrix(c *gin.Context) {
	roles := []string{common.RoleKeyRoot, common.RoleKeySupplierAdmin, common.RoleKeyDistributorAdmin, common.RoleKeyEnterpriseAdmin, common.RoleKeyUser}
	matrix := map[string]map[string]common.ResourceGrant{}
	for _, role := range roles {
		matrix[role] = common.ResourcePermissions(role)
	}
	c.Header("Cache-Control", "no-store")
	common.ApiSuccess(c, matrix)
}

func AssignResourceRole(c *gin.Context) {
	var input struct {
		UserId int    `json:"user_id"`
		Role   string `json:"role_key"`
	}
	if err := common.DecodeJson(c.Request.Body, &input); err != nil || input.UserId < 1 {
		c.AbortWithStatus(400)
		return
	}
	if err := model.AssignBusinessRole(c.GetInt("id"), input.UserId, input.Role); err != nil {
		c.JSON(409, gin.H{"success": false, "message": err.Error()})
		return
	}
	// DB auth_epoch is authoritative even if cache invalidation is unavailable.
	_ = model.InvalidateUserCache(input.UserId)
	common.ApiSuccess(c, nil)
}
