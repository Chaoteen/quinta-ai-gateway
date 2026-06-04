package service

import (
	"context"
	"errors"

	"github.com/Chaoteen/quinta-ai-gateway/model"
)

var ErrQuotaEngineNotImplemented = errors.New("quota engine foundation is not implemented")

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
	return QuotaDecision{}, ErrQuotaEngineNotImplemented
}

func (e *FoundationQuotaEngine) ReserveQuota(ctx context.Context, input QuotaReservationInput) (model.QuotaReservation, error) {
	return model.QuotaReservation{}, ErrQuotaEngineNotImplemented
}

func (e *FoundationQuotaEngine) CommitUsage(ctx context.Context, input QuotaCommitInput) (model.QuotaUsageRecord, error) {
	return model.QuotaUsageRecord{}, ErrQuotaEngineNotImplemented
}

func (e *FoundationQuotaEngine) RollbackReservation(ctx context.Context, input QuotaRollbackInput) error {
	return ErrQuotaEngineNotImplemented
}

func (e *FoundationQuotaEngine) ResetQuota(ctx context.Context, input QuotaResetInput) error {
	return ErrQuotaEngineNotImplemented
}
