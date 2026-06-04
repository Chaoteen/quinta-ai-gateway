package controller

import (
	"strconv"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/Chaoteen/quinta-ai-gateway/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

// ---- Shared types ----

type SubscriptionPlanDTO struct {
	Plan AlphaSubscriptionPlanDTO `json:"plan"`
}

type PublicSubscriptionPlanDTO struct {
	Plan PublicAlphaSubscriptionPlanDTO `json:"plan"`
}

type AlphaSubscriptionPlanDTO struct {
	Id           int     `json:"id"`
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	MonthlyPrice float64 `json:"monthly_price"`
	YearlyPrice  float64 `json:"yearly_price"`
	TokenQuota   int64   `json:"token_quota"`
	RequestQuota int64   `json:"request_quota"`
	ModelQuota   string  `json:"model_quota"`
	Status       string  `json:"status"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

type PublicAlphaSubscriptionPlanDTO struct {
	AlphaSubscriptionPlanDTO
	Title                   string  `json:"title"`
	Subtitle                string  `json:"subtitle"`
	PriceAmount             float64 `json:"price_amount"`
	Currency                string  `json:"currency"`
	DurationUnit            string  `json:"duration_unit"`
	DurationValue           int     `json:"duration_value"`
	CustomSeconds           int64   `json:"custom_seconds"`
	Enabled                 bool    `json:"enabled"`
	MaxPurchasePerUser      int     `json:"max_purchase_per_user"`
	UpgradeGroup            string  `json:"upgrade_group"`
	TotalAmount             int64   `json:"total_amount"`
	QuotaResetPeriod        string  `json:"quota_reset_period"`
	QuotaResetCustomSeconds int64   `json:"quota_reset_custom_seconds"`
	StripePriceId           string  `json:"stripe_price_id,omitempty"`
	CreemProductId          string  `json:"creem_product_id,omitempty"`
}

type UserSubscriptionDTO struct {
	Id                    int    `json:"id"`
	TenantId              int    `json:"tenant_id,omitempty"`
	OrganizationId        int    `json:"organization_id,omitempty"`
	DepartmentId          int    `json:"department_id,omitempty"`
	DistributionChannelId int    `json:"distribution_channel_id,omitempty"`
	UserId                int    `json:"user_id"`
	PlanId                int    `json:"plan_id"`
	PlanCode              string `json:"plan_code"`
	PlanName              string `json:"plan_name"`
	LifecycleStatus       string `json:"lifecycle_status"`
	StartTime             int64  `json:"start_time"`
	EndTime               int64  `json:"end_time"`
	TokenQuotaSnapshot    int64  `json:"token_quota_snapshot"`
	RequestQuotaSnapshot  int64  `json:"request_quota_snapshot"`
	ModelQuotaSnapshot    string `json:"model_quota_snapshot"`
	NextResetTime         int64  `json:"next_reset_time"`
	CreatedAt             int64  `json:"created_at"`
	UpdatedAt             int64  `json:"updated_at"`
}

type UserSubscriptionRecordDTO struct {
	Subscription UserSubscriptionDTO `json:"subscription"`
}

type SelfSubscriptionDTO struct {
	PlanCode             string `json:"plan_code"`
	PlanName             string `json:"plan_name"`
	LifecycleStatus      string `json:"lifecycle_status"`
	StartTime            int64  `json:"start_time"`
	EndTime              int64  `json:"end_time"`
	TokenQuotaSnapshot   int64  `json:"token_quota_snapshot"`
	RequestQuotaSnapshot int64  `json:"request_quota_snapshot"`
	ModelQuotaSnapshot   string `json:"model_quota_snapshot"`
	NextResetTime        int64  `json:"next_reset_time"`
	TokenQuota           int64  `json:"token_quota"`
	TokenUsed            int64  `json:"token_used"`
	TokenRemaining       int64  `json:"token_remaining"`
}

type SelfSubscriptionRecordDTO struct {
	Subscription SelfSubscriptionDTO `json:"subscription"`
}

type BillingPreferenceRequest struct {
	BillingPreference string `json:"billing_preference"`
}

func subscriptionPlanToDTO(plan model.SubscriptionPlan) AlphaSubscriptionPlanDTO {
	plan.NormalizeAlphaFields()
	return AlphaSubscriptionPlanDTO{
		Id:           plan.Id,
		Code:         plan.Code,
		Name:         plan.Name,
		Description:  plan.Description,
		MonthlyPrice: plan.MonthlyPrice,
		YearlyPrice:  plan.YearlyPrice,
		TokenQuota:   plan.TokenQuota,
		RequestQuota: plan.RequestQuota,
		ModelQuota:   plan.ModelQuota,
		Status:       plan.Status,
		CreatedAt:    plan.CreatedAt,
		UpdatedAt:    plan.UpdatedAt,
	}
}

func subscriptionPlansToDTO(plans []model.SubscriptionPlan) []SubscriptionPlanDTO {
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		result = append(result, SubscriptionPlanDTO{Plan: subscriptionPlanToDTO(p)})
	}
	return result
}

func publicSubscriptionPlansToDTO(plans []model.SubscriptionPlan) []PublicSubscriptionPlanDTO {
	result := make([]PublicSubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		result = append(result, PublicSubscriptionPlanDTO{
			Plan: PublicAlphaSubscriptionPlanDTO{
				AlphaSubscriptionPlanDTO: subscriptionPlanToDTO(p),
				Title:                    p.Title,
				Subtitle:                 p.Subtitle,
				PriceAmount:              p.PriceAmount,
				Currency:                 p.Currency,
				DurationUnit:             p.DurationUnit,
				DurationValue:            p.DurationValue,
				CustomSeconds:            p.CustomSeconds,
				Enabled:                  p.Enabled,
				MaxPurchasePerUser:       p.MaxPurchasePerUser,
				UpgradeGroup:             p.UpgradeGroup,
				TotalAmount:              p.TotalAmount,
				QuotaResetPeriod:         p.QuotaResetPeriod,
				QuotaResetCustomSeconds:  p.QuotaResetCustomSeconds,
				StripePriceId:            p.StripePriceId,
				CreemProductId:           p.CreemProductId,
			},
		})
	}
	return result
}

func applyPlanDTOToModel(dto AlphaSubscriptionPlanDTO, plan *model.SubscriptionPlan) {
	plan.Code = strings.TrimSpace(dto.Code)
	plan.Name = strings.TrimSpace(dto.Name)
	plan.Description = strings.TrimSpace(dto.Description)
	plan.MonthlyPrice = dto.MonthlyPrice
	plan.YearlyPrice = dto.YearlyPrice
	plan.TokenQuota = dto.TokenQuota
	plan.RequestQuota = dto.RequestQuota
	plan.ModelQuota = strings.TrimSpace(dto.ModelQuota)
	plan.Status = model.NormalizeSubscriptionPlanStatus(dto.Status, dto.Status == model.SubscriptionPlanStatusEnabled)
	plan.Enabled = plan.Status == model.SubscriptionPlanStatusEnabled
	if plan.Title == "" {
		plan.Title = plan.Name
	}
	if plan.Subtitle == "" {
		plan.Subtitle = plan.Description
	}
	plan.PriceAmount = dto.MonthlyPrice
	plan.Currency = "USD"
	plan.TotalAmount = dto.TokenQuota
	plan.NormalizeAlphaFields()
}

func validateAlphaSubscriptionPlan(plan *model.SubscriptionPlan) string {
	if strings.TrimSpace(plan.Code) == "" {
		return "套餐编码不能为空"
	}
	if strings.TrimSpace(plan.Name) == "" {
		return "套餐名称不能为空"
	}
	if plan.MonthlyPrice < 0 || plan.YearlyPrice < 0 || plan.PriceAmount < 0 {
		return "价格不能为负数"
	}
	if plan.MonthlyPrice > 9999 || plan.YearlyPrice > 99999 || plan.PriceAmount > 9999 {
		return "价格超过允许范围"
	}
	if plan.MaxPurchasePerUser < 0 {
		return "购买上限不能为负数"
	}
	if plan.TokenQuota < 0 || plan.RequestQuota < 0 || plan.TotalAmount < 0 {
		return "额度不能为负数"
	}
	if plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[plan.UpgradeGroup]; !ok {
			return "升级分组不存在"
		}
	}
	if plan.QuotaResetPeriod == model.SubscriptionResetCustom && plan.QuotaResetCustomSeconds <= 0 {
		return "自定义重置周期需大于0秒"
	}
	return ""
}

func subscriptionToDTO(sub model.UserSubscription, plans map[int]model.SubscriptionPlan, includeOwnership bool) UserSubscriptionRecordDTO {
	dto := UserSubscriptionDTO{
		Id:                   sub.Id,
		UserId:               sub.UserId,
		PlanId:               sub.PlanId,
		LifecycleStatus:      model.NormalizeSubscriptionLifecycle(sub.LifecycleStatus),
		StartTime:            sub.StartTime,
		EndTime:              sub.EndTime,
		TokenQuotaSnapshot:   sub.TokenQuotaSnapshot,
		RequestQuotaSnapshot: sub.RequestQuotaSnapshot,
		ModelQuotaSnapshot:   sub.ModelQuotaSnapshot,
		NextResetTime:        sub.NextResetTime,
		CreatedAt:            sub.CreatedAt,
		UpdatedAt:            sub.UpdatedAt,
	}
	if includeOwnership {
		dto.TenantId = sub.TenantId
		dto.OrganizationId = sub.OrganizationId
		dto.DepartmentId = sub.DepartmentId
		dto.DistributionChannelId = sub.DistributionChannelId
	}
	if plan, ok := plans[sub.PlanId]; ok {
		plan.NormalizeAlphaFields()
		dto.PlanCode = plan.Code
		dto.PlanName = plan.Name
	}
	return UserSubscriptionRecordDTO{Subscription: dto}
}

func subscriptionToSelfDTO(sub model.UserSubscription, plans map[int]model.SubscriptionPlan) SelfSubscriptionRecordDTO {
	tokenQuota := sub.TokenQuotaSnapshot
	if tokenQuota == 0 {
		tokenQuota = sub.AmountTotal
	}
	tokenUsed := sub.AmountUsed
	tokenRemaining := int64(0)
	if tokenQuota > 0 {
		tokenRemaining = tokenQuota - tokenUsed
		if tokenRemaining < 0 {
			tokenRemaining = 0
		}
	}
	dto := SelfSubscriptionDTO{
		LifecycleStatus:      model.NormalizeSubscriptionLifecycle(sub.LifecycleStatus),
		StartTime:            sub.StartTime,
		EndTime:              sub.EndTime,
		TokenQuotaSnapshot:   sub.TokenQuotaSnapshot,
		RequestQuotaSnapshot: sub.RequestQuotaSnapshot,
		ModelQuotaSnapshot:   sub.ModelQuotaSnapshot,
		NextResetTime:        sub.NextResetTime,
		TokenQuota:           tokenQuota,
		TokenUsed:            tokenUsed,
		TokenRemaining:       tokenRemaining,
	}
	if plan, ok := plans[sub.PlanId]; ok {
		plan.NormalizeAlphaFields()
		dto.PlanCode = plan.Code
		dto.PlanName = plan.Name
	}
	return SelfSubscriptionRecordDTO{Subscription: dto}
}

func buildUserSubscriptionDTOs(subs []model.UserSubscription, includeOwnership bool) []UserSubscriptionRecordDTO {
	plans := loadSubscriptionPlansForSubs(subs)
	result := make([]UserSubscriptionRecordDTO, 0, len(subs))
	for _, sub := range subs {
		result = append(result, subscriptionToDTO(sub, plans, includeOwnership))
	}
	return result
}

func buildSelfSubscriptionDTOs(summaries []model.SubscriptionSummary) []SelfSubscriptionRecordDTO {
	subs := make([]model.UserSubscription, 0, len(summaries))
	for _, summary := range summaries {
		if summary.Subscription == nil {
			continue
		}
		subs = append(subs, *summary.Subscription)
	}
	plans := loadSubscriptionPlansForSubs(subs)
	result := make([]SelfSubscriptionRecordDTO, 0, len(subs))
	for _, sub := range subs {
		result = append(result, subscriptionToSelfDTO(sub, plans))
	}
	return result
}

func loadSubscriptionPlansForSubs(subs []model.UserSubscription) map[int]model.SubscriptionPlan {
	planIds := make([]int, 0)
	seen := map[int]struct{}{}
	for _, sub := range subs {
		if sub.PlanId > 0 {
			if _, ok := seen[sub.PlanId]; !ok {
				seen[sub.PlanId] = struct{}{}
				planIds = append(planIds, sub.PlanId)
			}
		}
	}
	plans := map[int]model.SubscriptionPlan{}
	if len(planIds) > 0 {
		var planRows []model.SubscriptionPlan
		if err := model.DB.Where("id IN ?", planIds).Find(&planRows).Error; err == nil {
			for _, plan := range planRows {
				plans[plan.Id] = plan
			}
		}
	}
	return plans
}

func parseAdminPagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit
}

// ---- User APIs ----

func GetSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Where("enabled = ?", true).Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, publicSubscriptionPlansToDTO(plans))
}

func GetSubscriptionSelf(c *gin.Context) {
	userId := c.GetInt("id")
	settingMap, _ := model.GetUserSetting(userId, false)
	pref := common.NormalizeBillingPreference(settingMap.BillingPreference)

	// Get all subscriptions (including expired)
	allSubscriptions, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		allSubscriptions = []model.SubscriptionSummary{}
	}

	// Get active subscriptions for backward compatibility
	activeSubscriptions, err := model.GetAllActiveUserSubscriptions(userId)
	if err != nil {
		activeSubscriptions = []model.SubscriptionSummary{}
	}

	common.ApiSuccess(c, gin.H{
		"billing_preference": pref,
		"subscriptions":      buildSelfSubscriptionDTOs(activeSubscriptions),
		"all_subscriptions":  buildSelfSubscriptionDTOs(allSubscriptions),
	})
}

func UpdateSubscriptionPreference(c *gin.Context) {
	userId := c.GetInt("id")
	var req BillingPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	pref := common.NormalizeBillingPreference(req.BillingPreference)

	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	current := user.GetSetting()
	current.BillingPreference = pref
	user.SetSetting(current)
	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"billing_preference": pref})
}

// ---- Admin APIs ----

func AdminListSubscriptionPlans(c *gin.Context) {
	page, limit := parseAdminPagination(c)
	q := strings.TrimSpace(c.Query("q"))
	status := strings.TrimSpace(c.Query("status"))
	var plans []model.SubscriptionPlan
	query := model.DB.Model(&model.SubscriptionPlan{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("code LIKE ? OR name LIKE ? OR title LIKE ?", like, like, like)
	}
	if status != "" {
		query = query.Where("status = ?", model.NormalizeSubscriptionPlanStatus(status, status == model.SubscriptionPlanStatusEnabled))
	}
	if err := query.Order("sort_order desc, id desc").Limit(limit).Offset((page - 1) * limit).Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, subscriptionPlansToDTO(plans))
}

type AdminUpsertSubscriptionPlanRequest struct {
	Plan AlphaSubscriptionPlanDTO `json:"plan"`
}

func AdminCreateSubscriptionPlan(c *gin.Context) {
	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	plan := model.SubscriptionPlan{Enabled: true, Status: model.SubscriptionPlanStatusEnabled}
	applyPlanDTOToModel(req.Plan, &plan)
	if msg := validateAlphaSubscriptionPlan(&plan); msg != "" {
		common.ApiErrorMsg(c, msg)
		return
	}
	if err := model.EnsureSubscriptionPlanCodeAvailable(model.DB, plan.Code, 0); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DB.Create(&plan).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(plan.Id)
	common.ApiSuccess(c, SubscriptionPlanDTO{Plan: subscriptionPlanToDTO(plan)})
}

func AdminUpdateSubscriptionPlan(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	var plan model.SubscriptionPlan
	if err := model.DB.Where("id = ?", id).First(&plan).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	applyPlanDTOToModel(req.Plan, &plan)
	if msg := validateAlphaSubscriptionPlan(&plan); msg != "" {
		common.ApiErrorMsg(c, msg)
		return
	}
	if err := model.EnsureSubscriptionPlanCodeAvailable(model.DB, plan.Code, id); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DB.Save(&plan).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, SubscriptionPlanDTO{Plan: subscriptionPlanToDTO(plan)})
}

type AdminUpdateSubscriptionPlanStatusRequest struct {
	Enabled *bool  `json:"enabled"`
	Status  string `json:"status"`
}

func AdminUpdateSubscriptionPlanStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpdateSubscriptionPlanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		if err != nil || strings.TrimSpace(req.Status) == "" {
			common.ApiErrorMsg(c, "参数错误")
			return
		}
	}
	status := model.NormalizeSubscriptionPlanStatus(req.Status, req.Enabled != nil && *req.Enabled)
	enabled := status == model.SubscriptionPlanStatusEnabled
	if err := model.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Updates(map[string]interface{}{
		"enabled":    enabled,
		"status":     status,
		"updated_at": common.GetTimestamp(),
	}).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminBindSubscriptionRequest struct {
	UserId int `json:"user_id"`
	PlanId int `json:"plan_id"`
}

func AdminBindSubscription(c *gin.Context) {
	var req AdminBindSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.UserId <= 0 || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	scope := model.TenantScopeFromContext(c)
	if !ensureAdminTargetUserInTenant(c, req.UserId, scope) {
		return
	}
	msg, err := model.AdminBindSubscription(req.UserId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin: user subscription management ----

func ensureAdminTargetUserInTenant(c *gin.Context, userId int, scope model.TenantScope) bool {
	if scope.IsRoot {
		return true
	}
	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiErrorMsg(c, "用户不存在或无权访问")
		return false
	}
	if !scope.AllowsTenant(user.TenantId) {
		common.ApiErrorMsg(c, "用户不存在或无权访问")
		return false
	}
	return true
}

func ensureAdminTargetUserInAccessScope(c *gin.Context, userId int, scope model.AccessScope) bool {
	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiErrorMsg(c, "用户不存在或无权访问")
		return false
	}
	if !model.AllowsOwnership(scope, user.TenantId, user.OrganizationId, user.DepartmentId) {
		common.ApiErrorMsg(c, "用户不存在或无权访问")
		return false
	}
	return true
}

func ensureAdminSubscriptionInAccessScope(c *gin.Context, subId int, scope model.AccessScope) bool {
	if scope.IsRoot {
		return true
	}
	var sub model.UserSubscription
	if err := model.DB.Where("id = ?", subId).First(&sub).Error; err != nil {
		common.ApiErrorMsg(c, "用户不存在或无权访问")
		return false
	}
	if !model.AllowsOwnership(scope, sub.TenantId, sub.OrganizationId, sub.DepartmentId) {
		common.ApiErrorMsg(c, "用户不存在或无权访问")
		return false
	}
	return true
}

func ensureTenantAdminWriteScope(c *gin.Context) (model.AccessScope, bool) {
	scope := model.AccessScopeFromContext(c)
	if scope.IsRoot {
		return scope, true
	}
	if common.GetContextKeyString(c, constant.ContextKeyUserRoleKey) != common.RoleKeyTenantAdmin {
		common.ApiErrorMsg(c, "权限不足")
		return scope, false
	}
	return scope, true
}

func AdminListAllUserSubscriptions(c *gin.Context) {
	page, limit := parseAdminPagination(c)
	status := strings.TrimSpace(c.Query("status"))
	userId, _ := strconv.Atoi(c.Query("user_id"))
	scope, ok := operationalReadAccessScope(c)
	if !ok {
		return
	}
	query := model.DB.Model(&model.UserSubscription{})
	query = model.ApplyOwnershipScope(query, "user_subscriptions", scope)
	if userId > 0 {
		if !ensureAdminTargetUserInAccessScope(c, userId, scope) {
			return
		}
		query = query.Where("user_id = ?", userId)
	}
	if status != "" {
		query = query.Where("lifecycle_status = ? OR status = ?", model.NormalizeSubscriptionLifecycle(status), model.NormalizeSubscriptionLifecycle(status))
	}
	var subs []model.UserSubscription
	if err := query.Order("id desc").Limit(limit).Offset((page - 1) * limit).Find(&subs).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildUserSubscriptionDTOs(subs, scope.IsRoot))
}

func AdminListUserSubscriptions(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	scope, ok := operationalReadAccessScope(c)
	if !ok {
		return
	}
	if !ensureAdminTargetUserInAccessScope(c, userId, scope) {
		return
	}
	var subs []model.UserSubscription
	query := model.DB.Where("user_id = ?", userId)
	query = model.ApplyOwnershipScope(query, "user_subscriptions", scope)
	if err := query.Order("end_time desc, id desc").Find(&subs).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildUserSubscriptionDTOs(subs, scope.IsRoot))
}

type AdminCreateUserSubscriptionRequest struct {
	PlanId int `json:"plan_id"`
}

// AdminCreateUserSubscription creates a new user subscription from a plan (no payment).
func AdminCreateUserSubscription(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	var req AdminCreateUserSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	scope, ok := ensureTenantAdminWriteScope(c)
	if !ok {
		return
	}
	if !ensureAdminTargetUserInAccessScope(c, userId, scope) {
		return
	}
	msg, err := model.AdminBindSubscription(userId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminCancelUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	scope, ok := ensureTenantAdminWriteScope(c)
	if !ok {
		return
	}
	if !ensureAdminSubscriptionInAccessScope(c, subId, scope) {
		return
	}
	msg, err := model.AdminInvalidateUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminSuspendUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	scope, ok := ensureTenantAdminWriteScope(c)
	if !ok {
		return
	}
	if !ensureAdminSubscriptionInAccessScope(c, subId, scope) {
		return
	}
	msg, err := model.AdminSuspendUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminRenewUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	scope, ok := ensureTenantAdminWriteScope(c)
	if !ok {
		return
	}
	if !ensureAdminSubscriptionInAccessScope(c, subId, scope) {
		return
	}
	sub, err := model.AdminRenewUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildUserSubscriptionDTOs([]model.UserSubscription{*sub}, scope.IsRoot)[0])
}

// AdminInvalidateUserSubscription is retained for the legacy route.
func AdminInvalidateUserSubscription(c *gin.Context) {
	AdminCancelUserSubscription(c)
}

// AdminDeleteUserSubscription is retained for the legacy route, but it no
// longer hard-deletes business subscription records.
func AdminDeleteUserSubscription(c *gin.Context) {
	AdminCancelUserSubscription(c)
}
