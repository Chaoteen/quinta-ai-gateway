package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type adminConsoleTenantRequest struct {
	Name   string `json:"name"`
	Status *int   `json:"status,omitempty"`
}

type adminConsoleOrganizationRequest struct {
	Name     string `json:"name"`
	TenantId int    `json:"tenant_id"`
	Status   *int   `json:"status,omitempty"`
}

type adminConsoleDepartmentRequest struct {
	Name           string `json:"name"`
	TenantId       int    `json:"tenant_id"`
	OrganizationId int    `json:"organization_id"`
	Status         *int   `json:"status,omitempty"`
}

type adminConsoleDistributionChannelRequest struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	TenantId int    `json:"tenant_id"`
	Status   *int   `json:"status,omitempty"`
}

type adminConsoleStatusRequest struct {
	Status int `json:"status"`
}

func CreateAdminConsoleTenant(c *gin.Context) {
	var req adminConsoleTenantRequest
	if !bindAdminConsoleJSON(c, &req) {
		return
	}
	name, ok := normalizeRequiredAdminConsoleText(c, req.Name, "name")
	if !ok {
		return
	}
	status, ok := normalizeAdminConsoleStatus(c, req.Status)
	if !ok {
		return
	}

	item, err := model.CreateTenantManagement(name, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func UpdateAdminConsoleTenant(c *gin.Context) {
	id, ok := parseAdminConsoleID(c)
	if !ok {
		return
	}
	var req adminConsoleTenantRequest
	if !bindAdminConsoleJSON(c, &req) {
		return
	}
	name, ok := normalizeRequiredAdminConsoleText(c, req.Name, "name")
	if !ok {
		return
	}
	status, ok := normalizeAdminConsoleStatus(c, req.Status)
	if !ok {
		return
	}

	item, err := model.UpdateTenantManagement(id, name, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func UpdateAdminConsoleTenantStatus(c *gin.Context) {
	id, status, ok := parseAdminConsoleStatusUpdate(c)
	if !ok {
		return
	}
	item, err := model.UpdateTenantStatusManagement(id, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func CreateAdminConsoleOrganization(c *gin.Context) {
	var req adminConsoleOrganizationRequest
	if !bindAdminConsoleJSON(c, &req) {
		return
	}
	name, ok := normalizeRequiredAdminConsoleText(c, req.Name, "name")
	if !ok {
		return
	}
	status, ok := normalizeAdminConsoleStatus(c, req.Status)
	if !ok {
		return
	}

	item, err := model.CreateOrganizationManagement(name, req.TenantId, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func UpdateAdminConsoleOrganization(c *gin.Context) {
	id, ok := parseAdminConsoleID(c)
	if !ok {
		return
	}
	var req adminConsoleOrganizationRequest
	if !bindAdminConsoleJSON(c, &req) {
		return
	}
	name, ok := normalizeRequiredAdminConsoleText(c, req.Name, "name")
	if !ok {
		return
	}
	status, ok := normalizeAdminConsoleStatus(c, req.Status)
	if !ok {
		return
	}

	item, err := model.UpdateOrganizationManagement(id, name, req.TenantId, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func UpdateAdminConsoleOrganizationStatus(c *gin.Context) {
	id, status, ok := parseAdminConsoleStatusUpdate(c)
	if !ok {
		return
	}
	item, err := model.UpdateOrganizationStatusManagement(id, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func CreateAdminConsoleDepartment(c *gin.Context) {
	var req adminConsoleDepartmentRequest
	if !bindAdminConsoleJSON(c, &req) {
		return
	}
	name, ok := normalizeRequiredAdminConsoleText(c, req.Name, "name")
	if !ok {
		return
	}
	status, ok := normalizeAdminConsoleStatus(c, req.Status)
	if !ok {
		return
	}

	item, err := model.CreateDepartmentManagement(name, req.TenantId, req.OrganizationId, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func UpdateAdminConsoleDepartment(c *gin.Context) {
	id, ok := parseAdminConsoleID(c)
	if !ok {
		return
	}
	var req adminConsoleDepartmentRequest
	if !bindAdminConsoleJSON(c, &req) {
		return
	}
	name, ok := normalizeRequiredAdminConsoleText(c, req.Name, "name")
	if !ok {
		return
	}
	status, ok := normalizeAdminConsoleStatus(c, req.Status)
	if !ok {
		return
	}

	item, err := model.UpdateDepartmentManagement(id, name, req.TenantId, req.OrganizationId, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func UpdateAdminConsoleDepartmentStatus(c *gin.Context) {
	id, status, ok := parseAdminConsoleStatusUpdate(c)
	if !ok {
		return
	}
	item, err := model.UpdateDepartmentStatusManagement(id, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func CreateAdminConsoleDistributionChannel(c *gin.Context) {
	var req adminConsoleDistributionChannelRequest
	if !bindAdminConsoleJSON(c, &req) {
		return
	}
	name, ok := normalizeRequiredAdminConsoleText(c, req.Name, "name")
	if !ok {
		return
	}
	code, ok := normalizeRequiredAdminConsoleText(c, req.Code, "code")
	if !ok {
		return
	}
	status, ok := normalizeAdminConsoleStatus(c, req.Status)
	if !ok {
		return
	}

	item, err := model.CreateDistributionChannelManagement(name, code, req.TenantId, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func UpdateAdminConsoleDistributionChannel(c *gin.Context) {
	id, ok := parseAdminConsoleID(c)
	if !ok {
		return
	}
	var req adminConsoleDistributionChannelRequest
	if !bindAdminConsoleJSON(c, &req) {
		return
	}
	name, ok := normalizeRequiredAdminConsoleText(c, req.Name, "name")
	if !ok {
		return
	}
	code, ok := normalizeRequiredAdminConsoleText(c, req.Code, "code")
	if !ok {
		return
	}
	status, ok := normalizeAdminConsoleStatus(c, req.Status)
	if !ok {
		return
	}

	item, err := model.UpdateDistributionChannelManagement(id, name, code, req.TenantId, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func UpdateAdminConsoleDistributionChannelStatus(c *gin.Context) {
	id, status, ok := parseAdminConsoleStatusUpdate(c)
	if !ok {
		return
	}
	item, err := model.UpdateDistributionChannelStatusManagement(id, status)
	writeAdminConsoleMutationResponse(c, item, err)
}

func bindAdminConsoleJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return false
	}
	return true
}

func parseAdminConsoleID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid id")
		return 0, false
	}
	return id, true
}

func parseAdminConsoleStatusUpdate(c *gin.Context) (int, int, bool) {
	id, ok := parseAdminConsoleID(c)
	if !ok {
		return 0, 0, false
	}
	var req adminConsoleStatusRequest
	if !bindAdminConsoleJSON(c, &req) {
		return 0, 0, false
	}
	status, ok := normalizeAdminConsoleStatus(c, &req.Status)
	if !ok {
		return 0, 0, false
	}
	return id, status, true
}

func normalizeRequiredAdminConsoleText(c *gin.Context, value string, field string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		common.ApiErrorMsg(c, field+" is required")
		return "", false
	}
	return trimmed, true
}

func normalizeAdminConsoleStatus(c *gin.Context, status *int) (int, bool) {
	value := model.DefaultAdminConsoleStatus(status)
	if value != common.UserStatusEnabled && value != common.UserStatusDisabled {
		common.ApiErrorMsg(c, "status must be 1 or 2")
		return 0, false
	}
	return value, true
}

func writeAdminConsoleMutationResponse(c *gin.Context, item any, err error) {
	if err == nil {
		common.ApiSuccess(c, item)
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		common.ApiErrorMsg(c, "resource not found")
		return
	}
	common.ApiError(c, err)
}
