package controller

import (
	"net/http"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func TestPlaygroundModelsRespectGroupTenantAndEmptyState(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{Id: 19731, Username: "playground-model-fixture", AffCode: "pg-fixture", TenantId: 801, Group: "default"}).Error)
	for _, ability := range []model.Ability{
		{TenantId: 801, Group: "default", Model: "available-model", ChannelId: 1, Enabled: true},
		{TenantId: 802, Group: "default", Model: "other-tenant", ChannelId: 2, Enabled: true},
		{TenantId: 801, Group: "default", Model: "disabled-model", ChannelId: 3, Enabled: false},
		{TenantId: 801, Group: "private-group", Model: "private-model", ChannelId: 4, Enabled: true},
	} {
		require.NoError(t, db.Create(&ability).Error)
	}
	c, w := newInvoiceControllerContext(http.MethodGet, "/api/user/models?group=default", "", 19731, 801, common.RoleKeyUser)
	GetUserModels(c)
	require.Equal(t, http.StatusOK, w.Code)
	var response struct {
		Success bool     `json:"success"`
		Data    []string `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, []string{"available-model"}, response.Data)

	c, w = newInvoiceControllerContext(http.MethodGet, "/api/user/models?group=private-group", "", 19731, 801, common.RoleKeyUser)
	GetUserModels(c)
	require.Equal(t, http.StatusForbidden, w.Code)
	require.NotContains(t, w.Body.String(), "private-model")

	require.NoError(t, db.Model(&model.Ability{}).Where("channel_id = ?", 1).Update("enabled", false).Error)
	c, w = newInvoiceControllerContext(http.MethodGet, "/api/user/models?group=default", "", 19731, 801, common.RoleKeyUser)
	GetUserModels(c)
	require.Contains(t, w.Body.String(), `"data":[]`)
}
