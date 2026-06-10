package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"gorm.io/gorm"
)

var (
	ErrVoucherInvalidBatch       = errors.New("invalid voucher batch")
	ErrVoucherInvalidType        = errors.New("invalid voucher type")
	ErrVoucherInvalidFulfillment = errors.New("invalid voucher fulfillment")
	ErrVoucherNotFound           = errors.New("voucher not found")
	ErrVoucherExpired            = errors.New("voucher expired")
	ErrVoucherDisabled           = errors.New("voucher disabled")
	ErrVoucherAlreadyRedeemed    = errors.New("voucher already redeemed")
	ErrVoucherOutOfScope         = errors.New("voucher is outside access scope")
)

type VoucherService struct{}

func NewVoucherService() *VoucherService {
	return &VoucherService{}
}

type VoucherActor struct {
	UserId int
	Scope  model.AccessScope
}

type CreateVoucherBatchInput struct {
	Name                  string
	Description           string
	VoucherType           string
	Quantity              int
	Status                string
	TenantId              int
	OrganizationId        int
	DepartmentId          int
	DistributionChannelId int
	CreatedBy             int
}

type GenerateVouchersInput struct {
	BatchId            int
	Quantity           int
	QuotaAmount        int64
	SubscriptionPlanId int
	ExpiredAt          int64
	Codes              []string
}

type VoucherListInput struct {
	Page        int
	PageSize    int
	Keyword     string
	Status      string
	VoucherType string
	BatchId     int
	StartTime   int64
	EndTime     int64
	UserId      int
}

type VoucherPage[T any] struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
	Items    []T   `json:"items"`
}

func (s *VoucherService) CreateVoucherBatch(ctx context.Context, actor VoucherActor, input CreateVoucherBatchInput) (*model.VoucherBatch, error) {
	voucherType := model.NormalizeVoucherType(input.VoucherType)
	if voucherType != model.VoucherTypeToken && voucherType != model.VoucherTypeSubscription {
		return nil, ErrVoucherInvalidType
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, ErrVoucherInvalidBatch
	}
	ownership := model.NormalizeOwnership(model.OwnershipSnapshot{
		TenantId:              input.TenantId,
		OrganizationId:        input.OrganizationId,
		DepartmentId:          input.DepartmentId,
		DistributionChannelId: input.DistributionChannelId,
	})
	if !actor.Scope.IsRoot {
		ownership = model.NormalizeOwnership(model.OwnershipSnapshot{
			TenantId:              actor.Scope.TenantId,
			OrganizationId:        actor.Scope.OrganizationId,
			DepartmentId:          actor.Scope.DepartmentId,
			DistributionChannelId: input.DistributionChannelId,
		})
	}
	if err := model.ValidateOwnershipHierarchy(ownership); err != nil {
		return nil, err
	}
	if !model.AllowsOwnership(actor.Scope, ownership.TenantId, ownership.OrganizationId, ownership.DepartmentId) {
		return nil, ErrVoucherOutOfScope
	}
	status := model.NormalizeVoucherBatchStatus(input.Status)
	if strings.TrimSpace(input.Status) == "" {
		status = model.VoucherBatchStatusActive
	}
	batch := &model.VoucherBatch{
		BatchNo:     generateVoucherBatchNo(input.CreatedBy),
		Name:        input.Name,
		Description: input.Description,
		VoucherType: voucherType,
		Quantity:    input.Quantity,
		Status:      status,
		CreatedBy:   input.CreatedBy,
	}
	ownership.ApplyTo(batch)
	if err := model.DB.WithContext(ctx).Create(batch).Error; err != nil {
		return nil, err
	}
	return batch, nil
}

func (s *VoucherService) GenerateVouchers(ctx context.Context, actor VoucherActor, input GenerateVouchersInput) ([]model.Voucher, error) {
	if input.BatchId <= 0 || input.Quantity < 0 {
		return nil, ErrVoucherInvalidBatch
	}
	if len(input.Codes) > 0 {
		input.Quantity = len(input.Codes)
	}
	if input.Quantity <= 0 {
		return nil, ErrVoucherInvalidBatch
	}

	var vouchers []model.Voucher
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		batch, err := s.getBatchForUpdateTx(tx, input.BatchId, actor.Scope)
		if err != nil {
			return err
		}
		if batch.Status == model.VoucherBatchStatusDisabled {
			return ErrVoucherDisabled
		}
		if err := validateVoucherFulfillmentTx(tx, batch.VoucherType, input.QuotaAmount, input.SubscriptionPlanId); err != nil {
			return err
		}
		codes, err := buildVoucherCodesTx(tx, input.Codes, input.Quantity)
		if err != nil {
			return err
		}
		vouchers = make([]model.Voucher, 0, input.Quantity)
		for _, code := range codes {
			vouchers = append(vouchers, model.Voucher{
				BatchId:            batch.Id,
				VoucherCode:        code,
				VoucherType:        batch.VoucherType,
				QuotaAmount:        input.QuotaAmount,
				SubscriptionPlanId: input.SubscriptionPlanId,
				Status:             model.VoucherStatusUnused,
				ExpiredAt:          input.ExpiredAt,
			})
		}
		if err := tx.Create(&vouchers).Error; err != nil {
			return err
		}
		return tx.Model(&model.VoucherBatch{}).
			Where("id = ?", batch.Id).
			Update("quantity", gorm.Expr("quantity + ?", len(vouchers))).Error
	})
	if err != nil {
		return nil, err
	}
	return vouchers, nil
}

func (s *VoucherService) RedeemVoucher(ctx context.Context, userId int, code string) (*model.VoucherRedemption, error) {
	code = model.NormalizeVoucherCode(code)
	if userId <= 0 || code == "" {
		return nil, ErrVoucherNotFound
	}
	var out model.VoucherRedemption
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var voucher model.Voucher
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("voucher_code = ?", code).First(&voucher).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrVoucherNotFound
			}
			return err
		}
		var batch model.VoucherBatch
		if err := tx.Where("id = ?", voucher.BatchId).First(&batch).Error; err != nil {
			return err
		}
		if batch.Status == model.VoucherBatchStatusDisabled || voucher.Status == model.VoucherStatusDisabled {
			return ErrVoucherDisabled
		}
		now := common.GetTimestamp()
		if voucher.ExpiredAt > 0 && voucher.ExpiredAt < now {
			if voucher.Status == model.VoucherStatusUnused {
				_ = tx.Model(&model.Voucher{}).Where("id = ?", voucher.Id).Update("status", model.VoucherStatusExpired).Error
			}
			return ErrVoucherExpired
		}
		if voucher.Status == model.VoucherStatusRedeemed {
			var existing model.VoucherRedemption
			err := tx.Where("voucher_id = ? AND redemption_result = ?", voucher.Id, model.VoucherRedemptionResultSuccess).
				Order("id desc").First(&existing).Error
			if err == nil && existing.UserId == userId {
				out = existing
				return nil
			}
			return ErrVoucherAlreadyRedeemed
		}
		if voucher.Status != model.VoucherStatusUnused {
			return ErrVoucherAlreadyRedeemed
		}
		ownership, err := getVoucherUserOwnershipTx(tx, userId)
		if err != nil {
			return err
		}
		switch voucher.VoucherType {
		case model.VoucherTypeToken:
			if voucher.QuotaAmount <= 0 {
				return ErrVoucherInvalidFulfillment
			}
			if err := tx.Model(&model.User{}).Where("id = ?", userId).
				Update("quota", gorm.Expr("quota + ?", voucher.QuotaAmount)).Error; err != nil {
				return err
			}
		case model.VoucherTypeSubscription:
			if voucher.SubscriptionPlanId <= 0 {
				return ErrVoucherInvalidFulfillment
			}
			plan, err := getVoucherSubscriptionPlanTx(tx, voucher.SubscriptionPlanId)
			if err != nil {
				return err
			}
			if _, err := model.CreateUserSubscriptionFromPlanWithOwnershipTx(tx, userId, plan, "voucher", ownership); err != nil {
				return err
			}
		default:
			return ErrVoucherInvalidType
		}
		voucher.Status = model.VoucherStatusRedeemed
		voucher.ActivatedBy = userId
		voucher.ActivatedAt = now
		if err := tx.Save(&voucher).Error; err != nil {
			return err
		}
		redemption := model.VoucherRedemption{
			VoucherId:        voucher.Id,
			VoucherCode:      voucher.VoucherCode,
			UserId:           userId,
			RedemptionType:   voucher.VoucherType,
			RedemptionResult: model.VoucherRedemptionResultSuccess,
		}
		ownership.ApplyTo(&redemption)
		if err := tx.Create(&redemption).Error; err != nil {
			return err
		}
		out = redemption
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *VoucherService) DisableVoucher(ctx context.Context, actor VoucherActor, voucherId int) (*model.Voucher, error) {
	var voucher model.Voucher
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", voucherId).First(&voucher).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrVoucherNotFound
			}
			return err
		}
		if _, err := s.getBatchForUpdateTx(tx, voucher.BatchId, actor.Scope); err != nil {
			return err
		}
		if voucher.Status == model.VoucherStatusRedeemed {
			return ErrVoucherAlreadyRedeemed
		}
		voucher.Status = model.VoucherStatusDisabled
		return tx.Save(&voucher).Error
	})
	if err != nil {
		return nil, err
	}
	return &voucher, nil
}

func (s *VoucherService) DisableBatch(ctx context.Context, actor VoucherActor, batchId int) (*model.VoucherBatch, error) {
	var batch *model.VoucherBatch
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := s.getBatchForUpdateTx(tx, batchId, actor.Scope)
		if err != nil {
			return err
		}
		locked.Status = model.VoucherBatchStatusDisabled
		if err := tx.Save(locked).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Voucher{}).
			Where("batch_id = ? AND status = ?", locked.Id, model.VoucherStatusUnused).
			Update("status", model.VoucherStatusDisabled).Error; err != nil {
			return err
		}
		batch = locked
		return nil
	})
	if err != nil {
		return nil, err
	}
	return batch, nil
}

func (s *VoucherService) ListBatches(ctx context.Context, actor VoucherActor, input VoucherListInput) (VoucherPage[model.VoucherBatch], error) {
	var page VoucherPage[model.VoucherBatch]
	setVoucherPageDefaults(&input)
	page.Page = input.Page
	page.PageSize = input.PageSize
	query := voucherScopedQuery(model.DB.WithContext(ctx).Model(&model.VoucherBatch{}), "voucher_batches", actor.Scope)
	if input.Keyword != "" {
		kw := "%" + strings.TrimSpace(input.Keyword) + "%"
		query = query.Where("name LIKE ? OR batch_no LIKE ?", kw, kw)
	}
	if input.Status != "" {
		query = query.Where("status = ?", model.NormalizeVoucherBatchStatus(input.Status))
	}
	if input.VoucherType != "" {
		query = query.Where("voucher_type = ?", model.NormalizeVoucherType(input.VoucherType))
	}
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.VoucherBatch
	if err := query.Order("id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func (s *VoucherService) ListVouchers(ctx context.Context, actor VoucherActor, input VoucherListInput) (VoucherPage[model.Voucher], error) {
	var page VoucherPage[model.Voucher]
	setVoucherPageDefaults(&input)
	page.Page = input.Page
	page.PageSize = input.PageSize
	query := model.DB.WithContext(ctx).Model(&model.Voucher{}).
		Joins("JOIN voucher_batches ON voucher_batches.id = vouchers.batch_id")
	query = voucherScopedQuery(query, "voucher_batches", actor.Scope)
	if input.BatchId > 0 {
		query = query.Where("vouchers.batch_id = ?", input.BatchId)
	}
	if input.Keyword != "" {
		query = query.Where("vouchers.voucher_code LIKE ?", "%"+strings.TrimSpace(input.Keyword)+"%")
	}
	if input.Status != "" {
		query = query.Where("vouchers.status = ?", model.NormalizeVoucherStatus(input.Status))
	}
	if input.VoucherType != "" {
		query = query.Where("vouchers.voucher_type = ?", model.NormalizeVoucherType(input.VoucherType))
	}
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.Voucher
	if err := query.Order("vouchers.id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func (s *VoucherService) ListRedemptions(ctx context.Context, actor VoucherActor, input VoucherListInput) (VoucherPage[model.VoucherRedemption], error) {
	var page VoucherPage[model.VoucherRedemption]
	setVoucherPageDefaults(&input)
	page.Page = input.Page
	page.PageSize = input.PageSize
	query := voucherScopedQuery(model.DB.WithContext(ctx).Model(&model.VoucherRedemption{}), "voucher_redemptions", actor.Scope)
	if !actor.Scope.IsRoot && !common.IsTenantAdminRole(actor.Scope.RoleKey) && actor.Scope.RoleKey != common.RoleKeyFinance {
		query = query.Where("voucher_redemptions.user_id = ?", actor.UserId)
	}
	if input.UserId > 0 {
		query = query.Where("voucher_redemptions.user_id = ?", input.UserId)
	}
	if input.Keyword != "" {
		query = query.Where("voucher_redemptions.voucher_code LIKE ?", "%"+strings.TrimSpace(input.Keyword)+"%")
	}
	if input.VoucherType != "" {
		query = query.Where("voucher_redemptions.redemption_type = ?", model.NormalizeVoucherType(input.VoucherType))
	}
	if input.Status != "" {
		query = query.Where("voucher_redemptions.redemption_result = ?", model.NormalizeVoucherRedemptionResult(input.Status))
	}
	if input.StartTime > 0 {
		query = query.Where("voucher_redemptions.created_at >= ?", input.StartTime)
	}
	if input.EndTime > 0 {
		query = query.Where("voucher_redemptions.created_at <= ?", input.EndTime)
	}
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.VoucherRedemption
	if err := query.Order("voucher_redemptions.id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func (s *VoucherService) getBatchForUpdateTx(tx *gorm.DB, batchId int, scope model.AccessScope) (*model.VoucherBatch, error) {
	var batch model.VoucherBatch
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", batchId).First(&batch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVoucherInvalidBatch
		}
		return nil, err
	}
	if !model.AllowsOwnership(scope, batch.TenantId, batch.OrganizationId, batch.DepartmentId) {
		return nil, ErrVoucherOutOfScope
	}
	return &batch, nil
}

func validateVoucherFulfillmentTx(tx *gorm.DB, voucherType string, quotaAmount int64, planId int) error {
	switch voucherType {
	case model.VoucherTypeToken:
		if quotaAmount <= 0 {
			return ErrVoucherInvalidFulfillment
		}
	case model.VoucherTypeSubscription:
		if planId <= 0 {
			return ErrVoucherInvalidFulfillment
		}
		if _, err := getVoucherSubscriptionPlanTx(tx, planId); err != nil {
			return err
		}
	default:
		return ErrVoucherInvalidType
	}
	return nil
}

func buildVoucherCodesTx(tx *gorm.DB, requested []string, quantity int) ([]string, error) {
	codes := make([]string, 0, quantity)
	seen := map[string]struct{}{}
	for _, raw := range requested {
		code := model.NormalizeVoucherCode(raw)
		if code == "" {
			return nil, ErrVoucherNotFound
		}
		if _, ok := seen[code]; ok {
			return nil, ErrVoucherAlreadyRedeemed
		}
		var count int64
		if err := tx.Model(&model.Voucher{}).Where("voucher_code = ?", code).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrVoucherAlreadyRedeemed
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	for len(codes) < quantity {
		code := generateVoucherCode()
		if _, ok := seen[code]; ok {
			continue
		}
		var count int64
		if err := tx.Model(&model.Voucher{}).Where("voucher_code = ?", code).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	return codes, nil
}

func getVoucherSubscriptionPlanTx(tx *gorm.DB, planId int) (*model.SubscriptionPlan, error) {
	var plan model.SubscriptionPlan
	if err := tx.Where("id = ?", planId).First(&plan).Error; err != nil {
		return nil, err
	}
	plan.NormalizeAlphaFields()
	return &plan, nil
}

func getVoucherUserOwnershipTx(tx *gorm.DB, userId int) (model.OwnershipSnapshot, error) {
	var user model.User
	if err := tx.Select("tenant_id", "organization_id", "department_id", "distribution_channel_id").
		Where("id = ?", userId).First(&user).Error; err != nil {
		return model.OwnershipSnapshot{}, err
	}
	return model.NormalizeOwnership(model.OwnershipSnapshot{
		TenantId:              user.TenantId,
		OrganizationId:        user.OrganizationId,
		DepartmentId:          user.DepartmentId,
		DistributionChannelId: user.DistributionChannelId,
	}), nil
}

func voucherScopedQuery(db *gorm.DB, table string, scope model.AccessScope) *gorm.DB {
	return model.ApplyOwnershipScope(db, table, scope)
}

func setVoucherPageDefaults(input *VoucherListInput) {
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 20
	}
	if input.PageSize > 100 {
		input.PageSize = 100
	}
}

func generateVoucherBatchNo(userId int) string {
	return fmt.Sprintf("VB%d%s%s", userId, time.Now().Format("20060102150405"), strings.ToUpper(common.GetRandomString(6)))
}

func generateVoucherCode() string {
	return "VCH" + strings.ToUpper(common.GetRandomString(16))
}
