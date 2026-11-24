package middleware

import (
	"github.com/gin-gonic/gin"
	"net"
	"os"
	"strconv"
	"strings"
)

// Full script CSP can be staged in report-only mode. HSTS is opt-in for a
// dedicated DNS host; it applies to the entire host, never just a URL prefix.
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
		https := c.Request.TLS != nil || trustedForwardedHTTPS(c)
		if https && hstsHost != "" && net.ParseIP(hstsHost) == nil && strings.EqualFold(host, hstsHost) && hstsAge >= 300 && hstsAge <= 31536000 {
			c.Header("Strict-Transport-Security", "max-age="+strconv.Itoa(hstsAge))
		}
		c.Next()
	}
}

// Forwarded headers are assertions by an explicitly configured ingress proxy.
// Never let a direct client opt itself into the HTTPS policy.
func trustedForwardedHTTPS(c *gin.Context) bool {
	if c.GetHeader("X-Forwarded-Proto") != "https" {
		return false
	}
	remote, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(remote)
	if ip == nil {
		return false
	}
	for _, entry := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		entry = strings.TrimSpace(entry)
		if candidate := net.ParseIP(entry); candidate != nil && candidate.Equal(ip) {
			return true
		}
		if _, block, err := net.ParseCIDR(entry); err == nil && block.Contains(ip) {
			return true
		}
	}
	return false
}
