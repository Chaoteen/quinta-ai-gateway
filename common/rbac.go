package common

import "strings"

const (
	RoleKeyRoot              = "root"
	RoleKeySupplierAdmin     = "supplier_admin"
	RoleKeyDistributorAdmin  = "distributor_admin"
	RoleKeyEnterpriseAdmin   = "enterprise_admin"
	RoleKeyTenantAdmin       = "tenant_admin"
	RoleKeyOrganizationAdmin = "organization_admin"
	RoleKeyFinance           = "finance"
	RoleKeyOps               = "ops"
	RoleKeyAuditor           = "auditor"
	RoleKeyUser              = "user"
)

func NormalizeRoleKey(roleKey string) string {
	switch strings.ToLower(strings.TrimSpace(roleKey)) {
	case RoleKeyRoot:
		return RoleKeyRoot
	case RoleKeySupplierAdmin, RoleKeyDistributorAdmin, RoleKeyEnterpriseAdmin:
		return strings.ToLower(strings.TrimSpace(roleKey))
	case RoleKeyTenantAdmin:
		return RoleKeyTenantAdmin
	case RoleKeyOrganizationAdmin:
		return RoleKeyOrganizationAdmin
	case RoleKeyFinance:
		return RoleKeyFinance
	case RoleKeyOps:
		return RoleKeyOps
	case RoleKeyAuditor:
		return RoleKeyAuditor
	case RoleKeyUser:
		return RoleKeyUser
	default:
		return RoleKeyUser
	}
}

func LegacyRoleToRoleKey(role int) string {
	switch role {
	case RoleRootUser:
		return RoleKeyRoot
	case RoleAdminUser:
		return RoleKeyTenantAdmin
	default:
		return RoleKeyUser
	}
}

func RoleKeyToLegacyRole(roleKey string) int {
	switch NormalizeRoleKey(roleKey) {
	case RoleKeyRoot:
		return RoleRootUser
	case RoleKeyTenantAdmin, RoleKeySupplierAdmin, RoleKeyDistributorAdmin, RoleKeyEnterpriseAdmin:
		return RoleAdminUser
	default:
		return RoleCommonUser
	}
}

func NormalizeRoleConsistency(role int, roleKey string) (int, string) {
	normalizedRoleKey := NormalizeRoleKey(roleKey)
	if strings.TrimSpace(roleKey) == "" {
		normalizedRoleKey = LegacyRoleToRoleKey(role)
	}
	return RoleKeyToLegacyRole(normalizedRoleKey), normalizedRoleKey
}

func IsRootRole(roleKey string) bool {
	return NormalizeRoleKey(roleKey) == RoleKeyRoot
}

func IsTenantAdminRole(roleKey string) bool {
	return NormalizeRoleKey(roleKey) == RoleKeyTenantAdmin
}

func IsOrganizationAdminRole(roleKey string) bool {
	return NormalizeRoleKey(roleKey) == RoleKeyOrganizationAdmin
}

func IsScopedAdminRole(roleKey string) bool {
	roleKey = NormalizeRoleKey(roleKey)
	return roleKey == RoleKeyRoot || roleKey == RoleKeyTenantAdmin || roleKey == RoleKeyOrganizationAdmin
}

func HasRole(roleKey string, requiredRole string) bool {
	roleKey = NormalizeRoleKey(roleKey)
	requiredRole = NormalizeRoleKey(requiredRole)
	if requiredRole == RoleKeyRoot {
		return roleKey == RoleKeyRoot
	}
	if roleKey == RoleKeyRoot {
		return true
	}
	return roleKey == requiredRole
}

func HasAnyRole(roleKey string, requiredRoles ...string) bool {
	for _, requiredRole := range requiredRoles {
		if HasRole(roleKey, requiredRole) {
			return true
		}
	}
	return false
}
