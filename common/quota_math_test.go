package common

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestSecurityQuotaConversionNeverWrapsPositiveCharge(t *testing.T) {
	for _, value := range []float64{math.MaxFloat64, math.Inf(1), math.Exp2(64), float64(MaxBillingQuota) + 1} {
		require.Equal(t, MaxBillingQuota, QuotaFromFloat(value))
		require.Equal(t, MaxBillingQuota, QuotaRound(value))
	}
	require.Equal(t, 0, QuotaFromFloat(math.NaN()))
	require.Equal(t, -MaxBillingQuota, QuotaFromFloat(math.Inf(-1)))
	require.Equal(t, MaxBillingQuota, QuotaFromDecimal(decimal.RequireFromString("1e100")))
	require.Equal(t, -MaxBillingQuota, QuotaFromDecimal(decimal.RequireFromString("-1e100")))
	for _, v := range []float64{0, 0.49, 0.5, 1.5, -0.5, -1.5, 500000} {
		require.Equal(t, int(math.Round(v)), QuotaRound(v))
		require.Equal(t, int(v), QuotaFromFloat(v))
		require.Equal(t, QuotaRound(v), QuotaFromDecimal(decimal.NewFromFloat(v)))
	}
}
