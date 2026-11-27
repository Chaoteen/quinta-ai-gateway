package model

import (
	"errors"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestRoleAssignmentAtomicAndScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	old := DB
	DB = db
	t.Cleanup(func() { DB = old; s, _ := db.DB(); s.Close() })
	require.NoError(t, db.AutoMigrate(&User{}, &Tenant{}, &RoleAssignmentAudit{}))
	for _, u := range []User{{Id: 1, Username: "root", AffCode: "a", Role: 100, RoleKey: "root", Status: 1, TenantId: 1}, {Id: 2, Username: "member", AffCode: "b", Role: 1, RoleKey: "user", Status: 1, TenantId: 2}, {Id: 3, Username: "personal", AffCode: "c", Role: 1, RoleKey: "user", Status: 1, TenantId: 1}} {
		require.NoError(t, db.Create(&u).Error)
	}
	require.NoError(t, db.Create(&Tenant{Id: 2, Name: "enterprise", Status: 1}).Error)
	require.Error(t, AssignBusinessRole(2, 2, common.RoleKeyEnterpriseAdmin))
	require.Error(t, AssignBusinessRole(1, 1, common.RoleKeyEnterpriseAdmin))
	require.Error(t, AssignBusinessRole(1, 3, common.RoleKeyEnterpriseAdmin))
	require.Error(t, AssignBusinessRole(1, 2, common.RoleKeyRoot))
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("reject:audit", func(tx *gorm.DB) {
		if tx.Statement.Table == "role_assignment_audits" {
			tx.AddError(errors.New("audit failed"))
		}
	}))
	require.Error(t, AssignBusinessRole(1, 2, common.RoleKeyEnterpriseAdmin))
	var u User
	require.NoError(t, db.First(&u, 2).Error)
	require.Equal(t, "user", u.RoleKey)
	require.Zero(t, u.AuthEpoch)
	require.NoError(t, db.Callback().Create().Remove("reject:audit"))
	require.NoError(t, AssignBusinessRole(1, 2, common.RoleKeyEnterpriseAdmin))
	require.NoError(t, db.First(&u, 2).Error)
	require.Equal(t, common.RoleKeyEnterpriseAdmin, u.RoleKey)
	require.EqualValues(t, 1, u.AuthEpoch)
	require.Equal(t, 2, u.TenantId)
	var count int64
	db.Model(&RoleAssignmentAudit{}).Count(&count)
	require.EqualValues(t, 1, count)
}
