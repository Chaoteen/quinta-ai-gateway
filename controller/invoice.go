package controller

import (
	"strconv"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/Chaoteen/quinta-ai-gateway/service"
	"github.com/gin-gonic/gin"
)

type createInvoiceProfileRequest struct {
	UserId           int    `json:"user_id"`
	ProfileType      string `json:"profile_type"`
	Title            string `json:"title"`
	TaxNo            string `json:"tax_no"`
	BankName         string `json:"bank_name"`
	BankAccount      string `json:"bank_account"`
	CompanyAddress   string `json:"company_address"`
	CompanyPhone     string `json:"company_phone"`
	RecipientName    string `json:"recipient_name"`
	RecipientPhone   string `json:"recipient_phone"`
	RecipientEmail   string `json:"recipient_email"`
	RecipientAddress string `json:"recipient_address"`
	IsDefault        bool   `json:"is_default"`
}

type createInvoiceApplicationRequest struct {
	InvoiceProfileId int     `json:"invoice_profile_id"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	InvoiceType      string  `json:"invoice_type"`
	SourceType       string  `json:"source_type"`
	SourceId         int     `json:"source_id"`
}

type reviewInvoiceApplicationRequest struct {
	Approved   bool   `json:"approved"`
	ReviewNote string `json:"review_note"`
}

type issueInvoiceRequest struct {
	InvoiceNo   string `json:"invoice_no"`
	InvoiceDate int64  `json:"invoice_date"`
	FileName    string `json:"file_name"`
	FileUrl     string `json:"file_url"`
	FileType    string `json:"file_type"`
}

func CreateInvoiceProfile(c *gin.Context) {
	var req createInvoiceProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.UserId = 0
	profile, err := service.NewInvoiceService().CreateInvoiceProfile(c.Request.Context(), invoiceActorFromContext(c), invoiceProfileInput(req))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, profile)
}

func ListInvoiceProfiles(c *gin.Context) {
	page, err := service.NewInvoiceService().ListInvoiceProfiles(c.Request.Context(), invoiceActorFromContext(c), invoiceListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func DisableInvoiceProfile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的开票资料ID")
		return
	}
	profile, err := service.NewInvoiceService().DisableInvoiceProfile(c.Request.Context(), invoiceActorFromContext(c), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, profile)
}

func CreateInvoiceApplication(c *gin.Context) {
	var req createInvoiceApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	app, err := service.NewInvoiceService().CreateInvoiceApplication(c.Request.Context(), invoiceActorFromContext(c), service.CreateInvoiceApplicationInput{
		InvoiceProfileId: req.InvoiceProfileId,
		Amount:           req.Amount,
		Currency:         req.Currency,
		InvoiceType:      req.InvoiceType,
		SourceType:       req.SourceType,
		SourceId:         req.SourceId,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, app)
}

func ListInvoiceApplications(c *gin.Context) {
	page, err := service.NewInvoiceService().ListInvoiceApplications(c.Request.Context(), invoiceActorFromContext(c), invoiceListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func ListInvoiceFiles(c *gin.Context) {
	page, err := service.NewInvoiceService().ListInvoiceFiles(c.Request.Context(), invoiceActorFromContext(c), invoiceListInputFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func AdminCreateInvoiceProfile(c *gin.Context) {
	var req createInvoiceProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	profile, err := service.NewInvoiceService().CreateInvoiceProfile(c.Request.Context(), invoiceActorFromContext(c), invoiceProfileInput(req))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, profile)
}

func AdminListInvoiceProfiles(c *gin.Context) {
	ListInvoiceProfiles(c)
}

func AdminListInvoiceApplications(c *gin.Context) {
	ListInvoiceApplications(c)
}

func AdminListInvoiceFiles(c *gin.Context) {
	ListInvoiceFiles(c)
}

func AdminReviewInvoiceApplication(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的发票申请ID")
		return
	}
	var req reviewInvoiceApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	app, err := service.NewInvoiceService().ReviewInvoiceApplication(c.Request.Context(), invoiceActorFromContext(c), id, service.ReviewInvoiceApplicationInput{
		ReviewerId: c.GetInt("id"),
		Approved:   req.Approved,
		ReviewNote: req.ReviewNote,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, app)
}

func AdminIssueInvoice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的发票申请ID")
		return
	}
	var req issueInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	app, err := service.NewInvoiceService().IssueInvoice(c.Request.Context(), invoiceActorFromContext(c), id, service.IssueInvoiceInput{
		UploadedBy:  c.GetInt("id"),
		InvoiceNo:   req.InvoiceNo,
		InvoiceDate: req.InvoiceDate,
		FileName:    req.FileName,
		FileUrl:     req.FileUrl,
		FileType:    req.FileType,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, app)
}

func invoiceActorFromContext(c *gin.Context) service.InvoiceActor {
	return service.InvoiceActor{
		UserId: c.GetInt("id"),
		Scope:  model.AccessScopeFromContext(c),
	}
}

func invoiceProfileInput(req createInvoiceProfileRequest) service.CreateInvoiceProfileInput {
	return service.CreateInvoiceProfileInput{
		UserId:           req.UserId,
		ProfileType:      req.ProfileType,
		Title:            req.Title,
		TaxNo:            req.TaxNo,
		BankName:         req.BankName,
		BankAccount:      req.BankAccount,
		CompanyAddress:   req.CompanyAddress,
		CompanyPhone:     req.CompanyPhone,
		RecipientName:    req.RecipientName,
		RecipientPhone:   req.RecipientPhone,
		RecipientEmail:   req.RecipientEmail,
		RecipientAddress: req.RecipientAddress,
		IsDefault:        req.IsDefault,
	}
}

func invoiceListInputFromContext(c *gin.Context) service.InvoiceListInput {
	pageInfo := common.GetPageQuery(c)
	userId, _ := strconv.Atoi(c.Query("user_id"))
	sourceId, _ := strconv.Atoi(c.Query("source_id"))
	startTime, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)
	return service.InvoiceListInput{
		Page:      pageInfo.GetPage(),
		PageSize:  pageInfo.GetPageSize(),
		Status:    strings.TrimSpace(c.Query("status")),
		UserId:    userId,
		SourceId:  sourceId,
		Keyword:   strings.TrimSpace(c.Query("keyword")),
		StartTime: startTime,
		EndTime:   endTime,
	}
}
