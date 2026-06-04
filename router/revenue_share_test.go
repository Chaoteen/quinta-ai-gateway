package router

import (
	"net/http"
	"testing"
)

func TestRevenueShareRuleWriteDeniedForReadOnlyAndUserRoles(t *testing.T) {
	r := setupRoleAdoptionRouter(t)
	body := `{"tenant_id":1,"rule_name":"deny","rule_scope":"global","platform_share_rate":100,"master_distributor_share_rate":0,"distributor_share_rate":0,"enabled":true}`
	roles := []string{"organization_admin", "finance", "auditor", "user"}
	routes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/revenue-share/rules", body},
		{http.MethodPut, "/api/revenue-share/rules/1", body},
		{http.MethodPost, "/api/revenue-share/rules/1/enable", ""},
		{http.MethodPost, "/api/revenue-share/rules/1/disable", ""},
	}

	for _, roleName := range roles {
		for _, route := range routes {
			assertRoleAdoptionMethodRejected(t, r, route.method, route.path, route.body, roleAdoptionUsers[roleName])
		}
	}
}
