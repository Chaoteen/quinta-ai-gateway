package controller

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPasskeyReadyAccountAndClockBinding(t *testing.T) {
	for _, tc := range []struct {
		name   string
		uid    int
		offset int64
		want   bool
	}{{"valid", 1, 0, true}, {"future", 1, 60, false}, {"expired", 1, -61, false}, {"other", 2, 0, false}} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(sessions.Sessions("test", cookie.NewStore([]byte("test-secret"))))
			r.GET("/", func(c *gin.Context) {
				c.Set("id", 1)
				s := sessions.Default(c)
				s.Set(PasskeyReadySessionKey, time.Now().Unix()+tc.offset)
				s.Set("secure_passkey_ready_user_id", tc.uid)
				ok, _ := consumePasskeyReady(c)
				if ok != tc.want {
					t.Fatalf("got %v want %v", ok, tc.want)
				}
				repeated, _ := consumePasskeyReady(c)
				if repeated {
					t.Fatal("passkey marker replayed")
				}
			})
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
		})
	}
}
