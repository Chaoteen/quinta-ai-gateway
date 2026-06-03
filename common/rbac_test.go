package common

import "testing"

func TestLegacyRoleToRoleKey(t *testing.T) {
	tests := []struct {
		name string
		role int
		want string
	}{
		{name: "legacy root maps to root", role: RoleRootUser, want: RoleKeyRoot},
		{name: "legacy admin maps to tenant admin", role: RoleAdminUser, want: RoleKeyTenantAdmin},
		{name: "legacy user maps to user", role: RoleCommonUser, want: RoleKeyUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LegacyRoleToRoleKey(tt.role); got != tt.want {
				t.Fatalf("LegacyRoleToRoleKey(%d) = %q, want %q", tt.role, got, tt.want)
			}
		})
	}
}

func TestRoleKeyToLegacyRole(t *testing.T) {
	tests := []struct {
		name    string
		roleKey string
		want    int
	}{
		{name: "root maps to legacy root", roleKey: RoleKeyRoot, want: RoleRootUser},
		{name: "tenant admin maps to legacy admin", roleKey: RoleKeyTenantAdmin, want: RoleAdminUser},
		{name: "organization admin maps to legacy common", roleKey: RoleKeyOrganizationAdmin, want: RoleCommonUser},
		{name: "finance maps to legacy common", roleKey: RoleKeyFinance, want: RoleCommonUser},
		{name: "ops maps to legacy common", roleKey: RoleKeyOps, want: RoleCommonUser},
		{name: "auditor maps to legacy common", roleKey: RoleKeyAuditor, want: RoleCommonUser},
		{name: "user maps to legacy common", roleKey: RoleKeyUser, want: RoleCommonUser},
		{name: "unknown maps to legacy common", roleKey: "unknown", want: RoleCommonUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoleKeyToLegacyRole(tt.roleKey); got != tt.want {
				t.Fatalf("RoleKeyToLegacyRole(%q) = %d, want %d", tt.roleKey, got, tt.want)
			}
		})
	}
}

func TestNormalizeRoleConsistency(t *testing.T) {
	tests := []struct {
		name        string
		role        int
		roleKey     string
		wantRole    int
		wantRoleKey string
	}{
		{name: "blank root maps from legacy role", role: RoleRootUser, roleKey: "", wantRole: RoleRootUser, wantRoleKey: RoleKeyRoot},
		{name: "blank admin maps from legacy role", role: RoleAdminUser, roleKey: "", wantRole: RoleAdminUser, wantRoleKey: RoleKeyTenantAdmin},
		{name: "blank common maps from legacy role", role: RoleCommonUser, roleKey: "", wantRole: RoleCommonUser, wantRoleKey: RoleKeyUser},
		{name: "root role key is authoritative", role: RoleCommonUser, roleKey: RoleKeyRoot, wantRole: RoleRootUser, wantRoleKey: RoleKeyRoot},
		{name: "tenant admin role key is authoritative", role: RoleCommonUser, roleKey: RoleKeyTenantAdmin, wantRole: RoleAdminUser, wantRoleKey: RoleKeyTenantAdmin},
		{name: "finance role key stays scoped common", role: RoleAdminUser, roleKey: RoleKeyFinance, wantRole: RoleCommonUser, wantRoleKey: RoleKeyFinance},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRole, gotRoleKey := NormalizeRoleConsistency(tt.role, tt.roleKey)
			if gotRole != tt.wantRole || gotRoleKey != tt.wantRoleKey {
				t.Fatalf("NormalizeRoleConsistency(%d, %q) = (%d, %q), want (%d, %q)", tt.role, tt.roleKey, gotRole, gotRoleKey, tt.wantRole, tt.wantRoleKey)
			}
		})
	}
}

func TestHasRole(t *testing.T) {
	if !HasRole(RoleKeyRoot, RoleKeyRoot) {
		t.Fatal("root should have root role")
	}
	if HasRole(RoleKeyTenantAdmin, RoleKeyRoot) {
		t.Fatal("tenant_admin should not have root role")
	}
	if !HasRole(RoleKeyTenantAdmin, RoleKeyTenantAdmin) {
		t.Fatal("tenant_admin should have tenant_admin role")
	}
	if HasRole(RoleKeyUser, RoleKeyTenantAdmin) {
		t.Fatal("user should not have tenant_admin role")
	}
}

func TestRoleKeyHelpers(t *testing.T) {
	if !IsOrganizationAdminRole(RoleKeyOrganizationAdmin) {
		t.Fatal("organization_admin should be recognized")
	}
	if IsOrganizationAdminRole(RoleKeyTenantAdmin) {
		t.Fatal("tenant_admin should not be recognized as organization_admin")
	}
	if !IsScopedAdminRole(RoleKeyRoot) {
		t.Fatal("root should be a scoped admin role")
	}
	if !IsScopedAdminRole(RoleKeyTenantAdmin) {
		t.Fatal("tenant_admin should be a scoped admin role")
	}
	if !IsScopedAdminRole(RoleKeyOrganizationAdmin) {
		t.Fatal("organization_admin should be a scoped admin role")
	}
	if IsScopedAdminRole(RoleKeyFinance) {
		t.Fatal("finance should not be a scoped admin role")
	}
	if IsScopedAdminRole(RoleKeyUser) {
		t.Fatal("user should not be a scoped admin role")
	}
}
