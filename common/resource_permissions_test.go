package common

import "testing"

func TestResourcePermissionBoundaries(t *testing.T) {
	for _, role := range []string{RoleKeySupplierAdmin, RoleKeyDistributorAdmin, RoleKeyEnterpriseAdmin, RoleKeyUser} {
		for _, action := range []string{"view", "use", "create", "operate", "update", "delete"} {
			if ResourceAllowed(role, "platform", action) || ResourceAllowed(role, "supplier_procurement", action) || ResourceAllowed(role, "invoice_issuance", action) {
				t.Fatalf("%s escalated", role)
			}
		}
		if ResourceAllowed(role, "payments", "delete") || ResourceAllowed(role, "audit", "delete") {
			t.Fatal("business evidence deletion allowed")
		}

	}
	if !ResourceAllowed(RoleKeySupplierAdmin, "supply_prices", "update") || ResourceAllowed(RoleKeyEnterpriseAdmin, "supply_prices", "update") {
		t.Fatal("pricing boundary")
	}
	if !ResourceAllowed(RoleKeyEnterpriseAdmin, "invoice_requests", "create") || ResourceAllowed(RoleKeyEnterpriseAdmin, "invoice_issuance", "operate") {
		t.Fatal("invoice request/issue boundary")
	}
	for _, role := range []string{"", "invented", "ROOT"} {
		if ResourceAllowed(role, "keys", "view") {
			t.Fatal("unknown role allowed")
		}
	}
	if ResourceAllowed(RoleKeyRoot, "invented", "delete") || ResourceAllowed(RoleKeyRoot, "keys", "invented") {
		t.Fatal("unknown permission allowed")
	}
	if !ResourceAllowed(RoleKeyRoot, "platform", "operate") {
		t.Fatal("root denied")
	}
}
