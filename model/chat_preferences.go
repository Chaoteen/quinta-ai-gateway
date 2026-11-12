package model

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
