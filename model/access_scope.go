package model

import (
	"fmt"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AccessScope struct {
	TenantId       int
	OrganizationId int
	DepartmentId   int
	RoleKey        string
	IsRoot         bool
}

func AccessScopeFromContext(c *gin.Context) AccessScope {
	if c == nil {
		return AccessScope{TenantId: normalizeTenantId(0), RoleKey: common.RoleKeyUser}
	}
	roleKey := common.GetContextKeyString(c, constant.ContextKeyUserRoleKey)
	if roleKey == "" && c != nil {
		roleKey = common.LegacyRoleToRoleKey(c.GetInt("role"))
	}
	roleKey = common.NormalizeRoleKey(roleKey)
	scope := AccessScope{
		TenantId: normalizeTenantId(common.GetContextKeyInt(c, constant.ContextKeyTenantId)),
		RoleKey:  roleKey,
		IsRoot:   common.IsRootRole(roleKey) || (c != nil && c.GetInt("role") == common.RoleRootUser),
	}
	if scope.IsRoot {
		return scope
	}
	if common.IsOrganizationAdminRole(roleKey) {
		scope.OrganizationId = common.GetContextKeyInt(c, constant.ContextKeyOrganizationId)
		return scope
	}
	return scope
}

func ApplyOwnershipScope(db *gorm.DB, tableAliasOrName string, scope AccessScope) *gorm.DB {
	if scope.IsRoot {
		return db
	}
	if common.IsOrganizationAdminRole(scope.RoleKey) && scope.OrganizationId <= 0 {
		return db.Where("1 = 0")
	}
	scope.TenantId = normalizeTenantId(scope.TenantId)
	db = db.Where(ownershipScopeColumn(tableAliasOrName, "tenant_id")+" = ?", scope.TenantId)
	if scope.OrganizationId > 0 {
		db = db.Where(ownershipScopeColumn(tableAliasOrName, "organization_id")+" = ?", scope.OrganizationId)
	}
	if scope.DepartmentId > 0 {
		db = db.Where(ownershipScopeColumn(tableAliasOrName, "department_id")+" = ?", scope.DepartmentId)
	}
	return db
}

func AllowsOwnership(scope AccessScope, tenantId int, organizationId int, departmentId int) bool {
	if scope.IsRoot {
		return true
	}
	if common.IsOrganizationAdminRole(scope.RoleKey) && scope.OrganizationId <= 0 {
		return false
	}
	if normalizeTenantId(scope.TenantId) != normalizeTenantId(tenantId) {
		return false
	}
	if scope.OrganizationId > 0 && scope.OrganizationId != organizationId {
		return false
	}
	if scope.DepartmentId > 0 && scope.DepartmentId != departmentId {
		return false
	}
	return true
}

func ownershipScopeColumn(tableAliasOrName string, column string) string {
	if tableAliasOrName == "" {
		return column
	}
	return fmt.Sprintf("%s.%s", tableAliasOrName, column)
}
