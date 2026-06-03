package model

import "strings"

type ReadonlyTenant struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Status    int    `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

type ReadonlyOrganization struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	TenantId  int    `json:"tenant_id"`
	Status    int    `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

type ReadonlyDepartment struct {
	Id             int    `json:"id"`
	Name           string `json:"name"`
	TenantId       int    `json:"tenant_id"`
	OrganizationId int    `json:"organization_id"`
	Status         int    `json:"status"`
	CreatedAt      int64  `json:"created_at"`
}

type ReadonlyDistributionChannel struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	TenantId  int    `json:"tenant_id"`
	Status    int    `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

type AdminConsoleReadonlyFilters struct {
	Keyword  string
	TenantId int
}

func GetAllTenantsReadonly(offset, limit int, filters ...AdminConsoleReadonlyFilters) ([]ReadonlyTenant, int64, error) {
	filter := firstAdminConsoleReadonlyFilter(filters)
	query := DB.Model(&Tenant{})
	if filter.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+filter.Keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tenants []ReadonlyTenant
	err := query.
		Select("id", "name", "status", "created_at").
		Order("id asc").
		Offset(offset).
		Limit(limit).
		Find(&tenants).Error
	return tenants, total, err
}

func GetAllOrganizationsReadonly(offset, limit int, filters ...AdminConsoleReadonlyFilters) ([]ReadonlyOrganization, int64, error) {
	filter := firstAdminConsoleReadonlyFilter(filters)
	query := DB.Model(&Organization{})
	if filter.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+filter.Keyword+"%")
	}
	if filter.TenantId > 0 {
		query = query.Where("tenant_id = ?", filter.TenantId)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var organizations []ReadonlyOrganization
	err := query.
		Select("id", "name", "tenant_id", "status", "created_at").
		Order("tenant_id asc").
		Order("id asc").
		Offset(offset).
		Limit(limit).
		Find(&organizations).Error
	return organizations, total, err
}

func GetAllDepartmentsReadonly(offset, limit int) ([]ReadonlyDepartment, int64, error) {
	var total int64
	if err := DB.Model(&Department{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var departments []ReadonlyDepartment
	err := DB.Model(&Department{}).
		Select("id", "name", "tenant_id", "organization_id", "status", "created_at").
		Order("tenant_id asc").
		Order("organization_id asc").
		Order("id asc").
		Offset(offset).
		Limit(limit).
		Find(&departments).Error
	return departments, total, err
}

func GetAllDistributionChannelsReadonly(offset, limit int) ([]ReadonlyDistributionChannel, int64, error) {
	var total int64
	if err := DB.Model(&DistributionChannel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var channels []ReadonlyDistributionChannel
	err := DB.Model(&DistributionChannel{}).
		Select("id", "name", "code", "tenant_id", "status", "created_at").
		Order("tenant_id asc").
		Order("id asc").
		Offset(offset).
		Limit(limit).
		Find(&channels).Error
	return channels, total, err
}

func firstAdminConsoleReadonlyFilter(filters []AdminConsoleReadonlyFilters) AdminConsoleReadonlyFilters {
	if len(filters) == 0 {
		return AdminConsoleReadonlyFilters{}
	}
	filter := filters[0]
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	return filter
}
