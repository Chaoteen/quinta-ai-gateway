package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersAllowOnlySameOriginEmbedding(t *testing.T) {
	r := gin.New()
	r.Use(SecurityResponseHeaders())
	r.GET("/", func(c *gin.Context) { c.String(200, "ok") })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" || rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Content-Security-Policy") != "frame-ancestors 'self'; object-src 'none'; base-uri 'self'" {
		t.Fatal("security headers missing")
	}
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("unexpected shared-host HTTPS policy")
	}
}
