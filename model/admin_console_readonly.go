package model

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
	TenantId  int    `json:"tenant_id"`
	Status    int    `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

func GetAllTenantsReadonly(offset, limit int) ([]ReadonlyTenant, int64, error) {
	var total int64
	if err := DB.Model(&Tenant{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tenants []ReadonlyTenant
	err := DB.Model(&Tenant{}).
		Select("id", "name", "status", "created_at").
		Order("id asc").
		Offset(offset).
		Limit(limit).
		Find(&tenants).Error
	return tenants, total, err
}

func GetAllOrganizationsReadonly(offset, limit int) ([]ReadonlyOrganization, int64, error) {
	var total int64
	if err := DB.Model(&Organization{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var organizations []ReadonlyOrganization
	err := DB.Model(&Organization{}).
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
		Select("id", "name", "tenant_id", "status", "created_at").
		Order("tenant_id asc").
		Order("id asc").
		Offset(offset).
		Limit(limit).
		Find(&channels).Error
	return channels, total, err
}
