package model

import (
	"errors"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestPasswordUpdatesRevokeSessionsAtomically(t *testing.T) {
	for _, mode := range []string{"update", "edit", "email"} {
		t.Run(mode, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			require.NoError(t, err)
			old := DB
			DB = db
			oldRedis := common.RedisEnabled
			common.RedisEnabled = false
			t.Cleanup(func() { DB = old; common.RedisEnabled = oldRedis; s, _ := db.DB(); _ = s.Close() })
			require.NoError(t, db.AutoMigrate(&User{}))
			hash, err := common.Password2Hash("old-password")
			require.NoError(t, err)
			require.NoError(t, db.Create(&User{Id: 1, Username: "fixture", AffCode: "fixture", Password: hash, Email: "fixture@example.invalid", Role: 1, Status: 1}).Error)
			change := func() error {
				u := User{Id: 1, Password: "new-password", Username: "fixture"}
				switch mode {
				case "update":
					return u.Update(true)
				case "edit":
					return u.Edit(true)
				default:
					return ResetUserPasswordByEmail("fixture@example.invalid", "new-password")
				}
			}
			require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:epoch-failure", func(tx *gorm.DB) {
				if _, ok := tx.Statement.Dest.(map[string]interface{}); ok {
					m := tx.Statement.Dest.(map[string]interface{})
					if _, has := m["auth_epoch"]; has {
						tx.AddError(errors.New("epoch update failed"))
					}
				}
			}))
			require.Error(t, change())
			var u User
			require.NoError(t, db.First(&u, 1).Error)
			require.Zero(t, u.AuthEpoch)
			require.True(t, common.ValidatePasswordAndHash("old-password", u.Password))
			require.NoError(t, db.Callback().Update().Remove("test:epoch-failure"))
			require.NoError(t, change())
			require.NoError(t, db.First(&u, 1).Error)
			require.EqualValues(t, 1, u.AuthEpoch)
			require.True(t, common.ValidatePasswordAndHash("new-password", u.Password))
			u.Password = "must-not-be-stored"
			require.NoError(t, u.Update(false))
			require.NoError(t, db.First(&u, 1).Error)
			require.EqualValues(t, 1, u.AuthEpoch)
			require.True(t, common.ValidatePasswordAndHash("new-password", u.Password))
		})
	}
}
