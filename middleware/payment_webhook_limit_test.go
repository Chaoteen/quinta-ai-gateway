package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSecurityPaymentWebhookBodyBound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, lengthKnown := range []bool{true, false} {
		r := gin.New()
		r.Use(PaymentWebhookBodyLimit())
		r.POST("/api/stripe/webhook", func(c *gin.Context) {
			data, err := io.ReadAll(c.Request.Body)
			require.Error(t, err)
			require.LessOrEqual(t, int64(len(data)), MaxPaymentWebhookBytes)
			c.Status(413)
		})
		req := httptest.NewRequest("POST", "/api/stripe/webhook", strings.NewReader(strings.Repeat("x", int(MaxPaymentWebhookBytes)+1)))
		if !lengthKnown {
			req.ContentLength = -1
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, 413, w.Code)
	}
}
