package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Exercise the shared gate used after OAuth/Passkey identity verification with
// actual cookie round trips, without contacting external identity providers.
func TestPrimaryIdentityRequiresLocalSecondFactor(t *testing.T) {
	db := setupMFALoginDB(t)
	require.NoError(t, db.Create(&model.User{Id: 1, Username: "fixture", AffCode: "fixture", Status: common.UserStatusEnabled, Role: common.RoleRootUser}).Error)
	require.NoError(t, db.Create(&model.TwoFA{UserId: 1, Secret: "fixture", IsEnabled: true}).Error)
	require.NoError(t, model.CreateBackupCodes(1, []string{"ABCD-EFGH"}))
	r := gin.New()
	r.Use(sessions.Sessions("test", cookie.NewStore([]byte("fixture-login-gate"))))
	r.GET("/primary", func(c *gin.Context) {
		sessions.Default(c).Set("id", 999)
		setupLogin(&model.User{Id: 1}, c)
	})
	r.POST("/factor", Verify2FALogin)
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(200, gin.H{"id": sessions.Default(c).Get("id"), "pending": sessions.Default(c).Get("pending_user_id")})
	})
	call := func(method, path, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	primary := call("GET", "/primary", "", nil)
	var challenge struct {
		Success bool `json:"success"`
		Data    struct {
			Require2FA bool `json:"require_2fa"`
			Id         int  `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(primary.Body.Bytes(), &challenge))
	require.True(t, challenge.Success)
	require.True(t, challenge.Data.Require2FA)
	require.Zero(t, challenge.Data.Id)
	probe := call("GET", "/probe", "", primary.Result().Cookies())
	require.JSONEq(t, `{"id":null,"pending":1}`, probe.Body.String())
	factor := call("POST", "/factor", `{"code":"ABCD-EFGH"}`, primary.Result().Cookies())
	var login struct {
		Success bool `json:"success"`
		Data    struct {
			Id int `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(factor.Body.Bytes(), &login))
	require.True(t, login.Success)
	require.Equal(t, 1, login.Data.Id)
	probe = call("GET", "/probe", "", factor.Result().Cookies())
	require.JSONEq(t, `{"id":1,"pending":null}`, probe.Body.String())
}

func TestPrimaryIdentityGateUsesCurrentStatusAndFailsClosed(t *testing.T) {
	for _, scenario := range []string{"disabled", "missing MFA table", "not enrolled"} {
		t.Run(scenario, func(t *testing.T) {
			db := setupMFALoginDB(t)
			status := common.UserStatusEnabled
			if scenario == "disabled" {
				status = common.UserStatusDisabled
			}
			require.NoError(t, db.Create(&model.User{Id: 1, Username: "fixture", AffCode: "fixture", Status: status, Role: common.RoleCommonUser}).Error)
			if scenario == "missing MFA table" {
				require.NoError(t, db.Migrator().DropTable(&model.TwoFA{}))
			}
			r := gin.New()
			r.Use(sessions.Sessions("test", cookie.NewStore([]byte("fixture-login-gate"))))
			r.GET("/", func(c *gin.Context) {
				setupLogin(&model.User{Id: 1, Status: common.UserStatusEnabled}, c)
				if scenario != "not enrolled" {
					require.Nil(t, sessions.Default(c).Get("id"))
				}
			})
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
			var result struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &result))
			require.Equal(t, scenario == "not enrolled", result.Success)
		})
	}
}
