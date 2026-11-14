package model

import (
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
)

type Tenant struct {
	Id        int            `json:"id"`
	Name      string         `json:"name" gorm:"type:varchar(128);not null;uniqueIndex"`
	Status    int            `json:"status" gorm:"type:int;default:1;index"`
	Remark    string         `json:"remark,omitempty" gorm:"type:varchar(255)"`
	CreatedAt int64          `json:"created_at" gorm:"bigint"`
	UpdatedAt int64          `json:"updated_at" gorm:"bigint"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (tenant *Tenant) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	tenant.CreatedAt = now
	tenant.UpdatedAt = now
	return nil
}

func (tenant *Tenant) BeforeUpdate(tx *gorm.DB) error {
	tenant.UpdatedAt = common.GetTimestamp()
	return nil
}

// TenantAccessEnabled reads current state instead of caching authorization.
// Older installations may not have a Tenant row for the implicit platform tenant 1.
// This compatibility exception never applies to explicit disabled/deleted rows.
func TenantAccessEnabled(tenantID int) (bool, error) {
	tenantID = normalizeTenantId(tenantID)
	if tenantID < 1 {
		return false, nil
	}
	var tenant Tenant
	result := DB.Unscoped().Select("id", "status", "deleted_at").Where("id = ?", tenantID).Limit(1).Find(&tenant)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return tenantID == 1, nil
	}
	return tenant.Status == 1 && !tenant.DeletedAt.Valid, nil
}
