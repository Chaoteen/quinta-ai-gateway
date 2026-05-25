package model

import (
	"errors"
	"fmt"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TenantScope struct {
	TenantId int
	IsRoot   bool
}

func normalizeTenantId(tenantId int) int {
	if tenantId == 0 {
		return 1
	}
	return tenantId
}

func TenantScopeFromContext(c *gin.Context) TenantScope {
	scope := TenantScope{
		TenantId: common.GetContextKeyInt(c, constant.ContextKeyTenantId),
		IsRoot:   c.GetInt("role") == common.RoleRootUser,
	}
	scope.TenantId = normalizeTenantId(scope.TenantId)
	return scope
}

// RelayTenantScopeFromContext rejects relay selection without an authenticated
// tenant context. Legacy user ownership is normalized before it reaches here.
func RelayTenantScopeFromContext(c *gin.Context) (TenantScope, error) {
	if c == nil {
		return TenantScope{}, errors.New("relay tenant context is missing")
	}
	scope := TenantScope{
		TenantId: common.GetContextKeyInt(c, constant.ContextKeyTenantId),
		IsRoot:   c.GetInt("role") == common.RoleRootUser,
	}
	if scope.IsRoot {
		scope.TenantId = normalizeTenantId(scope.TenantId)
		return scope, nil
	}
	if scope.TenantId == 0 {
		return TenantScope{}, errors.New("relay tenant context is missing")
	}
	return scope, nil
}

func (scope TenantScope) AllowsTenant(tenantId int) bool {
	if scope.IsRoot {
		return true
	}
	return normalizeTenantId(scope.TenantId) == normalizeTenantId(tenantId)
}

func (scope TenantScope) Apply(db *gorm.DB, tableAliasOrName string) *gorm.DB {
	if scope.IsRoot {
		return db
	}
	scope.TenantId = normalizeTenantId(scope.TenantId)
	column := "tenant_id"
	if tableAliasOrName != "" {
		column = fmt.Sprintf("%s.tenant_id", tableAliasOrName)
	}
	return db.Where(column+" = ?", scope.TenantId)
}
