package model

import (
	"net/http/httptest"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayTenantScopeRejectsMissingTenantUnlessRoot(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, err := RelayTenantScopeFromContext(ctx)
	require.Error(t, err)

	common.SetContextKey(ctx, constant.ContextKeyTenantId, 2)
	scope, err := RelayTenantScopeFromContext(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, scope.TenantId)
	require.False(t, scope.IsRoot)

	rootCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	rootCtx.Set("role", common.RoleRootUser)
	scope, err = RelayTenantScopeFromContext(rootCtx)
	require.NoError(t, err)
	require.True(t, scope.IsRoot)
}

func seedRelayIsolationChannels(t *testing.T) {
	t.Helper()
	previousGroupCol := commonGroupCol
	commonGroupCol = "`group`"
	require.NoError(t, DB.AutoMigrate(&Ability{}))
	require.NoError(t, DB.Exec("DELETE FROM abilities").Error)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)
	t.Cleanup(func() {
		commonGroupCol = previousGroupCol
		_ = DB.Exec("DELETE FROM abilities").Error
		_ = DB.Exec("DELETE FROM channels").Error
	})

	priorityTenant1 := int64(10)
	priorityTenant2 := int64(20)
	weight := uint(100)
	require.NoError(t, DB.Create(&[]Channel{
		{Id: 901, TenantId: 1, Name: "tenant-1", Status: common.ChannelStatusEnabled, Group: "default", Models: "isolation-model", Priority: &priorityTenant1, Weight: &weight, Key: "key-1"},
		{Id: 902, TenantId: 2, Name: "tenant-2", Status: common.ChannelStatusEnabled, Group: "default", Models: "isolation-model", Priority: &priorityTenant2, Weight: &weight, Key: "key-2"},
	}).Error)
	require.NoError(t, DB.Create(&[]Ability{
		{TenantId: 1, Group: "default", Model: "isolation-model", ChannelId: 901, Enabled: true, Priority: &priorityTenant1, Weight: weight},
		{TenantId: 2, Group: "default", Model: "isolation-model", ChannelId: 902, Enabled: true, Priority: &priorityTenant2, Weight: weight},
	}).Error)
}

func TestGetChannelTenantIsolation(t *testing.T) {
	seedRelayIsolationChannels(t)

	channel, err := GetChannel("default", "isolation-model", 0, TenantScope{TenantId: 1})
	require.NoError(t, err)
	require.Equal(t, 901, channel.Id)

	channel, err = GetChannel("default", "isolation-model", 0, TenantScope{IsRoot: true})
	require.NoError(t, err)
	require.Equal(t, 902, channel.Id)
}

func TestCachedChannelSelectionTenantIsolation(t *testing.T) {
	seedRelayIsolationChannels(t)

	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		tenantGroup2model2channels = nil
		channelsIDM = nil
	})
	InitChannelCache()

	channel, err := GetRandomSatisfiedChannel("default", "isolation-model", 0, TenantScope{TenantId: 1})
	require.NoError(t, err)
	require.Equal(t, 901, channel.Id)

	channel, err = GetRandomSatisfiedChannel("default", "isolation-model", 0, TenantScope{IsRoot: true})
	require.NoError(t, err)
	require.Equal(t, 902, channel.Id)
}
