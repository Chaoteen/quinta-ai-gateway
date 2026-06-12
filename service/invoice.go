package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrInvoiceProfileInvalid      = errors.New("invalid invoice profile")
	ErrInvoiceProfileNotFound     = errors.New("invoice profile not found")
	ErrInvoiceApplicationInvalid  = errors.New("invalid invoice application")
	ErrInvoiceApplicationNotFound = errors.New("invoice application not found")
	ErrInvoiceSourceInvalid       = errors.New("invalid invoice source")
	ErrInvoiceAmountExceeded      = errors.New("invoice amount exceeds available amount")
	ErrInvoiceStatusInvalid       = errors.New("invoice status invalid")
	ErrInvoiceOutOfScope          = errors.New("invoice is outside access scope")
)

type InvoiceService struct{}

func NewInvoiceService() *InvoiceService {
	return &InvoiceService{}
}

type InvoiceActor struct {
	UserId int
	Scope  model.AccessScope
}

type CreateInvoiceProfileInput struct {
	UserId           int
	ProfileType      string
	Title            string
	TaxNo            string
	BankName         string
	BankAccount      string
	CompanyAddress   string
	CompanyPhone     string
	RecipientName    string
	RecipientPhone   string
	RecipientEmail   string
	RecipientAddress string
	IsDefault        bool
}

type CreateInvoiceApplicationInput struct {
	InvoiceProfileId int
	Amount           float64
	Currency         string
	InvoiceType      string
	SourceType       string
	SourceId         int
}

type ReviewInvoiceApplicationInput struct {
	ReviewerId int
	Approved   bool
	ReviewNote string
}

type IssueInvoiceInput struct {
	UploadedBy  int
	InvoiceNo   string
	InvoiceDate int64
	FileName    string
	FileUrl     string
	FileType    string
}

type InvoiceListInput struct {
	Page      int
	PageSize  int
	Status    string
	UserId    int
	SourceId  int
	Keyword   string
	StartTime int64
	EndTime   int64
}

type InvoicePage[T any] struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
	Items    []T   `json:"items"`
}

func (s *InvoiceService) CreateInvoiceProfile(ctx context.Context, actor InvoiceActor, input CreateInvoiceProfileInput) (*model.InvoiceProfile, error) {
	if actor.UserId <= 0 {
		return nil, ErrInvoiceProfileInvalid
	}
	userId := input.UserId
	if userId <= 0 {
		userId = actor.UserId
	}
	if !invoiceCanManageUser(actor, userId) {
		return nil, ErrInvoiceOutOfScope
	}
	profileType := model.NormalizeInvoiceProfileType(input.ProfileType)
	title := strings.TrimSpace(input.Title)
	taxNo := strings.TrimSpace(input.TaxNo)
	if title == "" || (profileType == model.InvoiceProfileTypeCompany && taxNo == "") {
		return nil, ErrInvoiceProfileInvalid
	}
	ownership, err := invoiceOwnershipByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}
	if !model.AllowsOwnership(actor.Scope, ownership.TenantId, ownership.OrganizationId, ownership.DepartmentId) {
		return nil, ErrInvoiceOutOfScope
	}
	profile := &model.InvoiceProfile{
		UserId:           userId,
		ProfileType:      profileType,
		Title:            title,
		TaxNo:            taxNo,
		BankName:         input.BankName,
		BankAccount:      input.BankAccount,
		CompanyAddress:   input.CompanyAddress,
		CompanyPhone:     input.CompanyPhone,
		RecipientName:    input.RecipientName,
		RecipientPhone:   input.RecipientPhone,
		RecipientEmail:   input.RecipientEmail,
		RecipientAddress: input.RecipientAddress,
		IsDefault:        input.IsDefault,
		Status:           model.InvoiceProfileStatusActive,
	}
	ownership.ApplyTo(profile)
	err = model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if profile.IsDefault {
			if err := tx.Model(&model.InvoiceProfile{}).Where("user_id = ?", userId).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(profile).Error
	})
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *InvoiceService) ListInvoiceProfiles(ctx context.Context, actor InvoiceActor, input InvoiceListInput) (InvoicePage[model.InvoiceProfile], error) {
	var page InvoicePage[model.InvoiceProfile]
	setInvoicePageDefaults(&input)
	page.Page, page.PageSize = input.Page, input.PageSize
	query := invoiceScopedQuery(model.DB.WithContext(ctx).Model(&model.InvoiceProfile{}), "invoice_profiles", actor.Scope)
	if !invoiceIsAdmin(actor) {
		query = query.Where("invoice_profiles.user_id = ?", actor.UserId)
	}
	if input.UserId > 0 {
		query = query.Where("invoice_profiles.user_id = ?", input.UserId)
	}
	if input.Status != "" {
		query = query.Where("invoice_profiles.status = ?", model.NormalizeInvoiceProfileStatus(input.Status))
	}
	if input.Keyword != "" {
		kw := "%" + strings.TrimSpace(input.Keyword) + "%"
		query = query.Where("invoice_profiles.title LIKE ? OR invoice_profiles.tax_no LIKE ?", kw, kw)
	}
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.InvoiceProfile
	if err := query.Order("invoice_profiles.id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func (s *InvoiceService) DisableInvoiceProfile(ctx context.Context, actor InvoiceActor, profileId int) (*model.InvoiceProfile, error) {
	var profile model.InvoiceProfile
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", profileId).First(&profile).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceProfileNotFound
			}
			return err
		}
		if !invoiceCanAccessProfile(actor, &profile) {
			return ErrInvoiceOutOfScope
		}
		profile.Status = model.InvoiceProfileStatusDisabled
		profile.IsDefault = false
		return tx.Save(&profile).Error
	})
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *InvoiceService) CreateInvoiceApplication(ctx context.Context, actor InvoiceActor, input CreateInvoiceApplicationInput) (*model.InvoiceApplication, error) {
	if actor.UserId <= 0 || input.InvoiceProfileId <= 0 || input.Amount <= 0 {
		return nil, ErrInvoiceApplicationInvalid
	}
	sourceType := model.NormalizeInvoiceSourceType(input.SourceType)
	if sourceType != model.InvoiceSourcePaymentOrder || input.SourceId <= 0 {
		return nil, ErrInvoiceSourceInvalid
	}
	var out model.InvoiceApplication
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", input.SourceId).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceSourceInvalid
			}
			return err
		}
		if order.Status != model.PaymentOrderStatusPaid {
			return ErrInvoiceSourceInvalid
		}
		if !invoiceIsAdmin(actor) && order.UserId != actor.UserId {
			return ErrInvoiceOutOfScope
		}
		if !model.AllowsOwnership(actor.Scope, order.TenantId, order.OrganizationId, order.DepartmentId) {
			return ErrInvoiceOutOfScope
		}
		var profile model.InvoiceProfile
		if err := tx.Where("id = ? AND user_id = ? AND status = ?", input.InvoiceProfileId, order.UserId, model.InvoiceProfileStatusActive).First(&profile).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceProfileNotFound
			}
			return err
		}
		if !model.AllowsOwnership(actor.Scope, profile.TenantId, profile.OrganizationId, profile.DepartmentId) {
			return ErrInvoiceOutOfScope
		}
		if input.Currency != "" && strings.ToUpper(strings.TrimSpace(input.Currency)) != order.Currency {
			return ErrInvoiceApplicationInvalid
		}
		available, err := availableInvoiceAmountTx(tx, order.Id, order.Amount, 0)
		if err != nil {
			return err
		}
		if decimal.NewFromFloat(input.Amount).GreaterThan(available) {
			return ErrInvoiceAmountExceeded
		}
		app := model.InvoiceApplication{
			ApplicationNo:    generateInvoiceApplicationNo(order.UserId),
			UserId:           order.UserId,
			InvoiceProfileId: profile.Id,
			Amount:           input.Amount,
			Currency:         order.Currency,
			InvoiceType:      input.InvoiceType,
			Status:           model.InvoiceStatusPending,
			SourceType:       sourceType,
			SourceId:         order.Id,
		}
		model.NormalizeOwnership(model.OwnershipSnapshot{
			TenantId:              order.TenantId,
			OrganizationId:        order.OrganizationId,
			DepartmentId:          order.DepartmentId,
			DistributionChannelId: order.DistributionChannelId,
		}).ApplyTo(&app)
		if err := tx.Create(&app).Error; err != nil {
			return err
		}
		out = app
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *InvoiceService) ReviewInvoiceApplication(ctx context.Context, actor InvoiceActor, applicationId int, input ReviewInvoiceApplicationInput) (*model.InvoiceApplication, error) {
	if !invoiceCanReview(actor) {
		return nil, ErrInvoiceOutOfScope
	}
	var out model.InvoiceApplication
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app model.InvoiceApplication
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", applicationId).First(&app).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceApplicationNotFound
			}
			return err
		}
		if !model.AllowsOwnership(actor.Scope, app.TenantId, app.OrganizationId, app.DepartmentId) {
			return ErrInvoiceOutOfScope
		}
		if app.Status != model.InvoiceStatusPending {
			return ErrInvoiceStatusInvalid
		}
		if input.Approved {
			var order model.PaymentOrder
			if err := tx.Where("id = ? AND status = ?", app.SourceId, model.PaymentOrderStatusPaid).First(&order).Error; err != nil {
				return ErrInvoiceSourceInvalid
			}
			available, err := availableInvoiceAmountTx(tx, order.Id, order.Amount, app.Id)
			if err != nil {
				return err
			}
			if decimal.NewFromFloat(app.Amount).GreaterThan(available) {
				return ErrInvoiceAmountExceeded
			}
			app.Status = model.InvoiceStatusApproved
		} else {
			app.Status = model.InvoiceStatusRejected
		}
		app.ReviewerId = input.ReviewerId
		app.ReviewedAt = common.GetTimestamp()
		app.ReviewNote = input.ReviewNote
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		out = app
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *InvoiceService) IssueInvoice(ctx context.Context, actor InvoiceActor, applicationId int, input IssueInvoiceInput) (*model.InvoiceApplication, error) {
	if !invoiceCanReview(actor) {
		return nil, ErrInvoiceOutOfScope
	}
	if strings.TrimSpace(input.FileName) == "" || strings.TrimSpace(input.FileUrl) == "" || strings.TrimSpace(input.InvoiceNo) == "" {
		return nil, ErrInvoiceApplicationInvalid
	}
	var out model.InvoiceApplication
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app model.InvoiceApplication
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", applicationId).First(&app).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceApplicationNotFound
			}
			return err
		}
		if !model.AllowsOwnership(actor.Scope, app.TenantId, app.OrganizationId, app.DepartmentId) {
			return ErrInvoiceOutOfScope
		}
		if app.Status != model.InvoiceStatusApproved {
			return ErrInvoiceStatusInvalid
		}
		now := common.GetTimestamp()
		app.Status = model.InvoiceStatusIssued
		app.InvoiceNo = input.InvoiceNo
		app.InvoiceDate = input.InvoiceDate
		if app.InvoiceDate == 0 {
			app.InvoiceDate = now
		}
		app.IssuedAt = now
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		file := model.InvoiceFile{
			InvoiceApplicationId: app.Id,
			FileName:             input.FileName,
			FileUrl:              input.FileUrl,
			FileType:             input.FileType,
			UploadedBy:           input.UploadedBy,
		}
		if err := tx.Create(&file).Error; err != nil {
			return err
		}
		out = app
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *InvoiceService) ListInvoiceApplications(ctx context.Context, actor InvoiceActor, input InvoiceListInput) (InvoicePage[model.InvoiceApplication], error) {
	var page InvoicePage[model.InvoiceApplication]
	setInvoicePageDefaults(&input)
	page.Page, page.PageSize = input.Page, input.PageSize
	query := invoiceScopedQuery(model.DB.WithContext(ctx).Model(&model.InvoiceApplication{}), "invoice_applications", actor.Scope)
	if !invoiceIsAdmin(actor) {
		query = query.Where("invoice_applications.user_id = ?", actor.UserId)
	}
	if input.UserId > 0 {
		query = query.Where("invoice_applications.user_id = ?", input.UserId)
	}
	if input.SourceId > 0 {
		query = query.Where("invoice_applications.source_id = ?", input.SourceId)
	}
	if input.Status != "" {
		query = query.Where("invoice_applications.status = ?", model.NormalizeInvoiceStatus(input.Status))
	}
	if input.StartTime > 0 {
		query = query.Where("invoice_applications.created_at >= ?", input.StartTime)
	}
	if input.EndTime > 0 {
		query = query.Where("invoice_applications.created_at <= ?", input.EndTime)
	}
	if input.Keyword != "" {
		kw := "%" + strings.TrimSpace(input.Keyword) + "%"
		query = query.Where("invoice_applications.application_no LIKE ? OR invoice_applications.invoice_no LIKE ?", kw, kw)
	}
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.InvoiceApplication
	if err := query.Order("invoice_applications.id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func (s *InvoiceService) ListInvoiceFiles(ctx context.Context, actor InvoiceActor, input InvoiceListInput) (InvoicePage[model.InvoiceFile], error) {
	var page InvoicePage[model.InvoiceFile]
	setInvoicePageDefaults(&input)
	page.Page, page.PageSize = input.Page, input.PageSize
	query := model.DB.WithContext(ctx).Model(&model.InvoiceFile{}).
		Joins("JOIN invoice_applications ON invoice_applications.id = invoice_files.invoice_application_id")
	query = invoiceScopedQuery(query, "invoice_applications", actor.Scope)
	if !invoiceIsAdmin(actor) {
		query = query.Where("invoice_applications.user_id = ?", actor.UserId)
	}
	if input.UserId > 0 {
		query = query.Where("invoice_applications.user_id = ?", input.UserId)
	}
	if input.SourceId > 0 {
		query = query.Where("invoice_applications.source_id = ?", input.SourceId)
	}
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	var rows []model.InvoiceFile
	if err := query.Order("invoice_files.id desc").Limit(input.PageSize).Offset((input.Page - 1) * input.PageSize).Find(&rows).Error; err != nil {
		return page, err
	}
	page.Items = rows
	return page, nil
}

func availableInvoiceAmountTx(tx *gorm.DB, paymentOrderId int, paidAmount float64, excludeApplicationId int) (decimal.Decimal, error) {
	query := tx.Model(&model.InvoiceApplication{}).
		Where("source_type = ? AND source_id = ? AND status IN ?", model.InvoiceSourcePaymentOrder, paymentOrderId, []string{
			model.InvoiceStatusPending,
			model.InvoiceStatusApproved,
			model.InvoiceStatusIssued,
		})
	if excludeApplicationId > 0 {
		query = query.Where("id <> ?", excludeApplicationId)
	}
	var used float64
	if err := query.Select("COALESCE(SUM(amount), 0)").Scan(&used).Error; err != nil {
		return decimal.Zero, err
	}
	return decimal.NewFromFloat(paidAmount).Sub(decimal.NewFromFloat(used)), nil
}

func invoiceOwnershipByUserId(ctx context.Context, userId int) (model.OwnershipSnapshot, error) {
	var user model.User
	if err := model.DB.WithContext(ctx).Select("tenant_id", "organization_id", "department_id", "distribution_channel_id").Where("id = ?", userId).First(&user).Error; err != nil {
		return model.OwnershipSnapshot{}, err
	}
	return model.NormalizeOwnership(model.OwnershipSnapshot{
		TenantId:              user.TenantId,
		OrganizationId:        user.OrganizationId,
		DepartmentId:          user.DepartmentId,
		DistributionChannelId: user.DistributionChannelId,
	}), nil
}

func invoiceScopedQuery(db *gorm.DB, table string, scope model.AccessScope) *gorm.DB {
	if scope.IsRoot || scope.RoleKey == common.RoleKeyFinance {
		return db
	}
	return model.ApplyOwnershipScope(db, table, scope)
}

func invoiceIsAdmin(actor InvoiceActor) bool {
	return actor.Scope.IsRoot || actor.Scope.RoleKey == common.RoleKeyFinance || common.IsTenantAdminRole(actor.Scope.RoleKey)
}

func invoiceCanReview(actor InvoiceActor) bool {
	return actor.Scope.IsRoot || actor.Scope.RoleKey == common.RoleKeyFinance || common.IsTenantAdminRole(actor.Scope.RoleKey)
}

func invoiceCanManageUser(actor InvoiceActor, userId int) bool {
	return invoiceIsAdmin(actor) || actor.UserId == userId
}

func invoiceCanAccessProfile(actor InvoiceActor, profile *model.InvoiceProfile) bool {
	if profile == nil {
		return false
	}
	if !invoiceIsAdmin(actor) && profile.UserId != actor.UserId {
		return false
	}
	return model.AllowsOwnership(actor.Scope, profile.TenantId, profile.OrganizationId, profile.DepartmentId)
}

func setInvoicePageDefaults(input *InvoiceListInput) {
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

func generateInvoiceApplicationNo(userId int) string {
	return fmt.Sprintf("INV%d%s%s", userId, time.Now().Format("20060102150405"), strings.ToUpper(common.GetRandomString(6)))
}
