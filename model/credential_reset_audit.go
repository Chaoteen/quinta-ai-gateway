package model

import (
	"errors"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CredentialResetAudit is local transactional evidence, not immutable storage.
// Never store seeds, credential material, recovery codes or request bodies here.
type CredentialResetAudit struct {
	Id             int    `json:"id" gorm:"primaryKey"`
	ActorId        int    `json:"actor_id" gorm:"index;not null"`
	TargetId       int    `json:"target_id" gorm:"index;not null"`
	TargetTenantId int    `json:"target_tenant_id" gorm:"index;not null"`
	CredentialType string `json:"credential_type" gorm:"type:varchar(16);not null"`
	CreatedAt      int64  `json:"created_at" gorm:"index;not null"`
}

func AdminResetCredential(actorId, targetId int, kind string) error {
	if kind != "totp" && kind != "passkey" {
		return errors.New("invalid credential type")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		// Lock accounts in stable order and recheck authorization inside the mutation.
		var users []User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", []int{actorId, targetId}).Order("id").Find(&users).Error; err != nil {
			return err
		}
		var actor, target *User
		for i := range users {
			if users[i].Id == actorId {
				actor = &users[i]
			}
			if users[i].Id == targetId {
				target = &users[i]
			}
		}
		if actor == nil || target == nil || actorId == targetId || actor.Status != common.UserStatusEnabled || actor.Role < common.RoleAdminUser || target.Role >= actor.Role || common.IsRootRole(target.NormalizedRoleKey()) || (actor.Role != common.RoleRootUser && normalizeTenantId(actor.TenantId) != normalizeTenantId(target.TenantId)) {
			return errors.New("credential reset denied")
		}
		if kind == "totp" {
			result := tx.Unscoped().Where("user_id = ?", targetId).Delete(&TwoFA{})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return ErrTwoFANotEnabled
			}
			if err := tx.Unscoped().Where("user_id = ?", targetId).Delete(&TwoFABackupCode{}).Error; err != nil {
				return err
			}
		} else {
			result := tx.Unscoped().Where("user_id = ?", targetId).Delete(&PasskeyCredential{})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return ErrPasskeyNotFound
			}
		}
		return tx.Create(&CredentialResetAudit{ActorId: actorId, TargetId: targetId, TargetTenantId: target.TenantId, CredentialType: kind, CreatedAt: common.GetTimestamp()}).Error
	})
}

// ListCredentialResetAudits uses a bounded ID cursor; no credential joins.
func ListCredentialResetAudits(before int) ([]CredentialResetAudit, bool, error) {
	rows := []CredentialResetAudit{}
	q := DB.Model(&CredentialResetAudit{})
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	if err := q.Order("id DESC").Limit(21).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	more := len(rows) > 20
	if more {
		rows = rows[:20]
	}
	return rows, more, nil
}
