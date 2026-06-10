package service

import (
	"context"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"gorm.io/gorm"
)

const BillingPortalCurrencyQuota = "QUOTA"

type BillingPortalService struct{}

func NewBillingPortalService() *BillingPortalService {
	return &BillingPortalService{}
}

type BillingPortalActor struct {
	UserId int
	Scope  model.AccessScope
}

type BillingPortalListInput struct {
	Page          int
	PageSize      int
	StartTime     int64
	EndTime       int64
	Status        string
	Provider      string
	Model         string
	PaymentMethod string
	TenantId      int
	Subscription  string
}

type BillingPortalSummary struct {
	BalanceQuota               int64                      `json:"balance_quota"`
	CurrentSubscriptions       []model.UserSubscription   `json:"current_subscriptions"`
	TotalRechargeAmount        float64                    `json:"total_recharge_amount"`
	TotalRechargeCurrency      string                     `json:"total_recharge_currency"`
	TotalConsumptionAmount     int64                      `json:"total_consumption_amount"`
	ConsumptionCurrency        string                     `json:"consumption_currency"`
	TotalTokens                int64                      `json:"total_tokens"`
	TotalRequests              int64                      `json:"total_requests"`
	Recent30dConsumption       int64                      `json:"recent_30d_consumption"`
	Recent30dTokens            int64                      `json:"recent_30d_tokens"`
	Recent30dRequests          int64                      `json:"recent_30d_requests"`
	ModelConsumptionRanking    []BillingPortalRankingItem `json:"model_consumption_ranking"`
	ProviderConsumptionRanking []BillingPortalRankingItem `json:"provider_consumption_ranking"`
}

type BillingPortalRankingItem struct {
	Name         string `json:"name"`
	QuotaCharged int64  `json:"quota_charged"`
	TotalTokens  int64  `json:"total_tokens"`
	RequestCount int64  `json:"request_count"`
}

type BillingPortalPage[T any] struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
	Items    []T   `json:"items"`
}

func (s *BillingPortalService) Summary(ctx context.Context, actor BillingPortalActor) (BillingPortalSummary, error) {
	var out BillingPortalSummary
	out.TotalRechargeCurrency = "USD"
	out.ConsumptionCurrency = BillingPortalCurrencyQuota

	userQuery := billingPortalScopedUserQuery(actor)
	var balance int64
	if err := userQuery.WithContext(ctx).Select("COALESCE(SUM(quota), 0)").Scan(&balance).Error; err != nil {
		return out, err
	}
	out.BalanceQuota = balance

	subQuery := billingPortalScopedQuery(model.DB.WithContext(ctx).Model(&model.UserSubscription{}), "user_subscriptions", actor).
		Where("status = ? AND end_time > ?", model.SubscriptionLifecycleActive, common.GetTimestamp())
	if err := subQuery.Order("end_time desc, id desc").Limit(10).Find(&out.CurrentSubscriptions).Error; err != nil {
		return out, err
	}

	paymentQuery := billingPortalScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor).
		Where("status = ?", model.PaymentOrderStatusPaid)
	if err := paymentQuery.Select("COALESCE(SUM(amount), 0)").Scan(&out.TotalRechargeAmount).Error; err != nil {
		return out, err
	}

	billingQuery := billingPortalScopedQuery(model.DB.WithContext(ctx).Model(&model.BillingRecord{}), "billing_records", actor)
	if err := billingQuery.Select("COALESCE(SUM(quota_charged), 0), COALESCE(SUM(total_tokens), 0), COALESCE(SUM(request_count), 0)").
		Row().Scan(&out.TotalConsumptionAmount, &out.TotalTokens, &out.TotalRequests); err != nil {
		return out, err
	}

	recentSince := common.GetTimestamp() - 30*24*60*60
	recentQuery := billingPortalScopedQuery(model.DB.WithContext(ctx).Model(&model.BillingRecord{}), "billing_records", actor).
		Where("created_at >= ?", recentSince)
	if err := recentQuery.Select("COALESCE(SUM(quota_charged), 0), COALESCE(SUM(total_tokens), 0), COALESCE(SUM(request_count), 0)").
		Row().Scan(&out.Recent30dConsumption, &out.Recent30dTokens, &out.Recent30dRequests); err != nil {
		return out, err
	}

	modelRanks, err := billingPortalRanking(ctx, actor, "model_name")
	if err != nil {
		return out, err
	}
	out.ModelConsumptionRanking = modelRanks
	providerRanks, err := billingPortalRanking(ctx, actor, "provider_name")
	if err != nil {
		return out, err
	}
	out.ProviderConsumptionRanking = providerRanks
	return out, nil
}

func (s *BillingPortalService) PaymentHistory(ctx context.Context, actor BillingPortalActor, input BillingPortalListInput) (BillingPortalPage[model.PaymentOrder], error) {
	var page BillingPortalPage[model.PaymentOrder]
	setPageDefaults(&input)
	page.Page = input.Page
	page.PageSize = input.PageSize
	query := billingPortalScopedQuery(model.DB.WithContext(ctx).Model(&model.PaymentOrder{}), "payment_orders", actor)
	query = applyPaymentPortalFilters(query, input)
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.PaymentOrder
	if err := query.Order("id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func (s *BillingPortalService) UsageHistory(ctx context.Context, actor BillingPortalActor, input BillingPortalListInput) (BillingPortalPage[model.QuotaUsageRecord], error) {
	var page BillingPortalPage[model.QuotaUsageRecord]
	setPageDefaults(&input)
	page.Page = input.Page
	page.PageSize = input.PageSize
	query := billingPortalScopedQuery(model.DB.WithContext(ctx).Model(&model.QuotaUsageRecord{}), "quota_usage_records", actor)
	query = applyUsagePortalFilters(query, input)
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.QuotaUsageRecord
	if err := query.Order("occurred_at desc, id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func (s *BillingPortalService) BillingHistory(ctx context.Context, actor BillingPortalActor, input BillingPortalListInput) (BillingPortalPage[model.BillingRecord], error) {
	var page BillingPortalPage[model.BillingRecord]
	setPageDefaults(&input)
	page.Page = input.Page
	page.PageSize = input.PageSize
	query := billingPortalScopedQuery(model.DB.WithContext(ctx).Model(&model.BillingRecord{}), "billing_records", actor)
	query = applyBillingPortalFilters(query, input)
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

func (s *BillingPortalService) SubscriptionHistory(ctx context.Context, actor BillingPortalActor, input BillingPortalListInput) (BillingPortalPage[model.UserSubscription], error) {
	var page BillingPortalPage[model.UserSubscription]
	setPageDefaults(&input)
	page.Page = input.Page
	page.PageSize = input.PageSize
	now := common.GetTimestamp()
	query := billingPortalScopedQuery(model.DB.WithContext(ctx).Model(&model.UserSubscription{}), "user_subscriptions", actor)
	switch strings.ToLower(strings.TrimSpace(input.Subscription)) {
	case "active":
		query = query.Where("status = ? AND end_time > ?", model.SubscriptionLifecycleActive, now)
	case "expiring":
		query = query.Where("status = ? AND end_time > ? AND end_time <= ?", model.SubscriptionLifecycleActive, now, now+7*24*60*60)
	case "history":
		// all rows
	default:
		// all rows
	}
	if input.StartTime > 0 {
		query = query.Where("created_at >= ?", input.StartTime)
	}
	if input.EndTime > 0 {
		query = query.Where("created_at <= ?", input.EndTime)
	}
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.UserSubscription
	if err := query.Order("end_time desc, id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func billingPortalScopedQuery(db *gorm.DB, table string, actor BillingPortalActor) *gorm.DB {
	if actor.Scope.IsRoot {
		return db
	}
	db = model.ApplyOwnershipScope(db, table, actor.Scope)
	if !common.IsTenantAdminRole(actor.Scope.RoleKey) && !common.IsOrganizationAdminRole(actor.Scope.RoleKey) {
		db = db.Where(table+".user_id = ?", actor.UserId)
	}
	return db
}

func billingPortalScopedUserQuery(actor BillingPortalActor) *gorm.DB {
	db := model.DB.Model(&model.User{})
	if actor.Scope.IsRoot {
		return db
	}
	db = model.ApplyOwnershipScope(db, "users", actor.Scope)
	if !common.IsTenantAdminRole(actor.Scope.RoleKey) && !common.IsOrganizationAdminRole(actor.Scope.RoleKey) {
		db = db.Where("users.id = ?", actor.UserId)
	}
	return db
}

func billingPortalRanking(ctx context.Context, actor BillingPortalActor, column string) ([]BillingPortalRankingItem, error) {
	var rows []BillingPortalRankingItem
	nameExpr := column
	if column != "model_name" && column != "provider_name" {
		nameExpr = "model_name"
	}
	query := billingPortalScopedQuery(model.DB.WithContext(ctx).Model(&model.BillingRecord{}), "billing_records", actor).
		Select(nameExpr + " AS name, COALESCE(SUM(quota_charged), 0) AS quota_charged, COALESCE(SUM(total_tokens), 0) AS total_tokens, COALESCE(SUM(request_count), 0) AS request_count").
		Where(nameExpr + " <> ''").
		Group(nameExpr).
		Order("quota_charged desc, total_tokens desc").
		Limit(10)
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func applyPaymentPortalFilters(query *gorm.DB, input BillingPortalListInput) *gorm.DB {
	if input.StartTime > 0 {
		query = query.Where("created_at >= ?", input.StartTime)
	}
	if input.EndTime > 0 {
		query = query.Where("created_at <= ?", input.EndTime)
	}
	if input.Status != "" {
		query = query.Where("status = ?", model.NormalizePaymentOrderStatus(input.Status))
	}
	if input.PaymentMethod != "" {
		query = query.Where("provider = ?", model.NormalizePaymentProvider(input.PaymentMethod))
	}
	return query
}

func applyUsagePortalFilters(query *gorm.DB, input BillingPortalListInput) *gorm.DB {
	if input.StartTime > 0 {
		query = query.Where("occurred_at >= ?", input.StartTime)
	}
	if input.EndTime > 0 {
		query = query.Where("occurred_at <= ?", input.EndTime)
	}
	if input.Provider != "" {
		query = query.Where("provider_name = ?", strings.TrimSpace(input.Provider))
	}
	if input.Model != "" {
		query = query.Where("model_name = ?", strings.TrimSpace(input.Model))
	}
	return query
}

func applyBillingPortalFilters(query *gorm.DB, input BillingPortalListInput) *gorm.DB {
	if input.StartTime > 0 {
		query = query.Where("created_at >= ?", input.StartTime)
	}
	if input.EndTime > 0 {
		query = query.Where("created_at <= ?", input.EndTime)
	}
	if input.Provider != "" {
		query = query.Where("provider_name = ?", strings.TrimSpace(input.Provider))
	}
	if input.Model != "" {
		query = query.Where("model_name = ?", strings.TrimSpace(input.Model))
	}
	if input.TenantId > 0 {
		query = query.Where("tenant_id = ?", input.TenantId)
	}
	return query
}

func setPageDefaults(input *BillingPortalListInput) {
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 20
	}
	if input.PageSize > 100 {
		input.PageSize = 100
	}
}
