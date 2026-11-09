package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	ErrInvoiceSnapshotInvalid     = errors.New("invoice requires a trusted buyer and amount snapshot")
	ErrInvoiceDuplicateNumber     = errors.New("invoice number is already registered")
)

type InvoiceService struct{}

func NewInvoiceService() *InvoiceService {
	return &InvoiceService{}
}

type InvoiceActor struct {
	UserId int
	Scope  model.AccessScope
}

type InvoiceCapabilities struct {
	ActorUserId int  `json:"actor_user_id"`
	CanReview   bool `json:"can_review"`
	CanRegister bool `json:"can_register"`
}

func (s *InvoiceService) Capabilities(actor InvoiceActor) InvoiceCapabilities {
	allowed := invoiceCanReview(actor)
	return InvoiceCapabilities{ActorUserId: actor.UserId, CanReview: allowed, CanRegister: allowed}
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
	if !invoiceAllowsOwnership(actor, ownership.TenantId, ownership.OrganizationId, ownership.DepartmentId) {
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
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", profileId).First(&profile).Error; err != nil {
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
	amountMinor, amountErr := invoiceAmountMinor(input.Amount)
	if actor.UserId <= 0 || input.InvoiceProfileId <= 0 || amountErr != nil {
		return nil, ErrInvoiceApplicationInvalid
	}
	invoiceType := strings.ToUpper(strings.TrimSpace(input.InvoiceType))
	if invoiceType == "" {
		invoiceType = model.InvoiceTypeVATNormal
	}
	if invoiceType != model.InvoiceTypeVATNormal && invoiceType != model.InvoiceTypeVATSpecial {
		return nil, ErrInvoiceApplicationInvalid
	}
	if source := strings.ToUpper(strings.TrimSpace(input.SourceType)); source != "" && source != model.InvoiceSourcePaymentOrder {
		return nil, ErrInvoiceSourceInvalid
	}
	sourceType := model.NormalizeInvoiceSourceType(input.SourceType)
	if sourceType != model.InvoiceSourcePaymentOrder || input.SourceId <= 0 {
		return nil, ErrInvoiceSourceInvalid
	}
	var out model.InvoiceApplication
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", input.SourceId).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceSourceInvalid
			}
			return err
		}
		if order.Status != model.PaymentOrderStatusPaid || order.Currency != "CNY" {
			return ErrInvoiceSourceInvalid
		}
		if !invoiceCanManageUser(actor, order.UserId) {
			return ErrInvoiceOutOfScope
		}
		if !invoiceAllowsOwnership(actor, order.TenantId, order.OrganizationId, order.DepartmentId) {
			return ErrInvoiceOutOfScope
		}
		var profile model.InvoiceProfile
		if err := tx.Where("id = ? AND user_id = ? AND status = ?", input.InvoiceProfileId, order.UserId, model.InvoiceProfileStatusActive).First(&profile).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceProfileNotFound
			}
			return err
		}
		if profile.TenantId != order.TenantId || !invoiceAllowsOwnership(actor, profile.TenantId, profile.OrganizationId, profile.DepartmentId) {
			return ErrInvoiceOutOfScope
		}
		if profile.ProfileType == model.InvoiceProfileTypePersonal && invoiceType == model.InvoiceTypeVATSpecial {
			return ErrInvoiceProfileInvalid
		}
		if strings.TrimSpace(profile.Title) == "" || (profile.ProfileType == model.InvoiceProfileTypeCompany && strings.TrimSpace(profile.TaxNo) == "") {
			return ErrInvoiceProfileInvalid
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
		buyerSnapshot, err := common.Marshal(profile)
		if err != nil {
			return err
		}
		app := model.InvoiceApplication{
			ApplicationNo:    generateInvoiceApplicationNo(order.UserId),
			UserId:           order.UserId,
			InvoiceProfileId: profile.Id,
			Amount:           input.Amount,
			AmountMinor:      &amountMinor,
			SnapshotVersion:  1,
			BuyerSnapshot:    string(buyerSnapshot),
			Currency:         order.Currency,
			InvoiceType:      invoiceType,
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
	if !invoiceCanReview(actor) || input.ReviewerId != actor.UserId {
		return nil, ErrInvoiceOutOfScope
	}
	var out model.InvoiceApplication
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app model.InvoiceApplication
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", applicationId).First(&app).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceApplicationNotFound
			}
			return err
		}
		if !invoiceAllowsOwnership(actor, app.TenantId, app.OrganizationId, app.DepartmentId) || app.UserId == actor.UserId {
			return ErrInvoiceOutOfScope
		}
		if app.Status != model.InvoiceStatusPending {
			return ErrInvoiceStatusInvalid
		}
		if input.Approved {
			if err := validateInvoiceSnapshot(&app); err != nil {
				return err
			}
			var order model.PaymentOrder
			if err := tx.Where("id = ? AND status = ? AND user_id = ? AND tenant_id = ? AND currency = ?", app.SourceId,
				model.PaymentOrderStatusPaid, app.UserId, app.TenantId, app.Currency).First(&order).Error; err != nil {
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
		result := tx.Model(&app).Where("status = ?", model.InvoiceStatusPending).Select("status", "reviewer_id", "reviewed_at", "review_note", "updated_at").Updates(&app)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvoiceStatusInvalid
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
	if !invoiceCanReview(actor) || input.UploadedBy != actor.UserId {
		return nil, ErrInvoiceOutOfScope
	}
	if strings.TrimSpace(input.FileName) == "" || strings.TrimSpace(input.FileUrl) == "" || strings.TrimSpace(input.InvoiceNo) == "" {
		return nil, ErrInvoiceApplicationInvalid
	}
	invoiceNo := strings.ToUpper(strings.TrimSpace(input.InvoiceNo))
	if len(invoiceNo) > 128 || strings.ContainsAny(invoiceNo, "\r\n\t") || !validInvoiceFileURL(input.FileUrl) {
		return nil, ErrInvoiceApplicationInvalid
	}
	var out model.InvoiceApplication
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app model.InvoiceApplication
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", applicationId).First(&app).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceApplicationNotFound
			}
			return err
		}
		if !invoiceAllowsOwnership(actor, app.TenantId, app.OrganizationId, app.DepartmentId) || app.UserId == actor.UserId {
			return ErrInvoiceOutOfScope
		}
		if app.Status != model.InvoiceStatusApproved {
			return ErrInvoiceStatusInvalid
		}
		if err := validateInvoiceSnapshot(&app); err != nil {
			return err
		}
		var order model.PaymentOrder
		if err := tx.Where("id = ? AND status = ? AND user_id = ? AND tenant_id = ? AND currency = ?", app.SourceId,
			model.PaymentOrderStatusPaid, app.UserId, app.TenantId, app.Currency).First(&order).Error; err != nil {
			return ErrInvoiceSourceInvalid
		}
		available, err := availableInvoiceAmountTx(tx, order.Id, order.Amount, app.Id)
		if err != nil {
			return err
		}
		if decimal.NewFromFloat(app.Amount).GreaterThan(available) {
			return ErrInvoiceAmountExceeded
		}
		// Check legacy registrations too. The nullable unique identity below
		// arbitrates races between new registrations without conflicting on blanks.
		var duplicate int64
		if err := tx.Model(&model.InvoiceApplication{}).Where("id <> ? AND status = ? AND UPPER(TRIM(invoice_no)) = ?",
			app.Id, model.InvoiceStatusIssued, invoiceNo).Count(&duplicate).Error; err != nil {
			return err
		}
		if duplicate > 0 {
			return ErrInvoiceDuplicateNumber
		}
		now := common.GetTimestamp()
		app.Status = model.InvoiceStatusIssued
		app.InvoiceNo = invoiceNo
		app.IssuedIdentity = &invoiceNo
		app.InvoiceDate = input.InvoiceDate
		if app.InvoiceDate == 0 {
			app.InvoiceDate = now
		}
		app.IssuedAt = now
		result := tx.Model(&app).Where("status = ?", model.InvoiceStatusApproved).
			Select("status", "invoice_no", "issued_identity", "invoice_date", "issued_at", "updated_at").Updates(&app)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvoiceStatusInvalid
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
	paidMinor, err := invoiceAmountMinor(paidAmount)
	if err != nil {
		return decimal.Zero, err
	}
	// Sum validated integer amounts in Go. SQLite's SUM(decimal) may otherwise
	// accumulate binary floating point error at a one-cent boundary.
	var applications []model.InvoiceApplication
	if err := query.Select("amount", "amount_minor").Find(&applications).Error; err != nil {
		return decimal.Zero, err
	}
	remaining := paidMinor
	for _, app := range applications {
		minor, err := invoiceAmountMinor(app.Amount)
		if err != nil || (app.AmountMinor != nil && *app.AmountMinor != minor) {
			return decimal.Zero, ErrInvoiceSnapshotInvalid
		}
		if minor > remaining {
			return decimal.Zero, ErrInvoiceAmountExceeded
		}
		remaining -= minor
	}
	return decimal.NewFromInt(remaining).Div(decimal.NewFromInt(100)), nil
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
	if scope.IsRoot || invoicePlatformReader(scope) {
		return db
	}
	return model.ApplyOwnershipScope(db, table, scope)
}

func invoiceIsAdmin(actor InvoiceActor) bool {
	return actor.Scope.IsRoot || actor.Scope.RoleKey == common.RoleKeyFinance || actor.Scope.RoleKey == common.RoleKeyAuditor || common.IsTenantAdminRole(actor.Scope.RoleKey)
}

func invoiceCanReview(actor InvoiceActor) bool {
	return actor.UserId > 0 && (actor.Scope.IsRoot || invoicePlatformFinance(actor.Scope))
}

// Enterprise customer administrators cannot issue invoices on behalf of the
// platform seller. Platform finance requires an explicitly configured tenant.
func invoicePlatformFinance(scope model.AccessScope) bool {
	platformTenantId := common.GetEnvOrDefault("PLATFORM_TENANT_ID", 0)
	return platformTenantId > 0 && scope.TenantId == platformTenantId && scope.RoleKey == common.RoleKeyFinance
}

func invoicePlatformReader(scope model.AccessScope) bool {
	platformTenantId := common.GetEnvOrDefault("PLATFORM_TENANT_ID", 0)
	return platformTenantId > 0 && scope.TenantId == platformTenantId &&
		(scope.RoleKey == common.RoleKeyFinance || scope.RoleKey == common.RoleKeyAuditor)
}

func invoiceAllowsOwnership(actor InvoiceActor, tenantId, organizationId, departmentId int) bool {
	return invoicePlatformFinance(actor.Scope) || model.AllowsOwnership(actor.Scope, tenantId, organizationId, departmentId)
}

func invoiceAmountMinor(amount float64) (int64, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 || amount > maxPaymentAmount {
		return 0, ErrInvoiceApplicationInvalid
	}
	value := decimal.NewFromFloat(amount)
	if !value.Equal(value.Truncate(2)) {
		return 0, ErrInvoiceApplicationInvalid
	}
	return value.Mul(decimal.NewFromInt(100)).IntPart(), nil
}

func validateInvoiceSnapshot(app *model.InvoiceApplication) error {
	minor, err := invoiceAmountMinor(app.Amount)
	if err != nil || app.AmountMinor == nil || *app.AmountMinor != minor || app.SnapshotVersion != 1 || app.Currency != "CNY" || app.SourceType != model.InvoiceSourcePaymentOrder ||
		(app.InvoiceType != model.InvoiceTypeVATNormal && app.InvoiceType != model.InvoiceTypeVATSpecial) {
		return ErrInvoiceSnapshotInvalid
	}
	var profile model.InvoiceProfile
	if err := common.UnmarshalJsonStr(app.BuyerSnapshot, &profile); err != nil || profile.Id != app.InvoiceProfileId || profile.UserId != app.UserId || profile.TenantId != app.TenantId || strings.TrimSpace(profile.Title) == "" ||
		(profile.ProfileType != model.InvoiceProfileTypeCompany && profile.ProfileType != model.InvoiceProfileTypePersonal) ||
		(profile.ProfileType == model.InvoiceProfileTypeCompany && strings.TrimSpace(profile.TaxNo) == "") ||
		(profile.ProfileType == model.InvoiceProfileTypePersonal && app.InvoiceType != model.InvoiceTypeVATNormal) {
		return ErrInvoiceSnapshotInvalid
	}
	return nil
}

func validInvoiceFileURL(value string) bool {
	u, err := url.Parse(strings.TrimSpace(value))
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.Fragment == ""
}

func invoiceCanManageUser(actor InvoiceActor, userId int) bool {
	return actor.Scope.IsRoot || actor.Scope.RoleKey == common.RoleKeyFinance || common.IsTenantAdminRole(actor.Scope.RoleKey) || actor.UserId == userId
}

func invoiceCanAccessProfile(actor InvoiceActor, profile *model.InvoiceProfile) bool {
	if profile == nil {
		return false
	}
	if !invoiceCanManageUser(actor, profile.UserId) {
		return false
	}
	return invoiceAllowsOwnership(actor, profile.TenantId, profile.OrganizationId, profile.DepartmentId)
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
