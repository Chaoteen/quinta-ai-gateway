package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func TestQuotaEngineCheckQuotaSubscriptionLifecycle(t *testing.T) {
	ctx := context.Background()
	engine := NewFoundationQuotaEngine()

	t.Run("active subscription allow", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{
			UserId:       5101,
			TokenQuota:   1000,
			RequestQuota: 10,
			ModelQuota:   "",
		})

		decision, err := engine.CheckQuota(ctx, quotaCheckInput(user.Id, sub.Id, "gpt-4o", 100, 1, sub))
		require.NoError(t, err)
		require.True(t, decision.Allowed)
		require.Equal(t, int64(1000), decision.TokenLimit)
		require.Equal(t, int64(1000), decision.TokenRemaining)
		require.Equal(t, int64(10), decision.RequestLimit)
		require.Equal(t, int64(10), decision.RequestRemaining)
	})

	t.Run("inactive subscription deny", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{
			UserId:       5102,
			TokenQuota:   1000,
			RequestQuota: 10,
			Status:       model.SubscriptionLifecycleSuspended,
		})

		decision, err := engine.CheckQuota(ctx, quotaCheckInput(user.Id, sub.Id, "gpt-4o", 100, 1, sub))
		require.NoError(t, err)
		require.False(t, decision.Allowed)
		require.Equal(t, "no_active_subscription", decision.Reason)
	})

	t.Run("expired subscription deny", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{
			UserId:       5103,
			TokenQuota:   1000,
			RequestQuota: 10,
			EndTime:      time.Now().Add(-time.Hour).Unix(),
		})

		decision, err := engine.CheckQuota(ctx, quotaCheckInput(user.Id, sub.Id, "gpt-4o", 100, 1, sub))
		require.NoError(t, err)
		require.False(t, decision.Allowed)
		require.Equal(t, "no_active_subscription", decision.Reason)
	})
}

func TestQuotaEngineCheckQuotaModelAllowlist(t *testing.T) {
	ctx := context.Background()
	engine := NewFoundationQuotaEngine()

	t.Run("model allowlist allow", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{
			UserId:       5201,
			TokenQuota:   1000,
			RequestQuota: 10,
			ModelQuota:   `{"allow":["gpt-4o"]}`,
		})

		decision, err := engine.CheckQuota(ctx, quotaCheckInput(user.Id, sub.Id, "gpt-4o", 100, 1, sub))
		require.NoError(t, err)
		require.True(t, decision.Allowed)
		require.True(t, decision.ModelAllowed)
	})

	t.Run("model allowlist deny", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{
			UserId:       5202,
			TokenQuota:   1000,
			RequestQuota: 10,
			ModelQuota:   `{"allow":["gpt-4o"]}`,
		})

		decision, err := engine.CheckQuota(ctx, quotaCheckInput(user.Id, sub.Id, "claude-3-5", 100, 1, sub))
		require.NoError(t, err)
		require.False(t, decision.Allowed)
		require.Equal(t, "model_not_allowed", decision.Reason)
	})

	t.Run("unrestricted model allow", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{
			UserId:       5203,
			TokenQuota:   1000,
			RequestQuota: 10,
			ModelQuota:   "",
		})

		decision, err := engine.CheckQuota(ctx, quotaCheckInput(user.Id, sub.Id, "any-model", 100, 1, sub))
		require.NoError(t, err)
		require.True(t, decision.Allowed)
		require.True(t, decision.ModelAllowed)
	})
}

func TestQuotaEngineCheckQuotaTokenAndRequestLimits(t *testing.T) {
	ctx := context.Background()
	engine := NewFoundationQuotaEngine()

	t.Run("token quota enough", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{UserId: 5301, TokenQuota: 100, RequestQuota: 10})
		seedCommittedQuotaUsage(t, sub, 80, 0)

		decision, err := engine.CheckQuota(ctx, quotaCheckInput(user.Id, sub.Id, "gpt-4o", 20, 1, sub))
		require.NoError(t, err)
		require.True(t, decision.Allowed)
		require.Equal(t, int64(80), decision.TokenUsed)
	})

	t.Run("token quota insufficient", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{UserId: 5302, TokenQuota: 100, RequestQuota: 10})
		seedCommittedQuotaUsage(t, sub, 90, 0)

		decision, err := engine.CheckQuota(ctx, quotaCheckInput(user.Id, sub.Id, "gpt-4o", 20, 1, sub))
		require.NoError(t, err)
		require.False(t, decision.Allowed)
		require.Equal(t, "token_quota_insufficient", decision.Reason)
	})

	t.Run("request quota enough", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{UserId: 5303, TokenQuota: 1000, RequestQuota: 3})
		seedCommittedQuotaUsage(t, sub, 0, 2)

		decision, err := engine.CheckQuota(ctx, quotaCheckInput(user.Id, sub.Id, "gpt-4o", 10, 1, sub))
		require.NoError(t, err)
		require.True(t, decision.Allowed)
		require.Equal(t, int64(2), decision.RequestUsed)
	})

	t.Run("request quota insufficient", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{UserId: 5304, TokenQuota: 1000, RequestQuota: 3})
		seedCommittedQuotaUsage(t, sub, 0, 3)

		decision, err := engine.CheckQuota(ctx, quotaCheckInput(user.Id, sub.Id, "gpt-4o", 10, 1, sub))
		require.NoError(t, err)
		require.False(t, decision.Allowed)
		require.Equal(t, "request_quota_insufficient", decision.Reason)
	})
}

func TestQuotaEngineReserveQuotaIdempotencyByRequestId(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	engine := NewFoundationQuotaEngine()
	user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{UserId: 5401, TokenQuota: 1000, RequestQuota: 10})

	input := QuotaReservationInput{
		QuotaCheckInput: quotaCheckInput(user.Id, sub.Id, "gpt-4o", 100, 1, sub),
		ReservationId:   "reservation-a",
		ExpiresAt:       time.Now().Add(time.Hour).Unix(),
	}
	input.RequestId = "request-idempotent"

	first, err := engine.ReserveQuota(ctx, input)
	require.NoError(t, err)

	input.ReservationId = "reservation-b"
	second, err := engine.ReserveQuota(ctx, input)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)
	require.Equal(t, "reservation-a", second.ReservationId)

	var reservations int64
	require.NoError(t, model.DB.Model(&model.QuotaReservation{}).Where("request_id = ?", input.RequestId).Count(&reservations).Error)
	require.Equal(t, int64(1), reservations)

	var reservedRecords int64
	require.NoError(t, model.DB.Model(&model.QuotaUsageRecord{}).
		Where("request_id = ? AND status = ?", input.RequestId, model.QuotaUsageStatusReserved).
		Count(&reservedRecords).Error)
	require.Equal(t, int64(1), reservedRecords)
}

func TestQuotaEngineCommitAndRollbackStateMachine(t *testing.T) {
	ctx := context.Background()
	engine := NewFoundationQuotaEngine()

	t.Run("commit idempotency by reservation_id", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{UserId: 5501, TokenQuota: 1000, RequestQuota: 10})
		reservation := reserveQuotaForTest(t, engine, ctx, user.Id, sub, "commit-reservation", "commit-request")

		first, err := engine.CommitUsage(ctx, QuotaCommitInput{ReservationId: reservation.ReservationId, TokenAmount: 120, RequestAmount: 1})
		require.NoError(t, err)
		second, err := engine.CommitUsage(ctx, QuotaCommitInput{ReservationId: reservation.ReservationId, TokenAmount: 120, RequestAmount: 1})
		require.NoError(t, err)
		require.Equal(t, first.Id, second.Id)

		var count int64
		require.NoError(t, model.DB.Model(&model.QuotaUsageRecord{}).
			Where("reservation_id = ? AND status = ?", reservation.ReservationId, model.QuotaUsageStatusCommitted).
			Count(&count).Error)
		require.Equal(t, int64(1), count)
	})

	t.Run("rollback idempotency by reservation_id", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{UserId: 5502, TokenQuota: 1000, RequestQuota: 10})
		reservation := reserveQuotaForTest(t, engine, ctx, user.Id, sub, "rollback-reservation", "rollback-request")

		require.NoError(t, engine.RollbackReservation(ctx, QuotaRollbackInput{ReservationId: reservation.ReservationId}))
		require.NoError(t, engine.RollbackReservation(ctx, QuotaRollbackInput{ReservationId: reservation.ReservationId}))

		var count int64
		require.NoError(t, model.DB.Model(&model.QuotaUsageRecord{}).
			Where("reservation_id = ? AND status = ?", reservation.ReservationId, model.QuotaUsageStatusRolledBack).
			Count(&count).Error)
		require.Equal(t, int64(1), count)
	})

	t.Run("committed cannot rollback", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{UserId: 5503, TokenQuota: 1000, RequestQuota: 10})
		reservation := reserveQuotaForTest(t, engine, ctx, user.Id, sub, "committed-no-rollback", "committed-no-rollback-request")
		_, err := engine.CommitUsage(ctx, QuotaCommitInput{ReservationId: reservation.ReservationId})
		require.NoError(t, err)

		err = engine.RollbackReservation(ctx, QuotaRollbackInput{ReservationId: reservation.ReservationId})
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrQuotaInvalidState))
	})

	t.Run("rolled_back cannot commit", func(t *testing.T) {
		truncate(t)
		user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{UserId: 5504, TokenQuota: 1000, RequestQuota: 10})
		reservation := reserveQuotaForTest(t, engine, ctx, user.Id, sub, "rolled-no-commit", "rolled-no-commit-request")
		require.NoError(t, engine.RollbackReservation(ctx, QuotaRollbackInput{ReservationId: reservation.ReservationId}))

		_, err := engine.CommitUsage(ctx, QuotaCommitInput{ReservationId: reservation.ReservationId})
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrQuotaInvalidState))
	})
}

func TestQuotaEngineResetWritesResetUsageRecord(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	engine := NewFoundationQuotaEngine()
	_, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{
		UserId:       5601,
		TokenQuota:   1000,
		RequestQuota: 10,
		AmountUsed:   77,
		NextReset:    time.Now().Add(-time.Hour).Unix(),
	})
	plan := model.SubscriptionPlan{
		Id:                      5601,
		Code:                    "reset-plan",
		Name:                    "Reset Plan",
		Title:                   "Reset Plan",
		Status:                  model.SubscriptionPlanStatusEnabled,
		Enabled:                 true,
		QuotaResetPeriod:        model.SubscriptionResetCustom,
		QuotaResetCustomSeconds: 3600,
	}
	require.NoError(t, model.DB.Create(&plan).Error)
	require.NoError(t, model.DB.Model(&model.UserSubscription{}).Where("id = ?", sub.Id).Update("plan_id", plan.Id).Error)

	resetAt := time.Now().Unix()
	require.NoError(t, engine.ResetQuota(ctx, QuotaResetInput{UserSubscriptionId: sub.Id, ResetAt: resetAt, Reason: "test"}))

	var record model.QuotaUsageRecord
	require.NoError(t, model.DB.Where("user_subscription_id = ? AND status = ?", sub.Id, model.QuotaUsageStatusReset).First(&record).Error)
	require.Equal(t, model.QuotaDimensionReset, record.QuotaDimension)
	require.Equal(t, resetAt, record.OccurredAt)

	var reloaded model.UserSubscription
	require.NoError(t, model.DB.Where("id = ?", sub.Id).First(&reloaded).Error)
	require.Equal(t, int64(77), reloaded.AmountUsed)
	require.Equal(t, resetAt, reloaded.LastResetTime)
	require.Greater(t, reloaded.NextResetTime, resetAt)
}

func TestQuotaEngineTenantIsolation(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	engine := NewFoundationQuotaEngine()
	user, _, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{
		UserId:       5701,
		TenantId:     2,
		TokenQuota:   1000,
		RequestQuota: 10,
	})

	input := quotaCheckInput(user.Id, sub.Id, "gpt-4o", 100, 1, sub)
	input.Ownership.TenantId = 1
	decision, err := engine.CheckQuota(ctx, input)
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	require.Equal(t, "no_active_subscription", decision.Reason)
}

func TestQuotaEngineDoesNotMutateWalletTokenOrLegacySubscriptionQuota(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	engine := NewFoundationQuotaEngine()
	user, token, sub := seedQuotaRuntimeFixture(t, quotaRuntimeFixture{
		UserId:       5801,
		TokenQuota:   1000,
		RequestQuota: 10,
		UserQuota:    9000,
		TokenRemain:  8000,
		AmountUsed:   66,
	})

	reservation := reserveQuotaForTest(t, engine, ctx, user.Id, sub, "safety-reservation", "safety-request")
	_, err := engine.CommitUsage(ctx, QuotaCommitInput{ReservationId: reservation.ReservationId})
	require.NoError(t, err)
	require.NoError(t, engine.ResetQuota(ctx, QuotaResetInput{UserSubscriptionId: sub.Id, ResetAt: time.Now().Unix()}))

	var reloadedUser model.User
	require.NoError(t, model.DB.Select("quota").Where("id = ?", user.Id).First(&reloadedUser).Error)
	require.Equal(t, user.Quota, reloadedUser.Quota)

	var reloadedToken model.Token
	require.NoError(t, model.DB.Select("remain_quota").Where("id = ?", token.Id).First(&reloadedToken).Error)
	require.Equal(t, token.RemainQuota, reloadedToken.RemainQuota)

	var reloadedSub model.UserSubscription
	require.NoError(t, model.DB.Select("amount_used").Where("id = ?", sub.Id).First(&reloadedSub).Error)
	require.Equal(t, sub.AmountUsed, reloadedSub.AmountUsed)
}

type quotaRuntimeFixture struct {
	UserId       int
	TenantId     int
	TokenQuota   int64
	RequestQuota int64
	ModelQuota   string
	Status       string
	EndTime      int64
	NextReset    int64
	UserQuota    int
	TokenRemain  int
	AmountUsed   int64
}

func seedQuotaRuntimeFixture(t *testing.T, fixture quotaRuntimeFixture) (model.User, model.Token, model.UserSubscription) {
	t.Helper()
	if fixture.TenantId == 0 {
		fixture.TenantId = 1
	}
	if fixture.Status == "" {
		fixture.Status = model.SubscriptionLifecycleActive
	}
	if fixture.EndTime == 0 {
		fixture.EndTime = time.Now().Add(24 * time.Hour).Unix()
	}
	if fixture.UserQuota == 0 {
		fixture.UserQuota = 5000
	}
	if fixture.TokenRemain == 0 {
		fixture.TokenRemain = 4000
	}

	user := model.User{
		Id:       fixture.UserId,
		TenantId: fixture.TenantId,
		Username: "quota-user-" + time.Now().Format("150405.000000"),
		Quota:    fixture.UserQuota,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(&user).Error)

	token := model.Token{
		Id:          fixture.UserId + 10000,
		TenantId:    fixture.TenantId,
		UserId:      user.Id,
		Key:         "quota-token-" + time.Now().Format("150405.000000"),
		Name:        "quota token",
		Status:      common.TokenStatusEnabled,
		RemainQuota: fixture.TokenRemain,
	}
	require.NoError(t, model.DB.Create(&token).Error)

	sub := model.UserSubscription{
		Id:                   fixture.UserId + 20000,
		TenantId:             fixture.TenantId,
		UserId:               user.Id,
		PlanId:               fixture.UserId + 30000,
		AmountTotal:          100000,
		AmountUsed:           fixture.AmountUsed,
		StartTime:            time.Now().Add(-time.Hour).Unix(),
		EndTime:              fixture.EndTime,
		Status:               fixture.Status,
		LifecycleStatus:      fixture.Status,
		TokenQuotaSnapshot:   fixture.TokenQuota,
		RequestQuotaSnapshot: fixture.RequestQuota,
		ModelQuotaSnapshot:   fixture.ModelQuota,
		NextResetTime:        fixture.NextReset,
	}
	require.NoError(t, model.DB.Create(&sub).Error)
	return user, token, sub
}

func quotaCheckInput(userId int, subId int, modelName string, tokenAmount int64, requestAmount int64, sub model.UserSubscription) QuotaCheckInput {
	return QuotaCheckInput{
		UserId:             userId,
		UserSubscriptionId: subId,
		RequestId:          "quota-check",
		ModelName:          modelName,
		TokenAmount:        tokenAmount,
		RequestAmount:      requestAmount,
		Ownership: model.OwnershipSnapshot{
			TenantId:              sub.TenantId,
			OrganizationId:        sub.OrganizationId,
			DepartmentId:          sub.DepartmentId,
			DistributionChannelId: sub.DistributionChannelId,
		},
	}
}

func seedCommittedQuotaUsage(t *testing.T, sub model.UserSubscription, tokenDelta int64, requestDelta int64) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.QuotaUsageRecord{
		TenantId:              sub.TenantId,
		OrganizationId:        sub.OrganizationId,
		DepartmentId:          sub.DepartmentId,
		DistributionChannelId: sub.DistributionChannelId,
		UserId:                sub.UserId,
		UserSubscriptionId:    sub.Id,
		RequestId:             "committed-usage-" + time.Now().Format("150405.000000"),
		ReservationId:         "committed-reservation-" + time.Now().Format("150405.000000"),
		ModelName:             "gpt-4o",
		QuotaDimension:        model.QuotaDimensionToken,
		TokenDelta:            tokenDelta,
		RequestDelta:          requestDelta,
		Status:                model.QuotaUsageStatusCommitted,
	}).Error)
}

func reserveQuotaForTest(t *testing.T, engine QuotaEngine, ctx context.Context, userId int, sub model.UserSubscription, reservationId string, requestId string) model.QuotaReservation {
	t.Helper()
	input := quotaCheckInput(userId, sub.Id, "gpt-4o", 100, 1, sub)
	if requestId != "" {
		input.RequestId = requestId
	}
	reservation, err := engine.ReserveQuota(ctx, QuotaReservationInput{
		QuotaCheckInput: input,
		ReservationId:   reservationId,
		ExpiresAt:       time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, err)
	return reservation
}
