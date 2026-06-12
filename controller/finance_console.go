package controller

import (
	"strconv"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/Chaoteen/quinta-ai-gateway/service"
	"github.com/gin-gonic/gin"
)

func GetFinanceSummary(c *gin.Context) {
	summary, err := service.NewFinanceConsoleService().Summary(c.Request.Context(), financeConsoleActorFromContext(c), financeConsoleListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func GetFinanceTopTenants(c *gin.Context) {
	page, err := service.NewFinanceConsoleService().TopTenants(c.Request.Context(), financeConsoleActorFromContext(c), financeConsoleListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetFinanceTopModels(c *gin.Context) {
	page, err := service.NewFinanceConsoleService().TopModels(c.Request.Context(), financeConsoleActorFromContext(c), financeConsoleListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetFinanceTopProviders(c *gin.Context) {
	page, err := service.NewFinanceConsoleService().TopProviders(c.Request.Context(), financeConsoleActorFromContext(c), financeConsoleListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetFinanceTopChannels(c *gin.Context) {
	page, err := service.NewFinanceConsoleService().TopChannels(c.Request.Context(), financeConsoleActorFromContext(c), financeConsoleListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetFinanceRecentPayments(c *gin.Context) {
	page, err := service.NewFinanceConsoleService().RecentPayments(c.Request.Context(), financeConsoleActorFromContext(c), financeConsoleListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetFinanceRecentRedemptions(c *gin.Context) {
	page, err := service.NewFinanceConsoleService().RecentRedemptions(c.Request.Context(), financeConsoleActorFromContext(c), financeConsoleListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetFinanceRecentSubscriptions(c *gin.Context) {
	page, err := service.NewFinanceConsoleService().RecentSubscriptions(c.Request.Context(), financeConsoleActorFromContext(c), financeConsoleListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetFinanceRecentBilling(c *gin.Context) {
	page, err := service.NewFinanceConsoleService().RecentBilling(c.Request.Context(), financeConsoleActorFromContext(c), financeConsoleListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func financeConsoleActorFromContext(c *gin.Context) service.FinanceConsoleActor {
	return service.FinanceConsoleActor{
		UserId: c.GetInt("id"),
		Scope:  model.AccessScopeFromContext(c),
	}
}

func financeConsoleListInputFromContext(c *gin.Context) service.FinanceConsoleListInput {
	pageInfo := common.GetPageQuery(c)
	days, _ := strconv.Atoi(c.Query("days"))
	return service.FinanceConsoleListInput{
		Page:     pageInfo.GetPage(),
		PageSize: pageInfo.GetPageSize(),
		Days:     days,
	}
}
