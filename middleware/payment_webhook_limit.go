package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const MaxPaymentWebhookBytes int64 = 1 << 20

// Apply before every public payment webhook, including chunked requests whose
// Content-Length is unknown. Signature verification still sees the exact body.
func PaymentWebhookBodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost && strings.HasSuffix(c.Request.URL.Path, "/webhook") {
			if c.Request.ContentLength > MaxPaymentWebhookBytes {
				c.AbortWithStatus(http.StatusRequestEntityTooLarge)
				return
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxPaymentWebhookBytes)
		}
		c.Next()
	}
}
