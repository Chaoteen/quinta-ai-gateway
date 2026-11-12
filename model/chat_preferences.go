package model

import (
	"errors"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/dto"
	"github.com/Chaoteen/quinta-ai-gateway/setting/operation_setting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ConfigureChatKey(userId int, app string, tokenId int, create bool) (int, dto.UserSetting, error) {
	var out dto.UserSetting
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userId).Error; err != nil {
			return err
		}
		if user.Status != common.UserStatusEnabled {
			return errors.New("Account is not enabled")
		}
		raw := map[string]any{}
		if user.Setting != "" {
			if err := common.UnmarshalJsonStr(user.Setting, &raw); err != nil {
				return err
			}
		}
		if raw == nil {
			raw = map[string]any{}
		}
		out = user.GetSetting()
		if out.ChatKeyBindings == nil {
			out.ChatKeyBindings = map[string]int{}
		}
		var token Token
		usable := func(t Token) bool {
			return t.Status == common.TokenStatusEnabled && (t.ExpiredTime == -1 || t.ExpiredTime > common.GetTimestamp()) && (t.UnlimitedQuota || t.RemainQuota > 0)
		}
		// Reuse a valid existing tool key on repeated create clicks.
		if create && out.ChatKeyBindings[app] > 0 {
			if tx.Where("id = ? AND user_id = ? AND tenant_id = ?", out.ChatKeyBindings[app], userId, user.TenantId).First(&token).Error == nil && usable(token) {
				tokenId = token.Id
				create = false
			}
		}
		if create {
			if user.Quota <= 0 {
				return errors.New("Account balance is empty. Add quota before creating a tool key.")
			}
			var count int64
			if err := tx.Model(&Token{}).Where("user_id = ?", userId).Count(&count).Error; err != nil {
				return err
			}
			if count >= int64(operation_setting.GetMaxUserTokens()) {
				return errors.New("API key limit reached")
			}
			key, err := common.GenerateKey()
			if err != nil {
				return err
			}
			name := []rune(app + " - chat")
			if len(name) > 50 {
				name = name[:50]
			}
			token = Token{UserId: userId, TenantId: user.TenantId, OrganizationId: user.OrganizationId, DepartmentId: user.DepartmentId, DistributionChannelId: user.DistributionChannelId, Name: string(name), Key: key, Status: common.TokenStatusEnabled, Group: user.Group, RemainQuota: user.Quota, ExpiredTime: common.GetTimestamp() + 30*86400, CreatedTime: common.GetTimestamp(), AccessedTime: common.GetTimestamp()}
			if err := tx.Create(&token).Error; err != nil {
				return err
			}
			tokenId = token.Id
		} else {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ? AND tenant_id = ?", tokenId, userId, user.TenantId).First(&token).Error; err != nil {
				return errors.New("Selected API key is unavailable")
			}
			if !usable(token) {
				return errors.New("Selected API key is disabled, expired or exhausted")
			}
		}
		out.ChatKeyBindings[app] = tokenId
		if !common.StringsContains(out.EnabledChatApps, app) {
			out.EnabledChatApps = append(out.EnabledChatApps, app)
		}
		raw["chat_key_bindings"] = out.ChatKeyBindings
		raw["enabled_chat_apps"] = out.EnabledChatApps
		encoded, err := common.Marshal(raw)
		if err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", userId).Update("setting", string(encoded)).Error
	})
	if err == nil {
		err = InvalidateUserCache(userId)
	}
	return tokenId, out, err
}

// UpdateChatPreferences changes only this user's app preferences; other settings survive.
func UpdateChatPreferences(userId int, apps []string) error {
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userId).Error; err != nil {
			return err
		}
		settings := map[string]any{}
		if user.Setting != "" {
			if err := common.UnmarshalJsonStr(user.Setting, &settings); err != nil {
				return err
			}
		}
		if settings == nil {
			settings = map[string]any{}
		}
		settings["enabled_chat_apps"] = apps
		data, err := common.Marshal(settings)
		if err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", userId).Update("setting", string(data)).Error
	})
	if err != nil {
		return err
	}
	return InvalidateUserCache(userId)
}
