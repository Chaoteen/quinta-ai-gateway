package controller

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestChatPreferencesAreOwnerScopedAndPreserveOtherSettings(t *testing.T) {
	db := setupInvoiceControllerTestDB(t)
	for _, user := range []model.User{
		{Id: 26001, Username: "chat-pref-owner", AffCode: "chat-pref-owner", TenantId: 1, Setting: `{"language":"zh","quota_warning_threshold":999,"custom_option":"keep"}`},
		{Id: 26002, Username: "chat-pref-other", AffCode: "chat-pref-other", TenantId: 2, Setting: `{"language":"en"}`},
	} {
		require.NoError(t, db.Create(&user).Error)
	}
	c, w := newInvoiceControllerContext(http.MethodPut, "/api/user/self/chat-preferences", `{"user_id":26002,"enabled_chat_apps":["Cherry Studio","Cherry Studio","CC Switch"]}`, 26001, 1, common.RoleKeyUser)
	UpdateChatPreferences(c)
	require.Equal(t, http.StatusOK, w.Code)
	var owner, other model.User
	require.NoError(t, db.First(&owner, 26001).Error)
	require.NoError(t, db.First(&other, 26002).Error)
	require.Equal(t, []string{"Cherry Studio", "CC Switch"}, owner.GetSetting().EnabledChatApps)
	require.Contains(t, owner.Setting, `"custom_option":"keep"`)
	require.Equal(t, "zh", owner.GetSetting().Language)
	require.Equal(t, `{"language":"en"}`, other.Setting)
	for _, body := range []string{`{}`, `{"enabled_chat_apps":null}`, `{"enabled_chat_apps":["unknown-app"]}`} {
		c, w = newInvoiceControllerContext(http.MethodPut, "/fixture", body, 26001, 1, common.RoleKeyUser)
		UpdateChatPreferences(c)
		require.Equal(t, http.StatusBadRequest, w.Code)
	}
	c, w = newInvoiceControllerContext(http.MethodPut, "/fixture", `{"enabled_chat_apps":[]}`, 26001, 1, common.RoleKeyUser)
	UpdateChatPreferences(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, db.First(&owner, 26001).Error)
	require.Empty(t, owner.GetSetting().EnabledChatApps)
}
