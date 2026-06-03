package model

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var adminConsoleReadonlyMigrateOnce sync.Once
var adminConsoleReadonlyMigrateErr error

func resetAdminConsoleReadonlyTables(t *testing.T) {
	t.Helper()

	var migrateErr error
	adminConsoleReadonlyMigrateOnce.Do(func() {
		adminConsoleReadonlyMigrateErr = DB.AutoMigrate(&Tenant{}, &Organization{}, &Department{}, &DistributionChannel{})
	})
	migrateErr = adminConsoleReadonlyMigrateErr
	require.NoError(t, migrateErr)

	cleanup := func() {
		DB.Exec("DELETE FROM departments")
		DB.Exec("DELETE FROM organizations")
		DB.Exec("DELETE FROM distribution_channels")
		DB.Exec("DELETE FROM tenants")
	}
	cleanup()
	t.Cleanup(cleanup)
}

func TestGetAllTenantsReadonlyReturnsSafePagedProjection(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	require.NoError(t, DB.Create(&Tenant{Name: "tenant-a", Status: 1, Remark: "private"}).Error)
	require.NoError(t, DB.Create(&Tenant{Name: "tenant-b", Status: 2, Remark: "private"}).Error)
	require.NoError(t, DB.Create(&Tenant{Name: "tenant-c", Status: 1, Remark: "private"}).Error)

	items, total, err := GetAllTenantsReadonly(1, 1)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, items, 1)
	require.Equal(t, "tenant-b", items[0].Name)
	require.Equal(t, 2, items[0].Status)
	require.NotZero(t, items[0].CreatedAt)
}

func TestGetAllOrganizationsReadonlyReturnsSafePagedProjection(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	require.NoError(t, DB.Create(&Organization{TenantId: 2, Name: "org-b", Status: 1, Remark: "private"}).Error)
	require.NoError(t, DB.Create(&Organization{TenantId: 1, Name: "org-a", Status: 2, Remark: "private"}).Error)

	items, total, err := GetAllOrganizationsReadonly(0, 10)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, items, 2)
	require.Equal(t, "org-a", items[0].Name)
	require.Equal(t, 1, items[0].TenantId)
	require.Equal(t, 2, items[0].Status)
	require.NotZero(t, items[0].CreatedAt)
}

func TestGetAllDepartmentsReadonlyReturnsSafePagedProjection(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	require.NoError(t, DB.Create(&Department{
		TenantId:       1,
		OrganizationId: 2,
		Name:           "dept-b",
		Status:         1,
		ParentId:       99,
		Remark:         "private",
	}).Error)
	require.NoError(t, DB.Create(&Department{
		TenantId:       1,
		OrganizationId: 1,
		Name:           "dept-a",
		Status:         2,
		ParentId:       99,
		Remark:         "private",
	}).Error)

	items, total, err := GetAllDepartmentsReadonly(0, 10)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, items, 2)
	require.Equal(t, "dept-a", items[0].Name)
	require.Equal(t, 1, items[0].TenantId)
	require.Equal(t, 1, items[0].OrganizationId)
	require.Equal(t, 2, items[0].Status)
	require.NotZero(t, items[0].CreatedAt)
}

func TestGetAllDistributionChannelsReadonlyReturnsSafePagedProjection(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	require.NoError(t, DB.Create(&DistributionChannel{
		TenantId:       2,
		Name:           "channel-b",
		Code:           "channel-b",
		Status:         1,
		OwnerUserId:    100,
		CommissionRate: 0.15,
		Remark:         "private",
	}).Error)
	require.NoError(t, DB.Create(&DistributionChannel{
		TenantId:       1,
		Name:           "channel-a",
		Code:           "channel-a",
		Status:         2,
		OwnerUserId:    100,
		CommissionRate: 0.15,
		Remark:         "private",
	}).Error)

	items, total, err := GetAllDistributionChannelsReadonly(0, 10)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, items, 2)
	require.Equal(t, "channel-a", items[0].Name)
	require.Equal(t, 1, items[0].TenantId)
	require.Equal(t, 2, items[0].Status)
	require.NotZero(t, items[0].CreatedAt)
}
