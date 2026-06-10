package controller

import (
	"strconv"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/Chaoteen/quinta-ai-gateway/service"
	"github.com/gin-gonic/gin"
)

type redeemVoucherRequest struct {
	VoucherCode string `json:"voucher_code"`
}

type createVoucherBatchRequest struct {
	Name                  string `json:"name"`
	Description           string `json:"description"`
	VoucherType           string `json:"voucher_type"`
	Quantity              int    `json:"quantity"`
	Status                string `json:"status"`
	TenantId              int    `json:"tenant_id"`
	OrganizationId        int    `json:"organization_id"`
	DepartmentId          int    `json:"department_id"`
	DistributionChannelId int    `json:"distribution_channel_id"`
}

type generateVouchersRequest struct {
	Quantity           int      `json:"quantity"`
	QuotaAmount        int64    `json:"quota_amount"`
	SubscriptionPlanId int      `json:"subscription_plan_id"`
	ExpiredAt          int64    `json:"expired_at"`
	Codes              []string `json:"codes"`
}

func RedeemVoucher(c *gin.Context) {
	var req redeemVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	redemption, err := service.NewVoucherService().RedeemVoucher(c.Request.Context(), c.GetInt("id"), req.VoucherCode)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, redemption)
}

func ListVoucherHistory(c *gin.Context) {
	actor := voucherActorFromContext(c)
	actor.Scope.RoleKey = common.RoleKeyUser
	actor.Scope.IsRoot = false
	page, err := service.NewVoucherService().ListRedemptions(c.Request.Context(), actor, voucherListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func AdminCreateVoucherBatch(c *gin.Context) {
	var req createVoucherBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	actor := voucherActorFromContext(c)
	batch, err := service.NewVoucherService().CreateVoucherBatch(c.Request.Context(), actor, service.CreateVoucherBatchInput{
		Name:                  req.Name,
		Description:           req.Description,
		VoucherType:           req.VoucherType,
		Quantity:              req.Quantity,
		Status:                req.Status,
		TenantId:              req.TenantId,
		OrganizationId:        req.OrganizationId,
		DepartmentId:          req.DepartmentId,
		DistributionChannelId: req.DistributionChannelId,
		CreatedBy:             c.GetInt("id"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, batch)
}

func AdminGenerateVouchers(c *gin.Context) {
	batchId, _ := strconv.Atoi(c.Param("id"))
	if batchId <= 0 {
		common.ApiErrorMsg(c, "无效的批次ID")
		return
	}
	var req generateVouchersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	vouchers, err := service.NewVoucherService().GenerateVouchers(c.Request.Context(), voucherActorFromContext(c), service.GenerateVouchersInput{
		BatchId:            batchId,
		Quantity:           req.Quantity,
		QuotaAmount:        req.QuotaAmount,
		SubscriptionPlanId: req.SubscriptionPlanId,
		ExpiredAt:          req.ExpiredAt,
		Codes:              req.Codes,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, vouchers)
}

func AdminListVoucherBatches(c *gin.Context) {
	page, err := service.NewVoucherService().ListBatches(c.Request.Context(), voucherActorFromContext(c), voucherListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func AdminListVouchers(c *gin.Context) {
	page, err := service.NewVoucherService().ListVouchers(c.Request.Context(), voucherActorFromContext(c), voucherListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func AdminListVoucherRedemptions(c *gin.Context) {
	page, err := service.NewVoucherService().ListRedemptions(c.Request.Context(), voucherActorFromContext(c), voucherListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func AdminDisableVoucher(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的兑换码ID")
		return
	}
	voucher, err := service.NewVoucherService().DisableVoucher(c.Request.Context(), voucherActorFromContext(c), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, voucher)
}

func AdminDisableVoucherBatch(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的批次ID")
		return
	}
	batch, err := service.NewVoucherService().DisableBatch(c.Request.Context(), voucherActorFromContext(c), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, batch)
}

func voucherActorFromContext(c *gin.Context) service.VoucherActor {
	return service.VoucherActor{
		UserId: c.GetInt("id"),
		Scope:  model.AccessScopeFromContext(c),
	}
}

func voucherListInputFromContext(c *gin.Context) service.VoucherListInput {
	pageInfo := common.GetPageQuery(c)
	batchId, _ := strconv.Atoi(c.Query("batch_id"))
	userId, _ := strconv.Atoi(c.Query("user_id"))
	startTime, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)
	return service.VoucherListInput{
		Page:        pageInfo.GetPage(),
		PageSize:    pageInfo.GetPageSize(),
		Keyword:     strings.TrimSpace(c.Query("keyword")),
		Status:      strings.TrimSpace(c.Query("status")),
		VoucherType: strings.TrimSpace(c.Query("voucher_type")),
		BatchId:     batchId,
		StartTime:   startTime,
		EndTime:     endTime,
		UserId:      userId,
	}
}
