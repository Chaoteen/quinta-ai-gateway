package model

import "github.com/Chaoteen/quinta-ai-gateway/common"

type AdminSecurityAccount struct {
	Id         int    `json:"id"`
	Username   string `json:"username"`
	RoleKey    string `json:"role_key"`
	HasTOTP    bool   `json:"has_totp"`
	HasPasskey bool   `json:"has_passkey"`
}
type AdminSecurityReadiness struct {
	Accounts           []AdminSecurityAccount `json:"accounts"`
	EnrollmentComplete bool                   `json:"enrollment_complete"`
	Truncated          bool                   `json:"truncated"`
	EvaluatedAt        int64                  `json:"evaluated_at"`
}

// This is enrollment evidence only. It does not assert recovery readiness or
// enable an enforcement policy. Secrets, public keys and backup codes are never read.
func GetAdminSecurityReadiness() (AdminSecurityReadiness, error) {
	result := AdminSecurityReadiness{Accounts: []AdminSecurityAccount{}, EvaluatedAt: common.GetTimestamp()}
	var users []User
	roles := []string{common.RoleKeyRoot, common.RoleKeyTenantAdmin, common.RoleKeyOrganizationAdmin, common.RoleKeyFinance, common.RoleKeyOps, common.RoleKeyAuditor}
	err := DB.Select("id", "username", "role", "role_key").Where("status = ? AND (role >= ? OR role_key IN ?)", common.UserStatusEnabled, common.RoleAdminUser, roles).Order("id").Limit(501).Find(&users).Error
	if err != nil {
		return result, err
	}
	result.Truncated = len(users) > 500
	if result.Truncated {
		users = users[:500]
	}
	if len(users) == 0 {
		return result, nil
	}
	ids := make([]int, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.Id)
	}
	var totpIDs, passkeyIDs []int
	if err := DB.Model(&TwoFA{}).Where("user_id IN ? AND is_enabled = ?", ids, true).Pluck("user_id", &totpIDs).Error; err != nil {
		return result, err
	}
	if err := DB.Model(&PasskeyCredential{}).Where("user_id IN ? AND clone_warning = ?", ids, false).Pluck("user_id", &passkeyIDs).Error; err != nil {
		return result, err
	}
	totp, passkey := map[int]bool{}, map[int]bool{}
	for _, id := range totpIDs {
		totp[id] = true
	}
	for _, id := range passkeyIDs {
		passkey[id] = true
	}
	result.EnrollmentComplete = !result.Truncated
	for _, u := range users {
		account := AdminSecurityAccount{Id: u.Id, Username: u.Username, RoleKey: u.NormalizedRoleKey(), HasTOTP: totp[u.Id], HasPasskey: passkey[u.Id]}
		result.Accounts = append(result.Accounts, account)
		if !account.HasTOTP && !account.HasPasskey {
			result.EnrollmentComplete = false
		}
	}
	return result, nil
}
