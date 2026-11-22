package middleware

import (
	"github.com/gin-gonic/gin"
	"net"
	"os"
	"strconv"
	"strings"
)

// This limited policy protects embedding and base URLs. It is not a complete
// script CSP, and does not change the shared host's HTTPS/HSTS policy.
func SecurityResponseHeaders() gin.HandlerFunc {
	strict := os.Getenv("SECURITY_CSP_ENFORCE") == "true"
	hstsHost := strings.ToLower(strings.TrimSpace(os.Getenv("SECURITY_HSTS_HOST")))
	hstsAge, _ := strconv.Atoi(os.Getenv("SECURITY_HSTS_MAX_AGE"))
	const baseline = "frame-ancestors 'self'; object-src 'none'; base-uri 'self'"
	const policy = "default-src 'self'; script-src 'self' https://challenges.cloudflare.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-src https://challenges.cloudflare.com; worker-src 'self' blob:; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'self'"

	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		if strict {
			c.Header("Content-Security-Policy", policy)
		} else {
			c.Header("Content-Security-Policy", baseline)
			c.Header("Content-Security-Policy-Report-Only", policy)
		}
		host := c.Request.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		https := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
		if https && hstsHost != "" && net.ParseIP(hstsHost) == nil && strings.EqualFold(host, hstsHost) && hstsAge >= 300 && hstsAge <= 31536000 {
			c.Header("Strict-Transport-Security", "max-age="+strconv.Itoa(hstsAge))
		}
		c.Next()
	}
}
