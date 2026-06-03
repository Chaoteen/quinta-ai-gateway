package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type adminConsoleTenantListAPIResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items []model.ReadonlyTenant `json:"items"`
		Total int64                  `json:"total"`
		Page  int                    `json:"page"`
		Limit int                    `json:"limit"`
	} `json:"data"`
}

type adminConsoleOrganizationListAPIResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items []model.ReadonlyOrganization `json:"items"`
		Total int64                        `json:"total"`
		Page  int                          `json:"page"`
		Limit int                          `json:"limit"`
	} `json:"data"`
}

func setupAdminConsoleReadonlyControllerTest(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Tenant{}, &model.Organization{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	router := gin.New()
	router.GET("/tenants", GetReadonlyTenants)
	router.GET("/organizations", GetReadonlyOrganizations)
	return router
}

func TestGetReadonlyTenantsAPIKeywordSearch(t *testing.T) {
	router := setupAdminConsoleReadonlyControllerTest(t)
	require.NoError(t, model.DB.Create(&model.Tenant{Name: "alpha tenant", Status: 1}).Error)
	require.NoError(t, model.DB.Create(&model.Tenant{Name: "beta tenant", Status: 1}).Error)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenants?q=alpha", nil)
	router.ServeHTTP(recorder, req)

	var response adminConsoleTenantListAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.EqualValues(t, 1, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	require.Equal(t, "alpha tenant", response.Data.Items[0].Name)
}

func TestGetReadonlyOrganizationsAPITenantFilter(t *testing.T) {
	router := setupAdminConsoleReadonlyControllerTest(t)
	require.NoError(t, model.DB.Create(&model.Organization{TenantId: 1, Name: "tenant-one-org", Status: 1}).Error)
	require.NoError(t, model.DB.Create(&model.Organization{TenantId: 2, Name: "tenant-two-org", Status: 1}).Error)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations?tenant_id=2", nil)
	router.ServeHTTP(recorder, req)

	var response adminConsoleOrganizationListAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.EqualValues(t, 1, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	require.Equal(t, "tenant-two-org", response.Data.Items[0].Name)
	require.Equal(t, 2, response.Data.Items[0].TenantId)
}

func TestGetReadonlyTenantsAPIDefaultAndMaxPagination(t *testing.T) {
	router := setupAdminConsoleReadonlyControllerTest(t)

	defaultRecorder := httptest.NewRecorder()
	defaultReq := httptest.NewRequest(http.MethodGet, "/tenants", nil)
	router.ServeHTTP(defaultRecorder, defaultReq)

	var defaultResponse adminConsoleTenantListAPIResponse
	require.NoError(t, common.Unmarshal(defaultRecorder.Body.Bytes(), &defaultResponse))
	require.True(t, defaultResponse.Success)
	require.Equal(t, 1, defaultResponse.Data.Page)
	require.Equal(t, 50, defaultResponse.Data.Limit)

	maxRecorder := httptest.NewRecorder()
	maxReq := httptest.NewRequest(http.MethodGet, "/tenants?page=3&limit=999", nil)
	router.ServeHTTP(maxRecorder, maxReq)

	var maxResponse adminConsoleTenantListAPIResponse
	require.NoError(t, common.Unmarshal(maxRecorder.Body.Bytes(), &maxResponse))
	require.True(t, maxResponse.Success)
	require.Equal(t, 3, maxResponse.Data.Page)
	require.Equal(t, 200, maxResponse.Data.Limit)
}
