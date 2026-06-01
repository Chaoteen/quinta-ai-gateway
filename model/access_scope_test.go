package model

import (
	"net/http/httptest"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newAccessScopeTestContext(role int, roleKey string, tenantId int, organizationId int, departmentId int) *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("role", role)
	common.SetContextKey(ctx, constant.ContextKeyUserRoleKey, roleKey)
	common.SetContextKey(ctx, constant.ContextKeyTenantId, tenantId)
	common.SetContextKey(ctx, constant.ContextKeyOrganizationId, organizationId)
	common.SetContextKey(ctx, constant.ContextKeyDepartmentId, departmentId)
	return ctx
}

func TestAccessScopeRootAccessesAllOwnership(t *testing.T) {
	scope := AccessScopeFromContext(newAccessScopeTestContext(common.RoleRootUser, common.RoleKeyRoot, 1, 0, 0))

	require.True(t, scope.IsRoot)
	require.True(t, AllowsOwnership(scope, 1, 10, 100))
	require.True(t, AllowsOwnership(scope, 2, 20, 200))

	var count int64
	require.NoError(t, ApplyOwnershipScope(DB.Model(&User{}), "users", scope).Count(&count).Error)
}

func TestAccessScopeTenantAdminAccessesTenant(t *testing.T) {
	resetAccessScopeUsers(t)
	require.NoError(t, DB.Create(&[]User{
		{Id: 4101, TenantId: 1, OrganizationId: 10, Username: "tenant-admin-org-10", Password: "password123", Role: common.RoleCommonUser, RoleKey: common.RoleKeyUser, Status: common.UserStatusEnabled, AffCode: "scope-4101"},
		{Id: 4102, TenantId: 1, OrganizationId: 20, Username: "tenant-admin-org-20", Password: "password123", Role: common.RoleCommonUser, RoleKey: common.RoleKeyUser, Status: common.UserStatusEnabled, AffCode: "scope-4102"},
		{Id: 4103, TenantId: 2, OrganizationId: 10, Username: "tenant-admin-tenant-2", Password: "password123", Role: common.RoleCommonUser, RoleKey: common.RoleKeyUser, Status: common.UserStatusEnabled, AffCode: "scope-4103"},
	}).Error)

	scope := AccessScopeFromContext(newAccessScopeTestContext(common.RoleAdminUser, common.RoleKeyTenantAdmin, 1, 10, 0))
	require.False(t, scope.IsRoot)
	require.Equal(t, 1, scope.TenantId)
	require.Zero(t, scope.OrganizationId)
	require.True(t, AllowsOwnership(scope, 1, 10, 0))
	require.True(t, AllowsOwnership(scope, 1, 20, 0))
	require.False(t, AllowsOwnership(scope, 2, 10, 0))

	var users []User
	require.NoError(t, ApplyOwnershipScope(DB.Model(&User{}), "users", scope).Order("id").Find(&users).Error)
	require.Len(t, users, 2)
	require.Equal(t, 4101, users[0].Id)
	require.Equal(t, 4102, users[1].Id)
}

func TestAccessScopeOrganizationAdminAccessesOrganization(t *testing.T) {
	resetAccessScopeUsers(t)
	require.NoError(t, DB.Create(&[]User{
		{Id: 4201, TenantId: 1, OrganizationId: 10, Username: "org-admin-org-10", Password: "password123", Role: common.RoleCommonUser, RoleKey: common.RoleKeyUser, Status: common.UserStatusEnabled, AffCode: "scope-4201"},
		{Id: 4202, TenantId: 1, OrganizationId: 20, Username: "org-admin-org-20", Password: "password123", Role: common.RoleCommonUser, RoleKey: common.RoleKeyUser, Status: common.UserStatusEnabled, AffCode: "scope-4202"},
		{Id: 4203, TenantId: 2, OrganizationId: 10, Username: "org-admin-tenant-2", Password: "password123", Role: common.RoleCommonUser, RoleKey: common.RoleKeyUser, Status: common.UserStatusEnabled, AffCode: "scope-4203"},
	}).Error)

	scope := AccessScopeFromContext(newAccessScopeTestContext(common.RoleCommonUser, common.RoleKeyOrganizationAdmin, 1, 10, 99))
	require.False(t, scope.IsRoot)
	require.Equal(t, 1, scope.TenantId)
	require.Equal(t, 10, scope.OrganizationId)
	require.Zero(t, scope.DepartmentId)
	require.True(t, AllowsOwnership(scope, 1, 10, 0))
	require.False(t, AllowsOwnership(scope, 1, 20, 0))
	require.False(t, AllowsOwnership(scope, 2, 10, 0))

	var users []User
	require.NoError(t, ApplyOwnershipScope(DB.Model(&User{}), "users", scope).Find(&users).Error)
	require.Len(t, users, 1)
	require.Equal(t, 4201, users[0].Id)
}

func TestAccessScopeOrganizationAdminWithoutOrganizationFailsClosed(t *testing.T) {
	resetAccessScopeUsers(t)
	require.NoError(t, DB.Create(&User{
		Id: 4301, TenantId: 1, OrganizationId: 10, Username: "org-admin-fail-closed", Password: "password123", Role: common.RoleCommonUser, RoleKey: common.RoleKeyUser, Status: common.UserStatusEnabled, AffCode: "scope-4301",
	}).Error)

	scope := AccessScopeFromContext(newAccessScopeTestContext(common.RoleCommonUser, common.RoleKeyOrganizationAdmin, 1, 0, 0))
	require.False(t, AllowsOwnership(scope, 1, 10, 0))

	var count int64
	require.NoError(t, ApplyOwnershipScope(DB.Model(&User{}), "users", scope).Count(&count).Error)
	require.Zero(t, count)
}

func TestAccessScopeDepartmentFilterIsAppliedWhenExplicit(t *testing.T) {
	resetAccessScopeUsers(t)
	require.NoError(t, DB.Create(&[]User{
		{Id: 4401, TenantId: 1, OrganizationId: 10, DepartmentId: 100, Username: "dept-100", Password: "password123", Role: common.RoleCommonUser, RoleKey: common.RoleKeyUser, Status: common.UserStatusEnabled, AffCode: "scope-4401"},
		{Id: 4402, TenantId: 1, OrganizationId: 10, DepartmentId: 200, Username: "dept-200", Password: "password123", Role: common.RoleCommonUser, RoleKey: common.RoleKeyUser, Status: common.UserStatusEnabled, AffCode: "scope-4402"},
	}).Error)

	scope := AccessScope{TenantId: 1, OrganizationId: 10, DepartmentId: 100, RoleKey: common.RoleKeyOrganizationAdmin}
	require.True(t, AllowsOwnership(scope, 1, 10, 100))
	require.False(t, AllowsOwnership(scope, 1, 10, 200))

	var users []User
	require.NoError(t, ApplyOwnershipScope(DB.Model(&User{}), "users", scope).Find(&users).Error)
	require.Len(t, users, 1)
	require.Equal(t, 4401, users[0].Id)
}

func resetAccessScopeUsers(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
	t.Cleanup(func() {
		_ = DB.Exec("DELETE FROM users").Error
	})
}
