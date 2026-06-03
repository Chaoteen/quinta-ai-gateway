package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newReadonlyPaginationTestContext(target string) *gin.Context {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("GET", target, nil)
	ctx.Request = req
	return ctx
}

func TestGetReadonlyPaginationDefaults(t *testing.T) {
	ctx := newReadonlyPaginationTestContext("/api/admin_console/tenants")

	page, limit, offset := getReadonlyPagination(ctx)

	require.Equal(t, 1, page)
	require.Equal(t, 50, limit)
	require.Equal(t, 0, offset)
}

func TestGetReadonlyPaginationUsesLimitAndPage(t *testing.T) {
	ctx := newReadonlyPaginationTestContext("/api/admin_console/tenants?page=3&limit=25")

	page, limit, offset := getReadonlyPagination(ctx)

	require.Equal(t, 3, page)
	require.Equal(t, 25, limit)
	require.Equal(t, 50, offset)
}

func TestGetReadonlyPaginationCapsLimit(t *testing.T) {
	ctx := newReadonlyPaginationTestContext("/api/admin_console/tenants?page=2&limit=500")

	page, limit, offset := getReadonlyPagination(ctx)

	require.Equal(t, 2, page)
	require.Equal(t, 200, limit)
	require.Equal(t, 200, offset)
}

func TestGetReadonlyPaginationRejectsInvalidValues(t *testing.T) {
	ctx := newReadonlyPaginationTestContext("/api/admin_console/tenants?page=-1&limit=0")

	page, limit, offset := getReadonlyPagination(ctx)

	require.Equal(t, 1, page)
	require.Equal(t, 50, limit)
	require.Equal(t, 0, offset)
}
