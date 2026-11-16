package controller

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/Chaoteen/quinta-ai-gateway/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func supplierDecode(c *gin.Context, v interface{}) bool {
	c.Header("Cache-Control", "no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32*1024)
	if err := common.DecodeJson(c.Request.Body, v); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid supplier request"})
		return false
	}
	return true
}
func supplierResponse(c *gin.Context, data interface{}, err error) {
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, data)
}
func PlatformSupplierSummary(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	data, err := service.SupplierSummary()
	supplierResponse(c, data, err)
}
func SavePlatformSupplier(c *gin.Context) {
	var request struct {
		Supplier model.PlatformSupplier `json:"supplier"`
		Review   bool                   `json:"review"`
	}
	if !supplierDecode(c, &request) {
		return
	}
	row, err := service.SavePlatformSupplier(request.Supplier, c.GetInt("id"), request.Review)
	supplierResponse(c, row, err)
}
func CreateSupplierAccount(c *gin.Context) {
	var input model.SupplierAccount
	if !supplierDecode(c, &input) {
		return
	}
	row, err := service.CreateSupplierAccount(input, c.GetInt("id"))
	supplierResponse(c, row, err)
}
func SupplierAccountStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.AbortWithStatus(400)
		return
	}
	var request struct {
		Status string `json:"status"`
	}
	if !supplierDecode(c, &request) {
		return
	}
	supplierResponse(c, nil, service.SetSupplierAccountStatus(id, request.Status, c.GetInt("id")))
}
func CreateSupplierPrice(c *gin.Context) {
	var input service.SupplierPriceInput
	if !supplierDecode(c, &input) {
		return
	}
	row, err := service.CreateSupplierPrice(input, c.GetInt("id"))
	supplierResponse(c, row, err)
}
func CreateSupplierStatement(c *gin.Context) {
	var input service.SupplierStatementInput
	if !supplierDecode(c, &input) {
		return
	}
	row, err := service.CreateSupplierStatement(input, c.GetInt("id"))
	supplierResponse(c, row, err)
}
func ReviewSupplierStatement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.AbortWithStatus(400)
		return
	}
	var input struct {
		Note string `json:"note"`
	}
	if !supplierDecode(c, &input) {
		return
	}
	supplierResponse(c, nil, service.ReviewSupplierStatement(id, input.Note, c.GetInt("id")))
}
func RecordSupplierPayment(c *gin.Context) {
	var input struct {
		StatementId int    `json:"statement_id"`
		Reference   string `json:"reference"`
		Amount      string `json:"amount"`
	}
	if !supplierDecode(c, &input) {
		return
	}
	row, err := service.RecordSupplierPayment(input.StatementId, input.Reference, input.Amount, c.GetInt("id"))
	supplierResponse(c, row, err)
}
