package service

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func subsidyQuoteFixture() (EnterpriseSubsidyPolicy, EnterpriseSubsidyBudget, EnterpriseSubsidyCharge, time.Time) {
	at := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	policy := EnterpriseSubsidyPolicy{Id: 8, Version: 1, Currency: "CNY", RateBasisPoints: 3000,
		MonthlyCapMinor: 10000000, StartsAt: at.Add(-time.Hour).Unix(), EndsAt: at.AddDate(0, 6, 0).Unix(),
		TimeZone: "Asia/Shanghai", DiscountMode: SubsidyAfterDiscount}
	budget := EnterpriseSubsidyBudget{TenantId: 42, PolicyId: policy.Id, Currency: "CNY", Month: "2026-11"}
	charge := EnterpriseSubsidyCharge{TenantId: 42, Currency: "CNY", GrossMinor: 10000, DiscountMinor: 2000}
	return policy, budget, charge, at
}

func TestEnterpriseSubsidyDiscountAndReservation(t *testing.T) {
	p, b, c, at := subsidyQuoteFixture()
	q, err := QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Equal(t, int64(2400), q.SubsidyMinor)
	require.Equal(t, int64(5600), q.CustomerPaysMinor)
	// Pending reservations occupy the same cap as completed reductions.
	b.UsedMinor = p.MonthlyCapMinor - 1500
	b.ReservedMinor = 1000
	q, err = QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Equal(t, int64(500), q.SubsidyMinor)
	require.Equal(t, int64(7500), q.CustomerPaysMinor)
	require.Zero(t, q.RemainingCapMinor)
	// A new policy version cannot reset the existing policy/month budget.
	p.Version++
	q, err = QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Equal(t, int64(500), q.SubsidyMinor)
	// A different enterprise has its own independently scoped monthly cap.
	b.TenantId, c.TenantId = 43, 43
	b.UsedMinor, b.ReservedMinor = 0, 0
	q, err = QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Equal(t, int64(2400), q.SubsidyMinor)
}

func TestEnterpriseSubsidyRejectsWrongLedgerScope(t *testing.T) {
	for _, field := range []string{"tenant", "policy", "currency", "charge_currency", "month"} {
		t.Run(field, func(t *testing.T) {
			p, b, c, at := subsidyQuoteFixture()
			switch field {
			case "tenant":
				b.TenantId++
			case "policy":
				b.PolicyId++
			case "currency":
				b.Currency = "USD"
			case "charge_currency":
				c.Currency = "USD"
			case "month":
				b.Month = "2026-10"
			}
			q, err := QuoteEnterpriseSubsidy(p, b, c, at)
			require.ErrorIs(t, err, ErrSubsidyScope)
			require.Nil(t, q)
		})
	}
}

func TestEnterpriseSubsidyExclusiveDiscount(t *testing.T) {
	p, b, c, at := subsidyQuoteFixture()
	p.DiscountMode = SubsidyWithoutDiscount
	for _, flag := range []bool{false, true} {
		c.HasOtherDiscount = flag
		q, err := QuoteEnterpriseSubsidy(p, b, c, at)
		require.NoError(t, err)
		require.Zero(t, q.SubsidyMinor)
		require.Equal(t, int64(8000), q.CustomerPaysMinor)
		require.Equal(t, "commercial_discount_excluded", q.Reason)
	}
	// A promotion may exist even if its effect on this charge rounds to zero.
	c.DiscountMinor = 0
	q, err := QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Zero(t, q.SubsidyMinor)
	c.HasOtherDiscount = false
	q, err = QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Equal(t, int64(3000), q.SubsidyMinor)
}

func TestEnterpriseSubsidyCalendarAndPeriodBoundaries(t *testing.T) {
	p, b, c, at := subsidyQuoteFixture()
	at = time.Date(2026, 11, 30, 16, 0, 0, 0, time.UTC) // December in Shanghai.
	_, err := QuoteEnterpriseSubsidy(p, b, c, at)
	require.ErrorIs(t, err, ErrSubsidyScope)
	b.Month = "2026-12"
	p.StartsAt = at.Unix()
	p.EndsAt = at.Add(time.Second).Unix()
	q, err := QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Positive(t, q.SubsidyMinor)
	for _, when := range []time.Time{at.Add(-time.Nanosecond), at.Add(time.Second)} {
		// The previous nanosecond is still November in the policy's timezone.
		loc, loadErr := time.LoadLocation(p.TimeZone)
		require.NoError(t, loadErr)
		b.Month = when.In(loc).Format("2006-01")
		q, err = QuoteEnterpriseSubsidy(p, b, c, when)
		require.NoError(t, err)
		require.Zero(t, q.SubsidyMinor)
		require.Equal(t, "outside_policy_period", q.Reason)
	}
}

func TestEnterpriseSubsidyRoundingAndMaximumAmounts(t *testing.T) {
	p, b, c, at := subsidyQuoteFixture()
	c.GrossMinor, c.DiscountMinor = 1, 0
	q, err := QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Zero(t, q.SubsidyMinor)
	require.Equal(t, int64(1), q.CustomerPaysMinor)
	p.Currency, b.Currency, c.Currency = "USD", "USD", "USD"
	p.MonthlyCapMinor, c.GrossMinor, p.RateBasisPoints = math.MaxInt64, math.MaxInt64, 10000
	q, err = QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Equal(t, int64(math.MaxInt64), q.SubsidyMinor)
	require.Zero(t, q.CustomerPaysMinor)
	p.RateBasisPoints = 9999
	q, err = QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Equal(t, int64(9222449699651090329), q.SubsidyMinor)
	require.Equal(t, c.GrossMinor, q.SubsidyMinor+q.CustomerPaysMinor)
	b.ReservedMinor = math.MaxInt64
	q, err = QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Zero(t, q.SubsidyMinor)
	require.Equal(t, "monthly_cap_reached", q.Reason)
	b.UsedMinor = 1
	_, err = QuoteEnterpriseSubsidy(p, b, c, at)
	require.ErrorIs(t, err, ErrSubsidyAmount)
}

func TestEnterpriseSubsidySequentialQuotesRespectCap(t *testing.T) {
	p, b, c, at := subsidyQuoteFixture()
	p.MonthlyCapMinor = 1000
	c.GrossMinor, c.DiscountMinor = 1000, 0
	var reductions int64
	for range 12 {
		q, err := QuoteEnterpriseSubsidy(p, b, c, at)
		require.NoError(t, err)
		require.Equal(t, c.GrossMinor, q.CustomerPaysMinor+q.SubsidyMinor)
		require.GreaterOrEqual(t, q.CustomerPaysMinor, int64(0))
		// Simulate the caller's serialized reservation; the calculator alone
		// deliberately does not make concurrent reservations safe.
		b.ReservedMinor += q.SubsidyMinor
		reductions += q.SubsidyMinor
	}
	require.Equal(t, p.MonthlyCapMinor, reductions)
	// Settlement transfers a reservation to used without freeing cap.
	b.UsedMinor, b.ReservedMinor = b.ReservedMinor, 0
	q, err := QuoteEnterpriseSubsidy(p, b, c, at)
	require.NoError(t, err)
	require.Zero(t, q.SubsidyMinor)
}

func TestEnterpriseSubsidyRejectsInvalidInputs(t *testing.T) {
	for _, field := range []string{"id", "version", "rate", "cap", "dates", "timezone", "discount_mode", "amount", "discount", "used", "reserved"} {
		t.Run(field, func(t *testing.T) {
			p, b, c, at := subsidyQuoteFixture()
			want := ErrSubsidyPolicy
			switch field {
			case "id":
				p.Id = 0
			case "version":
				p.Version = 0
			case "rate":
				p.RateBasisPoints = 10001
			case "cap":
				p.MonthlyCapMinor = 0
			case "dates":
				p.EndsAt = p.StartsAt
			case "timezone":
				p.TimeZone = "Unknown/Zone"
			case "discount_mode":
				p.DiscountMode = ""
			case "amount":
				c.GrossMinor, want = -1, ErrSubsidyAmount
			case "discount":
				c.DiscountMinor, want = c.GrossMinor+1, ErrSubsidyAmount
			case "used":
				b.UsedMinor, want = -1, ErrSubsidyAmount
			case "reserved":
				b.ReservedMinor, want = -1, ErrSubsidyAmount
			}
			q, err := QuoteEnterpriseSubsidy(p, b, c, at)
			require.ErrorIs(t, err, want)
			require.Nil(t, q)
		})
	}
}
