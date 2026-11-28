package model

import (
	"errors"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RBACPolicy struct {
	Family      string `json:"family" gorm:"primaryKey;size:32"`
	Permissions string `json:"-" gorm:"type:text;not null"`
	Delegate    bool   `json:"delegate"`
	Version     int    `json:"version"`
}
type ScopedRole struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	TenantId    int    `json:"tenant_id" gorm:"index;not null"`
	Family      string `json:"family" gorm:"size:32;not null"`
	Name        string `json:"name" gorm:"size:128;not null"`
	Permissions string `json:"-" gorm:"type:text;not null"`
	Active      bool   `json:"active"`
	Version     int    `json:"version"`
}
type RBACBinding struct {
	UserId int `json:"user_id" gorm:"primaryKey"`
	RoleId int `json:"role_id" gorm:"index;not null"`
}
type RBACAudit struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	ActorId      int    `json:"actor_id"`
	TargetUserId int    `json:"target_user_id"`
	RoleId       int    `json:"role_id"`
	TenantId     int    `json:"tenant_id" gorm:"index"`
	Action       string `json:"action" gorm:"size:32"`
	Snapshot     string `json:"snapshot" gorm:"type:text"`
	CreatedAt    int64  `json:"created_at" gorm:"index"`
}
type ScopedRoleInput struct {
	Id       int                             `json:"id"`
	TenantId int                             `json:"tenant_id"`
	Family   string                          `json:"family"`
	Name     string                          `json:"name"`
	Active   bool                            `json:"active"`
	Version  int                             `json:"version"`
	Grants   map[string]common.ResourceGrant `json:"grants"`
}
type RBACState struct {
	Family      string                          `json:"family"`
	TenantId    int                             `json:"tenant_id"`
	RoleId      int                             `json:"role_id"`
	Grants      map[string]common.ResourceGrant `json:"grants"`
	CanDelegate bool                            `json:"can_delegate"`
}

type RBACPolicyView struct {
	Family   string                          `json:"family"`
	Delegate bool                            `json:"delegate"`
	Version  int                             `json:"version"`
	Grants   map[string]common.ResourceGrant `json:"grants"`
	Ceiling  map[string]common.ResourceGrant `json:"ceiling"`
}
type ScopedRoleView struct {
	ScopedRole
	Grants map[string]common.ResourceGrant `json:"grants"`
}
type RBACWorkspace struct {
	State     RBACState        `json:"state"`
	Policies  []RBACPolicyView `json:"policies"`
	Roles     []ScopedRoleView `json:"roles"`
	Audits    []RBACAudit      `json:"audits"`
	Truncated bool             `json:"truncated"`
}

func GetRBACWorkspace(userID, tenantFilter int) (RBACWorkspace, error) {
	state, err := EffectiveRBAC(userID)
	if err != nil {
		return RBACWorkspace{}, err
	}
	result := RBACWorkspace{State: state, Policies: []RBACPolicyView{}, Roles: []ScopedRoleView{}, Audits: []RBACAudit{}}
	root := common.IsRootRole(state.Family)
	if !root && !BusinessRoleFamily(state.Family) {
		return result, ErrRBACDenied
	}
	families := []string{state.Family}
	if root {
		families = []string{common.RoleKeySupplierAdmin, common.RoleKeyDistributorAdmin, common.RoleKeyEnterpriseAdmin}
	}
	for _, family := range families {
		policy, grants, err := readPolicy(DB, family, false)
		if err != nil {
			return result, err
		}
		result.Policies = append(result.Policies, RBACPolicyView{Family: family, Delegate: policy.Delegate, Version: policy.Version, Grants: grants, Ceiling: common.ResourcePermissions(family)})
	}
	q := DB.Model(&ScopedRole{})
	if root {
		if tenantFilter > 0 {
			q = q.Where("tenant_id = ?", tenantFilter)
		}
	} else {
		if tenantFilter > 0 && tenantFilter != state.TenantId {
			return result, ErrRBACDenied
		}
		q = q.Where("tenant_id = ? AND family = ?", state.TenantId, state.Family)
		if !state.CanDelegate {
			q = q.Where("id = ?", state.RoleId)
		}
	}
	var rows []ScopedRole
	if err = q.Order("id DESC").Limit(101).Find(&rows).Error; err != nil {
		return result, err
	}
	if len(rows) > 100 {
		result.Truncated = true
		rows = rows[:100]
	}
	for _, row := range rows {
		view := ScopedRoleView{ScopedRole: row}
		if err = common.UnmarshalJsonStr(row.Permissions, &view.Grants); err != nil {
			return result, err
		}
		result.Roles = append(result.Roles, view)
	}
	if root || state.CanDelegate {
		audit := DB.Model(&RBACAudit{})
		if !root {
			audit = audit.Where("tenant_id = ?", state.TenantId)
		} else if tenantFilter > 0 {
			audit = audit.Where("tenant_id = ?", tenantFilter)
		}
		if err = audit.Order("id DESC").Limit(20).Find(&result.Audits).Error; err != nil {
			return result, err
		}
	}
	return result, nil
}

var ErrRBACDenied = errors.New("role operation denied")
var ErrRBACConflict = errors.New("role changed; refresh before retrying")

func BusinessRoleFamily(family string) bool {
	return family == common.RoleKeySupplierAdmin || family == common.RoleKeyDistributorAdmin || family == common.RoleKeyEnterpriseAdmin
}
func intersectGrant(a, b common.ResourceGrant) common.ResourceGrant {
	return common.ResourceGrant{View: a.View && b.View, Use: a.Use && b.Use, Create: a.Create && b.Create, Operate: a.Operate && b.Operate, Update: a.Update && b.Update, Delete: a.Delete && b.Delete, Scope: b.Scope}
}
func validateGrants(input, ceiling map[string]common.ResourceGrant) (map[string]common.ResourceGrant, error) {
	result := make(map[string]common.ResourceGrant, len(ceiling))
	for resource, cap := range ceiling {
		result[resource] = common.ResourceGrant{Scope: cap.Scope}
	}
	for resource, g := range input {
		cap, ok := ceiling[resource]
		if !ok || (g.Scope != "" && g.Scope != cap.Scope) {
			return nil, ErrRBACDenied
		}
		g.Scope = cap.Scope
		if intersectGrant(g, cap) != g || (!g.View && (g.Use || g.Create || g.Operate || g.Update || g.Delete)) {
			return nil, ErrRBACDenied
		}
		result[resource] = g
	}
	return result, nil
}
func readPolicy(tx *gorm.DB, family string, lock bool) (RBACPolicy, map[string]common.ResourceGrant, error) {
	policy := RBACPolicy{Family: family}
	caps := common.ResourcePermissions(family)
	q := tx.Where("family = ?", family)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	lookup := q.Limit(1).Find(&policy)
	err := lookup.Error
	if err == nil && lookup.RowsAffected == 0 {
		return policy, caps, nil
	}
	if err != nil {
		return policy, nil, err
	}
	var stored map[string]common.ResourceGrant
	if err = common.UnmarshalJsonStr(policy.Permissions, &stored); err != nil {
		return policy, nil, err
	}
	for resource, cap := range caps {
		caps[resource] = intersectGrant(stored[resource], cap)
	}
	return policy, caps, nil
}
func EffectiveRBAC(userID int) (RBACState, error) {
	var user User
	if err := DB.Select("id", "role", "role_key", "tenant_id", "status").First(&user, userID).Error; err != nil {
		return RBACState{}, err
	}
	family := user.NormalizedRoleKey()
	state := RBACState{Family: family, TenantId: user.TenantId, Grants: common.ResourcePermissions(family), CanDelegate: common.IsRootRole(family)}
	if user.Status != common.UserStatusEnabled {
		return RBACState{}, ErrRBACDenied
	}
	if !BusinessRoleFamily(family) {
		return state, nil
	}
	if user.TenantId <= 1 {
		return RBACState{}, ErrRBACDenied
	}
	policy, caps, err := readPolicy(DB, family, false)
	if err != nil {
		return RBACState{}, err
	}
	state.Grants = caps
	var binding RBACBinding
	lookup := DB.Where("user_id = ?", userID).Limit(1).Find(&binding)
	err = lookup.Error
	if err == nil && lookup.RowsAffected == 0 {
		state.CanDelegate = policy.Delegate
		return state, nil
	}
	if err != nil {
		return RBACState{}, err
	}
	var role ScopedRole
	if err = DB.First(&role, binding.RoleId).Error; err != nil {
		return RBACState{}, err
	}
	if role.TenantId != user.TenantId || role.Family != family {
		return RBACState{}, ErrRBACDenied
	}
	state.RoleId = role.Id
	var stored map[string]common.ResourceGrant
	if err = common.UnmarshalJsonStr(role.Permissions, &stored); err != nil {
		return RBACState{}, err
	}
	for resource, cap := range caps {
		if role.Active {
			state.Grants[resource] = intersectGrant(stored[resource], cap)
		} else {
			state.Grants[resource] = common.ResourceGrant{Scope: cap.Scope}
		}
	}
	return state, nil
}
func auditRBAC(tx *gorm.DB, actor, target, role, tenant int, action string, snapshot interface{}) error {
	data, err := common.Marshal(snapshot)
	if err != nil {
		return err
	}
	return tx.Create(&RBACAudit{ActorId: actor, TargetUserId: target, RoleId: role, TenantId: tenant, Action: action, Snapshot: string(data), CreatedAt: common.GetTimestamp()}).Error
}
func lockRBACActor(tx *gorm.DB, id int) (User, error) {
	var actor User
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&actor, id).Error
	if err == nil && actor.Status != common.UserStatusEnabled {
		err = ErrRBACDenied
	}
	return actor, err
}
func SaveRBACPolicy(actorID int, family string, input map[string]common.ResourceGrant, delegate bool, version int) (RBACPolicy, error) {
	var result RBACPolicy
	if !BusinessRoleFamily(family) {
		return result, ErrRBACDenied
	}
	grants, err := validateGrants(input, common.ResourcePermissions(family))
	if err != nil {
		return result, err
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		actor, err := lockRBACActor(tx, actorID)
		if err != nil {
			return err
		}
		if !common.IsRootRole(actor.NormalizedRoleKey()) {
			return ErrRBACDenied
		}
		current, _, err := readPolicy(tx, family, true)
		if err != nil {
			return err
		}
		if current.Version != version {
			return ErrRBACConflict
		}
		encoded, err := common.Marshal(grants)
		if err != nil {
			return err
		}
		result = RBACPolicy{Family: family, Permissions: string(encoded), Delegate: delegate, Version: version + 1}
		if version == 0 {
			err = tx.Create(&result).Error
		} else {
			update := tx.Model(&RBACPolicy{}).Where("family = ? AND version = ?", family, version).Updates(map[string]interface{}{"permissions": string(encoded), "delegate": delegate, "version": version + 1})
			err = update.Error
			if err == nil && update.RowsAffected != 1 {
				return ErrRBACConflict
			}
		}
		if err != nil {
			return err
		}
		return auditRBAC(tx, actorID, 0, 0, 0, "policy.update", map[string]interface{}{"family": family, "version": version + 1, "delegate": delegate, "grants": grants})
	})
	return result, err
}
func managerCaps(tx *gorm.DB, actor User, family string, tenant int) (map[string]common.ResourceGrant, error) {
	if !BusinessRoleFamily(family) || tenant <= 1 {
		return nil, ErrRBACDenied
	}
	var t Tenant
	if err := tx.Where("id = ? AND status = ?", tenant, 1).First(&t).Error; err != nil {
		return nil, ErrRBACDenied
	}
	policy, caps, err := readPolicy(tx, family, true)
	if err != nil {
		return nil, err
	}
	if common.IsRootRole(actor.NormalizedRoleKey()) {
		return caps, nil
	}
	if actor.TenantId != tenant || actor.NormalizedRoleKey() != family || !policy.Delegate {
		return nil, ErrRBACDenied
	}
	var n int64
	if err = tx.Model(&RBACBinding{}).Where("user_id = ?", actor.Id).Count(&n).Error; err != nil {
		return nil, err
	}
	if n != 0 {
		return nil, ErrRBACDenied
	}
	return caps, nil
}
func SaveScopedRole(actorID int, input ScopedRoleInput) (ScopedRole, error) {
	var result ScopedRole
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 128 || input.Version < 0 {
		return result, errors.New("invalid role name or version")
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		actor, err := lockRBACActor(tx, actorID)
		if err != nil {
			return err
		}
		caps, err := managerCaps(tx, actor, input.Family, input.TenantId)
		if err != nil {
			return err
		}
		grants, err := validateGrants(input.Grants, caps)
		if err != nil {
			return err
		}
		encoded, err := common.Marshal(grants)
		if err != nil {
			return err
		}
		result = ScopedRole{Id: input.Id, TenantId: input.TenantId, Family: input.Family, Name: input.Name, Permissions: string(encoded), Active: input.Active, Version: input.Version + 1}
		action := "role.create"
		if input.Id == 0 {
			if input.Version != 0 {
				return ErrRBACConflict
			}
			if err = tx.Create(&result).Error; err != nil {
				return err
			}
		} else {
			action = "role.update"
			var current ScopedRole
			if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, input.Id).Error; err != nil {
				return err
			}
			if current.TenantId != input.TenantId || current.Family != input.Family {
				return ErrRBACDenied
			}
			if current.Version != input.Version {
				return ErrRBACConflict
			}
			update := tx.Model(&current).Where("version = ?", input.Version).Updates(map[string]interface{}{"name": result.Name, "permissions": result.Permissions, "active": result.Active, "version": result.Version})
			if update.Error != nil {
				return update.Error
			}
			if update.RowsAffected != 1 {
				return ErrRBACConflict
			}
		}
		return auditRBAC(tx, actorID, 0, result.Id, input.TenantId, action, map[string]interface{}{"family": input.Family, "name": input.Name, "active": input.Active, "version": result.Version, "grants": grants})
	})
	return result, err
}
func AssignScopedRole(actorID, targetID, roleID int) error {
	return changeScopedRole(actorID, targetID, roleID, false)
}
func RevokeScopedRole(actorID, targetID int) error {
	return changeScopedRole(actorID, targetID, 0, true)
}
func changeScopedRole(actorID, targetID, roleID int, revoke bool) error {
	if actorID == targetID || targetID <= 0 {
		return ErrRBACDenied
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
		if actor == nil || target == nil || actor.Status != common.UserStatusEnabled || target.Status != common.UserStatusEnabled || target.NormalizedRoleKey() == common.RoleKeyRoot || target.Role == common.RoleRootUser {
			return ErrRBACDenied
		}
		var binding RBACBinding
		lookup := tx.Where("user_id = ?", targetID).Limit(1).Find(&binding)
		bindingErr := lookup.Error
		if bindingErr == nil && lookup.RowsAffected == 0 {
			bindingErr = gorm.ErrRecordNotFound
		}
		if bindingErr != nil && !errors.Is(bindingErr, gorm.ErrRecordNotFound) {
			return bindingErr
		}
		if revoke {
			if bindingErr != nil {
				return ErrRBACDenied
			}
			roleID = binding.RoleId
		}
		var peek ScopedRole
		if err := tx.First(&peek, roleID).Error; err != nil {
			return err
		}
		if peek.TenantId != target.TenantId {
			return ErrRBACDenied
		}
		if _, err := managerCaps(tx, *actor, peek.Family, peek.TenantId); err != nil {
			return err
		}
		var role ScopedRole
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&role, roleID).Error; err != nil {
			return err
		}
		if !role.Active && !revoke {
			return ErrRBACDenied
		}
		if !common.IsRootRole(actor.NormalizedRoleKey()) {
			if bindingErr == nil {
				var previous ScopedRole
				if err := tx.First(&previous, binding.RoleId).Error; err != nil {
					return err
				}
				if previous.TenantId != peek.TenantId || previous.Family != peek.Family {
					return ErrRBACDenied
				}
			} else if target.NormalizedRoleKey() != common.RoleKeyUser || target.Role >= common.RoleAdminUser {
				return ErrRBACDenied
			}
		}
		family := role.Family
		action := "assignment.create"
		if revoke {
			family = common.RoleKeyUser
			action = "assignment.revoke"
			if err := tx.Where("user_id = ?", targetID).Delete(&RBACBinding{}).Error; err != nil {
				return err
			}
		} else {
			next := RBACBinding{UserId: targetID, RoleId: roleID}
			if bindingErr == nil {
				if err := tx.Model(&binding).Update("role_id", roleID).Error; err != nil {
					return err
				}
			} else if err := tx.Create(&next).Error; err != nil {
				return err
			}
		}
		if err := tx.Table("users").Where("id = ?", targetID).Updates(map[string]interface{}{"role_key": family, "role": common.RoleKeyToLegacyRole(family), "auth_epoch": gorm.Expr("auth_epoch + ?", 1), "access_token": nil}).Error; err != nil {
			return err
		}
		return auditRBAC(tx, actorID, targetID, roleID, target.TenantId, action, map[string]interface{}{"family": family, "role_version": role.Version})
	})
}

// Customer inference keys do not use dashboard sessions. Check live role state
// before upstream distribution as well, so an existing key cannot bypass RBAC.
func RBACInferenceAllowed(userID int) (bool, error) {
	state, err := EffectiveRBAC(userID)
	if err != nil {
		return false, err
	}
	if !BusinessRoleFamily(state.Family) {
		return true, nil
	}
	return state.Grants["model_calls"].Use, nil
}
