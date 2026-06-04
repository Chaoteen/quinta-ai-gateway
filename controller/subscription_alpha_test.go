package controller

import (
	"strings"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionPlanDTODoesNotExposePaymentFields(t *testing.T) {
	dto := SubscriptionPlanDTO{Plan: subscriptionPlanToDTO(model.SubscriptionPlan{
		Id:             1,
		Code:           "alpha-dto",
		Name:           "Alpha DTO",
		Title:          "Alpha DTO",
		MonthlyPrice:   10,
		StripePriceId:  "price_should_not_leak",
		CreemProductId: "product_should_not_leak",
		Enabled:        true,
	})}

	body, err := common.Marshal(dto)
	require.NoError(t, err)
	require.NotContains(t, string(body), `"stripe_price_id":`)
	require.NotContains(t, string(body), `"creem_product_id":`)
	require.NotContains(t, string(body), "price_should_not_leak")
	require.NotContains(t, string(body), `"title":`)
	require.NotContains(t, string(body), `"subtitle":`)
	require.NotContains(t, string(body), `"price_amount":`)
	require.NotContains(t, string(body), `"currency":`)
	require.NotContains(t, string(body), `"duration_unit":`)
	require.NotContains(t, string(body), `"duration_value":`)
	require.NotContains(t, string(body), `"custom_seconds":`)
	require.NotContains(t, string(body), `"enabled":`)
	require.NotContains(t, string(body), `"sort_order":`)
	require.NotContains(t, string(body), `"max_purchase_per_user":`)
	require.NotContains(t, string(body), `"upgrade_group":`)
	require.NotContains(t, string(body), `"total_amount":`)
	require.NotContains(t, string(body), `"quota_reset_period":`)
	require.NotContains(t, string(body), `"quota_reset_custom_seconds":`)
	require.Contains(t, string(body), "alpha-dto")
}

func TestPublicSubscriptionPlanDTOKeepsPurchaseFields(t *testing.T) {
	dto := publicSubscriptionPlansToDTO([]model.SubscriptionPlan{{
		Id:             1,
		Code:           "public-alpha",
		Name:           "Public Alpha",
		Title:          "Public Alpha",
		MonthlyPrice:   10,
		PriceAmount:    10,
		Currency:       "USD",
		DurationUnit:   model.SubscriptionDurationMonth,
		DurationValue:  1,
		Enabled:        true,
		StripePriceId:  "price_public",
		CreemProductId: "product_public",
	}})

	body, err := common.Marshal(dto)
	require.NoError(t, err)
	require.Contains(t, string(body), "price_amount")
	require.Contains(t, string(body), "currency")
	require.Contains(t, string(body), "duration_unit")
	require.Contains(t, string(body), "duration_value")
	require.Contains(t, string(body), "enabled")
	require.Contains(t, string(body), "stripe_price_id")
	require.Contains(t, string(body), "creem_product_id")
}

func TestUserSubscriptionDTODoesNotExposeLegacyBillingFields(t *testing.T) {
	dto := buildUserSubscriptionDTOs([]model.UserSubscription{{
		Id:                   1,
		TenantId:             2,
		UserId:               3,
		PlanId:               4,
		Status:               model.SubscriptionLifecycleActive,
		LifecycleStatus:      model.SubscriptionLifecycleActive,
		Source:               "admin",
		AmountTotal:          1000,
		AmountUsed:           10,
		TokenQuotaSnapshot:   1000,
		RequestQuotaSnapshot: 50,
		UpgradeGroup:         "vip",
		PrevUserGroup:        "default",
	}}, false)

	body, err := common.Marshal(dto)
	require.NoError(t, err)
	require.NotContains(t, string(body), `"source":`)
	require.NotContains(t, string(body), `"amount_total":`)
	require.NotContains(t, string(body), `"amount_used":`)
	require.NotContains(t, string(body), `"upgrade_group":`)
	require.NotContains(t, string(body), `"prev_user_group":`)
	require.NotContains(t, string(body), `"tenant_id":`)
	require.Contains(t, string(body), "lifecycle_status")
}

func TestRootUserSubscriptionDTOCanExposeOwnershipFields(t *testing.T) {
	dto := buildUserSubscriptionDTOs([]model.UserSubscription{{
		Id:                    1,
		TenantId:              2,
		OrganizationId:        3,
		DepartmentId:          4,
		DistributionChannelId: 5,
		UserId:                6,
		PlanId:                7,
		LifecycleStatus:       model.SubscriptionLifecycleActive,
	}}, true)

	body, err := common.Marshal(dto)
	require.NoError(t, err)
	require.Contains(t, string(body), "tenant_id")
	require.Contains(t, string(body), "organization_id")
	require.Contains(t, string(body), "department_id")
	require.Contains(t, string(body), "distribution_channel_id")
}

func TestSelfSubscriptionDTODoesNotExposeFullUserSubscriptionModel(t *testing.T) {
	dto := buildSelfSubscriptionDTOs([]model.SubscriptionSummary{{
		Subscription: &model.UserSubscription{
			Id:                   1,
			UserId:               2,
			PlanId:               3,
			Status:               model.SubscriptionLifecycleActive,
			LifecycleStatus:      model.SubscriptionLifecycleActive,
			Source:               "order",
			AmountTotal:          1000,
			AmountUsed:           125,
			TokenQuotaSnapshot:   1000,
			RequestQuotaSnapshot: 50,
			UpgradeGroup:         "vip",
			PrevUserGroup:        "default",
		},
	}})

	body, err := common.Marshal(dto)
	require.NoError(t, err)
	require.NotContains(t, string(body), `"id":`)
	require.NotContains(t, string(body), `"user_id":`)
	require.NotContains(t, string(body), `"source":`)
	require.NotContains(t, string(body), `"amount_total":`)
	require.NotContains(t, string(body), `"amount_used":`)
	require.NotContains(t, string(body), `"upgrade_group":`)
	require.NotContains(t, string(body), `"prev_user_group":`)
	require.Contains(t, string(body), "token_quota")
	require.Contains(t, string(body), "token_used")
	require.Contains(t, string(body), "token_remaining")
}

func TestApplyPlanDTOToModelAlphaFieldMapping(t *testing.T) {
	var plan model.SubscriptionPlan
	applyPlanDTOToModel(AlphaSubscriptionPlanDTO{
		Code:         "alpha-map",
		Name:         "Alpha Map",
		Description:  "Mapped description",
		MonthlyPrice: 12,
		YearlyPrice:  120,
		TokenQuota:   3000,
		RequestQuota: 50,
		ModelQuota:   `{"gpt-4o":100}`,
		Status:       model.SubscriptionPlanStatusEnabled,
	}, &plan)

	require.Equal(t, "alpha-map", plan.Code)
	require.Equal(t, "Alpha Map", plan.Name)
	require.Equal(t, "Alpha Map", plan.Title)
	require.Equal(t, "Mapped description", plan.Description)
	require.Equal(t, float64(12), plan.MonthlyPrice)
	require.Equal(t, int64(3000), plan.TokenQuota)
	require.Equal(t, int64(3000), plan.TotalAmount)
	require.Equal(t, int64(50), plan.RequestQuota)
	require.Equal(t, model.SubscriptionPlanStatusEnabled, plan.Status)
	require.True(t, plan.Enabled)
	require.False(t, strings.Contains(plan.ModelQuota, "stripe"))
}
