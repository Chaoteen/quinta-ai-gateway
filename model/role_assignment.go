package model

import (
	"errors"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleAssignmentAudit struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	ActorId      int    `json:"actor_id"`
	TargetId     int    `json:"target_id"`
	TenantId     int    `json:"tenant_id"`
	PreviousRole string `json:"previous_role" gorm:"size:32"`
	NewRole      string `json:"new_role" gorm:"size:32"`
	CreatedAt    int64  `json:"created_at"`
}

// Assignment changes the role inside the existing tenant, never moves a user's
// funds, API keys, subscriptions or history into a different enterprise.
func AssignBusinessRole(actorID, targetID int, role string) error {
	if role != common.RoleKeySupplierAdmin && role != common.RoleKeyDistributorAdmin && role != common.RoleKeyEnterpriseAdmin && role != common.RoleKeyUser {
		return errors.New("unsupported business role")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var users []User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", []int{actorID, targetID}).Order("id").Find(&users).Error; err != nil {
			return err
		}
		var actor, target *User
		for i := range users {
			if users[i].Id == actorID {
				actor = &users[i]
			}
			if users[i].Id == targetID {
				target = &users[i]
			}
		}
		if actor == nil || target == nil || actorID == targetID || actor.Status != common.UserStatusEnabled || actor.NormalizedRoleKey() != common.RoleKeyRoot || target.NormalizedRoleKey() == common.RoleKeyRoot || target.Role == common.RoleRootUser {
			return errors.New("role assignment denied")
		}
		// Implicit platform tenant 1 is shared by personal registrations, so it may
		// not be used as the isolation boundary of a business administrator.
		if target.TenantId <= 1 {
			return errors.New("dedicated enterprise tenant required")
		}
		var tenant Tenant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", target.TenantId, 1).First(&tenant).Error; err != nil {
			return errors.New("active tenant required")
		}
		previous := target.NormalizedRoleKey()
		if previous == role {
			return nil
		}
		updates := map[string]interface{}{"role": common.RoleKeyToLegacyRole(role), "role_key": role, "auth_epoch": gorm.Expr("auth_epoch + ?", 1), "access_token": nil}
		if err := tx.Table("users").Where("id = ?", targetID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Create(&RoleAssignmentAudit{ActorId: actorID, TargetId: targetID, TenantId: target.TenantId, PreviousRole: previous, NewRole: role, CreatedAt: common.GetTimestamp()}).Error
	})
}
