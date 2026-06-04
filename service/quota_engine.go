package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"gorm.io/gorm"
)

var (
	ErrQuotaEngineNotImplemented = errors.New("quota engine foundation is not implemented")
	ErrQuotaDenied               = errors.New("quota denied")
	ErrQuotaReservationNotFound  = errors.New("quota reservation not found")
	ErrQuotaInvalidState         = errors.New("quota reservation state transition is not allowed")
)

type QuotaEngine interface {
	CheckQuota(ctx context.Context, input QuotaCheckInput) (QuotaDecision, error)
	ReserveQuota(ctx context.Context, input QuotaReservationInput) (model.QuotaReservation, error)
	CommitUsage(ctx context.Context, input QuotaCommitInput) (model.QuotaUsageRecord, error)
	RollbackReservation(ctx context.Context, input QuotaRollbackInput) error
	ResetQuota(ctx context.Context, input QuotaResetInput) error
}

type QuotaCheckInput struct {
	UserId             int
	UserSubscriptionId int
	RequestId          string
	ModelName          string
	TokenAmount        int64
	RequestAmount      int64
	Ownership          model.OwnershipSnapshot
}

type QuotaReservationInput struct {
	QuotaCheckInput
	ReservationId string
	ExpiresAt     int64
	Metadata      string
}

type QuotaCommitInput struct {
	ReservationId string
	RequestId     string
	ModelName     string
	TokenAmount   int64
	RequestAmount int64
	Metadata      string
}

type QuotaRollbackInput struct {
	ReservationId string
	RequestId     string
	Reason        string
	Metadata      string
}

type QuotaResetInput struct {
	UserSubscriptionId int
	ResetAt            int64
	Reason             string
	Metadata           string
}

type QuotaDecision struct {
	Allowed            bool
	Reason             string
	Message            string
	UserId             int
	UserSubscriptionId int
	PlanId             int
	ModelName          string
	TokenLimit         int64
	TokenUsed          int64
	TokenRemaining     int64
	RequestLimit       int64
	RequestUsed        int64
	RequestRemaining   int64
	ModelAllowed       bool
	ResetAt            int64
	Ownership          model.OwnershipSnapshot
}

type FoundationQuotaEngine struct{}

func NewFoundationQuotaEngine() QuotaEngine {
	return &FoundationQuotaEngine{}
}

func (e *FoundationQuotaEngine) CheckQuota(ctx context.Context, input QuotaCheckInput) (QuotaDecision, error) {
	sub, decision, err := findQuotaSubscription(ctx, input)
	if err != nil {
		return decision, err
	}
	if sub == nil {
		return decision, nil
	}

	if err := applyModelQuota(&decision, sub.ModelQuotaSnapshot, input.ModelName); err != nil {
		return decision, nil
	}
	if !decision.ModelAllowed {
		return decision, nil
	}

	now := common.GetTimestamp()
	tokenUsed, requestUsed, err := quotaRuntimeUsage(ctx, sub.Id, sub.LastResetTime, now)
	if err != nil {
		return decision, err
	}
	decision.TokenUsed = tokenUsed
	decision.RequestUsed = requestUsed
	decision.TokenRemaining = quotaRemaining(decision.TokenLimit, tokenUsed)
	decision.RequestRemaining = quotaRemaining(decision.RequestLimit, requestUsed)

	if quotaInsufficient(decision.TokenLimit, tokenUsed, input.TokenAmount) {
		decision.Reason = "token_quota_insufficient"
		decision.Message = "token quota is insufficient"
		return decision, nil
	}
	if quotaInsufficient(decision.RequestLimit, requestUsed, input.RequestAmount) {
		decision.Reason = "request_quota_insufficient"
		decision.Message = "request quota is insufficient"
		return decision, nil
	}

	decision.Allowed = true
	decision.Reason = "allowed"
	decision.Message = "quota allowed"
	return decision, nil
}

func (e *FoundationQuotaEngine) ReserveQuota(ctx context.Context, input QuotaReservationInput) (model.QuotaReservation, error) {
	input.RequestId = strings.TrimSpace(input.RequestId)
	input.ReservationId = strings.TrimSpace(input.ReservationId)
	if input.RequestId == "" {
		return model.QuotaReservation{}, errors.New("request_id is required")
	}
	if input.ReservationId == "" {
		input.ReservationId = input.RequestId
	}

	decision, err := e.CheckQuota(ctx, input.QuotaCheckInput)
	if err != nil {
		return model.QuotaReservation{}, err
	}
	if !decision.Allowed {
		if decision.Reason == "" {
			decision.Reason = "quota_denied"
		}
		return model.QuotaReservation{}, fmt.Errorf("%w: %s", ErrQuotaDenied, decision.Reason)
	}

	var reservation model.QuotaReservation
	err = model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := findExistingReservationTx(tx, input.ReservationId, input.RequestId)
		if err != nil {
			return err
		}
		if existing != nil {
			reservation = *existing
			return nil
		}

		reservation = model.QuotaReservation{
			TenantId:              decision.Ownership.TenantId,
			OrganizationId:        decision.Ownership.OrganizationId,
			DepartmentId:          decision.Ownership.DepartmentId,
			DistributionChannelId: decision.Ownership.DistributionChannelId,
			ReservationId:         input.ReservationId,
			RequestId:             input.RequestId,
			UserId:                decision.UserId,
			UserSubscriptionId:    decision.UserSubscriptionId,
			ModelName:             strings.TrimSpace(input.ModelName),
			TokenReserved:         input.TokenAmount,
			RequestReserved:       input.RequestAmount,
			Status:                model.QuotaReservationStatusActive,
			ExpiresAt:             input.ExpiresAt,
			Metadata:              input.Metadata,
		}
		if err := tx.Create(&reservation).Error; err != nil {
			if existing, findErr := findExistingReservationTx(tx, input.ReservationId, input.RequestId); findErr == nil && existing != nil {
				reservation = *existing
				return nil
			}
			return err
		}
		record := quotaUsageRecordFromReservation(reservation, model.QuotaUsageStatusReserved, input.Metadata)
		return createQuotaUsageRecordTx(tx, &record)
	})
	if err != nil {
		return model.QuotaReservation{}, err
	}
	return reservation, nil
}

func (e *FoundationQuotaEngine) CommitUsage(ctx context.Context, input QuotaCommitInput) (model.QuotaUsageRecord, error) {
	input.ReservationId = strings.TrimSpace(input.ReservationId)
	if input.ReservationId == "" {
		return model.QuotaUsageRecord{}, errors.New("reservation_id is required")
	}

	var usage model.QuotaUsageRecord
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reservation model.QuotaReservation
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("reservation_id = ?", input.ReservationId).
			First(&reservation).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrQuotaReservationNotFound
			}
			return err
		}

		if reservation.Status == model.QuotaReservationStatusCommitted {
			return firstOrCreateCommittedUsageTx(tx, reservation, input, &usage)
		}
		if reservation.Status != model.QuotaReservationStatusActive {
			return fmt.Errorf("%w: %s cannot commit", ErrQuotaInvalidState, reservation.Status)
		}

		now := common.GetTimestamp()
		reservation.Status = model.QuotaReservationStatusCommitted
		reservation.CommittedAt = now
		if err := tx.Save(&reservation).Error; err != nil {
			return err
		}

		usage = quotaUsageRecordFromReservation(reservation, model.QuotaUsageStatusCommitted, input.Metadata)
		usage.RequestId = firstNonEmpty(strings.TrimSpace(input.RequestId), reservation.RequestId)
		usage.ModelName = firstNonEmpty(strings.TrimSpace(input.ModelName), reservation.ModelName)
		usage.TokenDelta = firstNonZero(input.TokenAmount, reservation.TokenReserved)
		usage.RequestDelta = firstNonZero(input.RequestAmount, reservation.RequestReserved)
		return createQuotaUsageRecordTx(tx, &usage)
	})
	if err != nil {
		return model.QuotaUsageRecord{}, err
	}
	return usage, nil
}

func (e *FoundationQuotaEngine) RollbackReservation(ctx context.Context, input QuotaRollbackInput) error {
	input.ReservationId = strings.TrimSpace(input.ReservationId)
	if input.ReservationId == "" {
		return errors.New("reservation_id is required")
	}

	return model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reservation model.QuotaReservation
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("reservation_id = ?", input.ReservationId).
			First(&reservation).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrQuotaReservationNotFound
			}
			return err
		}

		if reservation.Status == model.QuotaReservationStatusRolledBack {
			return ensureRollbackUsageTx(tx, reservation, input)
		}
		if reservation.Status != model.QuotaReservationStatusActive {
			return fmt.Errorf("%w: %s cannot rollback", ErrQuotaInvalidState, reservation.Status)
		}

		reservation.Status = model.QuotaReservationStatusRolledBack
		reservation.RolledBackAt = common.GetTimestamp()
		if err := tx.Save(&reservation).Error; err != nil {
			return err
		}
		return ensureRollbackUsageTx(tx, reservation, input)
	})
}

func (e *FoundationQuotaEngine) ResetQuota(ctx context.Context, input QuotaResetInput) error {
	if input.UserSubscriptionId <= 0 {
		return errors.New("user_subscription_id is required")
	}
	resetAt := input.ResetAt
	if resetAt <= 0 {
		resetAt = common.GetTimestamp()
	}

	return model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sub model.UserSubscription
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("id = ?", input.UserSubscriptionId).
			First(&sub).Error; err != nil {
			return err
		}

		nextReset := sub.NextResetTime
		if sub.PlanId > 0 {
			var plan model.SubscriptionPlan
			if err := tx.Where("id = ?", sub.PlanId).First(&plan).Error; err == nil {
				nextReset = advanceQuotaResetWindow(time.Unix(resetAt, 0), &plan, sub.EndTime)
			} else if errors.Is(err, gorm.ErrRecordNotFound) && nextReset <= resetAt {
				nextReset = 0
			} else if err != nil {
				return err
			}
		} else if nextReset <= resetAt {
			nextReset = 0
		}

		if err := tx.Model(&model.UserSubscription{}).
			Where("id = ?", sub.Id).
			Updates(map[string]any{
				"last_reset_time": resetAt,
				"next_reset_time": nextReset,
			}).Error; err != nil {
			return err
		}

		record := model.QuotaUsageRecord{
			TenantId:              sub.TenantId,
			OrganizationId:        sub.OrganizationId,
			DepartmentId:          sub.DepartmentId,
			DistributionChannelId: sub.DistributionChannelId,
			UserId:                sub.UserId,
			UserSubscriptionId:    sub.Id,
			RequestId:             fmt.Sprintf("quota-reset-%d-%d", sub.Id, resetAt),
			QuotaDimension:        model.QuotaDimensionReset,
			Status:                model.QuotaUsageStatusReset,
			Metadata:              firstNonEmpty(input.Metadata, input.Reason),
			OccurredAt:            resetAt,
		}
		return createQuotaUsageRecordTx(tx, &record)
	})
}

func findQuotaSubscription(ctx context.Context, input QuotaCheckInput) (*model.UserSubscription, QuotaDecision, error) {
	now := common.GetTimestamp()
	ownership := model.NormalizeOwnership(input.Ownership)
	decision := QuotaDecision{
		Reason:       "no_active_subscription",
		Message:      "no active subscription",
		UserId:       input.UserId,
		ModelName:    strings.TrimSpace(input.ModelName),
		ModelAllowed: false,
		Ownership:    ownership,
	}
	if input.UserId <= 0 {
		decision.Reason = "invalid_user"
		decision.Message = "user_id is required"
		return nil, decision, nil
	}

	query := model.DB.WithContext(ctx).Where(
		"user_id = ? AND status = ? AND lifecycle_status = ? AND start_time <= ? AND end_time > ?",
		input.UserId,
		model.SubscriptionLifecycleActive,
		model.SubscriptionLifecycleActive,
		now,
		now,
	)
	query = applyOwnershipFilter(query, ownership)
	if input.UserSubscriptionId > 0 {
		query = query.Where("id = ?", input.UserSubscriptionId)
	}

	var sub model.UserSubscription
	if err := query.Order("end_time asc, id asc").First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, decision, nil
		}
		return nil, decision, err
	}

	decision.UserSubscriptionId = sub.Id
	decision.PlanId = sub.PlanId
	decision.TokenLimit = sub.TokenQuotaSnapshot
	decision.RequestLimit = sub.RequestQuotaSnapshot
	decision.ResetAt = sub.NextResetTime
	decision.Ownership = model.OwnershipSnapshot{
		TenantId:              sub.TenantId,
		OrganizationId:        sub.OrganizationId,
		DepartmentId:          sub.DepartmentId,
		DistributionChannelId: sub.DistributionChannelId,
	}
	return &sub, decision, nil
}

func applyOwnershipFilter(query *gorm.DB, ownership model.OwnershipSnapshot) *gorm.DB {
	return query.Where(
		"tenant_id = ? AND organization_id = ? AND department_id = ? AND distribution_channel_id = ?",
		ownership.TenantId,
		ownership.OrganizationId,
		ownership.DepartmentId,
		ownership.DistributionChannelId,
	)
}

func applyModelQuota(decision *QuotaDecision, snapshot string, modelName string) error {
	modelName = strings.TrimSpace(modelName)
	parsed, err := model.ParseModelQuotaSnapshot(snapshot)
	if err != nil {
		decision.Reason = "model_quota_invalid"
		decision.Message = "model quota snapshot is invalid"
		return err
	}
	if parsed.Unrestricted || modelName == "" {
		decision.ModelAllowed = true
		return nil
	}
	for _, allowed := range parsed.Allow {
		if allowed == modelName {
			decision.ModelAllowed = true
			return nil
		}
	}
	decision.Reason = "model_not_allowed"
	decision.Message = "model is not allowed by subscription"
	return nil
}

func quotaRuntimeUsage(ctx context.Context, userSubscriptionId int, resetAfter int64, now int64) (int64, int64, error) {
	var committed struct {
		TokenUsed   int64
		RequestUsed int64
	}
	committedQuery := model.DB.WithContext(ctx).Model(&model.QuotaUsageRecord{}).
		Select("COALESCE(SUM(token_delta), 0) AS token_used, COALESCE(SUM(request_delta), 0) AS request_used").
		Where("user_subscription_id = ? AND status = ?", userSubscriptionId, model.QuotaUsageStatusCommitted)
	if resetAfter > 0 {
		committedQuery = committedQuery.Where("occurred_at > ?", resetAfter)
	}
	if err := committedQuery.Scan(&committed).Error; err != nil {
		return 0, 0, err
	}

	var reserved struct {
		TokenUsed   int64
		RequestUsed int64
	}
	reservedQuery := model.DB.WithContext(ctx).Model(&model.QuotaReservation{}).
		Select("COALESCE(SUM(token_reserved), 0) AS token_used, COALESCE(SUM(request_reserved), 0) AS request_used").
		Where("user_subscription_id = ? AND status = ? AND (expires_at = 0 OR expires_at > ?)", userSubscriptionId, model.QuotaReservationStatusActive, now)
	if resetAfter > 0 {
		reservedQuery = reservedQuery.Where("created_at > ?", resetAfter)
	}
	if err := reservedQuery.Scan(&reserved).Error; err != nil {
		return 0, 0, err
	}
	return committed.TokenUsed + reserved.TokenUsed, committed.RequestUsed + reserved.RequestUsed, nil
}

func quotaRemaining(limit int64, used int64) int64 {
	if limit <= 0 {
		return 0
	}
	remaining := limit - used
	if remaining < 0 {
		return 0
	}
	return remaining
}

func quotaInsufficient(limit int64, used int64, requested int64) bool {
	if limit <= 0 || requested <= 0 {
		return false
	}
	return used+requested > limit
}

func findExistingReservationTx(tx *gorm.DB, reservationId string, requestId string) (*model.QuotaReservation, error) {
	var reservation model.QuotaReservation
	query := tx.Where("reservation_id = ?", reservationId)
	if requestId != "" {
		query = tx.Where("reservation_id = ? OR request_id = ?", reservationId, requestId)
	}
	if err := query.Order("id asc").First(&reservation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reservation, nil
}

func createQuotaUsageRecordTx(tx *gorm.DB, record *model.QuotaUsageRecord) error {
	if record.QuotaDimension == "" {
		record.QuotaDimension = quotaDimensionFor(record.TokenDelta, record.RequestDelta)
	}
	return tx.Create(record).Error
}

func quotaUsageRecordFromReservation(reservation model.QuotaReservation, status string, metadata string) model.QuotaUsageRecord {
	return model.QuotaUsageRecord{
		TenantId:              reservation.TenantId,
		OrganizationId:        reservation.OrganizationId,
		DepartmentId:          reservation.DepartmentId,
		DistributionChannelId: reservation.DistributionChannelId,
		UserId:                reservation.UserId,
		UserSubscriptionId:    reservation.UserSubscriptionId,
		RequestId:             reservation.RequestId,
		ReservationId:         reservation.ReservationId,
		ModelName:             reservation.ModelName,
		QuotaDimension:        quotaDimensionFor(reservation.TokenReserved, reservation.RequestReserved),
		TokenDelta:            reservation.TokenReserved,
		RequestDelta:          reservation.RequestReserved,
		Status:                status,
		Metadata:              metadata,
	}
}

func firstOrCreateCommittedUsageTx(tx *gorm.DB, reservation model.QuotaReservation, input QuotaCommitInput, usage *model.QuotaUsageRecord) error {
	if err := tx.Where("reservation_id = ? AND status = ?", reservation.ReservationId, model.QuotaUsageStatusCommitted).
		Order("id asc").
		First(usage).Error; err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	*usage = quotaUsageRecordFromReservation(reservation, model.QuotaUsageStatusCommitted, input.Metadata)
	usage.RequestId = firstNonEmpty(strings.TrimSpace(input.RequestId), reservation.RequestId)
	usage.ModelName = firstNonEmpty(strings.TrimSpace(input.ModelName), reservation.ModelName)
	usage.TokenDelta = firstNonZero(input.TokenAmount, reservation.TokenReserved)
	usage.RequestDelta = firstNonZero(input.RequestAmount, reservation.RequestReserved)
	return createQuotaUsageRecordTx(tx, usage)
}

func ensureRollbackUsageTx(tx *gorm.DB, reservation model.QuotaReservation, input QuotaRollbackInput) error {
	var existing model.QuotaUsageRecord
	if err := tx.Where("reservation_id = ? AND status = ?", reservation.ReservationId, model.QuotaUsageStatusRolledBack).
		Order("id asc").
		First(&existing).Error; err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	record := quotaUsageRecordFromReservation(reservation, model.QuotaUsageStatusRolledBack, firstNonEmpty(input.Metadata, input.Reason))
	record.RequestId = firstNonEmpty(strings.TrimSpace(input.RequestId), reservation.RequestId)
	record.TokenDelta = -reservation.TokenReserved
	record.RequestDelta = -reservation.RequestReserved
	return createQuotaUsageRecordTx(tx, &record)
}

func quotaDimensionFor(tokenDelta int64, requestDelta int64) string {
	if tokenDelta != 0 {
		return model.QuotaDimensionToken
	}
	if requestDelta != 0 {
		return model.QuotaDimensionRequest
	}
	return model.QuotaDimensionModel
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func advanceQuotaResetWindow(base time.Time, plan *model.SubscriptionPlan, endUnix int64) int64 {
	if plan == nil {
		return 0
	}
	next := nextQuotaResetTime(base, plan, endUnix)
	for next > 0 && next <= base.Unix() {
		next = nextQuotaResetTime(time.Unix(next, 0), plan, endUnix)
	}
	return next
}

func nextQuotaResetTime(base time.Time, plan *model.SubscriptionPlan, endUnix int64) int64 {
	switch model.NormalizeResetPeriod(plan.QuotaResetPeriod) {
	case model.SubscriptionResetDaily:
		return capResetAtEnd(time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).AddDate(0, 0, 1).Unix(), endUnix)
	case model.SubscriptionResetWeekly:
		weekday := int(base.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return capResetAtEnd(time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).AddDate(0, 0, 8-weekday).Unix(), endUnix)
	case model.SubscriptionResetMonthly:
		return capResetAtEnd(time.Date(base.Year(), base.Month(), 1, 0, 0, 0, 0, base.Location()).AddDate(0, 1, 0).Unix(), endUnix)
	case model.SubscriptionResetCustom:
		if plan.QuotaResetCustomSeconds <= 0 {
			return 0
		}
		return capResetAtEnd(base.Add(time.Duration(plan.QuotaResetCustomSeconds)*time.Second).Unix(), endUnix)
	default:
		return 0
	}
}

func capResetAtEnd(next int64, endUnix int64) int64 {
	if endUnix > 0 && next > endUnix {
		return 0
	}
	return next
}
