package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	relaycommon "github.com/Chaoteen/quinta-ai-gateway/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNewRelayRetryParamRejectsMissingTenantScope(t *testing.T) {
	info := &relaycommon.RelayInfo{
		TokenGroup:      "default",
		OriginModelName: "strict-scope-model",
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	param, err := newRelayRetryParam(ctx, info)
	require.Error(t, err)
	require.Nil(t, param)
	require.Contains(t, err.Error(), "relay tenant context is missing")

	common.SetContextKey(ctx, constant.ContextKeyTenantId, 2)
	param, err = newRelayRetryParam(ctx, info)
	require.NoError(t, err)
	require.Equal(t, 2, param.TenantScope.TenantId)
	require.False(t, param.TenantScope.IsRoot)

	rootCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	rootCtx.Set("role", common.RoleRootUser)
	param, err = newRelayRetryParam(rootCtx, info)
	require.NoError(t, err)
	require.True(t, param.TenantScope.IsRoot)
}
