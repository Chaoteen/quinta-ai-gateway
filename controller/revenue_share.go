package controller

import (
	"strconv"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/Chaoteen/quinta-ai-gateway/service"
	"github.com/gin-gonic/gin"
)

type revenueShareRuleRequest struct {
	TenantId                   int     `json:"tenant_id"`
	DistributionChannelId      int     `json:"distribution_channel_id"`
	RuleName                   string  `json:"rule_name"`
	RuleScope                  string  `json:"rule_scope"`
	ProviderName               string  `json:"provider_name"`
	ModelName                  string  `json:"model_name"`
	ProductType                string  `json:"product_type"`
	PlatformShareRate          float64 `json:"platform_share_rate"`
	MasterDistributorShareRate float64 `json:"master_distributor_share_rate"`
	DistributorShareRate       float64 `json:"distributor_share_rate"`
	EffectiveFrom              int64   `json:"effective_from"`
	EffectiveTo                int64   `json:"effective_to"`
	Enabled                    bool    `json:"enabled"`
}

func CreateRevenueShareRule(c *gin.Context) {
	var req revenueShareRuleRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	rule, err := service.CreateRevenueShareRule(c.Request.Context(), model.AccessScopeFromContext(c), revenueShareRuleInput(req))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"rule": rule})
}

func ListRevenueShareRules(c *gin.Context) {
	enabled, hasEnabled := parseOptionalBool(c.Query("enabled"))
	input := service.RevenueShareRuleListInput{
		TenantId:              queryInt(c, "tenant_id"),
		DistributionChannelId: queryInt(c, "distribution_channel_id"),
		RuleScope:             c.Query("rule_scope"),
		Page:                  queryInt(c, "page"),
		Limit:                 queryInt(c, "limit"),
	}
	if hasEnabled {
		input.Enabled = &enabled
	}
	rules, total, err := service.ListRevenueShareRules(c.Request.Context(), model.AccessScopeFromContext(c), input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"rules": rules, "total": total})
}

func UpdateRevenueShareRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid id")
		return
	}
	var req revenueShareRuleRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	rule, err := service.UpdateRevenueShareRule(c.Request.Context(), model.AccessScopeFromContext(c), id, revenueShareRuleInput(req))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"rule": rule})
}

func EnableRevenueShareRule(c *gin.Context) {
	updateRevenueShareRuleEnabled(c, true)
}

func DisableRevenueShareRule(c *gin.Context) {
	updateRevenueShareRuleEnabled(c, false)
}

func ListRevenueShareRecords(c *gin.Context) {
	records, total, err := service.ListRevenueShareRecords(c.Request.Context(), model.AccessScopeFromContext(c), service.RevenueShareRecordListInput{
		TenantId:              queryInt(c, "tenant_id"),
		DistributionChannelId: queryInt(c, "distribution_channel_id"),
		Status:                c.Query("status"),
		SourceType:            c.Query("source_type"),
		StartTime:             queryInt64(c, "start_time"),
		EndTime:               queryInt64(c, "end_time"),
		Page:                  queryInt(c, "page"),
		Limit:                 queryInt(c, "limit"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"records": records, "total": total})
}

func updateRevenueShareRuleEnabled(c *gin.Context, enabled bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid id")
		return
	}
	if enabled {
		err = service.EnableRevenueShareRule(c.Request.Context(), model.AccessScopeFromContext(c), id)
	} else {
		err = service.DisableRevenueShareRule(c.Request.Context(), model.AccessScopeFromContext(c), id)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": id, "enabled": enabled})
}

func revenueShareRuleInput(req revenueShareRuleRequest) service.RevenueShareRuleInput {
	return service.RevenueShareRuleInput{
		TenantId:                   req.TenantId,
		DistributionChannelId:      req.DistributionChannelId,
		RuleName:                   req.RuleName,
		RuleScope:                  req.RuleScope,
		ProviderName:               req.ProviderName,
		ModelName:                  req.ModelName,
		ProductType:                req.ProductType,
		PlatformShareRate:          req.PlatformShareRate,
		MasterDistributorShareRate: req.MasterDistributorShareRate,
		DistributorShareRate:       req.DistributorShareRate,
		EffectiveFrom:              req.EffectiveFrom,
		EffectiveTo:                req.EffectiveTo,
		Enabled:                    req.Enabled,
	}
}

func queryInt(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

func queryInt64(c *gin.Context, key string) int64 {
	value, _ := strconv.ParseInt(c.Query(key), 10, 64)
	return value
}

func parseOptionalBool(value string) (bool, bool) {
	if value == "" {
		return false, false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false
	}
	return parsed, true
}
