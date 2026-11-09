package service

import (
	"errors"
	"time"
)

var (
	ErrSubsidyPolicy = errors.New("invalid subsidy policy")
	ErrSubsidyScope  = errors.New("subsidy budget does not match enterprise, policy, currency or month")
	ErrSubsidyAmount = errors.New("invalid subsidy charge or monthly budget")
)

const (
	// SubsidyAfterDiscount applies the subsidy to the eligible discounted price.
	SubsidyAfterDiscount = "after_discount"
	// SubsidyWithoutDiscount rejects combining subsidy and commercial discounts.
	// It does not automatically choose or remove a customer's contract discount.
	SubsidyWithoutDiscount = "without_discount"
)

// EnterpriseSubsidyPolicy is a versioned calculation snapshot, not a cash balance.
// Currency amounts are integer minor units. No implicit foreign exchange occurs.
// Percentages use basis points: 3000 means 30%. Dates are [StartsAt, EndsAt).
// Callers must use a server-owned approved policy and verify enterprise eligibility.
type EnterpriseSubsidyPolicy struct {
	Id              int
	Version         int
	Currency        string
	RateBasisPoints int64
	MonthlyCapMinor int64
	StartsAt        int64
	EndsAt          int64
	TimeZone        string
	DiscountMode    string
}

// EnterpriseSubsidyBudget is a trusted ledger projection for one enterprise and
// calendar month. UsedMinor excludes ReservedMinor; both occupy the monthly cap.
// PolicyId is stable across versions so publishing a version cannot reset a cap.
type EnterpriseSubsidyBudget struct {
	TenantId      int
	PolicyId      int
	Currency      string
	Month         string
	UsedMinor     int64
	ReservedMinor int64
}

type EnterpriseSubsidyCharge struct {
	TenantId         int
	Currency         string
	GrossMinor       int64
	DiscountMinor    int64
	HasOtherDiscount bool
}

// EnterpriseSubsidyQuote only allocates a proposed reduction of the customer
// charge. Government reimbursement is a separate receivable and receipt process.
type EnterpriseSubsidyQuote struct {
	TenantId           int
	PolicyId           int
	PolicyVersion      int
	Currency           string
	Month              string
	BeforeSubsidyMinor int64
	SubsidyMinor       int64
	CustomerPaysMinor  int64
	RemainingCapMinor  int64
	Reason             string
}

// QuoteEnterpriseSubsidy is pure: it neither reserves the cap nor grants quota.
// An eventual caller MUST atomically lock/check the shared monthly budget and
// persist the quote with a unique charge identity before applying a reduction.
// Passing parallel independent snapshots here does not enforce a concurrent cap.
// Round down to the currency's minor unit so fragmented charges cannot create
// extra subsidy. This rounding rule is a proposal to freeze in each policy.
func QuoteEnterpriseSubsidy(policy EnterpriseSubsidyPolicy, budget EnterpriseSubsidyBudget, charge EnterpriseSubsidyCharge, at time.Time) (*EnterpriseSubsidyQuote, error) {
	if policy.Id <= 0 || policy.Version <= 0 || (policy.Currency != "CNY" && policy.Currency != "USD") ||
		policy.RateBasisPoints <= 0 || policy.RateBasisPoints > 10000 || policy.MonthlyCapMinor <= 0 ||
		policy.StartsAt < 0 || policy.EndsAt <= policy.StartsAt || policy.TimeZone == "" ||
		(policy.DiscountMode != SubsidyAfterDiscount && policy.DiscountMode != SubsidyWithoutDiscount) {
		return nil, ErrSubsidyPolicy
	}
	location, err := time.LoadLocation(policy.TimeZone)
	if err != nil {
		return nil, ErrSubsidyPolicy
	}
	month := at.In(location).Format("2006-01")
	if charge.TenantId <= 0 || budget.TenantId != charge.TenantId || budget.PolicyId != policy.Id ||
		charge.Currency != policy.Currency || budget.Currency != policy.Currency || budget.Month != month {
		return nil, ErrSubsidyScope
	}
	if charge.GrossMinor < 0 || charge.DiscountMinor < 0 || charge.DiscountMinor > charge.GrossMinor ||
		budget.UsedMinor < 0 || budget.ReservedMinor < 0 || budget.UsedMinor > policy.MonthlyCapMinor ||
		budget.ReservedMinor > policy.MonthlyCapMinor-budget.UsedMinor {
		return nil, ErrSubsidyAmount
	}
	remaining := policy.MonthlyCapMinor - budget.UsedMinor - budget.ReservedMinor
	net := charge.GrossMinor - charge.DiscountMinor
	quote := &EnterpriseSubsidyQuote{
		TenantId: charge.TenantId, PolicyId: policy.Id, PolicyVersion: policy.Version,
		Currency: policy.Currency, Month: month, BeforeSubsidyMinor: net,
		CustomerPaysMinor: net, RemainingCapMinor: remaining,
	}
	if at.Before(time.Unix(policy.StartsAt, 0)) || !at.Before(time.Unix(policy.EndsAt, 0)) {
		quote.Reason = "outside_policy_period"
		return quote, nil
	}
	if policy.DiscountMode == SubsidyWithoutDiscount && (charge.HasOtherDiscount || charge.DiscountMinor > 0) {
		quote.Reason = "commercial_discount_excluded"
		return quote, nil
	}
	if remaining == 0 {
		quote.Reason = "monthly_cap_reached"
		return quote, nil
	}
	// Split the product before multiplying to avoid int64 overflow and floating
	// point rounding. Both terms and their sum are bounded by net at <=100%.
	amount := (net/10000)*policy.RateBasisPoints + (net%10000)*policy.RateBasisPoints/10000
	if amount > remaining {
		amount = remaining
	}
	quote.SubsidyMinor = amount
	quote.CustomerPaysMinor = net - amount
	quote.RemainingCapMinor = remaining - amount
	quote.Reason = "quoted"
	return quote, nil
}
