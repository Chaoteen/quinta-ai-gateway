package model

import (
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/stretchr/testify/require"
)

func TestMigrateUserRoleKeyBackfillsLegacyDefaults(t *testing.T) {
	truncateTables(t)

	testCases := []struct {
		id       int
		username string
		role     int
		roleKey  any
		want     string
	}{
		{id: 301, username: "legacy-root-default-user", role: common.RoleRootUser, roleKey: common.RoleKeyUser, want: common.RoleKeyRoot},
		{id: 302, username: "legacy-admin-default-user", role: common.RoleAdminUser, roleKey: common.RoleKeyUser, want: common.RoleKeyTenantAdmin},
		{id: 303, username: "legacy-root-null", role: common.RoleRootUser, roleKey: nil, want: common.RoleKeyRoot},
		{id: 304, username: "legacy-admin-null", role: common.RoleAdminUser, roleKey: nil, want: common.RoleKeyTenantAdmin},
		{id: 305, username: "legacy-root-empty", role: common.RoleRootUser, roleKey: "", want: common.RoleKeyRoot},
		{id: 306, username: "legacy-admin-empty", role: common.RoleAdminUser, roleKey: "", want: common.RoleKeyTenantAdmin},
	}
	for _, testCase := range testCases {
		require.NoError(t, DB.Exec(
			"INSERT INTO users (id, username, password, role, role_key, status) VALUES (?, ?, ?, ?, ?, ?)",
			testCase.id, testCase.username, "password123", testCase.role, testCase.roleKey, common.UserStatusEnabled,
		).Error)
	}

	require.NoError(t, migrateUserRoleKey())

	for _, testCase := range testCases {
		var user User
		require.NoError(t, DB.Select("role_key").Where("id = ?", testCase.id).First(&user).Error)
		require.Equal(t, testCase.want, user.RoleKey)
	}
}

func TestMigrateUserRoleKeyDoesNotOverwriteExplicitRoleKeys(t *testing.T) {
	truncateTables(t)

	explicitRoles := []string{
		common.RoleKeyFinance,
		common.RoleKeyOps,
		common.RoleKeyAuditor,
		common.RoleKeyOrganizationAdmin,
	}
	for i, roleKey := range explicitRoles {
		require.NoError(t, DB.Exec(
			"INSERT INTO users (id, username, password, role, role_key, status) VALUES (?, ?, ?, ?, ?, ?)",
			320+i, "explicit-role-"+roleKey, "password123", common.RoleAdminUser, roleKey, common.UserStatusEnabled,
		).Error)
	}

	require.NoError(t, migrateUserRoleKey())

	for i, roleKey := range explicitRoles {
		var user User
		require.NoError(t, DB.Select("role_key").Where("id = ?", 320+i).First(&user).Error)
		require.Equal(t, roleKey, user.RoleKey)
	}
}
