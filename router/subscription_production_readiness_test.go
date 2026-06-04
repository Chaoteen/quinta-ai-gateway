package router

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
)

func assertSubscriptionPrivilegeDenied(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	success, message := decodeRoleAdoptionBasicResponse(t, recorder)
	if success {
		t.Fatalf("expected privilege denial, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(message, "权限不足") && !strings.Contains(message, "insufficient_privilege") {
		t.Fatalf("expected privilege denial message, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func assertSubscriptionTenantDenied(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	success, message := decodeRoleAdoptionBasicResponse(t, recorder)
	if success || message != roleAdoptionTenantDeniedMessage {
		t.Fatalf("expected tenant scope denial, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func requireSubscriptionSuccess(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if !decodeRoleAdoptionSuccess(t, recorder) {
		t.Fatalf("expected success, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func decodeSubscriptionPlanID(t *testing.T, recorder *httptest.ResponseRecorder) int {
	t.Helper()
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Plan struct {
				Id int `json:"id"`
			} `json:"plan"`
		} `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode plan response %q: %v", recorder.Body.String(), err)
	}
	if !response.Success || response.Data.Plan.Id <= 0 {
		t.Fatalf("expected plan response with id, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	return response.Data.Plan.Id
}

func decodeSubscriptionIDs(t *testing.T, recorder *httptest.ResponseRecorder) []int {
	t.Helper()
	var response struct {
		Success bool `json:"success"`
		Data    []struct {
			Subscription struct {
				Id int `json:"id"`
			} `json:"subscription"`
		} `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode subscription list %q: %v", recorder.Body.String(), err)
	}
	if !response.Success {
		t.Fatalf("expected subscription list success, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	ids := make([]int, 0, len(response.Data))
	for _, item := range response.Data {
		ids = append(ids, item.Subscription.Id)
	}
	return ids
}

func decodeSubscriptionPlanCodes(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()
	var response struct {
		Success bool `json:"success"`
		Data    []struct {
			Plan struct {
				Code string `json:"code"`
			} `json:"plan"`
		} `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode plan list %q: %v", recorder.Body.String(), err)
	}
	if !response.Success {
		t.Fatalf("expected plan list success, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	codes := make([]string, 0, len(response.Data))
	for _, item := range response.Data {
		codes = append(codes, item.Plan.Code)
	}
	return codes
}

func requireStringPresence(t *testing.T, values []string, value string, want bool) {
	t.Helper()
	for _, current := range values {
		if current == value {
			if !want {
				t.Fatalf("expected %q to be absent from %v", value, values)
			}
			return
		}
	}
	if want {
		t.Fatalf("expected %q to be present in %v", value, values)
	}
}

func subscriptionPlanBody(code string) string {
	return `{"plan":{"code":"` + code + `","name":"` + code + `","description":"test plan","monthly_price":12,"yearly_price":120,"token_quota":3000,"request_quota":50,"model_quota":"","status":"enabled"}}`
}

func TestSubscriptionRBACWriteDenyMatrix(t *testing.T) {
	denyRoles := []string{"organization_admin", "finance", "auditor", "user"}
	writeRoutes := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create plan", method: http.MethodPost, path: "/api/subscription/admin/plans", body: subscriptionPlanBody("deny-create")},
		{name: "update plan", method: http.MethodPut, path: "/api/subscription/admin/plans/1", body: subscriptionPlanBody("deny-update")},
		{name: "disable plan", method: http.MethodPatch, path: "/api/subscription/admin/plans/1", body: `{"status":"disabled"}`},
		{name: "assign subscription", method: http.MethodPost, path: "/api/subscription/admin/users/6/subscriptions", body: `{"plan_id":1}`},
		{name: "renew subscription", method: http.MethodPatch, path: "/api/subscription/admin/user-subscriptions/1/renew", body: ""},
		{name: "suspend subscription", method: http.MethodPatch, path: "/api/subscription/admin/user-subscriptions/1/suspend", body: ""},
		{name: "cancel subscription", method: http.MethodPatch, path: "/api/subscription/admin/user-subscriptions/1/cancel", body: ""},
	}

	for _, roleName := range denyRoles {
		for _, route := range writeRoutes {
			t.Run(roleName+" "+route.name, func(t *testing.T) {
				r := setupRoleAdoptionRouter(t)
				recorder := performRoleAdoptionRequest(r, route.method, route.path, route.body, roleAdoptionUsers[roleName])
				assertSubscriptionPrivilegeDenied(t, recorder)
			})
		}
	}
}

func TestSubscriptionRootOnlyPlanManagement(t *testing.T) {
	t.Run("root can create update disable and enable plan", func(t *testing.T) {
		r := setupRoleAdoptionRouter(t)
		root := roleAdoptionUsers["root"]

		createRecorder := performRoleAdoptionRequest(r, http.MethodPost, "/api/subscription/admin/plans", subscriptionPlanBody("root-plan"), root)
		requireSubscriptionSuccess(t, createRecorder)
		planID := decodeSubscriptionPlanID(t, createRecorder)

		updateRecorder := performRoleAdoptionRequest(r, http.MethodPut, "/api/subscription/admin/plans/"+strconv.Itoa(planID), subscriptionPlanBody("root-plan-updated"), root)
		requireSubscriptionSuccess(t, updateRecorder)

		disableRecorder := performRoleAdoptionRequest(r, http.MethodPatch, "/api/subscription/admin/plans/"+strconv.Itoa(planID), `{"status":"disabled"}`, root)
		requireSubscriptionSuccess(t, disableRecorder)

		enableRecorder := performRoleAdoptionRequest(r, http.MethodPatch, "/api/subscription/admin/plans/"+strconv.Itoa(planID), `{"enabled":true}`, root)
		requireSubscriptionSuccess(t, enableRecorder)
	})

	for _, roleName := range []string{"tenant_admin", "organization_admin", "finance", "auditor", "user"} {
		t.Run(roleName+" cannot mutate plan", func(t *testing.T) {
			r := setupRoleAdoptionRouter(t)
			for _, route := range []struct {
				method string
				path   string
				body   string
			}{
				{method: http.MethodPost, path: "/api/subscription/admin/plans", body: subscriptionPlanBody("non-root-create")},
				{method: http.MethodPut, path: "/api/subscription/admin/plans/1", body: subscriptionPlanBody("non-root-update")},
				{method: http.MethodPatch, path: "/api/subscription/admin/plans/1", body: `{"enabled":false}`},
			} {
				recorder := performRoleAdoptionRequest(r, route.method, route.path, route.body, roleAdoptionUsers[roleName])
				assertSubscriptionPrivilegeDenied(t, recorder)
			}
		})
	}
}

func TestSubscriptionTenantAdminTenantScope(t *testing.T) {
	t.Run("tenant_admin only lists own tenant subscriptions", func(t *testing.T) {
		r := setupRoleAdoptionRouter(t)
		recorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/subscription/admin/user-subscriptions", "", roleAdoptionUsers["tenant_admin"])
		ids := decodeSubscriptionIDs(t, recorder)
		requireRoleAdoptionIDPresence(t, ids, 1, true)
		requireRoleAdoptionIDPresence(t, ids, 3, true)
		requireRoleAdoptionIDPresence(t, ids, 4, true)
		requireRoleAdoptionIDPresence(t, ids, 2, false)
	})

	for _, route := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "assign own tenant", method: http.MethodPost, path: "/api/subscription/admin/users/6/subscriptions", body: `{"plan_id":1}`},
		{name: "renew own tenant", method: http.MethodPatch, path: "/api/subscription/admin/user-subscriptions/1/renew", body: ""},
		{name: "suspend own tenant", method: http.MethodPatch, path: "/api/subscription/admin/user-subscriptions/1/suspend", body: ""},
		{name: "cancel own tenant", method: http.MethodPatch, path: "/api/subscription/admin/user-subscriptions/1/cancel", body: ""},
	} {
		t.Run("tenant_admin can "+route.name, func(t *testing.T) {
			r := setupRoleAdoptionRouter(t)
			recorder := performRoleAdoptionRequest(r, route.method, route.path, route.body, roleAdoptionUsers["tenant_admin"])
			requireSubscriptionSuccess(t, recorder)
		})
	}

	for _, route := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "assign other tenant", method: http.MethodPost, path: "/api/subscription/admin/users/7/subscriptions", body: `{"plan_id":1}`},
		{name: "renew other tenant", method: http.MethodPatch, path: "/api/subscription/admin/user-subscriptions/2/renew", body: ""},
		{name: "suspend other tenant", method: http.MethodPatch, path: "/api/subscription/admin/user-subscriptions/2/suspend", body: ""},
		{name: "cancel other tenant", method: http.MethodPatch, path: "/api/subscription/admin/user-subscriptions/2/cancel", body: ""},
	} {
		t.Run("tenant_admin cannot "+route.name, func(t *testing.T) {
			r := setupRoleAdoptionRouter(t)
			recorder := performRoleAdoptionRequest(r, route.method, route.path, route.body, roleAdoptionUsers["tenant_admin"])
			assertSubscriptionTenantDenied(t, recorder)
		})
	}
}

func TestSubscriptionDTOBoundaryThroughRoutes(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	adminPlans := performRoleAdoptionRequest(r, http.MethodGet, "/api/subscription/admin/plans", "", roleAdoptionUsers["root"])
	requireSubscriptionSuccess(t, adminPlans)
	adminPlanBody := adminPlans.Body.String()
	for _, field := range []string{"stripe_price_id", "creem_product_id", "payment", "order_callback"} {
		if strings.Contains(adminPlanBody, field) {
			t.Fatalf("admin plan DTO leaked %q: %s", field, adminPlanBody)
		}
	}

	adminSubs := performRoleAdoptionRequest(r, http.MethodGet, "/api/subscription/admin/user-subscriptions", "", roleAdoptionUsers["root"])
	requireSubscriptionSuccess(t, adminSubs)
	adminSubBody := adminSubs.Body.String()
	for _, field := range []string{"source", "amount_total", "amount_used"} {
		if strings.Contains(adminSubBody, field) {
			t.Fatalf("admin user subscription DTO leaked %q: %s", field, adminSubBody)
		}
	}

	self := performRoleAdoptionRequest(r, http.MethodGet, "/api/subscription/self", "", roleAdoptionUsers["user"])
	requireSubscriptionSuccess(t, self)
	selfBody := self.Body.String()
	for _, field := range []string{`"id":`, `"user_id":`, `"plan_id":`, `"source":`, `"amount_total":`, `"amount_used":`} {
		if strings.Contains(selfBody, field) {
			t.Fatalf("self subscription DTO leaked GORM field %q: %s", field, selfBody)
		}
	}
}

func TestSubscriptionPublicPlansKeepPurchaseFields(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	recorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/subscription/plans", "", roleAdoptionUsers["user"])
	requireSubscriptionSuccess(t, recorder)
	body := recorder.Body.String()
	for _, field := range []string{"price_amount", "currency", "duration_unit", "duration_value", "stripe_price_id", "creem_product_id"} {
		if !strings.Contains(body, field) {
			t.Fatalf("public plans missing purchase field %q: %s", field, body)
		}
	}
}

func TestSubscriptionOwnershipVisibility(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	rootRecorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/subscription/admin/user-subscriptions", "", roleAdoptionUsers["root"])
	requireSubscriptionSuccess(t, rootRecorder)
	rootBody := rootRecorder.Body.String()
	for _, field := range []string{"tenant_id", "organization_id", "department_id", "distribution_channel_id"} {
		if !strings.Contains(rootBody, field) {
			t.Fatalf("root response missing ownership field %q: %s", field, rootBody)
		}
	}

	for _, roleName := range []string{"tenant_admin", "organization_admin", "finance", "auditor"} {
		t.Run(roleName+" ownership hidden", func(t *testing.T) {
			recorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/subscription/admin/user-subscriptions", "", roleAdoptionUsers[roleName])
			requireSubscriptionSuccess(t, recorder)
			body := recorder.Body.String()
			for _, field := range []string{"tenant_id", "organization_id", "department_id", "distribution_channel_id"} {
				if strings.Contains(body, field) {
					t.Fatalf("%s response leaked ownership field %q: %s", roleName, field, body)
				}
			}
		})
	}
}

func TestSubscriptionAdminQueryParameters(t *testing.T) {
	r := setupRoleAdoptionRouter(t)

	planRecorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/subscription/admin/plans?page=1&limit=1&q=disabled&status=disabled", "", roleAdoptionUsers["root"])
	planCodes := decodeSubscriptionPlanCodes(t, planRecorder)
	if len(planCodes) != 1 {
		t.Fatalf("expected one plan from q/status/page/limit filter, got %v", planCodes)
	}
	requireStringPresence(t, planCodes, "role-disabled", true)
	requireStringPresence(t, planCodes, "role-alpha", false)

	subRecorder := performRoleAdoptionRequest(r, http.MethodGet, "/api/subscription/admin/user-subscriptions?page=1&limit=1&status=active&user_id=6", "", roleAdoptionUsers["root"])
	subIDs := decodeSubscriptionIDs(t, subRecorder)
	if len(subIDs) != 1 {
		t.Fatalf("expected one subscription from status/user_id/page/limit filter, got %v", subIDs)
	}
	requireRoleAdoptionIDPresence(t, subIDs, 1, true)
	requireRoleAdoptionIDPresence(t, subIDs, 2, false)
}
