package controller

import (
	"strconv"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
)

const (
	readonlyDefaultLimit = 50
	readonlyMaxLimit     = 200
)

func getReadonlyPagination(c *gin.Context) (page int, limit int, offset int) {
	page = parsePositiveQueryInt(c.Query("page"), 1)
	limit = parsePositiveQueryInt(c.Query("limit"), readonlyDefaultLimit)
	if limit > readonlyMaxLimit {
		limit = readonlyMaxLimit
	}
	offset = (page - 1) * limit
	return page, limit, offset
}

func parsePositiveQueryInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func GetReadonlyTenants(c *gin.Context) {
	page, limit, offset := getReadonlyPagination(c)
	items, total, err := model.GetAllTenantsReadonly(offset, limit, model.AdminConsoleReadonlyFilters{
		Keyword: strings.TrimSpace(c.Query("q")),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

func GetReadonlyOrganizations(c *gin.Context) {
	page, limit, offset := getReadonlyPagination(c)
	items, total, err := model.GetAllOrganizationsReadonly(offset, limit, model.AdminConsoleReadonlyFilters{
		Keyword:  strings.TrimSpace(c.Query("q")),
		TenantId: parsePositiveQueryInt(c.Query("tenant_id"), 0),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

func GetReadonlyDepartments(c *gin.Context) {
	page, limit, offset := getReadonlyPagination(c)
	items, total, err := model.GetAllDepartmentsReadonly(offset, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

func GetReadonlyDistributionChannels(c *gin.Context) {
	page, limit, offset := getReadonlyPagination(c)
	items, total, err := model.GetAllDistributionChannelsReadonly(offset, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}
