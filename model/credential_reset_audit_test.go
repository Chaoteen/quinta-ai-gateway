package model

import (
	"errors"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestCredentialResetAuditIsAtomic(t *testing.T) {
	for _, kind := range []string{"totp", "passkey"} {
		t.Run(kind, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			require.NoError(t, err)
			old := DB
			DB = db
			t.Cleanup(func() { DB = old; s, _ := db.DB(); _ = s.Close() })
			require.NoError(t, db.AutoMigrate(&User{}, &TwoFA{}, &TwoFABackupCode{}, &PasskeyCredential{}, &CredentialResetAudit{}))
			require.NoError(t, db.Create(&User{Id: 1, Username: "actor", AffCode: "a", Role: 100, Status: 1}).Error)
			require.NoError(t, db.Create(&User{Id: 2, Username: "target", AffCode: "b", Role: 1, Status: 1, TenantId: 1}).Error)
			require.NoError(t, db.Create(&TwoFA{UserId: 2, Secret: "fixture", IsEnabled: true}).Error)
			require.NoError(t, db.Create(&TwoFABackupCode{UserId: 2, CodeHash: "fixture"}).Error)
			require.NoError(t, db.Create(&PasskeyCredential{UserID: 2, CredentialID: "fixture", PublicKey: "fixture"}).Error)
			require.NoError(t, db.Callback().Create().Before("gorm:create").Register("test:audit-fail", func(tx *gorm.DB) {
				if tx.Statement.Table == "credential_reset_audits" {
					tx.AddError(errors.New("audit unavailable"))
				}
			}))
			require.Error(t, AdminResetCredential(1, 2, kind))
			var n int64
			for _, m := range []interface{}{&TwoFA{}, &TwoFABackupCode{}, &PasskeyCredential{}} {
				require.NoError(t, db.Model(m).Count(&n).Error)
				require.EqualValues(t, 1, n)
			}
			require.NoError(t, db.Callback().Create().Remove("test:audit-fail"))
			require.Error(t, AdminResetCredential(2, 1, kind))
			require.NoError(t, AdminResetCredential(1, 2, kind))
			var rows []CredentialResetAudit
			require.NoError(t, db.Find(&rows).Error)
			require.Len(t, rows, 1)
			require.Equal(t, 1, rows[0].ActorId)
			require.Equal(t, 2, rows[0].TargetId)
			require.Equal(t, kind, rows[0].CredentialType)
		})
	}
}
