package middleware

import "github.com/gin-gonic/gin"

// This limited policy protects embedding and base URLs. It is not a complete
// script CSP, and does not change the shared host's HTTPS/HSTS policy.
func SecurityResponseHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Content-Security-Policy", "frame-ancestors 'self'; object-src 'none'; base-uri 'self'")
		c.Next()
	}
}
