package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"gorm.io/gorm"
)

var (
	ErrRevenueShareInvalidInput        = errors.New("revenue share input is invalid")
	ErrRevenueShareRuleNotFound        = errors.New("revenue share rule not found")
	ErrRevenueShareRecordNotFound      = errors.New("revenue share record not found")
	ErrRevenueShareTenantScopeDenied   = errors.New("revenue share tenant scope denied")
	ErrRevenueShareChannelLevelInvalid = errors.New("distribution channel level cannot exceed distributor")
)

type RevenueShareRuleInput struct {
	TenantId                   int
	DistributionChannelId      int
	RuleName                   string
	RuleScope                  string
	ProviderName               string
	ModelName                  string
	ProductType                string
	PlatformShareRate          float64
	MasterDistributorShareRate float64
	DistributorShareRate       float64
	EffectiveFrom              int64
	EffectiveTo                int64
	Enabled                    bool
}

type RevenueShareRuleListInput struct {
	TenantId              int
	DistributionChannelId int
	RuleScope             string
	Enabled               *bool
	Page                  int
	Limit                 int
}

type RevenueShareCalculationInput struct {
	TenantId              int
	BillingRecordId       int
	SourceId              int
	GrossAmount           float64
	DistributionChannelId int
	ProviderName          string
	ModelName             string
	ProductType           string
	Currency              string
	Now                   int64
}

type RevenueShareCalculationResult struct {
	PlatformAmount          float64
	MasterDistributorAmount float64
	DistributorAmount       float64
	MatchedRuleId           int
	MasterDistributorId     int
	DistributorId           int
}

type RevenueShareRecordListInput struct {
	TenantId              int
	DistributionChannelId int
	Status                string
	SourceType            string
	StartTime             int64
	EndTime               int64
	Page                  int
	Limit                 int
}

func CreateRevenueShareRule(ctx context.Context, scope model.AccessScope, input RevenueShareRuleInput) (model.RevenueShareRule, error) {
	rule := revenueShareRuleFromInput(scope, input, nil)
	if err := validateRevenueShareRule(scope, rule); err != nil {
		return model.RevenueShareRule{}, err
	}
	if err := validateRevenueShareChannel(ctx, rule.TenantId, rule.DistributionChannelId); err != nil {
		return model.RevenueShareRule{}, err
	}
	if err := model.DB.WithContext(ctx).Create(&rule).Error; err != nil {
		return model.RevenueShareRule{}, err
	}
	return rule, nil
}

func UpdateRevenueShareRule(ctx context.Context, scope model.AccessScope, id int, input RevenueShareRuleInput) (model.RevenueShareRule, error) {
	if id <= 0 {
		return model.RevenueShareRule{}, errors.Join(ErrRevenueShareRuleNotFound, errors.New("id is required"))
	}
	var rule model.RevenueShareRule
	if err := model.ApplyOwnershipScope(model.DB.WithContext(ctx), "revenue_share_rules", scope).Where("id = ?", id).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.RevenueShareRule{}, ErrRevenueShareRuleNotFound
		}
		return model.RevenueShareRule{}, err
	}
	updated := revenueShareRuleFromInput(scope, input, &rule)
	if err := validateRevenueShareRule(scope, updated); err != nil {
		return model.RevenueShareRule{}, err
	}
	if err := validateRevenueShareChannel(ctx, updated.TenantId, updated.DistributionChannelId); err != nil {
		return model.RevenueShareRule{}, err
	}
	if err := model.DB.WithContext(ctx).Save(&updated).Error; err != nil {
		return model.RevenueShareRule{}, err
	}
	return updated, nil
}

func EnableRevenueShareRule(ctx context.Context, scope model.AccessScope, id int) error {
	return setRevenueShareRuleEnabled(ctx, scope, id, true)
}

func DisableRevenueShareRule(ctx context.Context, scope model.AccessScope, id int) error {
	return setRevenueShareRuleEnabled(ctx, scope, id, false)
}

func ListRevenueShareRules(ctx context.Context, scope model.AccessScope, input RevenueShareRuleListInput) ([]model.RevenueShareRule, int64, error) {
	query := model.ApplyOwnershipScope(model.DB.WithContext(ctx).Model(&model.RevenueShareRule{}), "revenue_share_rules", scope)
	if scope.IsRoot && input.TenantId > 0 {
		query = query.Where("tenant_id = ?", input.TenantId)
	}
	if input.DistributionChannelId > 0 {
		query = query.Where("distribution_channel_id = ?", input.DistributionChannelId)
	}
	if strings.TrimSpace(input.RuleScope) != "" {
		query = query.Where("rule_scope = ?", model.NormalizeRevenueShareRuleScope(input.RuleScope))
	}
	if input.Enabled != nil {
		query = query.Where("enabled = ?", *input.Enabled)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, limit := normalizeRevenueSharePage(input.Page, input.Limit)
	var rules []model.RevenueShareRule
	if err := query.Order("id desc").Limit(limit).Offset((page - 1) * limit).Find(&rules).Error; err != nil {
		return nil, 0, err
	}
	return rules, total, nil
}

func CalculateRevenueShare(ctx context.Context, input RevenueShareCalculationInput) (RevenueShareCalculationResult, error) {
	if input.TenantId <= 0 {
		return RevenueShareCalculationResult{}, errors.Join(ErrRevenueShareInvalidInput, errors.New("tenant_id is required"))
	}
	if input.GrossAmount < 0 {
		return RevenueShareCalculationResult{}, errors.Join(ErrRevenueShareInvalidInput, errors.New("gross_amount cannot be negative"))
	}
	masterId, distributorId, err := resolveRevenueShareChannelHierarchy(ctx, input.TenantId, input.DistributionChannelId)
	if err != nil {
		return RevenueShareCalculationResult{}, err
	}
	rule, matched, err := matchRevenueShareRule(ctx, input)
	if err != nil {
		return RevenueShareCalculationResult{}, err
	}
	result := RevenueShareCalculationResult{
		PlatformAmount:      roundRevenueAmount(input.GrossAmount),
		MasterDistributorId: masterId,
		DistributorId:       distributorId,
	}
	if !matched {
		return result, nil
	}
	result.MatchedRuleId = rule.Id
	result.PlatformAmount = roundRevenueAmount(input.GrossAmount * rule.PlatformShareRate / 100)
	result.MasterDistributorAmount = roundRevenueAmount(input.GrossAmount * rule.MasterDistributorShareRate / 100)
	result.DistributorAmount = roundRevenueAmount(input.GrossAmount - result.PlatformAmount - result.MasterDistributorAmount)
	if rule.DistributorShareRate == 0 {
		result.DistributorAmount = 0
		result.PlatformAmount = roundRevenueAmount(input.GrossAmount - result.MasterDistributorAmount)
	}
	return result, nil
}

func CreateRevenueShareRecordFromBillingRecord(ctx context.Context, scope model.AccessScope, billingRecordId int) (model.RevenueShareRecord, error) {
	if billingRecordId <= 0 {
		return model.RevenueShareRecord{}, errors.Join(ErrRevenueShareRecordNotFound, errors.New("billing_record_id is required"))
	}
	var existing model.RevenueShareRecord
	err := model.DB.WithContext(ctx).Where("billing_record_id = ? AND source_type = ?", billingRecordId, model.RevenueShareSourceBilling).First(&existing).Error
	if err == nil {
		if !model.AllowsOwnership(scope, existing.TenantId, 0, 0) {
			return model.RevenueShareRecord{}, ErrRevenueShareTenantScopeDenied
		}
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.RevenueShareRecord{}, err
	}

	var billing model.BillingRecord
	if err := model.DB.WithContext(ctx).Where("id = ?", billingRecordId).First(&billing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.RevenueShareRecord{}, ErrBillingRecordNotFound
		}
		return model.RevenueShareRecord{}, err
	}
	if !model.AllowsOwnership(scope, billing.TenantId, billing.OrganizationId, billing.DepartmentId) {
		return model.RevenueShareRecord{}, ErrRevenueShareTenantScopeDenied
	}

	calculation, err := CalculateRevenueShare(ctx, RevenueShareCalculationInput{
		TenantId:              billing.TenantId,
		BillingRecordId:       billing.Id,
		SourceId:              billing.Id,
		GrossAmount:           float64(billing.QuotaCharged),
		DistributionChannelId: billing.DistributionChannelId,
		ProviderName:          billing.ProviderName,
		ModelName:             billing.ModelName,
		ProductType:           "billing",
		Currency:              billing.Currency,
	})
	if err != nil {
		return model.RevenueShareRecord{}, err
	}

	record := model.RevenueShareRecord{
		TenantId:                billing.TenantId,
		BillingRecordId:         billing.Id,
		SourceType:              model.RevenueShareSourceBilling,
		SourceId:                billing.Id,
		DistributionChannelId:   billing.DistributionChannelId,
		MasterDistributorId:     calculation.MasterDistributorId,
		DistributorId:           calculation.DistributorId,
		GrossAmount:             float64(billing.QuotaCharged),
		PlatformAmount:          calculation.PlatformAmount,
		MasterDistributorAmount: calculation.MasterDistributorAmount,
		DistributorAmount:       calculation.DistributorAmount,
		Currency:                billing.Currency,
		ShareRuleId:             calculation.MatchedRuleId,
		Status:                  model.RevenueShareStatusCalculated,
		CalculatedAt:            modelTimestamp(),
	}
	if record.Currency == "" {
		record.Currency = "QUOTA"
	}
	if err := model.DB.WithContext(ctx).Create(&record).Error; err != nil {
		var dup model.RevenueShareRecord
		if findErr := model.DB.WithContext(ctx).Where("billing_record_id = ? AND source_type = ?", billingRecordId, model.RevenueShareSourceBilling).First(&dup).Error; findErr == nil {
			return dup, nil
		}
		return model.RevenueShareRecord{}, err
	}
	return record, nil
}

func ListRevenueShareRecords(ctx context.Context, scope model.AccessScope, input RevenueShareRecordListInput) ([]model.RevenueShareRecord, int64, error) {
	query := model.ApplyOwnershipScope(model.DB.WithContext(ctx).Model(&model.RevenueShareRecord{}), "revenue_share_records", scope)
	if scope.IsRoot && input.TenantId > 0 {
		query = query.Where("tenant_id = ?", input.TenantId)
	}
	if input.DistributionChannelId > 0 {
		query = query.Where("distribution_channel_id = ?", input.DistributionChannelId)
	}
	if strings.TrimSpace(input.Status) != "" {
		query = query.Where("status = ?", model.NormalizeRevenueShareStatus(input.Status))
	}
	if strings.TrimSpace(input.SourceType) != "" {
		query = query.Where("source_type = ?", model.NormalizeRevenueShareSourceType(input.SourceType))
	}
	if input.StartTime > 0 {
		query = query.Where("created_at >= ?", input.StartTime)
	}
	if input.EndTime > 0 {
		query = query.Where("created_at <= ?", input.EndTime)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, limit := normalizeRevenueSharePage(input.Page, input.Limit)
	var records []model.RevenueShareRecord
	if err := query.Order("id desc").Limit(limit).Offset((page - 1) * limit).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func setRevenueShareRuleEnabled(ctx context.Context, scope model.AccessScope, id int, enabled bool) error {
	if id <= 0 {
		return errors.Join(ErrRevenueShareRuleNotFound, errors.New("id is required"))
	}
	result := model.ApplyOwnershipScope(model.DB.WithContext(ctx).Model(&model.RevenueShareRule{}), "revenue_share_rules", scope).
		Where("id = ?", id).
		Update("enabled", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRevenueShareRuleNotFound
	}
	return nil
}

func revenueShareRuleFromInput(scope model.AccessScope, input RevenueShareRuleInput, existing *model.RevenueShareRule) model.RevenueShareRule {
	rule := model.RevenueShareRule{}
	if existing != nil {
		rule = *existing
	}
	if scope.IsRoot {
		rule.TenantId = input.TenantId
	} else {
		rule.TenantId = scope.TenantId
	}
	rule.DistributionChannelId = input.DistributionChannelId
	rule.RuleName = strings.TrimSpace(input.RuleName)
	rule.RuleScope = model.NormalizeRevenueShareRuleScope(input.RuleScope)
	rule.ProviderName = strings.TrimSpace(input.ProviderName)
	rule.ModelName = strings.TrimSpace(input.ModelName)
	rule.ProductType = strings.TrimSpace(input.ProductType)
	rule.PlatformShareRate = input.PlatformShareRate
	rule.MasterDistributorShareRate = input.MasterDistributorShareRate
	rule.DistributorShareRate = input.DistributorShareRate
	rule.EffectiveFrom = input.EffectiveFrom
	rule.EffectiveTo = input.EffectiveTo
	rule.Enabled = input.Enabled
	if existing == nil && rule.RuleName == "" {
		rule.RuleName = fmt.Sprintf("%s revenue share", rule.RuleScope)
	}
	if rule.TenantId == 0 {
		rule.TenantId = 1
	}
	return rule
}

func validateRevenueShareRule(scope model.AccessScope, rule model.RevenueShareRule) error {
	if !model.AllowsOwnership(scope, rule.TenantId, 0, 0) {
		return ErrRevenueShareTenantScopeDenied
	}
	if strings.TrimSpace(rule.RuleName) == "" {
		return errors.Join(ErrRevenueShareInvalidInput, errors.New("rule_name is required"))
	}
	if rule.PlatformShareRate < 0 || rule.MasterDistributorShareRate < 0 || rule.DistributorShareRate < 0 {
		return errors.Join(ErrRevenueShareInvalidInput, errors.New("share rates cannot be negative"))
	}
	if math.Abs(rule.PlatformShareRate+rule.MasterDistributorShareRate+rule.DistributorShareRate-100) > 0.000001 {
		return errors.Join(ErrRevenueShareInvalidInput, errors.New("share rates must sum to 100"))
	}
	if rule.EffectiveTo > 0 && rule.EffectiveFrom > 0 && rule.EffectiveTo < rule.EffectiveFrom {
		return errors.Join(ErrRevenueShareInvalidInput, errors.New("effective_to must be greater than effective_from"))
	}
	switch model.NormalizeRevenueShareRuleScope(rule.RuleScope) {
	case model.RevenueShareRuleScopeModel:
		if strings.TrimSpace(rule.ModelName) == "" {
			return errors.Join(ErrRevenueShareInvalidInput, errors.New("model_name is required for model scope"))
		}
	case model.RevenueShareRuleScopeProvider:
		if strings.TrimSpace(rule.ProviderName) == "" {
			return errors.Join(ErrRevenueShareInvalidInput, errors.New("provider_name is required for provider scope"))
		}
	case model.RevenueShareRuleScopeSubscription, model.RevenueShareRuleScopeAgent, model.RevenueShareRuleScopeSkill, model.RevenueShareRuleScopeVideo:
		if strings.TrimSpace(rule.ProductType) == "" {
			return errors.Join(ErrRevenueShareInvalidInput, errors.New("product_type is required for product scope"))
		}
	}
	return nil
}

func matchRevenueShareRule(ctx context.Context, input RevenueShareCalculationInput) (model.RevenueShareRule, bool, error) {
	priorities := []struct {
		scope string
		field string
		value string
	}{
		{model.RevenueShareRuleScopeModel, "model_name", input.ModelName},
		{model.RevenueShareRuleScopeProvider, "provider_name", input.ProviderName},
		{model.NormalizeRevenueShareRuleScope(input.ProductType), "product_type", input.ProductType},
		{model.RevenueShareRuleScopeGlobal, "", ""},
	}
	now := input.Now
	if now == 0 {
		now = modelTimestamp()
	}
	for _, priority := range priorities {
		if priority.scope == "" {
			continue
		}
		if priority.scope != model.RevenueShareRuleScopeGlobal && strings.TrimSpace(priority.value) == "" {
			continue
		}
		query := model.DB.WithContext(ctx).
			Where("tenant_id = ? AND enabled = ? AND rule_scope = ?", input.TenantId, true, priority.scope).
			Where("distribution_channel_id IN ?", []int{0, input.DistributionChannelId}).
			Where("(effective_from = 0 OR effective_from <= ?) AND (effective_to = 0 OR effective_to >= ?)", now, now)
		if priority.field != "" {
			query = query.Where(priority.field+" = ?", strings.TrimSpace(priority.value))
		}
		var rule model.RevenueShareRule
		if err := query.Order("distribution_channel_id desc, id desc").First(&rule).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return model.RevenueShareRule{}, false, err
		}
		return rule, true, nil
	}
	return model.RevenueShareRule{}, false, nil
}

func resolveRevenueShareChannelHierarchy(ctx context.Context, tenantId int, channelId int) (int, int, error) {
	if channelId <= 0 {
		return 0, 0, nil
	}
	var channel model.DistributionChannel
	if err := model.DB.WithContext(ctx).Where("id = ? AND tenant_id = ?", channelId, tenantId).First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, ErrRevenueShareTenantScopeDenied
		}
		return 0, 0, err
	}
	if channel.ParentId <= 0 {
		return channel.Id, 0, nil
	}
	var parent model.DistributionChannel
	if err := model.DB.WithContext(ctx).Where("id = ? AND tenant_id = ?", channel.ParentId, tenantId).First(&parent).Error; err != nil {
		return 0, 0, err
	}
	if parent.ParentId > 0 {
		return 0, 0, ErrRevenueShareChannelLevelInvalid
	}
	return parent.Id, channel.Id, nil
}

func validateRevenueShareChannel(ctx context.Context, tenantId int, channelId int) error {
	_, _, err := resolveRevenueShareChannelHierarchy(ctx, tenantId, channelId)
	return err
}

func normalizeRevenueSharePage(page int, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func roundRevenueAmount(value float64) float64 {
	return math.Round(value*1000000) / 1000000
}

func modelTimestamp() int64 {
	return common.GetTimestamp()
}
