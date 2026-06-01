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
