package model

import (
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedSubscriptionAlphaUser(t *testing.T, id int, tenantId int) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:       id,
		TenantId: tenantId,
		Username: "subscription-alpha-user",
		Password: "password123",
		Role:     common.RoleCommonUser,
		RoleKey:  common.RoleKeyUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "subscription-alpha-user",
	}).Error)
}

func TestSubscriptionPlanAlphaCodeUniqueHelper(t *testing.T) {
	truncateTables(t)

	plan := SubscriptionPlan{
		Code:          "alpha-basic",
		Name:          "Alpha Basic",
		Title:         "Alpha Basic",
		MonthlyPrice:  10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
	}
	require.NoError(t, DB.Create(&plan).Error)

	assert.Error(t, EnsureSubscriptionPlanCodeAvailable(DB, "alpha-basic", 0))
	assert.NoError(t, EnsureSubscriptionPlanCodeAvailable(DB, "alpha-basic", plan.Id))
	assert.NoError(t, EnsureSubscriptionPlanCodeAvailable(DB, "alpha-pro", 0))
}

func TestSubscriptionPlanStatusNormalization(t *testing.T) {
	plan := SubscriptionPlan{
		Code:          "alpha-status",
		Name:          "Alpha Status",
		MonthlyPrice:  10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       false,
		Status:        SubscriptionPlanStatusDisabled,
		TokenQuota:    1000,
	}
	plan.NormalizeAlphaFields()

	assert.Equal(t, SubscriptionPlanStatusDisabled, plan.Status)
	assert.False(t, plan.Enabled)
	assert.Equal(t, "Alpha Status", plan.Title)
	assert.Equal(t, int64(1000), plan.TotalAmount)
}

func TestUserSubscriptionLifecycleAndQuotaSnapshot(t *testing.T) {
	truncateTables(t)
	seedSubscriptionAlphaUser(t, 901, 1)
	plan := SubscriptionPlan{
		Code:          "alpha-snapshot",
		Name:          "Alpha Snapshot",
		Title:         "Alpha Snapshot",
		MonthlyPrice:  20,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TokenQuota:    3000,
		RequestQuota:  40,
		ModelQuota:    `{"gpt-4o":100}`,
	}
	require.NoError(t, DB.Create(&plan).Error)

	sub, err := CreateUserSubscriptionFromPlanTx(DB, 901, &plan, "admin")
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, SubscriptionLifecycleActive, sub.Status)
	assert.Equal(t, SubscriptionLifecycleActive, sub.LifecycleStatus)
	assert.Equal(t, int64(3000), sub.TokenQuotaSnapshot)
	assert.Equal(t, int64(40), sub.RequestQuotaSnapshot)
	assert.Equal(t, `{"gpt-4o":100}`, sub.ModelQuotaSnapshot)

	msg, err := AdminSuspendUserSubscription(sub.Id)
	require.NoError(t, err)
	assert.Empty(t, msg)
	var suspended UserSubscription
	require.NoError(t, DB.Where("id = ?", sub.Id).First(&suspended).Error)
	assert.Equal(t, SubscriptionLifecycleSuspended, suspended.Status)
	assert.Equal(t, SubscriptionLifecycleSuspended, suspended.LifecycleStatus)

	renewed, err := AdminRenewUserSubscription(sub.Id)
	require.NoError(t, err)
	require.NotNil(t, renewed)
	assert.Equal(t, SubscriptionLifecycleActive, renewed.LifecycleStatus)
	assert.Equal(t, int64(3000), renewed.TokenQuotaSnapshot)
}
