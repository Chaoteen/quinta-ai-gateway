package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/Chaoteen/quinta-ai-gateway/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createPaymentOrderRequest struct {
	Provider     string  `json:"provider"`
	BusinessType string  `json:"business_type"`
	BusinessId   int     `json:"business_id"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Subject      string  `json:"subject"`
	Description  string  `json:"description"`
	ExpiredAt    int64   `json:"expired_at"`
}

type createBankTransferRequest struct {
	PaymentOrderId      int     `json:"payment_order_id"`
	BankAccountName     string  `json:"bank_account_name"`
	BankAccountNoMasked string  `json:"bank_account_no_masked"`
	TransferAmount      float64 `json:"transfer_amount"`
	TransferTime        int64   `json:"transfer_time"`
	ProofUrl            string  `json:"proof_url"`
}

type reviewBankTransferRequest struct {
	Approved     bool   `json:"approved"`
	ReviewNote   string `json:"review_note"`
	FailedStatus bool   `json:"failed_status"`
}

func CreatePaymentOrder(c *gin.Context) {
	var req createPaymentOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	ownership := model.OwnershipFromContext(c)
	order, err := service.NewPaymentGatewayService().CreatePaymentOrder(c.Request.Context(), service.CreatePaymentOrderInput{
		UserId:                c.GetInt("id"),
		TenantId:              ownership.TenantId,
		OrganizationId:        ownership.OrganizationId,
		DepartmentId:          ownership.DepartmentId,
		DistributionChannelId: ownership.DistributionChannelId,
		Provider:              req.Provider,
		BusinessType:          req.BusinessType,
		BusinessId:            req.BusinessId,
		Amount:                req.Amount,
		Currency:              req.Currency,
		Subject:               req.Subject,
		Description:           req.Description,
		ExpiredAt:             req.ExpiredAt,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, order)
}

func ListUserPaymentOrders(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId := c.GetInt("id")
	query := model.DB.Model(&model.PaymentOrder{}).Where("user_id = ?", userId)
	query = applyPaymentOrderFilters(c, query)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var orders []model.PaymentOrder
	if err := query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&orders).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(orders)
	common.ApiSuccess(c, pageInfo)
}

func GetUserPaymentOrder(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的订单ID")
		return
	}
	var order model.PaymentOrder
	if err := model.DB.Where("id = ? AND user_id = ?", id, c.GetInt("id")).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "订单不存在或无权访问")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, order)
}

func CreateBankTransferRecord(c *gin.Context) {
	var req createBankTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	record, err := service.NewPaymentGatewayService().CreateBankTransferRecord(c.Request.Context(), service.CreateBankTransferRecordInput{
		PaymentOrderId:      req.PaymentOrderId,
		UserId:              c.GetInt("id"),
		BankAccountName:     req.BankAccountName,
		BankAccountNoMasked: req.BankAccountNoMasked,
		TransferAmount:      req.TransferAmount,
		TransferTime:        req.TransferTime,
		ProofUrl:            req.ProofUrl,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, record)
}

func AdminListPaymentOrders(c *gin.Context) {
	scope := paymentAdminAccessScope(c)
	pageInfo := common.GetPageQuery(c)
	query := model.DB.Model(&model.PaymentOrder{})
	query = model.ApplyOwnershipScope(query, "payment_orders", scope)
	query = applyPaymentOrderFilters(c, query)
	if userId, _ := strconv.Atoi(c.Query("user_id")); userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var orders []model.PaymentOrder
	if err := query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&orders).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(orders)
	common.ApiSuccess(c, pageInfo)
}

func AdminListPaymentCallbackLogs(c *gin.Context) {
	scope := paymentAdminAccessScope(c)
	pageInfo := common.GetPageQuery(c)
	query := model.DB.Model(&model.PaymentCallbackLog{}).
		Joins("JOIN payment_orders ON payment_orders.id = payment_callback_logs.payment_order_id")
	query = model.ApplyOwnershipScope(query, "payment_orders", scope)
	if provider := strings.TrimSpace(c.Query("provider")); provider != "" {
		query = query.Where("payment_callback_logs.provider = ?", model.NormalizePaymentProvider(provider))
	}
	if orderNo := strings.TrimSpace(c.Query("order_no")); orderNo != "" {
		query = query.Where("payment_callback_logs.order_no = ?", orderNo)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var logs []model.PaymentCallbackLog
	if err := query.Order("payment_callback_logs.id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&logs).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}

func AdminListBankTransfers(c *gin.Context) {
	scope := paymentAdminAccessScope(c)
	pageInfo := common.GetPageQuery(c)
	query := model.DB.Model(&model.BankTransferRecord{}).
		Joins("JOIN payment_orders ON payment_orders.id = bank_transfer_records.payment_order_id")
	query = model.ApplyOwnershipScope(query, "payment_orders", scope)
	if status := strings.TrimSpace(c.Query("review_status")); status != "" {
		query = query.Where("bank_transfer_records.review_status = ?", model.NormalizeBankTransferReviewStatus(status))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var records []model.BankTransferRecord
	if err := query.Order("bank_transfer_records.id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&records).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(records)
	common.ApiSuccess(c, pageInfo)
}

func AdminReviewBankTransfer(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的转账记录ID")
		return
	}
	var req reviewBankTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	scope := paymentAdminAccessScope(c)
	if !ensureBankTransferInScope(c, id, scope) {
		return
	}
	rawPayload := ""
	if payload, err := common.Marshal(req); err == nil {
		rawPayload = string(payload)
	}
	record, err := service.NewPaymentGatewayService().ReviewBankTransfer(c.Request.Context(), id, service.ReviewBankTransferInput{
		ReviewerId:   c.GetInt("id"),
		Approved:     req.Approved,
		ReviewNote:   req.ReviewNote,
		FailedStatus: req.FailedStatus,
		RawPayload:   rawPayload,
		EventType:    "bank_transfer.review",
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, record)
}

func applyPaymentOrderFilters(c *gin.Context, query *gorm.DB) *gorm.DB {
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", model.NormalizePaymentOrderStatus(status))
	}
	if provider := strings.TrimSpace(c.Query("provider")); provider != "" {
		query = query.Where("provider = ?", model.NormalizePaymentProvider(provider))
	}
	if businessType := strings.TrimSpace(c.Query("business_type")); businessType != "" {
		query = query.Where("business_type = ?", model.NormalizePaymentBusinessType(businessType))
	}
	if orderNo := strings.TrimSpace(c.Query("order_no")); orderNo != "" {
		query = query.Where("order_no = ?", orderNo)
	}
	return query
}

func paymentAdminAccessScope(c *gin.Context) model.AccessScope {
	return model.AccessScopeFromContext(c)
}

func ensureBankTransferInScope(c *gin.Context, id int, scope model.AccessScope) bool {
	if scope.IsRoot {
		return true
	}
	var record model.BankTransferRecord
	if err := model.DB.Where("id = ?", id).First(&record).Error; err != nil {
		common.ApiErrorMsg(c, "转账记录不存在或无权访问")
		return false
	}
	if !model.AllowsOwnership(scope, record.TenantId, 0, 0) {
		common.ApiErrorMsg(c, "转账记录不存在或无权访问")
		return false
	}
	roleKey := common.GetContextKeyString(c, constant.ContextKeyUserRoleKey)
	if roleKey == common.RoleKeyUser {
		common.ApiErrorMsg(c, "权限不足")
		return false
	}
	return true
}
