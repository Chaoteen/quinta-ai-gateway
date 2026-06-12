package service

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"gorm.io/gorm"
)

const FinanceConsoleCurrencyQuota = "QUOTA"

type FinanceConsoleService struct{}

func NewFinanceConsoleService() *FinanceConsoleService {
	return &FinanceConsoleService{}
}

type FinanceConsoleActor struct {
	UserId int
	Scope  model.AccessScope
}

type FinanceConsoleListInput struct {
	Page     int
	PageSize int
	Days     int
}

type FinanceConsolePage[T any] struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
	Items    []T   `json:"items"`
}

type FinanceSummary struct {
	Revenue      FinanceRevenueSummary      `json:"revenue"`
	Consumption  FinanceConsumptionSummary  `json:"consumption"`
	Activity     FinanceActivitySummary     `json:"activity"`
	Payment      FinancePaymentDashboard    `json:"payment"`
	Voucher      FinanceVoucherDashboard    `json:"voucher"`
	RevenueShare FinanceRevenueShareSummary `json:"revenue_share"`
	Tenant       FinanceTenantDashboard     `json:"tenant"`
}

type FinanceRevenueSummary struct {
	TotalRechargeAmount   float64 `json:"total_recharge_amount"`
	MonthRechargeAmount   float64 `json:"month_recharge_amount"`
	Recent30dRecharge     float64 `json:"recent_30d_recharge"`
	PaymentOrderCount     int64   `json:"payment_order_count"`
	PaidPaymentOrderCount int64   `json:"paid_payment_order_count"`
	PaymentSuccessRate    float64 `json:"payment_success_rate"`
	Currency              string  `json:"currency"`
}

type FinanceConsumptionSummary struct {
	TotalConsumptionAmount int64  `json:"total_consumption_amount"`
	MonthConsumptionAmount int64  `json:"month_consumption_amount"`
	Recent30dConsumption   int64  `json:"recent_30d_consumption"`
	TotalRequests          int64  `json:"total_requests"`
	TotalTokens            int64  `json:"total_tokens"`
	Currency               string `json:"currency"`
}

type FinanceActivitySummary struct {
	ActiveTenantCount       int64 `json:"active_tenant_count"`
	ActiveUserCount         int64 `json:"active_user_count"`
	ActiveSubscriptionCount int64 `json:"active_subscription_count"`
	ActiveChannelCount      int64 `json:"active_channel_count"`
}

type FinancePaymentDashboard struct {
	Days              int                      `json:"days"`
	TotalAmount       float64                  `json:"total_amount"`
	TotalOrders       int64                    `json:"total_orders"`
	PaidOrders        int64                    `json:"paid_orders"`
	SuccessRate       float64                  `json:"success_rate"`
	ProviderBreakdown []FinanceProviderPayment `json:"provider_breakdown"`
	DailyTrend        []FinanceDailyAmount     `json:"daily_trend"`
}

type FinanceProviderPayment struct {
	Provider string  `json:"provider"`
	Amount   float64 `json:"amount"`
	Orders   int64   `json:"orders"`
}

type FinanceDailyAmount struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Orders int64   `json:"orders"`
}

type FinanceVoucherDashboard struct {
	TotalIssued      int64   `json:"total_issued"`
	TotalRedeemed    int64   `json:"total_redeemed"`
	TotalUnused      int64   `json:"total_unused"`
	RedemptionRate   float64 `json:"redemption_rate"`
	BatchCount       int64   `json:"batch_count"`
	ActiveBatchCount int64   `json:"active_batch_count"`
}

type FinanceRevenueShareSummary struct {
	GrossAmount             float64                 `json:"gross_amount"`
	PlatformAmount          float64                 `json:"platform_amount"`
	MasterDistributorAmount float64                 `json:"master_distributor_amount"`
	DistributorAmount       float64                 `json:"distributor_amount"`
	Currency                string                  `json:"currency"`
	TopChannels             []FinanceTopChannelItem `json:"top_channels"`
}

type FinanceTenantDashboard struct {
	RechargeRanking     []FinanceTenantMetricItem `json:"recharge_ranking"`
	ConsumptionRanking  []FinanceTenantMetricItem `json:"consumption_ranking"`
	BalanceRanking      []FinanceTenantMetricItem `json:"balance_ranking"`
	SubscriptionRanking []FinanceTenantMetricItem `json:"subscription_ranking"`
}

type FinanceTenantMetricItem struct {
	TenantId int     `json:"tenant_id"`
	Name     string  `json:"name"`
	Amount   float64 `json:"amount"`
	Count    int64   `json:"count"`
}

type FinanceMetricItem struct {
	Name         string  `json:"name"`
	Amount       float64 `json:"amount"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
}

type FinanceTopChannelItem struct {
	DistributionChannelId int     `json:"distribution_channel_id"`
	Name                  string  `json:"name"`
	GrossAmount           float64 `json:"gross_amount"`
	PlatformAmount        float64 `json:"platform_amount"`
	RecordCount           int64   `json:"record_count"`
}

type financeTenantBillingRow struct {
	TenantId     int
	Amount       float64
	Count        int64
	RequestCount int64
}

type financeTenantAmountRow struct {
	TenantId int
	Amount   float64
	Count    int64
}

func (s *FinanceConsoleService) Summary(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput) (FinanceSummary, error) {
	var out FinanceSummary
	out.Revenue.Currency = "USD"
	out.Consumption.Currency = FinanceConsoleCurrencyQuota
	out.RevenueShare.Currency = FinanceConsoleCurrencyQuota

	monthStart := startOfCurrentMonthUnix()
	recent30d := common.GetTimestamp() - 30*24*60*60

	paymentQuery := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor)
	if err := paymentQuery.Count(&out.Revenue.PaymentOrderCount).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor).
		Where("status = ?", model.PaymentOrderStatusPaid).
		Count(&out.Revenue.PaidPaymentOrderCount).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor).
		Where("status = ?", model.PaymentOrderStatusPaid).
		Select("COALESCE(SUM(amount), 0)").Scan(&out.Revenue.TotalRechargeAmount).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor).
		Where("status = ? AND created_at >= ?", model.PaymentOrderStatusPaid, monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&out.Revenue.MonthRechargeAmount).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor).
		Where("status = ? AND created_at >= ?", model.PaymentOrderStatusPaid, recent30d).
		Select("COALESCE(SUM(amount), 0)").Scan(&out.Revenue.Recent30dRecharge).Error; err != nil {
		return out, err
	}
	out.Revenue.PaymentSuccessRate = percent(out.Revenue.PaidPaymentOrderCount, out.Revenue.PaymentOrderCount)

	billingQuery := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.BillingRecord{}), "billing_records", actor)
	if err := billingQuery.Select("COALESCE(SUM(quota_charged), 0), COALESCE(SUM(request_count), 0), COALESCE(SUM(total_tokens), 0)").
		Row().Scan(&out.Consumption.TotalConsumptionAmount, &out.Consumption.TotalRequests, &out.Consumption.TotalTokens); err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.BillingRecord{}), "billing_records", actor).
		Where("created_at >= ?", monthStart).Select("COALESCE(SUM(quota_charged), 0)").Scan(&out.Consumption.MonthConsumptionAmount).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.BillingRecord{}), "billing_records", actor).
		Where("created_at >= ?", recent30d).Select("COALESCE(SUM(quota_charged), 0)").Scan(&out.Consumption.Recent30dConsumption).Error; err != nil {
		return out, err
	}

	if err := financeScopedTenantQuery(model.DB.WithContext(ctx).Model(&model.Tenant{}), actor).Where("status = ?", 1).Count(&out.Activity.ActiveTenantCount).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.User{}), "users", actor).Where("status = ?", common.UserStatusEnabled).Count(&out.Activity.ActiveUserCount).Error; err != nil {
		return out, err
	}
	now := common.GetTimestamp()
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.UserSubscription{}), "user_subscriptions", actor).
		Where("status = ? AND end_time > ?", model.SubscriptionLifecycleActive, now).
		Count(&out.Activity.ActiveSubscriptionCount).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.Channel{}), "channels", actor).Where("status = ?", common.ChannelStatusEnabled).Count(&out.Activity.ActiveChannelCount).Error; err != nil {
		return out, err
	}

	paymentDashboard, err := s.PaymentDashboard(ctx, actor, input)
	if err != nil {
		return out, err
	}
	out.Payment = paymentDashboard
	voucherDashboard, err := s.VoucherDashboard(ctx, actor)
	if err != nil {
		return out, err
	}
	out.Voucher = voucherDashboard
	revenueShare, err := s.RevenueShareDashboard(ctx, actor)
	if err != nil {
		return out, err
	}
	out.RevenueShare = revenueShare
	tenantDashboard, err := s.TenantDashboard(ctx, actor)
	if err != nil {
		return out, err
	}
	out.Tenant = tenantDashboard
	return out, nil
}

func (s *FinanceConsoleService) PaymentDashboard(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput) (FinancePaymentDashboard, error) {
	days := normalizeFinanceDays(input.Days)
	since := common.GetTimestamp() - int64(days)*24*60*60
	out := FinancePaymentDashboard{Days: days}
	query := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor).Where("created_at >= ?", since)
	if err := query.Count(&out.TotalOrders).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor).
		Where("created_at >= ? AND status = ?", since, model.PaymentOrderStatusPaid).
		Select("COALESCE(SUM(amount), 0), COUNT(*)").Row().Scan(&out.TotalAmount, &out.PaidOrders); err != nil {
		return out, err
	}
	out.SuccessRate = percent(out.PaidOrders, out.TotalOrders)

	var breakdown []FinanceProviderPayment
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor).
		Where("created_at >= ?", since).
		Select("provider, COALESCE(SUM(CASE WHEN status = ? THEN amount ELSE 0 END), 0) AS amount, COUNT(*) AS orders", model.PaymentOrderStatusPaid).
		Group("provider").Order("amount desc").Scan(&breakdown).Error; err != nil {
		return out, err
	}
	out.ProviderBreakdown = breakdown

	var payments []model.PaymentOrder
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor).
		Where("created_at >= ? AND status = ?", since, model.PaymentOrderStatusPaid).
		Find(&payments).Error; err != nil {
		return out, err
	}
	trendMap := map[string]*FinanceDailyAmount{}
	for _, order := range payments {
		day := time.Unix(order.CreatedAt, 0).Format("2006-01-02")
		item := trendMap[day]
		if item == nil {
			item = &FinanceDailyAmount{Date: day}
			trendMap[day] = item
		}
		item.Amount += order.Amount
		item.Orders++
	}
	for _, item := range trendMap {
		out.DailyTrend = append(out.DailyTrend, *item)
	}
	sort.Slice(out.DailyTrend, func(i, j int) bool { return out.DailyTrend[i].Date < out.DailyTrend[j].Date })
	return out, nil
}

func (s *FinanceConsoleService) VoucherDashboard(ctx context.Context, actor FinanceConsoleActor) (FinanceVoucherDashboard, error) {
	var out FinanceVoucherDashboard
	voucherQuery := model.DB.WithContext(ctx).Model(&model.Voucher{}).Joins("JOIN voucher_batches ON voucher_batches.id = vouchers.batch_id")
	voucherQuery = financeScopedQuery(voucherQuery, "voucher_batches", actor)
	if err := voucherQuery.Count(&out.TotalIssued).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.Voucher{}).Joins("JOIN voucher_batches ON voucher_batches.id = vouchers.batch_id"), "voucher_batches", actor).
		Where("vouchers.status = ?", model.VoucherStatusRedeemed).Count(&out.TotalRedeemed).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.Voucher{}).Joins("JOIN voucher_batches ON voucher_batches.id = vouchers.batch_id"), "voucher_batches", actor).
		Where("vouchers.status = ?", model.VoucherStatusUnused).Count(&out.TotalUnused).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.VoucherBatch{}), "voucher_batches", actor).Count(&out.BatchCount).Error; err != nil {
		return out, err
	}
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.VoucherBatch{}), "voucher_batches", actor).Where("status = ?", model.VoucherBatchStatusActive).Count(&out.ActiveBatchCount).Error; err != nil {
		return out, err
	}
	out.RedemptionRate = percent(out.TotalRedeemed, out.TotalIssued)
	return out, nil
}

func (s *FinanceConsoleService) RevenueShareDashboard(ctx context.Context, actor FinanceConsoleActor) (FinanceRevenueShareSummary, error) {
	var out FinanceRevenueShareSummary
	out.Currency = FinanceConsoleCurrencyQuota
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.RevenueShareRecord{}), "revenue_share_records", actor).
		Select("COALESCE(SUM(gross_amount), 0), COALESCE(SUM(platform_amount), 0), COALESCE(SUM(master_distributor_amount), 0), COALESCE(SUM(distributor_amount), 0)").
		Row().Scan(&out.GrossAmount, &out.PlatformAmount, &out.MasterDistributorAmount, &out.DistributorAmount); err != nil {
		return out, err
	}
	top, err := s.TopChannels(ctx, actor, FinanceConsoleListInput{Page: 1, PageSize: 10})
	if err != nil {
		return out, err
	}
	out.TopChannels = top.Items
	return out, nil
}

func (s *FinanceConsoleService) TenantDashboard(ctx context.Context, actor FinanceConsoleActor) (FinanceTenantDashboard, error) {
	var out FinanceTenantDashboard
	recharge, err := s.tenantPaymentRanking(ctx, actor, "amount")
	if err != nil {
		return out, err
	}
	out.RechargeRanking = recharge
	consumption, err := s.TopTenants(ctx, actor, FinanceConsoleListInput{Page: 1, PageSize: 10})
	if err != nil {
		return out, err
	}
	out.ConsumptionRanking = consumption.Items
	balance, err := s.tenantUserQuotaRanking(ctx, actor)
	if err != nil {
		return out, err
	}
	out.BalanceRanking = balance
	subscriptions, err := s.tenantSubscriptionRanking(ctx, actor)
	if err != nil {
		return out, err
	}
	out.SubscriptionRanking = subscriptions
	return out, nil
}

func (s *FinanceConsoleService) TopTenants(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput) (FinanceConsolePage[FinanceTenantMetricItem], error) {
	var page FinanceConsolePage[FinanceTenantMetricItem]
	setFinancePageDefaults(&input)
	page.Page, page.PageSize = input.Page, input.PageSize
	var rows []financeTenantBillingRow
	query := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.BillingRecord{}), "billing_records", actor).
		Select("tenant_id, COALESCE(SUM(quota_charged), 0) AS amount, COUNT(*) AS count").
		Group("tenant_id").Order("amount desc")
	if err := query.Scan(&rows).Error; err != nil {
		return page, err
	}
	page.Total = int64(len(rows))
	rows = paginateSlice(rows, input.Page, input.PageSize)
	names := tenantNameMap(ctx, tenantIdsFromTopTenantRows(rows))
	for _, r := range rows {
		page.Items = append(page.Items, FinanceTenantMetricItem{TenantId: r.TenantId, Name: names[r.TenantId], Amount: r.Amount, Count: r.Count})
	}
	return page, nil
}

func (s *FinanceConsoleService) TopModels(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput) (FinanceConsolePage[FinanceMetricItem], error) {
	return groupedBillingMetric(ctx, actor, input, "model_name")
}

func (s *FinanceConsoleService) TopProviders(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput) (FinanceConsolePage[FinanceMetricItem], error) {
	return groupedBillingMetric(ctx, actor, input, "provider_name")
}

func (s *FinanceConsoleService) TopChannels(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput) (FinanceConsolePage[FinanceTopChannelItem], error) {
	var page FinanceConsolePage[FinanceTopChannelItem]
	setFinancePageDefaults(&input)
	page.Page, page.PageSize = input.Page, input.PageSize
	var rows []FinanceTopChannelItem
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.RevenueShareRecord{}), "revenue_share_records", actor).
		Select("distribution_channel_id, COALESCE(SUM(gross_amount), 0) AS gross_amount, COALESCE(SUM(platform_amount), 0) AS platform_amount, COUNT(*) AS record_count").
		Group("distribution_channel_id").Order("gross_amount desc").Scan(&rows).Error; err != nil {
		return page, err
	}
	page.Total = int64(len(rows))
	rows = paginateSlice(rows, input.Page, input.PageSize)
	names := channelNameMap(ctx, channelIdsFromTopChannelRows(rows))
	for i := range rows {
		rows[i].Name = names[rows[i].DistributionChannelId]
	}
	page.Items = rows
	return page, nil
}

func (s *FinanceConsoleService) RecentPayments(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput) (FinanceConsolePage[model.PaymentOrder], error) {
	var page FinanceConsolePage[model.PaymentOrder]
	setFinancePageDefaults(&input)
	page.Page, page.PageSize = input.Page, input.PageSize
	query := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor)
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.PaymentOrder
	if err := query.Order("created_at desc, id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func (s *FinanceConsoleService) RecentRedemptions(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput) (FinanceConsolePage[model.VoucherRedemption], error) {
	var page FinanceConsolePage[model.VoucherRedemption]
	setFinancePageDefaults(&input)
	page.Page, page.PageSize = input.Page, input.PageSize
	query := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.VoucherRedemption{}), "voucher_redemptions", actor)
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.VoucherRedemption
	if err := query.Order("created_at desc, id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func (s *FinanceConsoleService) RecentSubscriptions(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput) (FinanceConsolePage[model.UserSubscription], error) {
	var page FinanceConsolePage[model.UserSubscription]
	setFinancePageDefaults(&input)
	page.Page, page.PageSize = input.Page, input.PageSize
	query := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.UserSubscription{}), "user_subscriptions", actor)
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.UserSubscription
	if err := query.Order("created_at desc, id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func (s *FinanceConsoleService) RecentBilling(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput) (FinanceConsolePage[model.BillingRecord], error) {
	var page FinanceConsolePage[model.BillingRecord]
	setFinancePageDefaults(&input)
	page.Page, page.PageSize = input.Page, input.PageSize
	query := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.BillingRecord{}), "billing_records", actor)
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.BillingRecord
	if err := query.Order("created_at desc, id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func groupedBillingMetric(ctx context.Context, actor FinanceConsoleActor, input FinanceConsoleListInput, groupCol string) (FinanceConsolePage[FinanceMetricItem], error) {
	var page FinanceConsolePage[FinanceMetricItem]
	setFinancePageDefaults(&input)
	page.Page, page.PageSize = input.Page, input.PageSize
	var rows []FinanceMetricItem
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.BillingRecord{}), "billing_records", actor).
		Select(groupCol + " AS name, COALESCE(SUM(quota_charged), 0) AS amount, COALESCE(SUM(request_count), 0) AS request_count, COALESCE(SUM(total_tokens), 0) AS total_tokens").
		Group(groupCol).Order("amount desc").Scan(&rows).Error; err != nil {
		return page, err
	}
	page.Total = int64(len(rows))
	page.Items = paginateSlice(rows, input.Page, input.PageSize)
	return page, nil
}

func (s *FinanceConsoleService) tenantPaymentRanking(ctx context.Context, actor FinanceConsoleActor, _ string) ([]FinanceTenantMetricItem, error) {
	var rows []financeTenantAmountRow
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor).
		Where("status = ?", model.PaymentOrderStatusPaid).
		Select("tenant_id, COALESCE(SUM(amount), 0) AS amount, COUNT(*) AS count").
		Group("tenant_id").Order("amount desc").Limit(10).Scan(&rows).Error; err != nil {
		return nil, err
	}
	names := tenantNameMap(ctx, tenantIdsFromPaymentRows(rows))
	out := make([]FinanceTenantMetricItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, FinanceTenantMetricItem{TenantId: r.TenantId, Name: names[r.TenantId], Amount: r.Amount, Count: r.Count})
	}
	return out, nil
}

func (s *FinanceConsoleService) tenantUserQuotaRanking(ctx context.Context, actor FinanceConsoleActor) ([]FinanceTenantMetricItem, error) {
	var rows []financeTenantAmountRow
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.User{}), "users", actor).
		Select("tenant_id, COALESCE(SUM(quota), 0) AS amount, COUNT(*) AS count").
		Group("tenant_id").Order("amount desc").Limit(10).Scan(&rows).Error; err != nil {
		return nil, err
	}
	names := tenantNameMap(ctx, tenantIdsFromQuotaRows(rows))
	out := make([]FinanceTenantMetricItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, FinanceTenantMetricItem{TenantId: r.TenantId, Name: names[r.TenantId], Amount: r.Amount, Count: r.Count})
	}
	return out, nil
}

func (s *FinanceConsoleService) tenantSubscriptionRanking(ctx context.Context, actor FinanceConsoleActor) ([]FinanceTenantMetricItem, error) {
	var rows []financeTenantAmountRow
	if err := financeScopedQuery(model.DB.WithContext(ctx).Model(&model.UserSubscription{}), "user_subscriptions", actor).
		Where("status = ? AND end_time > ?", model.SubscriptionLifecycleActive, common.GetTimestamp()).
		Select("tenant_id, COUNT(*) AS amount, COUNT(*) AS count").
		Group("tenant_id").Order("amount desc").Limit(10).Scan(&rows).Error; err != nil {
		return nil, err
	}
	names := tenantNameMap(ctx, tenantIdsFromSubscriptionRows(rows))
	out := make([]FinanceTenantMetricItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, FinanceTenantMetricItem{TenantId: r.TenantId, Name: names[r.TenantId], Amount: r.Amount, Count: r.Count})
	}
	return out, nil
}

func financeScopedQuery(db *gorm.DB, table string, actor FinanceConsoleActor) *gorm.DB {
	if actor.Scope.IsRoot || actor.Scope.RoleKey == common.RoleKeyFinance {
		return db
	}
	return model.ApplyOwnershipScope(db, table, actor.Scope)
}

func financeScopedTenantQuery(db *gorm.DB, actor FinanceConsoleActor) *gorm.DB {
	if actor.Scope.IsRoot || actor.Scope.RoleKey == common.RoleKeyFinance {
		return db
	}
	if actor.Scope.TenantId > 0 {
		return db.Where("tenants.id = ?", actor.Scope.TenantId)
	}
	return db.Where("1 = 0")
}

func setFinancePageDefaults(input *FinanceConsoleListInput) {
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 10
	}
	if input.PageSize > 100 {
		input.PageSize = 100
	}
}

func normalizeFinanceDays(days int) int {
	switch days {
	case 7, 90:
		return days
	default:
		return 30
	}
}

func startOfCurrentMonthUnix() int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Unix()
}

func percent(num int64, denom int64) float64 {
	if denom <= 0 {
		return 0
	}
	return math.Round((float64(num)/float64(denom))*10000) / 100
}

func paginateSlice[T any](items []T, page int, size int) []T {
	start := (page - 1) * size
	if start >= len(items) {
		return []T{}
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func tenantNameMap(ctx context.Context, ids []int) map[int]string {
	names := make(map[int]string, len(ids))
	ids = uniquePositiveInts(ids)
	if len(ids) == 0 {
		return names
	}
	var tenants []model.Tenant
	if err := model.DB.WithContext(ctx).Where("id IN ?", ids).Find(&tenants).Error; err != nil {
		return names
	}
	for _, tenant := range tenants {
		names[tenant.Id] = tenant.Name
	}
	return names
}

func channelNameMap(ctx context.Context, ids []int) map[int]string {
	names := make(map[int]string, len(ids))
	ids = uniquePositiveInts(ids)
	if len(ids) == 0 {
		return names
	}
	var channels []model.DistributionChannel
	if err := model.DB.WithContext(ctx).Where("id IN ?", ids).Find(&channels).Error; err != nil {
		return names
	}
	for _, channel := range channels {
		names[channel.Id] = channel.Name
	}
	return names
}

func uniquePositiveInts(values []int) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func tenantIdsFromTopTenantRows(rows []financeTenantBillingRow) []int {
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TenantId)
	}
	return ids
}

func channelIdsFromTopChannelRows(rows []FinanceTopChannelItem) []int {
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.DistributionChannelId)
	}
	return ids
}

func tenantIdsFromPaymentRows(rows []financeTenantAmountRow) []int {
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TenantId)
	}
	return ids
}

func tenantIdsFromQuotaRows(rows []financeTenantAmountRow) []int {
	return tenantIdsFromPaymentRows(rows)
}

func tenantIdsFromSubscriptionRows(rows []financeTenantAmountRow) []int {
	return tenantIdsFromPaymentRows(rows)
}
