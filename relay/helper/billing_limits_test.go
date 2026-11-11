package helper

import (
	"math"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSecurityRelayOutputBudgetBounds(t *testing.T) {
	for _, value := range []uint{maxRequestOutputTokens + 1, math.MaxUint} {
		for _, req := range []dto.Request{
			&dto.GeneralOpenAIRequest{MaxTokens: &value},
			&dto.GeneralOpenAIRequest{MaxCompletionTokens: &value},
			&dto.ClaudeRequest{MaxTokens: &value},
			&dto.ClaudeRequest{MaxTokensToSample: &value},
			&dto.OpenAIResponsesRequest{MaxOutputTokens: &value},
		} {
			require.Error(t, validateRelayBudget(req))
		}
	}
	zero := uint(0)
	req := &dto.GeneralOpenAIRequest{MaxTokens: &zero}
	require.NoError(t, validateRelayBudget(req))
	encoded, err := common.Marshal(req)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"max_tokens":0`)
	require.NoError(t, validateRelayBudget(&dto.GeneralOpenAIRequest{}))
}

func TestSecurityImageCountsCannotBypassBillingLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, body := range []string{
		`{"model":"gpt-image-1","prompt":"fixture","n":18446744073709551615}`,
		`{"model":"gpt-image-1","prompt":"fixture","n":0}`,
		`{"model":"gpt-image-1","prompt":"fixture","parameters":{"n":18446744073709551615}}`,
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		_, err := GetAndValidOpenAIImageRequest(c, 0)
		require.Error(t, err)
	}
	for _, body := range []string{
		`{"model":"gpt-image-1","prompt":"fixture"}`,
		`{"model":"gpt-image-1","prompt":"fixture","n":128}`,
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		_, err := GetAndValidOpenAIImageRequest(c, 0)
		require.NoError(t, err)
	}
}
