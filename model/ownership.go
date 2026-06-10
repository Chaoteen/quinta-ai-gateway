package model

import (
	"fmt"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/gin-gonic/gin"
)

type OwnershipSnapshot struct {
	TenantId              int
	OrganizationId        int
	DepartmentId          int
	DistributionChannelId int
}

// ValidateOwnershipHierarchy validates explicitly requested ownership before
// root creates a resource on behalf of a tenant.
func ValidateOwnershipHierarchy(snapshot OwnershipSnapshot) error {
	if snapshot.OrganizationId != 0 {
		if snapshot.TenantId == 0 {
			return fmt.Errorf("organization_id requires an explicit tenant_id")
		}
		var organization Organization
		if err := DB.Select("id").Where("id = ? AND tenant_id = ?", snapshot.OrganizationId, snapshot.TenantId).First(&organization).Error; err != nil {
			return fmt.Errorf("organization_id does not belong to tenant_id")
		}
	}

	if snapshot.DepartmentId != 0 {
		if snapshot.TenantId == 0 || snapshot.OrganizationId == 0 {
			return fmt.Errorf("department_id requires explicit tenant_id and organization_id")
		}
		var department Department
		if err := DB.Select("id").Where(
			"id = ? AND tenant_id = ? AND organization_id = ?",
			snapshot.DepartmentId,
			snapshot.TenantId,
			snapshot.OrganizationId,
		).First(&department).Error; err != nil {
			return fmt.Errorf("department_id does not belong to organization_id and tenant_id")
		}
	}

	if snapshot.DistributionChannelId != 0 {
		if snapshot.TenantId == 0 {
			return fmt.Errorf("distribution_channel_id requires an explicit tenant_id")
		}
		var channel DistributionChannel
		if err := DB.Select("id").Where("id = ? AND tenant_id = ?", snapshot.DistributionChannelId, snapshot.TenantId).First(&channel).Error; err != nil {
			return fmt.Errorf("distribution_channel_id does not belong to tenant_id")
		}
	}

	return nil
}

func NormalizeOwnership(snapshot OwnershipSnapshot) OwnershipSnapshot {
	if snapshot.TenantId == 0 {
		snapshot.TenantId = 1
	}
	return snapshot
}

func OwnershipFromContext(c *gin.Context) OwnershipSnapshot {
	if c == nil {
		return NormalizeOwnership(OwnershipSnapshot{})
	}
	return NormalizeOwnership(OwnershipSnapshot{
		TenantId:              common.GetContextKeyInt(c, constant.ContextKeyTenantId),
		OrganizationId:        common.GetContextKeyInt(c, constant.ContextKeyOrganizationId),
		DepartmentId:          common.GetContextKeyInt(c, constant.ContextKeyDepartmentId),
		DistributionChannelId: common.GetContextKeyInt(c, constant.ContextKeyDistributionChannelId),
	})
}

func OwnershipByUserId(userId int) OwnershipSnapshot {
	if userId <= 0 {
		return NormalizeOwnership(OwnershipSnapshot{})
	}
	userCache, err := GetUserCache(userId)
	if err != nil || userCache == nil {
		return NormalizeOwnership(OwnershipSnapshot{})
	}
	return NormalizeOwnership(OwnershipSnapshot{
		TenantId:              userCache.TenantId,
		OrganizationId:        userCache.OrganizationId,
		DepartmentId:          userCache.DepartmentId,
		DistributionChannelId: userCache.DistributionChannelId,
	})
}

func RequiredOwnershipByUserId(userId int) (OwnershipSnapshot, error) {
	if userId <= 0 {
		return OwnershipSnapshot{}, fmt.Errorf("invalid user id %d for ownership", userId)
	}
	userCache, err := GetUserCache(userId)
	if err != nil {
		return OwnershipSnapshot{}, fmt.Errorf("get ownership for user %d: %w", userId, err)
	}
	if userCache == nil {
		return OwnershipSnapshot{}, fmt.Errorf("ownership user %d not found", userId)
	}
	return NormalizeOwnership(OwnershipSnapshot{
		TenantId:              userCache.TenantId,
		OrganizationId:        userCache.OrganizationId,
		DepartmentId:          userCache.DepartmentId,
		DistributionChannelId: userCache.DistributionChannelId,
	}), nil
}

func OwnershipFromChannel(channel *Channel) OwnershipSnapshot {
	if channel == nil {
		return NormalizeOwnership(OwnershipSnapshot{})
	}
	return NormalizeOwnership(OwnershipSnapshot{
		TenantId:              channel.TenantId,
		OrganizationId:        channel.OrganizationId,
		DepartmentId:          channel.DepartmentId,
		DistributionChannelId: channel.DistributionChannelId,
	})
}

func ownershipFromSubscriptionOrder(order *SubscriptionOrder) OwnershipSnapshot {
	if order == nil {
		return NormalizeOwnership(OwnershipSnapshot{})
	}
	return NormalizeOwnership(OwnershipSnapshot{
		TenantId:              order.TenantId,
		OrganizationId:        order.OrganizationId,
		DepartmentId:          order.DepartmentId,
		DistributionChannelId: order.DistributionChannelId,
	})
}

func ownershipFromUserSubscription(subscription *UserSubscription) OwnershipSnapshot {
	if subscription == nil {
		return NormalizeOwnership(OwnershipSnapshot{})
	}
	return NormalizeOwnership(OwnershipSnapshot{
		TenantId:              subscription.TenantId,
		OrganizationId:        subscription.OrganizationId,
		DepartmentId:          subscription.DepartmentId,
		DistributionChannelId: subscription.DistributionChannelId,
	})
}

func ApplyOwnershipFromContext(c *gin.Context, target any) {
	OwnershipFromContext(c).ApplyTo(target)
}

func ApplyOwnershipFromUser(userId int, target any) {
	OwnershipByUserId(userId).ApplyTo(target)
}

func (snapshot OwnershipSnapshot) ApplyTo(target any) {
	snapshot = NormalizeOwnership(snapshot)
	switch v := target.(type) {
	case *User:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *Token:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *Channel:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *Ability:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *Task:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *Log:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *TopUp:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *Redemption:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *SubscriptionOrder:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *UserSubscription:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *SubscriptionPreConsumeRecord:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *BillingRecord:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *RevenueShareRule:
		v.TenantId = snapshot.TenantId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *RevenueShareRecord:
		v.TenantId = snapshot.TenantId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *PaymentOrder:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *BankTransferRecord:
		v.TenantId = snapshot.TenantId
	case *VoucherBatch:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *VoucherRedemption:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	case *Midjourney:
		v.TenantId = snapshot.TenantId
		v.OrganizationId = snapshot.OrganizationId
		v.DepartmentId = snapshot.DepartmentId
		v.DistributionChannelId = snapshot.DistributionChannelId
	}
}
