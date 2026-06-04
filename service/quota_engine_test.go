package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFoundationQuotaEngineReturnsNotImplemented(t *testing.T) {
	engine := NewFoundationQuotaEngine()
	require.NotNil(t, engine)

	ctx := context.Background()

	_, err := engine.CheckQuota(ctx, QuotaCheckInput{})
	require.True(t, errors.Is(err, ErrQuotaEngineNotImplemented))

	_, err = engine.ReserveQuota(ctx, QuotaReservationInput{})
	require.True(t, errors.Is(err, ErrQuotaEngineNotImplemented))

	_, err = engine.CommitUsage(ctx, QuotaCommitInput{})
	require.True(t, errors.Is(err, ErrQuotaEngineNotImplemented))

	err = engine.RollbackReservation(ctx, QuotaRollbackInput{})
	require.True(t, errors.Is(err, ErrQuotaEngineNotImplemented))

	err = engine.ResetQuota(ctx, QuotaResetInput{})
	require.True(t, errors.Is(err, ErrQuotaEngineNotImplemented))
}
