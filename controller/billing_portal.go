package controller

import (
	"strconv"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/Chaoteen/quinta-ai-gateway/service"
	"github.com/gin-gonic/gin"
)

func GetBillingPortalSummary(c *gin.Context) {
	summary, err := service.NewBillingPortalService().Summary(c.Request.Context(), billingPortalActorFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func GetBillingPortalPayments(c *gin.Context) {
	page, err := service.NewBillingPortalService().PaymentHistory(c.Request.Context(), billingPortalActorFromContext(c), billingPortalListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetBillingPortalUsages(c *gin.Context) {
	page, err := service.NewBillingPortalService().UsageHistory(c.Request.Context(), billingPortalActorFromContext(c), billingPortalListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetBillingPortalRecords(c *gin.Context) {
	page, err := service.NewBillingPortalService().BillingHistory(c.Request.Context(), billingPortalActorFromContext(c), billingPortalListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetBillingPortalSubscriptions(c *gin.Context) {
	page, err := service.NewBillingPortalService().SubscriptionHistory(c.Request.Context(), billingPortalActorFromContext(c), billingPortalListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func billingPortalActorFromContext(c *gin.Context) service.BillingPortalActor {
	return service.BillingPortalActor{
		UserId: c.GetInt("id"),
		Scope:  model.AccessScopeFromContext(c),
	}
}

func billingPortalListInputFromContext(c *gin.Context) service.BillingPortalListInput {
	pageInfo := common.GetPageQuery(c)
	startTime, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)
	tenantId, _ := strconv.Atoi(c.Query("tenant_id"))
	return service.BillingPortalListInput{
		Page:          pageInfo.GetPage(),
		PageSize:      pageInfo.GetPageSize(),
		StartTime:     startTime,
		EndTime:       endTime,
		Status:        c.Query("status"),
		Provider:      c.Query("provider"),
		Model:         c.Query("model"),
		PaymentMethod: c.Query("payment_method"),
		TenantId:      tenantId,
		Subscription:  c.Query("subscription"),
	}
}
