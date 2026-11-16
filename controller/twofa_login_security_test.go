package controller

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/i18n"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMFALoginDB(t *testing.T) *gorm.DB {
	t.Helper()
	require.NoError(t, i18n.Init())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	previous := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previous; _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TwoFA{}, &model.TwoFABackupCode{}))
	return db
}

func TestPendingMFALoginExpiresAndChecksCurrentStatus(t *testing.T) {
	for _, tc := range []struct {
		name    string
		offset  int64
		missing bool
		status  int
		want    bool
	}{
		{"valid recovery", 0, false, common.UserStatusEnabled, true},
		{"missing timestamp", 0, true, common.UserStatusEnabled, false},
		{"expired", -301, false, common.UserStatusEnabled, false},
		{"future", 60, false, common.UserStatusEnabled, false},
		{"disabled since password", 0, false, common.UserStatusDisabled, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupMFALoginDB(t)
			require.NoError(t, db.Create(&model.User{Id: 1, Username: "fixture", AffCode: "fixture", Status: tc.status, Role: common.RoleCommonUser}).Error)
			require.NoError(t, db.Create(&model.TwoFA{UserId: 1, Secret: "fixture", IsEnabled: true}).Error)
			require.NoError(t, model.CreateBackupCodes(1, []string{"ABCD-EFGH"}))
			r := gin.New()
			r.Use(sessions.Sessions("test", cookie.NewStore([]byte("fixture-secret"))))
			r.POST("/", func(c *gin.Context) {
				s := sessions.Default(c)
				s.Set("pending_user_id", 1)
				if !tc.missing {
					s.Set("pending_2fa_started_at", common.GetTimestamp()+tc.offset)
				}
				// A stale step-up grant must not survive successful login.
				s.Set(SecureVerificationSessionKey, common.GetTimestamp())
				Verify2FALogin(c)
				if tc.want {
					require.Nil(t, s.Get(SecureVerificationSessionKey))
					require.Equal(t, 1, s.Get("id"))
				} else {
					require.Nil(t, s.Get("id"))
				}
			})
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest("POST", "/", strings.NewReader(`{"code":"ABCD-EFGH"}`)))
			var body struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tc.want, body.Success)
			count, err := model.GetUnusedBackupCodeCount(1)
			require.NoError(t, err)
			if tc.want {
				require.Zero(t, count)
			} else {
				require.Equal(t, 1, count)
			}
		})
	}
}

func TestPasswordLoginFailsClosedWhenMFAQueryFails(t *testing.T) {
	db := setupMFALoginDB(t)
	previous := common.PasswordLoginEnabled
	common.PasswordLoginEnabled = true
	t.Cleanup(func() { common.PasswordLoginEnabled = previous })
	hash, err := common.Password2Hash("fixture-password")
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.User{Id: 1, Username: "fixture", AffCode: "fixture", Password: hash, Status: common.UserStatusEnabled, Role: common.RoleCommonUser}).Error)
	require.NoError(t, db.Migrator().DropTable(&model.TwoFA{}))
	r := gin.New()
	r.Use(sessions.Sessions("test", cookie.NewStore([]byte("fixture-secret"))))
	r.POST("/", func(c *gin.Context) { Login(c); require.Nil(t, sessions.Default(c).Get("id")) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("POST", "/", strings.NewReader(`{"username":"fixture","password":"fixture-password"}`)))
	var body struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &body))
	require.False(t, body.Success)
}

func TestPasswordChallengeClearsPreviousIdentityAndStepUp(t *testing.T) {
	db := setupMFALoginDB(t)
	previous := common.PasswordLoginEnabled
	common.PasswordLoginEnabled = true
	t.Cleanup(func() { common.PasswordLoginEnabled = previous })
	hash, err := common.Password2Hash("fixture-password")
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.User{Id: 1, Username: "fixture", AffCode: "fixture", Password: hash, Status: common.UserStatusEnabled, Role: common.RoleCommonUser}).Error)
	require.NoError(t, db.Create(&model.TwoFA{UserId: 1, Secret: "fixture", IsEnabled: true}).Error)
	r := gin.New()
	r.Use(sessions.Sessions("test", cookie.NewStore([]byte("fixture-secret"))))
	r.POST("/", func(c *gin.Context) {
		s := sessions.Default(c)
		s.Set("id", 2)
		s.Set("username", "previous-user")
		s.Set(SecureVerificationSessionKey, common.GetTimestamp())
		Login(c)
		require.Nil(t, s.Get("id"))
		require.Nil(t, s.Get("username"))
		require.Nil(t, s.Get(SecureVerificationSessionKey))
		require.Equal(t, 1, s.Get("pending_user_id"))
		require.IsType(t, int64(0), s.Get("pending_2fa_started_at"))
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("POST", "/", strings.NewReader(`{"username":"fixture","password":"fixture-password"}`)))
	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Require2FA bool `json:"require_2fa"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.True(t, body.Data.Require2FA)
}
