package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
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

func TestStrictCSPAndScopedHSTS(t *testing.T) {
	t.Setenv("SECURITY_CSP_ENFORCE", "true")
	t.Setenv("SECURITY_HSTS_HOST", "gateway.example.invalid")
	t.Setenv("SECURITY_HSTS_MAX_AGE", "300")
	r := gin.New()
	r.Use(SecurityResponseHeaders())
	r.GET("/", func(c *gin.Context) { c.String(200, "ok") })
	for _, host := range []string{"gateway.example.invalid", "other.example.invalid", "47.110.42.3"} {
		req := httptest.NewRequest("GET", "https://"+host+"/", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if (rec.Header().Get("Strict-Transport-Security") != "") != (host == "gateway.example.invalid") {
			t.Fatal("HSTS scope incorrect")
		}
		if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "script-src 'self'") || rec.Header().Get("Content-Security-Policy-Report-Only") != "" {
			t.Fatal("strict CSP missing")
		}
	}
}
