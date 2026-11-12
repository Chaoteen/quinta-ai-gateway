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

func TestChatToolCreatesBindsAndReusesOnlyOwnedUsableKeys(t *testing.T) {
	db := setupInvoiceControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}))
	owner := model.User{Id: 26101, Username: "chat-key-owner", AffCode: "chat-key-owner", TenantId: 1, Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Quota: 68493, Group: "default", Setting: `{"language":"zh","custom_option":"keep"}`}
	other := model.User{Id: 26102, Username: "chat-key-other", AffCode: "chat-key-other", TenantId: 2, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&other).Error)
	foreign := model.Token{UserId: other.Id, TenantId: 2, Key: "foreign-fixture-not-a-real-secret", Status: 1, ExpiredTime: -1, UnlimitedQuota: true}
	require.NoError(t, db.Create(&foreign).Error)
	for attempt := 0; attempt < 2; attempt++ {
		c, w := newInvoiceControllerContext(http.MethodPost, "/api/user/self/chat-key", `{"app":"Cherry Studio","create":true}`, owner.Id, 1, common.RoleKeyUser)
		ConfigureChatKey(c)
		require.Contains(t, w.Body.String(), `"success":true`)
	}
	var tokens []model.Token
	require.NoError(t, db.Where("user_id = ?", owner.Id).Find(&tokens).Error)
	require.Len(t, tokens, 1)
	token := tokens[0]
	require.Equal(t, 1, token.TenantId)
	require.Equal(t, 68493, token.RemainQuota)
	require.False(t, token.UnlimitedQuota)
	require.Equal(t, "default", token.Group)
	require.InDelta(t, common.GetTimestamp()+30*86400, token.ExpiredTime, 5)
	require.NoError(t, db.First(&owner, owner.Id).Error)
	require.Equal(t, 68493, owner.Quota)
	require.Equal(t, token.Id, owner.GetSetting().ChatKeyBindings["Cherry Studio"])
	require.Equal(t, []string{"Cherry Studio"}, owner.GetSetting().EnabledChatApps)
	require.Contains(t, owner.Setting, `"custom_option":"keep"`)
	body, err := common.Marshal(map[string]any{"app": "AionUI", "token_id": foreign.Id})
	require.NoError(t, err)
	c, w := newInvoiceControllerContext(http.MethodPost, "/fixture", string(body), owner.Id, 1, common.RoleKeyUser)
	ConfigureChatKey(c)
	require.NotContains(t, w.Body.String(), `"success":true`)
	require.NotContains(t, w.Body.String(), foreign.Key)
	body, err = common.Marshal(map[string]any{"app": "AionUI", "token_id": token.Id})
	require.NoError(t, err)
	c, w = newInvoiceControllerContext(http.MethodPost, "/fixture", string(body), owner.Id, 1, common.RoleKeyUser)
	ConfigureChatKey(c)
	require.Contains(t, w.Body.String(), `"success":true`)
	require.NotContains(t, w.Body.String(), token.Key)
	require.NoError(t, db.First(&owner, owner.Id).Error)
	require.Equal(t, token.Id, owner.GetSetting().ChatKeyBindings["AionUI"])
	require.NoError(t, db.Model(&token).Update("status", common.TokenStatusDisabled).Error)
	c, w = newInvoiceControllerContext(http.MethodPost, "/fixture", string(body), owner.Id, 1, common.RoleKeyUser)
	ConfigureChatKey(c)
	require.NotContains(t, w.Body.String(), `"success":true`)
}
