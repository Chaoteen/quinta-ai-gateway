package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecureVerificationRejectsWrongAccountAndFutureTimestamp(t *testing.T) {
	now := time.Now().Unix()
	for _, tc := range []struct {
		name  string
		stamp int64
		uid   interface{}
		want  int
	}{
		{"valid", now, 1, 200}, {"future", now + 60, 1, 403}, {"expired", now - 301, 1, 403}, {"other_account", now, 2, 403}, {"unbound_legacy", now, nil, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(sessions.Sessions("test", cookie.NewStore([]byte("test-secret"))))
			r.Use(func(c *gin.Context) {
				c.Set("id", 1)
				s := sessions.Default(c)
				s.Set(SecureVerificationSessionKey, tc.stamp)
				if tc.uid != nil {
					s.Set(secureVerificationUserSessionKey, tc.uid)
				}
				c.Next()
			})
			r.GET("/", SecureVerificationRequired(), func(c *gin.Context) { c.Status(200) })
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
			if rec.Code != tc.want {
				t.Fatalf("got %d want %d", rec.Code, tc.want)
			}
		})
	}
}
