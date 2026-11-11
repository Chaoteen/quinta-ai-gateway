package common

import (
	"math"

	"github.com/shopspring/decimal"
)

// Single-request quota is bounded independently of the accumulated wallet.
// Check before conversion: converting out-of-range floats/decimals can turn a
// positive charge into a negative integer and credit the customer's account.
const MaxBillingQuota = math.MaxInt32

func QuotaFromFloat(value float64) int {
	switch {
	case math.IsNaN(value):
		SysError("billing quota conversion rejected NaN")
		return 0
	case value > MaxBillingQuota:
		SysError("billing quota conversion saturated positive overflow")
		return MaxBillingQuota
	case value < -MaxBillingQuota:
		SysError("billing quota conversion saturated negative overflow")
		return -MaxBillingQuota
	default:
		return int(value)
	}
}

func QuotaRound(value float64) int { return QuotaFromFloat(math.Round(value)) }

func QuotaFromDecimal(value decimal.Decimal) int {
	value = value.Round(0)
	bound := decimal.NewFromInt(MaxBillingQuota)
	if value.GreaterThan(bound) {
		SysError("billing decimal conversion saturated positive overflow")
		return MaxBillingQuota
	}
	if value.LessThan(bound.Neg()) {
		SysError("billing decimal conversion saturated negative overflow")
		return -MaxBillingQuota
	}
	return int(value.IntPart())
}
