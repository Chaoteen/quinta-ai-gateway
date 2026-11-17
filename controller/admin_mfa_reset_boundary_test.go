package controller

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestAdminCredentialResetPreservesProtectedAccounts(t *testing.T) {
	for _, method := range []string{"totp", "passkey"} {
		for _, tc := range []struct {
			name                    string
			actor, role, targetRole int
			targetKey               string
		}{
			{"self", 1, 100, 100, "root"}, {"peer root", 2, 100, 100, "root"},
			{"root role key", 2, 100, 1, "root"}, {"peer admin", 2, 10, 10, "tenant_admin"},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				db := setupMFALoginDB(t)
				require.NoError(t, db.AutoMigrate(&model.PasskeyCredential{}))
				require.NoError(t, db.Create(&model.User{Id: 1, Username: "fixture", AffCode: "fixture", Role: tc.targetRole, RoleKey: tc.targetKey, Status: common.UserStatusEnabled, TenantId: 1}).Error)
				require.NoError(t, db.Create(&model.TwoFA{UserId: 1, Secret: "fixture", IsEnabled: true}).Error)
				require.NoError(t, db.Create(&model.PasskeyCredential{UserID: 1, CredentialID: "fixture", PublicKey: "fixture"}).Error)
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Request = httptest.NewRequest("DELETE", "/", nil)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
				c.Set("id", tc.actor)
				c.Set("role", tc.role)
				if method == "totp" {
					AdminDisable2FA(c)
				} else {
					AdminResetPasskey(c)
				}
				var result struct {
					Success bool `json:"success"`
				}
				require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &result))
				require.False(t, result.Success)
				var count int64
				require.NoError(t, db.Model(&model.TwoFA{}).Count(&count).Error)
				require.EqualValues(t, 1, count)
				require.NoError(t, db.Model(&model.PasskeyCredential{}).Count(&count).Error)
				require.EqualValues(t, 1, count)
			})
		}
	}
}
