package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSecurityTenantCollectionRejectsGlobalCreationRoutes(t *testing.T) {
	t.Setenv("TENANT_INDEPENDENT_PAYMENTS", "true")
	for _, handler := range []gin.HandlerFunc{RequestEpay, RequestStripePay, RequestCreemPay, RequestWaffoPay,
		RequestWaffoPancakePay, SubscriptionRequestEpay, SubscriptionRequestStripePay, SubscriptionRequestCreemPay} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/legacy-payment", nil)
		handler(c)
		require.Equal(t, http.StatusConflict, recorder.Code)
	}
}
