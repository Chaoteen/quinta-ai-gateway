package model

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestAdminReadinessFiltersAccountsAndNeverExposesCredentials(t *testing.T) {
	old := DB
	db, err := gorm.Open(sqlite.Open("file:security-readiness?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	t.Cleanup(func() { DB = old; sql, _ := db.DB(); sql.Close() })
	require.NoError(t, db.AutoMigrate(&User{}, &TwoFA{}, &PasskeyCredential{}))
	for _, u := range []User{{Id: 1, Username: "root", AffCode: "r1", Role: 100, RoleKey: "root", Status: 1}, {Id: 2, Username: "finance", AffCode: "r2", Role: 1, RoleKey: "finance", Status: 1}, {Id: 3, Username: "user", AffCode: "r3", Role: 1, RoleKey: "user", Status: 1}, {Id: 4, Username: "disabled", AffCode: "r4", Role: 100, RoleKey: "root", Status: 2}} {
		require.NoError(t, db.Create(&u).Error)
	}
	require.NoError(t, db.Create(&TwoFA{UserId: 1, Secret: "fixture-private-seed", IsEnabled: true}).Error)
	require.NoError(t, db.Create(&PasskeyCredential{UserID: 2, CredentialID: "fixture-id", PublicKey: "fixture-private-marker", CloneWarning: true}).Error)
	result, err := GetAdminSecurityReadiness()
	require.NoError(t, err)
	require.Len(t, result.Accounts, 2)
	require.False(t, result.EnrollmentComplete)
	require.True(t, result.Accounts[0].HasTOTP)
	require.False(t, result.Accounts[1].HasPasskey)
	encoded, err := common.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "fixture-private")
	require.NoError(t, db.Model(&PasskeyCredential{}).Where("user_id = ?", 2).Update("clone_warning", false).Error)
	result, err = GetAdminSecurityReadiness()
	require.NoError(t, err)
	require.True(t, result.EnrollmentComplete)
	require.NoError(t, db.Where("user_id = ?", 1).Delete(&TwoFA{}).Error)
	result, err = GetAdminSecurityReadiness()
	require.NoError(t, err)
	require.False(t, result.EnrollmentComplete)
}
