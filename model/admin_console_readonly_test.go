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
		adminConsoleReadonlyMigrateErr = DB.AutoMigrate(
			&Tenant{},
			&Organization{},
			&Department{},
			&DistributionChannel{},
			&User{},
			&Channel{},
			&Token{},
			&TopUp{},
			&SubscriptionOrder{},
			&UserSubscription{},
			&SubscriptionPreConsumeRecord{},
			&Redemption{},
			&Midjourney{},
			&Task{},
			&Log{},
			&Ability{},
		)
	})
	migrateErr = adminConsoleReadonlyMigrateErr
	require.NoError(t, migrateErr)

	cleanup := func() {
		DB.Exec("DELETE FROM departments")
		DB.Exec("DELETE FROM organizations")
		DB.Exec("DELETE FROM distribution_channels")
		DB.Exec("DELETE FROM tenants")
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM channels")
		DB.Exec("DELETE FROM tokens")
		DB.Exec("DELETE FROM top_ups")
		DB.Exec("DELETE FROM subscription_orders")
		DB.Exec("DELETE FROM user_subscriptions")
		DB.Exec("DELETE FROM subscription_pre_consume_records")
		DB.Exec("DELETE FROM redemptions")
		DB.Exec("DELETE FROM midjourneys")
		DB.Exec("DELETE FROM tasks")
		DB.Exec("DELETE FROM logs")
		DB.Exec("DELETE FROM abilities")
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

func TestGetAllTenantsReadonlyFiltersByKeyword(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	require.NoError(t, DB.Create(&Tenant{Name: "alpha tenant", Status: 1}).Error)
	require.NoError(t, DB.Create(&Tenant{Name: "beta tenant", Status: 1}).Error)

	items, total, err := GetAllTenantsReadonly(0, 50, AdminConsoleReadonlyFilters{Keyword: "alpha"})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "alpha tenant", items[0].Name)
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

func TestGetAllOrganizationsReadonlyFiltersByKeywordAndTenant(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	require.NoError(t, DB.Create(&Organization{TenantId: 1, Name: "alpha org", Status: 1}).Error)
	require.NoError(t, DB.Create(&Organization{TenantId: 2, Name: "alpha org other tenant", Status: 1}).Error)
	require.NoError(t, DB.Create(&Organization{TenantId: 1, Name: "beta org", Status: 1}).Error)

	items, total, err := GetAllOrganizationsReadonly(0, 50, AdminConsoleReadonlyFilters{
		Keyword:  "alpha",
		TenantId: 1,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "alpha org", items[0].Name)
	require.Equal(t, 1, items[0].TenantId)
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
	require.Equal(t, "channel-a", items[0].Code)
	require.Equal(t, 1, items[0].TenantId)
	require.Equal(t, 2, items[0].Status)
	require.NotZero(t, items[0].CreatedAt)
}

func TestCreateOrganizationManagementRejectsMissingTenant(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	_, err := CreateOrganizationManagement("orphan-org", 999, 1)
	require.ErrorContains(t, err, "tenant_id 999 does not exist")
}

func TestCreateDepartmentManagementRejectsOrganizationTenantMismatch(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	require.NoError(t, DB.Create(&Tenant{Name: "tenant-a", Status: 1}).Error)
	require.NoError(t, DB.Create(&Tenant{Name: "tenant-b", Status: 1}).Error)

	var tenantA Tenant
	var tenantB Tenant
	require.NoError(t, DB.Where("name = ?", "tenant-a").First(&tenantA).Error)
	require.NoError(t, DB.Where("name = ?", "tenant-b").First(&tenantB).Error)

	org, err := CreateOrganizationManagement("org-a", tenantA.Id, 1)
	require.NoError(t, err)

	_, err = CreateDepartmentManagement("bad-dept", tenantB.Id, org.Id, 1)
	require.ErrorContains(t, err, "does not belong to tenant_id")
}

func TestUpdateOrganizationManagementRejectsTenantChangeWithDepartments(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	require.NoError(t, DB.Create(&Tenant{Name: "tenant-a", Status: 1}).Error)
	require.NoError(t, DB.Create(&Tenant{Name: "tenant-b", Status: 1}).Error)

	var tenantA Tenant
	var tenantB Tenant
	require.NoError(t, DB.Where("name = ?", "tenant-a").First(&tenantA).Error)
	require.NoError(t, DB.Where("name = ?", "tenant-b").First(&tenantB).Error)

	org, err := CreateOrganizationManagement("org-a", tenantA.Id, 1)
	require.NoError(t, err)
	_, err = CreateDepartmentManagement("dept-a", tenantA.Id, org.Id, 1)
	require.NoError(t, err)

	_, err = UpdateOrganizationManagement(org.Id, "org-moved", tenantB.Id, 1)
	require.ErrorContains(t, err, "departments.organization_id")
}

func TestUpdateOrganizationManagementRejectsTenantChangeWithUsers(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	tenantA, tenantB := createAdminConsoleReadonlyTenants(t)
	org, err := CreateOrganizationManagement("org-a", tenantA.Id, 1)
	require.NoError(t, err)
	require.NoError(t, DB.Create(&User{
		TenantId:       tenantA.Id,
		OrganizationId: org.Id,
		Username:       "org-user",
		Password:       "password123",
		AffCode:        "org-user-aff",
		Status:         1,
	}).Error)

	_, err = UpdateOrganizationManagement(org.Id, "org-moved", tenantB.Id, 1)
	require.ErrorContains(t, err, "users.organization_id")
}

func TestUpdateDepartmentManagementRejectsParentChangeWithUsers(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	tenantA, tenantB := createAdminConsoleReadonlyTenants(t)
	orgA, err := CreateOrganizationManagement("org-a", tenantA.Id, 1)
	require.NoError(t, err)
	orgB, err := CreateOrganizationManagement("org-b", tenantB.Id, 1)
	require.NoError(t, err)
	department, err := CreateDepartmentManagement("dept-a", tenantA.Id, orgA.Id, 1)
	require.NoError(t, err)
	require.NoError(t, DB.Create(&User{
		TenantId:       tenantA.Id,
		OrganizationId: orgA.Id,
		DepartmentId:   department.Id,
		Username:       "dept-user",
		Password:       "password123",
		AffCode:        "dept-user-aff",
		Status:         1,
	}).Error)

	_, err = UpdateDepartmentManagement(department.Id, "dept-moved", tenantB.Id, orgB.Id, 1)
	require.ErrorContains(t, err, "users.department_id")
}

func TestCreateDistributionChannelManagementRejectsMissingTenant(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	_, err := CreateDistributionChannelManagement("channel", "channel", 999, 1)
	require.ErrorContains(t, err, "tenant_id 999 does not exist")
}

func TestUpdateTenantStatusManagementSoftDisablesTenant(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	tenant, err := CreateTenantManagement("tenant-a", 1)
	require.NoError(t, err)

	updated, err := UpdateTenantStatusManagement(tenant.Id, 2)
	require.NoError(t, err)
	require.Equal(t, 2, updated.Status)
	require.Equal(t, tenant.Id, updated.Id)
}

func TestUpdateDistributionChannelManagementRejectsTenantChangeWithUsers(t *testing.T) {
	resetAdminConsoleReadonlyTables(t)

	tenantA, tenantB := createAdminConsoleReadonlyTenants(t)
	channel, err := CreateDistributionChannelManagement("channel-a", "channel-a", tenantA.Id, 1)
	require.NoError(t, err)
	require.NoError(t, DB.Create(&User{
		TenantId:              tenantA.Id,
		DistributionChannelId: channel.Id,
		Username:              "channel-user",
		Password:              "password123",
		AffCode:               "channel-user-aff",
		Status:                1,
	}).Error)

	_, err = UpdateDistributionChannelManagement(channel.Id, "channel-moved", "channel-moved", tenantB.Id, 1)
	require.ErrorContains(t, err, "users.distribution_channel_id")
}

func TestUpdateDistributionChannelManagementRejectsTenantChangeWithDependentModels(t *testing.T) {
	tests := []struct {
		name      string
		wantLabel string
		create    func(t *testing.T, tenantId int, channelId int)
	}{
		{
			name:      "organization",
			wantLabel: "organizations.distribution_channel_id",
			create: func(t *testing.T, tenantId int, channelId int) {
				require.NoError(t, DB.Create(&Organization{TenantId: tenantId, Name: "org-ref", DistributionChannelId: channelId, Status: 1}).Error)
			},
		},
		{
			name:      "department",
			wantLabel: "departments.distribution_channel_id",
			create: func(t *testing.T, tenantId int, channelId int) {
				require.NoError(t, DB.Create(&Department{TenantId: tenantId, Name: "dept-ref", DistributionChannelId: channelId, Status: 1}).Error)
			},
		},
		{
			name:      "channel",
			wantLabel: "channels.distribution_channel_id",
			create: func(t *testing.T, tenantId int, channelId int) {
				require.NoError(t, DB.Create(&Channel{TenantId: tenantId, Name: "channel-ref", Key: "key", DistributionChannelId: channelId, Status: 1}).Error)
			},
		},
		{
			name:      "token",
			wantLabel: "tokens.distribution_channel_id",
			create: func(t *testing.T, tenantId int, channelId int) {
				require.NoError(t, DB.Create(&Token{TenantId: tenantId, Name: "token-ref", Key: "token-key", DistributionChannelId: channelId, Status: 1}).Error)
			},
		},
		{
			name:      "topup",
			wantLabel: "top_ups.distribution_channel_id",
			create: func(t *testing.T, tenantId int, channelId int) {
				require.NoError(t, DB.Create(&TopUp{TenantId: tenantId, TradeNo: "topup-ref", DistributionChannelId: channelId, Status: "pending"}).Error)
			},
		},
		{
			name:      "subscription_order",
			wantLabel: "subscription_orders.distribution_channel_id",
			create: func(t *testing.T, tenantId int, channelId int) {
				require.NoError(t, DB.Create(&SubscriptionOrder{TenantId: tenantId, TradeNo: "sub-order-ref", DistributionChannelId: channelId, Status: "pending"}).Error)
			},
		},
		{
			name:      "user_subscription",
			wantLabel: "user_subscriptions.distribution_channel_id",
			create: func(t *testing.T, tenantId int, channelId int) {
				require.NoError(t, DB.Create(&UserSubscription{TenantId: tenantId, DistributionChannelId: channelId, Status: "active"}).Error)
			},
		},
		{
			name:      "subscription_pre_consume",
			wantLabel: "subscription_pre_consume_records.distribution_channel_id",
			create: func(t *testing.T, tenantId int, channelId int) {
				require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{TenantId: tenantId, RequestId: "pre-consume-ref", DistributionChannelId: channelId, Status: "consumed"}).Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminConsoleReadonlyTables(t)
			tenantA, tenantB := createAdminConsoleReadonlyTenants(t)
			channel, err := CreateDistributionChannelManagement("channel-a", "channel-a", tenantA.Id, 1)
			require.NoError(t, err)
			tt.create(t, tenantA.Id, channel.Id)

			_, err = UpdateDistributionChannelManagement(channel.Id, "channel-moved", "channel-moved", tenantB.Id, 1)
			require.ErrorContains(t, err, tt.wantLabel)
		})
	}
}

func createAdminConsoleReadonlyTenants(t *testing.T) (Tenant, Tenant) {
	t.Helper()
	require.NoError(t, DB.Create(&Tenant{Name: "tenant-a", Status: 1}).Error)
	require.NoError(t, DB.Create(&Tenant{Name: "tenant-b", Status: 1}).Error)

	var tenantA Tenant
	var tenantB Tenant
	require.NoError(t, DB.Where("name = ?", "tenant-a").First(&tenantA).Error)
	require.NoError(t, DB.Where("name = ?", "tenant-b").First(&tenantB).Error)
	return tenantA, tenantB
}
