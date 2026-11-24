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

func TestHSTSRejectsIPAndUntrustedForwardedHTTPS(t *testing.T) {
	t.Setenv("SECURITY_HSTS_HOST", "gateway.example.invalid")
	t.Setenv("SECURITY_HSTS_MAX_AGE", "300")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1,::1")
	r := gin.New()
	r.Use(SecurityResponseHeaders())
	r.GET("/", func(c *gin.Context) { c.String(200, "ok") })
	for _, tc := range []struct {
		remote, proto string
		allowed       bool
	}{
		{"127.0.0.1:41000", "https", true},
		{"[::1]:41000", "https", true},
		{"203.0.113.2:41000", "https", false},
		{"127.0.0.1:41000", "https,http", false},
		{"127.0.0.1:41000", "http", false},
	} {
		req := httptest.NewRequest("GET", "http://gateway.example.invalid/", nil)
		req.RemoteAddr = tc.remote
		req.Header.Set("X-Forwarded-Proto", tc.proto)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if (rec.Header().Get("Strict-Transport-Security") != "") != tc.allowed {
			t.Fatal("proxy trust boundary failed")
		}
	}
	t.Setenv("SECURITY_HSTS_HOST", "47.110.42.3")
	r = gin.New()
	r.Use(SecurityResponseHeaders())
	r.GET("/", func(c *gin.Context) { c.Status(200) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "https://47.110.42.3/", nil))
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("IP is not an HSTS host")
	}
}
