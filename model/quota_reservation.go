package model

import (
	"errors"
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"gorm.io/gorm"
)

const (
	QuotaReservationStatusActive     = "active"
	QuotaReservationStatusCommitted  = "committed"
	QuotaReservationStatusRolledBack = "rolled_back"
	QuotaReservationStatusExpired    = "expired"
)

// QuotaReservation stores idempotent quota reservation state. It does not
// perform wallet deduction or billing settlement.
type QuotaReservation struct {
	Id                    int    `json:"id"`
	TenantId              int    `json:"tenant_id" gorm:"index;default:1"`
	OrganizationId        int    `json:"organization_id" gorm:"index;default:0"`
	DepartmentId          int    `json:"department_id" gorm:"index;default:0"`
	DistributionChannelId int    `json:"distribution_channel_id" gorm:"index;default:0"`
	ReservationId         string `json:"reservation_id" gorm:"type:varchar(128);uniqueIndex"`
	RequestId             string `json:"request_id" gorm:"type:varchar(128);index"`
	UserId                int    `json:"user_id" gorm:"index"`
	UserSubscriptionId    int    `json:"user_subscription_id" gorm:"index"`
	ModelName             string `json:"model_name" gorm:"type:varchar(255);index"`
	TokenReserved         int64  `json:"token_reserved" gorm:"type:bigint;not null;default:0"`
	RequestReserved       int64  `json:"request_reserved" gorm:"type:bigint;not null;default:0"`
	Status                string `json:"status" gorm:"type:varchar(32);index"`
	ExpiresAt             int64  `json:"expires_at" gorm:"bigint;index"`
	CommittedAt           int64  `json:"committed_at" gorm:"bigint"`
	RolledBackAt          int64  `json:"rolled_back_at" gorm:"bigint"`
	Metadata              string `json:"metadata" gorm:"type:text"`
	CreatedAt             int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt             int64  `json:"updated_at" gorm:"bigint"`
}

func (r *QuotaReservation) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.ReservationId = strings.TrimSpace(r.ReservationId)
	r.RequestId = strings.TrimSpace(r.RequestId)
	r.ModelName = strings.TrimSpace(r.ModelName)
	r.Status = strings.TrimSpace(r.Status)
	if r.Status == "" {
		r.Status = QuotaReservationStatusActive
	}
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *QuotaReservation) BeforeUpdate(tx *gorm.DB) error {
	r.ReservationId = strings.TrimSpace(r.ReservationId)
	r.RequestId = strings.TrimSpace(r.RequestId)
	r.ModelName = strings.TrimSpace(r.ModelName)
	r.Status = strings.TrimSpace(r.Status)
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

type ModelQuotaSnapshot struct {
	Unrestricted bool
	Allow        []string
}

type modelQuotaSnapshotPayload struct {
	Allow []string `json:"allow"`
}

// ParseModelQuotaSnapshot parses the Alpha model entitlement snapshot.
// Empty input means unrestricted access. The parser intentionally supports
// allowlists only and does not implement pricing or billing rules.
func ParseModelQuotaSnapshot(raw string) (ModelQuotaSnapshot, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ModelQuotaSnapshot{Unrestricted: true}, nil
	}

	var payload modelQuotaSnapshotPayload
	if err := common.Unmarshal([]byte(raw), &payload); err != nil {
		return ModelQuotaSnapshot{}, err
	}
	seen := make(map[string]struct{}, len(payload.Allow))
	allow := make([]string, 0, len(payload.Allow))
	for _, modelName := range payload.Allow {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			continue
		}
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		allow = append(allow, modelName)
	}
	if len(allow) == 0 {
		return ModelQuotaSnapshot{}, errors.New("model quota allowlist is empty")
	}
	return ModelQuotaSnapshot{Allow: allow}, nil
}
