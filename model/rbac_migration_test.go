package model

import (
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/stretchr/testify/require"
)

func TestMigrateUserRoleKeyBackfillsLegacyDefaults(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Exec(
		"INSERT INTO users (id, username, password, role, role_key, status) VALUES (?, ?, ?, ?, ?, ?)",
		301, "legacy-root-default-user", "password123", common.RoleRootUser, common.RoleKeyUser, common.UserStatusEnabled,
	).Error)
	require.NoError(t, DB.Exec(
		"INSERT INTO users (id, username, password, role, role_key, status) VALUES (?, ?, ?, ?, ?, ?)",
		302, "legacy-admin-default-user", "password123", common.RoleAdminUser, common.RoleKeyUser, common.UserStatusEnabled,
	).Error)

	require.NoError(t, migrateUserRoleKey())

	var root User
	require.NoError(t, DB.Select("role_key").Where("id = ?", 301).First(&root).Error)
	require.Equal(t, common.RoleKeyRoot, root.RoleKey)

	var admin User
	require.NoError(t, DB.Select("role_key").Where("id = ?", 302).First(&admin).Error)
	require.Equal(t, common.RoleKeyTenantAdmin, admin.RoleKey)
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
