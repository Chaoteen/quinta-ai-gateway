package model

import (
	"errors"
	"fmt"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
)

func GetTenantReadonlyByID(id int) (ReadonlyTenant, error) {
	var tenant ReadonlyTenant
	err := DB.Model(&Tenant{}).
		Select("id", "name", "status", "created_at").
		Where("id = ?", id).
		First(&tenant).Error
	return tenant, err
}

func GetOrganizationReadonlyByID(id int) (ReadonlyOrganization, error) {
	var organization ReadonlyOrganization
	err := DB.Model(&Organization{}).
		Select("id", "name", "tenant_id", "status", "created_at").
		Where("id = ?", id).
		First(&organization).Error
	return organization, err
}

func GetDepartmentReadonlyByID(id int) (ReadonlyDepartment, error) {
	var department ReadonlyDepartment
	err := DB.Model(&Department{}).
		Select("id", "name", "tenant_id", "organization_id", "status", "created_at").
		Where("id = ?", id).
		First(&department).Error
	return department, err
}

func GetDistributionChannelReadonlyByID(id int) (ReadonlyDistributionChannel, error) {
	var channel ReadonlyDistributionChannel
	err := DB.Model(&DistributionChannel{}).
		Select("id", "name", "code", "tenant_id", "status", "created_at").
		Where("id = ?", id).
		First(&channel).Error
	return channel, err
}

func CreateTenantManagement(name string, status int) (ReadonlyTenant, error) {
	tenant := Tenant{Name: name, Status: status}
	if err := DB.Create(&tenant).Error; err != nil {
		return ReadonlyTenant{}, err
	}
	return GetTenantReadonlyByID(tenant.Id)
}

func UpdateTenantManagement(id int, name string, status int) (ReadonlyTenant, error) {
	result := DB.Model(&Tenant{}).Where("id = ?", id).Updates(map[string]any{
		"name":   name,
		"status": status,
	})
	if result.Error != nil {
		return ReadonlyTenant{}, result.Error
	}
	if result.RowsAffected == 0 {
		return ReadonlyTenant{}, gorm.ErrRecordNotFound
	}
	return GetTenantReadonlyByID(id)
}

func UpdateTenantStatusManagement(id int, status int) (ReadonlyTenant, error) {
	return updateAdminConsoleStatus[Tenant, ReadonlyTenant](id, status, GetTenantReadonlyByID)
}

func CreateOrganizationManagement(name string, tenantId int, status int) (ReadonlyOrganization, error) {
	if err := ensureTenantExists(tenantId); err != nil {
		return ReadonlyOrganization{}, err
	}
	organization := Organization{Name: name, TenantId: tenantId, Status: status}
	if err := DB.Create(&organization).Error; err != nil {
		return ReadonlyOrganization{}, err
	}
	return GetOrganizationReadonlyByID(organization.Id)
}

func UpdateOrganizationManagement(id int, name string, tenantId int, status int) (ReadonlyOrganization, error) {
	if err := ensureTenantExists(tenantId); err != nil {
		return ReadonlyOrganization{}, err
	}

	var organization Organization
	if err := DB.Select("id", "tenant_id").Where("id = ?", id).First(&organization).Error; err != nil {
		return ReadonlyOrganization{}, err
	}
	if organization.TenantId != tenantId {
		if err := ensureNoAdminConsoleReference("organization tenant_id", id, organizationReferenceChecks()); err != nil {
			return ReadonlyOrganization{}, err
		}
	}

	if err := DB.Model(&Organization{}).Where("id = ?", id).Updates(map[string]any{
		"name":      name,
		"tenant_id": tenantId,
		"status":    status,
	}).Error; err != nil {
		return ReadonlyOrganization{}, err
	}
	return GetOrganizationReadonlyByID(id)
}

func UpdateOrganizationStatusManagement(id int, status int) (ReadonlyOrganization, error) {
	return updateAdminConsoleStatus[Organization, ReadonlyOrganization](id, status, GetOrganizationReadonlyByID)
}

func CreateDepartmentManagement(name string, tenantId int, organizationId int, status int) (ReadonlyDepartment, error) {
	if err := ensureOrganizationBelongsToTenant(organizationId, tenantId); err != nil {
		return ReadonlyDepartment{}, err
	}
	department := Department{Name: name, TenantId: tenantId, OrganizationId: organizationId, Status: status}
	if err := DB.Create(&department).Error; err != nil {
		return ReadonlyDepartment{}, err
	}
	return GetDepartmentReadonlyByID(department.Id)
}

func UpdateDepartmentManagement(id int, name string, tenantId int, organizationId int, status int) (ReadonlyDepartment, error) {
	if err := ensureOrganizationBelongsToTenant(organizationId, tenantId); err != nil {
		return ReadonlyDepartment{}, err
	}
	var department Department
	if err := DB.Select("id", "tenant_id", "organization_id").Where("id = ?", id).First(&department).Error; err != nil {
		return ReadonlyDepartment{}, err
	}
	if department.TenantId != tenantId || department.OrganizationId != organizationId {
		if err := ensureNoAdminConsoleReference("department parent ownership", id, departmentReferenceChecks()); err != nil {
			return ReadonlyDepartment{}, err
		}
	}
	result := DB.Model(&Department{}).Where("id = ?", id).Updates(map[string]any{
		"name":            name,
		"tenant_id":       tenantId,
		"organization_id": organizationId,
		"status":          status,
	})
	if result.Error != nil {
		return ReadonlyDepartment{}, result.Error
	}
	if result.RowsAffected == 0 {
		return ReadonlyDepartment{}, gorm.ErrRecordNotFound
	}
	return GetDepartmentReadonlyByID(id)
}

func UpdateDepartmentStatusManagement(id int, status int) (ReadonlyDepartment, error) {
	return updateAdminConsoleStatus[Department, ReadonlyDepartment](id, status, GetDepartmentReadonlyByID)
}

func CreateDistributionChannelManagement(name string, code string, tenantId int, status int) (ReadonlyDistributionChannel, error) {
	if err := ensureTenantExists(tenantId); err != nil {
		return ReadonlyDistributionChannel{}, err
	}
	channel := DistributionChannel{Name: name, Code: code, TenantId: tenantId, Status: status}
	if err := DB.Create(&channel).Error; err != nil {
		return ReadonlyDistributionChannel{}, err
	}
	return GetDistributionChannelReadonlyByID(channel.Id)
}

func UpdateDistributionChannelManagement(id int, name string, code string, tenantId int, status int) (ReadonlyDistributionChannel, error) {
	if err := ensureTenantExists(tenantId); err != nil {
		return ReadonlyDistributionChannel{}, err
	}
	var channel DistributionChannel
	if err := DB.Select("id", "tenant_id").Where("id = ?", id).First(&channel).Error; err != nil {
		return ReadonlyDistributionChannel{}, err
	}
	if channel.TenantId != tenantId {
		if err := ensureNoAdminConsoleReference("distribution_channel tenant_id", id, distributionChannelReferenceChecks()); err != nil {
			return ReadonlyDistributionChannel{}, err
		}
	}
	result := DB.Model(&DistributionChannel{}).Where("id = ?", id).Updates(map[string]any{
		"name":      name,
		"code":      code,
		"tenant_id": tenantId,
		"status":    status,
	})
	if result.Error != nil {
		return ReadonlyDistributionChannel{}, result.Error
	}
	if result.RowsAffected == 0 {
		return ReadonlyDistributionChannel{}, gorm.ErrRecordNotFound
	}
	return GetDistributionChannelReadonlyByID(id)
}

func UpdateDistributionChannelStatusManagement(id int, status int) (ReadonlyDistributionChannel, error) {
	return updateAdminConsoleStatus[DistributionChannel, ReadonlyDistributionChannel](id, status, GetDistributionChannelReadonlyByID)
}

func ensureTenantExists(tenantId int) error {
	if tenantId <= 0 {
		return errors.New("tenant_id is required")
	}
	var tenant Tenant
	if err := DB.Select("id").Where("id = ?", tenantId).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("tenant_id %d does not exist", tenantId)
		}
		return err
	}
	return nil
}

func ensureOrganizationBelongsToTenant(organizationId int, tenantId int) error {
	if organizationId <= 0 {
		return errors.New("organization_id is required")
	}
	if err := ensureTenantExists(tenantId); err != nil {
		return err
	}
	var organization Organization
	if err := DB.Select("id").Where("id = ? AND tenant_id = ?", organizationId, tenantId).First(&organization).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("organization_id %d does not belong to tenant_id %d", organizationId, tenantId)
		}
		return err
	}
	return nil
}

func updateAdminConsoleStatus[T any, R any](id int, status int, reload func(int) (R, error)) (R, error) {
	result := DB.Model(new(T)).Where("id = ?", id).Updates(map[string]any{
		"status": status,
	})
	var empty R
	if result.Error != nil {
		return empty, result.Error
	}
	if result.RowsAffected == 0 {
		return empty, gorm.ErrRecordNotFound
	}
	return reload(id)
}

func DefaultAdminConsoleStatus(status *int) int {
	if status == nil {
		return common.UserStatusEnabled
	}
	return *status
}

type adminConsoleReferenceCheck struct {
	model any
	field string
	label string
}

func ensureNoAdminConsoleReference(change string, id int, checks []adminConsoleReferenceCheck) error {
	for _, check := range checks {
		var count int64
		if err := DB.Model(check.model).Where(check.field+" = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("%s cannot be changed while referenced by %s", change, check.label)
		}
	}
	return nil
}

func organizationReferenceChecks() []adminConsoleReferenceCheck {
	return []adminConsoleReferenceCheck{
		{model: &User{}, field: "organization_id", label: "users.organization_id"},
		{model: &Department{}, field: "organization_id", label: "departments.organization_id"},
		{model: &Channel{}, field: "organization_id", label: "channels.organization_id"},
		{model: &Token{}, field: "organization_id", label: "tokens.organization_id"},
		{model: &TopUp{}, field: "organization_id", label: "top_ups.organization_id"},
		{model: &SubscriptionOrder{}, field: "organization_id", label: "subscription_orders.organization_id"},
		{model: &UserSubscription{}, field: "organization_id", label: "user_subscriptions.organization_id"},
		{model: &SubscriptionPreConsumeRecord{}, field: "organization_id", label: "subscription_pre_consume_records.organization_id"},
		{model: &Redemption{}, field: "organization_id", label: "redemptions.organization_id"},
		{model: &Midjourney{}, field: "organization_id", label: "midjourneys.organization_id"},
		{model: &Task{}, field: "organization_id", label: "tasks.organization_id"},
		{model: &Log{}, field: "organization_id", label: "logs.organization_id"},
		{model: &Ability{}, field: "organization_id", label: "abilities.organization_id"},
	}
}

func departmentReferenceChecks() []adminConsoleReferenceCheck {
	return []adminConsoleReferenceCheck{
		{model: &User{}, field: "department_id", label: "users.department_id"},
		{model: &Channel{}, field: "department_id", label: "channels.department_id"},
		{model: &Token{}, field: "department_id", label: "tokens.department_id"},
		{model: &TopUp{}, field: "department_id", label: "top_ups.department_id"},
		{model: &SubscriptionOrder{}, field: "department_id", label: "subscription_orders.department_id"},
		{model: &UserSubscription{}, field: "department_id", label: "user_subscriptions.department_id"},
		{model: &SubscriptionPreConsumeRecord{}, field: "department_id", label: "subscription_pre_consume_records.department_id"},
		{model: &Redemption{}, field: "department_id", label: "redemptions.department_id"},
		{model: &Midjourney{}, field: "department_id", label: "midjourneys.department_id"},
		{model: &Task{}, field: "department_id", label: "tasks.department_id"},
		{model: &Log{}, field: "department_id", label: "logs.department_id"},
		{model: &Ability{}, field: "department_id", label: "abilities.department_id"},
	}
}

func distributionChannelReferenceChecks() []adminConsoleReferenceCheck {
	return []adminConsoleReferenceCheck{
		{model: &User{}, field: "distribution_channel_id", label: "users.distribution_channel_id"},
		{model: &Organization{}, field: "distribution_channel_id", label: "organizations.distribution_channel_id"},
		{model: &Department{}, field: "distribution_channel_id", label: "departments.distribution_channel_id"},
		{model: &Channel{}, field: "distribution_channel_id", label: "channels.distribution_channel_id"},
		{model: &Token{}, field: "distribution_channel_id", label: "tokens.distribution_channel_id"},
		{model: &TopUp{}, field: "distribution_channel_id", label: "top_ups.distribution_channel_id"},
		{model: &SubscriptionOrder{}, field: "distribution_channel_id", label: "subscription_orders.distribution_channel_id"},
		{model: &UserSubscription{}, field: "distribution_channel_id", label: "user_subscriptions.distribution_channel_id"},
		{model: &SubscriptionPreConsumeRecord{}, field: "distribution_channel_id", label: "subscription_pre_consume_records.distribution_channel_id"},
		{model: &Redemption{}, field: "distribution_channel_id", label: "redemptions.distribution_channel_id"},
		{model: &Midjourney{}, field: "distribution_channel_id", label: "midjourneys.distribution_channel_id"},
		{model: &Task{}, field: "distribution_channel_id", label: "tasks.distribution_channel_id"},
		{model: &Log{}, field: "distribution_channel_id", label: "logs.distribution_channel_id"},
		{model: &Ability{}, field: "distribution_channel_id", label: "abilities.distribution_channel_id"},
	}
}
