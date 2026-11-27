package common

// Resource grants are independent from legacy numeric compatibility levels. A grant
// never supplies ownership: controllers must additionally enforce its scope.
type ResourceGrant struct {
	View    bool   `json:"view"`
	Use     bool   `json:"use"`
	Create  bool   `json:"create"`
	Operate bool   `json:"operate"`
	Update  bool   `json:"update"`
	Delete  bool   `json:"delete"`
	Scope   string `json:"scope"`
}

var RBACResources = []string{"platform", "supplier_procurement", "supply_models", "supply_prices", "resale_models", "resale_prices", "members", "keys", "subscriptions", "payments", "invoice_requests", "invoice_issuance", "vouchers", "audit"}

func ResourcePermissions(role string) map[string]ResourceGrant {
	result := make(map[string]ResourceGrant, len(RBACResources))
	for _, resource := range RBACResources {
		result[resource] = ResourceGrant{Scope: "none"}
	}
	grant := func(resources []string, scope, actions string) {
		for _, resource := range resources {
			g := ResourceGrant{Scope: scope}
			for _, a := range actions {
				switch a {
				case 'v':
					g.View = true
				case 'u':
					g.Use = true
				case 'c':
					g.Create = true
				case 'o':
					g.Operate = true
				case 'e':
					g.Update = true
				case 'd':
					g.Delete = true
				}
			}
			result[resource] = g
		}
	}
	// Unknown identities fail closed instead of inheriting user grants.
	switch role {
	case RoleKeyRoot:
		grant(RBACResources, "platform", "vucoed")
	case RoleKeySupplierAdmin:
		grant([]string{"supply_models", "supply_prices"}, "supplier", "vucoe")
		grant([]string{"members", "keys"}, "tenant", "vucoed")
		grant([]string{"payments", "audit"}, "supplier", "v")
	case RoleKeyDistributorAdmin:
		grant([]string{"resale_models", "resale_prices"}, "distribution", "vucoe")
		grant([]string{"members", "keys"}, "tenant", "vucoed")
		grant([]string{"subscriptions", "payments", "invoice_requests", "vouchers"}, "tenant", "vucoe")
		grant([]string{"audit"}, "tenant", "v")
	case RoleKeyEnterpriseAdmin:
		grant([]string{"members", "keys"}, "tenant", "vucoed")
		grant([]string{"subscriptions", "payments", "invoice_requests"}, "tenant", "vucoe")
		grant([]string{"vouchers"}, "tenant", "vu")
		grant([]string{"audit"}, "tenant", "v")
	case RoleKeyUser:
		grant([]string{"keys"}, "self", "vucoed")
		grant([]string{"subscriptions", "payments", "invoice_requests"}, "self", "vucoe")
		grant([]string{"vouchers"}, "self", "vu")
	}
	return result
}

func ResourceAllowed(role, resource, action string) bool {
	g, ok := ResourcePermissions(role)[resource]
	if !ok {
		return false
	}
	switch action {
	case "view":
		return g.View
	case "use":
		return g.Use
	case "create":
		return g.Create
	case "operate":
		return g.Operate
	case "update":
		return g.Update
	case "delete":
		return g.Delete
	}
	return false
}
