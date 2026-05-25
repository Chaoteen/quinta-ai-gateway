package controller

import (
	"net/http"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"gorm.io/gorm"
)

func setupOwnershipCreationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := openTokenControllerTestDB(t)
	if err := db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.Ability{},
		&model.Organization{},
		&model.Department{},
		&model.DistributionChannel{},
	); err != nil {
		t.Fatalf("failed to migrate ownership creation test tables: %v", err)
	}

	oldQuotaForNewUser := common.QuotaForNewUser
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	common.QuotaForNewUser = 0
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.QuotaForNewUser = oldQuotaForNewUser
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
	return db
}

func TestRootAddChannelRequiresExplicitTenantID(t *testing.T) {
	db := setupOwnershipCreationTestDB(t)

	body := AddChannelRequest{
		Mode: "single",
		Channel: &model.Channel{
			Name:   "root-channel-without-tenant",
			Key:    "test-key",
			Models: "gpt-test",
			Group:  "default",
		},
	}
	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/channel/", body, 1)
	ctx.Set("role", common.RoleRootUser)
	AddChannel(ctx)

	response := decodeAPIResponse(t, recorder)
	if response.Success {
		t.Fatal("expected root channel creation without tenant_id to fail")
	}
	var count int64
	if err := db.Model(&model.Channel{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count channels: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no channel insert after validation failure, got %d", count)
	}
}

func TestRootCreateUserRejectsCrossTenantOwnership(t *testing.T) {
	db := setupOwnershipCreationTestDB(t)

	orgTenant1 := model.Organization{Name: "tenant-1-org", TenantId: 1}
	orgTenant2 := model.Organization{Name: "tenant-2-org", TenantId: 2}
	if err := db.Create(&orgTenant1).Error; err != nil {
		t.Fatalf("failed to create tenant 1 organization: %v", err)
	}
	if err := db.Create(&orgTenant2).Error; err != nil {
		t.Fatalf("failed to create tenant 2 organization: %v", err)
	}
	departmentTenant2 := model.Department{Name: "tenant-2-dept", TenantId: 2, OrganizationId: orgTenant2.Id}
	if err := db.Create(&departmentTenant2).Error; err != nil {
		t.Fatalf("failed to create cross-tenant department: %v", err)
	}
	departmentWrongOrg := model.Department{Name: "wrong-org-dept", TenantId: 1, OrganizationId: orgTenant2.Id}
	if err := db.Create(&departmentWrongOrg).Error; err != nil {
		t.Fatalf("failed to create wrong-organization department: %v", err)
	}
	distributionTenant2 := model.DistributionChannel{Name: "tenant-2-channel", Code: "t2", TenantId: 2}
	if err := db.Create(&distributionTenant2).Error; err != nil {
		t.Fatalf("failed to create cross-tenant distribution channel: %v", err)
	}

	testCases := []struct {
		name string
		user model.User
	}{
		{
			name: "organization from another tenant",
			user: model.User{TenantId: 1, OrganizationId: orgTenant2.Id},
		},
		{
			name: "department from another tenant",
			user: model.User{TenantId: 1, OrganizationId: orgTenant1.Id, DepartmentId: departmentTenant2.Id},
		},
		{
			name: "department from another organization",
			user: model.User{TenantId: 1, OrganizationId: orgTenant1.Id, DepartmentId: departmentWrongOrg.Id},
		},
		{
			name: "distribution channel from another tenant",
			user: model.User{TenantId: 1, DistributionChannelId: distributionTenant2.Id},
		},
	}

	for i, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.user.Username = "rejected-user-" + string(rune('a'+i))
			testCase.user.Password = "password1"
			testCase.user.Role = common.RoleCommonUser
			ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/user/", testCase.user, 1)
			ctx.Set("role", common.RoleRootUser)
			CreateUser(ctx)

			response := decodeAPIResponse(t, recorder)
			if response.Success {
				t.Fatalf("expected root user creation with invalid ownership to fail")
			}
			var count int64
			if err := db.Model(&model.User{}).Where("username = ?", testCase.user.Username).Count(&count).Error; err != nil {
				t.Fatalf("failed to count rejected user: %v", err)
			}
			if count != 0 {
				t.Fatalf("expected no inserted user for invalid ownership, got %d", count)
			}
		})
	}
}

func TestNonRootAdminCreationOverridesRequestedTenantID(t *testing.T) {
	db := setupOwnershipCreationTestDB(t)
	const contextTenantID = 7

	channelBody := AddChannelRequest{
		Mode: "single",
		Channel: &model.Channel{
			TenantId: 99,
			Name:     "admin-channel",
			Key:      "test-key",
			Models:   "gpt-test",
			Group:    "default",
		},
	}
	channelCtx, channelRecorder := newAuthenticatedContext(t, http.MethodPost, "/api/channel/", channelBody, 1)
	channelCtx.Set("role", common.RoleAdminUser)
	common.SetContextKey(channelCtx, constant.ContextKeyTenantId, contextTenantID)
	AddChannel(channelCtx)
	if response := decodeAPIResponse(t, channelRecorder); !response.Success {
		t.Fatalf("expected non-root channel creation to succeed, got %s", response.Message)
	}
	var channel model.Channel
	if err := db.Where("name = ?", "admin-channel").First(&channel).Error; err != nil {
		t.Fatalf("failed to load created channel: %v", err)
	}
	if channel.TenantId != contextTenantID {
		t.Fatalf("expected channel tenant_id %d from context, got %d", contextTenantID, channel.TenantId)
	}

	userBody := model.User{
		TenantId: 99,
		Username: "admin-created-user",
		Password: "password1",
		Role:     common.RoleCommonUser,
	}
	userCtx, userRecorder := newAuthenticatedContext(t, http.MethodPost, "/api/user/", userBody, 1)
	userCtx.Set("role", common.RoleAdminUser)
	common.SetContextKey(userCtx, constant.ContextKeyTenantId, contextTenantID)
	CreateUser(userCtx)
	if response := decodeAPIResponse(t, userRecorder); !response.Success {
		t.Fatalf("expected non-root user creation to succeed, got %s", response.Message)
	}
	var user model.User
	if err := db.Where("username = ?", userBody.Username).First(&user).Error; err != nil {
		t.Fatalf("failed to load created user: %v", err)
	}
	if user.TenantId != contextTenantID {
		t.Fatalf("expected user tenant_id %d from context, got %d", contextTenantID, user.TenantId)
	}
}
