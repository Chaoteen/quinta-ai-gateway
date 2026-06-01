package model

import "github.com/Chaoteen/quinta-ai-gateway/common"

func migrateUserRoleKey() error {
	if err := DB.Model(&User{}).
		Where("role = ? AND (role_key = ? OR role_key = ? OR role_key IS NULL)", common.RoleRootUser, "", common.RoleKeyUser).
		Update("role_key", common.RoleKeyRoot).Error; err != nil {
		return err
	}
	if err := DB.Model(&User{}).
		Where("role = ? AND (role_key = ? OR role_key = ? OR role_key IS NULL)", common.RoleAdminUser, "", common.RoleKeyUser).
		Update("role_key", common.RoleKeyTenantAdmin).Error; err != nil {
		return err
	}
	return DB.Model(&User{}).
		Where("role_key = ? OR role_key IS NULL", "").
		Update("role_key", common.RoleKeyUser).Error
}
