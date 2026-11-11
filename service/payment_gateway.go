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
	"gorm.io/gorm/clause"
)

var (
	ErrPaymentInvalidProvider     = errors.New("invalid payment provider")
	ErrPaymentInvalidBusinessType = errors.New("invalid payment business type")
	ErrPaymentInvalidAmount       = errors.New("payment amount must be greater than 0")
	ErrPaymentInvalidBusinessId   = errors.New("invalid payment business id")
	ErrPaymentOrderNotFound       = errors.New("payment order not found")
	ErrPaymentOrderStatusInvalid  = errors.New("payment order status invalid")
	ErrBankTransferNotFound       = errors.New("bank transfer record not found")
	ErrBankTransferStatusInvalid  = errors.New("bank transfer review status invalid")
)

type PaymentGatewayService struct{}

func NewPaymentGatewayService() *PaymentGatewayService {
	return &PaymentGatewayService{}
}

type CreatePaymentOrderInput struct {
	PaymentAccountId      int
	UserId                int
	TenantId              int
	OrganizationId        int
	DepartmentId          int
	DistributionChannelId int
	Provider              string
	BusinessType          string
	BusinessId            int
	Amount                float64
	Currency              string
	Subject               string
	Description           string
	ExpiredAt             int64
}

type ConfirmPaymentInput struct {
	PaymentAccountId int
	TenantId         int
	MerchantId       string
	AppId            string
	Currency         string
	AmountMinor      int64
	OrderNo          string
	Provider         string
	EventType        string
	RawPayload       string
	SignatureValid   bool
	ProcessMessage   string
}

type CreateBankTransferRecordInput struct {
	PaymentOrderId      int
	UserId              int
	BankAccountName     string
	BankAccountNoMasked string
	TransferAmount      float64
	TransferTime        int64
	ProofUrl            string
}

type ReviewBankTransferInput struct {
	BankTransactionReference string
	ReviewerId               int
	Approved                 bool
	ReviewNote               string
	RawPayload               string
	EventType                string
	FailedStatus             bool
}

func (s *PaymentGatewayService) CreatePaymentOrder(ctx context.Context, input CreatePaymentOrderInput) (*model.PaymentOrder, error) {
	provider := strings.ToUpper(strings.TrimSpace(input.Provider))
	if provider == model.PaymentProviderMock && !common.GetEnvOrDefaultBool("ENABLE_MOCK_PAYMENTS", false) {
		return nil, ErrPaymentInvalidProvider
	}
	if provider != model.PaymentProviderMock && provider != model.PaymentProviderBankTransfer {
		return nil, ErrPaymentInvalidProvider
	}
	businessType := strings.ToUpper(strings.TrimSpace(input.BusinessType))
	if err := validatePaymentBusiness(businessType, input.BusinessId); err != nil {
		return nil, err
	}
	if input.UserId <= 0 {
		return nil, errors.New("invalid user id")
	}
	if input.Amount <= 0 {
		return nil, ErrPaymentInvalidAmount
	}
	ownership, err := model.RequiredOwnershipByUserId(input.UserId)
	if err != nil {
		return nil, err
	}
	if (input.TenantId != 0 && input.TenantId != ownership.TenantId) ||
		(input.OrganizationId != 0 && input.OrganizationId != ownership.OrganizationId) ||
		(input.DepartmentId != 0 && input.DepartmentId != ownership.DepartmentId) ||
		(input.DistributionChannelId != 0 && input.DistributionChannelId != ownership.DistributionChannelId) {
		return nil, ErrPaymentOwnership
	}
	if err := model.ValidateOwnershipHierarchy(ownership); err != nil {
		return nil, err
	}
	// Expiry is controlled by the server, not by the customer.
	input.ExpiredAt = common.GetTimestamp() + 30*60
	order := &model.PaymentOrder{
		OrderNo:           generatePaymentOrderNo(input.UserId),
		UserId:            input.UserId,
		Provider:          provider,
		BusinessType:      businessType,
		BusinessId:        input.BusinessId,
		Amount:            input.Amount,
		Currency:          input.Currency,
		Status:            model.PaymentOrderStatusPending,
		Subject:           input.Subject,
		Description:       input.Description,
		ExpiredAt:         input.ExpiredAt,
		FulfillmentStatus: model.PaymentFulfillmentPending,
	}
	ownership.ApplyTo(order)
	if err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := bindTenantPaymentAccount(ctx, tx, order, input.PaymentAccountId); err != nil {
			return err
		}
		if err := quotePayment(ctx, tx, input, order); err != nil {
			return err
		}
		return tx.Create(order).Error
	}); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *PaymentGatewayService) ConfirmPayment(ctx context.Context, input ConfirmPaymentInput) (*model.PaymentOrder, error) {
	var out *model.PaymentOrder
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		out, err = s.confirmPaymentTx(ctx, tx, input)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := model.InvalidateUserCache(out.UserId); err != nil {
		common.SysError(fmt.Sprintf("payment cache invalidation failed for user %d", out.UserId))
	}
	return out, nil
}

// The verified adapter and manual bank review share a single transaction boundary.
func (s *PaymentGatewayService) confirmPaymentTx(ctx context.Context, tx *gorm.DB, input ConfirmPaymentInput) (*model.PaymentOrder, error) {
	if !input.SignatureValid {
		return nil, ErrPaymentInvalidSignature
	}
	provider := strings.ToUpper(strings.TrimSpace(input.Provider))
	if provider != model.PaymentProviderBankTransfer && provider != model.PaymentProviderMock {
		return nil, ErrPaymentInvalidProvider
	}
	if provider == model.PaymentProviderMock && !common.GetEnvOrDefaultBool("ENABLE_MOCK_PAYMENTS", false) {
		return nil, ErrPaymentInvalidProvider
	}
	var order model.PaymentOrder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_no = ?", strings.TrimSpace(input.OrderNo)).First(&order).Error; err != nil {
		return nil, err
	}
	if provider != order.Provider {
		return nil, ErrPaymentInvalidProvider
	}
	if err := validateTenantPaymentEventTx(tx, &order, input); err != nil {
		return nil, err
	}
	if order.Status == model.PaymentOrderStatusPaid && order.FulfillmentStatus == model.PaymentFulfillmentSuccess {
		return &order, createPaymentCallbackLogTx(tx, &order, input, model.PaymentCallbackProcessIgnored, "payment order already paid")
	}
	if order.Status != model.PaymentOrderStatusPending || (order.ExpiredAt > 0 && order.ExpiredAt <= common.GetTimestamp()) {
		return nil, ErrPaymentOrderStatusInvalid
	}
	if order.PricingVersion != paymentPricingVersion {
		return nil, ErrPaymentUntrustedQuote
	}
	// Claim the transition before touching balances, including on SQLite where
	// row-level SELECT FOR UPDATE is unavailable.
	claim := tx.Model(&model.PaymentOrder{}).
		Where("id = ? AND status = ? AND fulfillment_status <> ?", order.Id, model.PaymentOrderStatusPending, model.PaymentFulfillmentSuccess).
		Update("status", model.PaymentOrderStatusPaid)
	if claim.Error != nil {
		return nil, claim.Error
	}
	if claim.RowsAffected != 1 {
		return nil, ErrPaymentOrderStatusInvalid
	}
	order.Status = model.PaymentOrderStatusPaid
	if err := s.fulfillPaymentOrderTx(ctx, tx, &order); err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	order.PaidAt = now
	order.FulfillmentStatus = model.PaymentFulfillmentSuccess
	order.FulfilledAt = now
	order.FulfillmentMessage = "fulfilled"
	if err := tx.Save(&order).Error; err != nil {
		return nil, err
	}
	return &order, createPaymentCallbackLogTx(tx, &order, input, model.PaymentCallbackProcessSuccess, "payment fulfilled")
}

func (s *PaymentGatewayService) FulfillPaymentOrder(ctx context.Context, orderId int) (*model.PaymentOrder, error) {
	var out model.PaymentOrder
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderId).First(&order).Error; err != nil {
			return err
		}
		if order.Status != model.PaymentOrderStatusPaid {
			return ErrPaymentOrderStatusInvalid
		}
		if order.FulfillmentStatus == model.PaymentFulfillmentSuccess {
			out = order
			return nil
		}
		if err := s.fulfillPaymentOrderTx(ctx, tx, &order); err != nil {
			return err
		}
		now := common.GetTimestamp()
		order.FulfillmentStatus = model.PaymentFulfillmentSuccess
		order.FulfillmentMessage = "fulfilled"
		order.FulfilledAt = now
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		out = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *PaymentGatewayService) CreateBankTransferRecord(ctx context.Context, input CreateBankTransferRecordInput) (*model.BankTransferRecord, error) {
	if input.PaymentOrderId <= 0 || input.UserId <= 0 {
		return nil, errors.New("invalid bank transfer target")
	}
	if strings.TrimSpace(input.BankAccountName) == "" {
		return nil, errors.New("bank account name is required")
	}
	if !validPaymentTransferAmount(input.TransferAmount) {
		return nil, ErrPaymentInvalidAmount
	}
	var order model.PaymentOrder
	if err := model.DB.WithContext(ctx).Where("id = ? AND user_id = ?", input.PaymentOrderId, input.UserId).First(&order).Error; err != nil {
		return nil, err
	}
	if order.Status != model.PaymentOrderStatusPending || order.ExpiredAt <= common.GetTimestamp() || input.TransferAmount != order.Amount {
		return nil, ErrPaymentOrderStatusInvalid
	}
	if order.Provider != model.PaymentProviderBankTransfer {
		return nil, ErrPaymentInvalidProvider
	}
	record := &model.BankTransferRecord{
		PaymentAccountId:    order.PaymentAccountId,
		PaymentOrderId:      order.Id,
		UserId:              input.UserId,
		BankAccountName:     input.BankAccountName,
		BankAccountNoMasked: input.BankAccountNoMasked,
		TransferAmount:      input.TransferAmount,
		TransferTime:        input.TransferTime,
		ProofUrl:            input.ProofUrl,
		ReviewStatus:        model.BankTransferReviewPending,
	}
	model.NormalizeOwnership(model.OwnershipSnapshot{TenantId: order.TenantId}).ApplyTo(record)
	if err := model.DB.WithContext(ctx).Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *PaymentGatewayService) ReviewBankTransfer(ctx context.Context, recordId int, input ReviewBankTransferInput) (*model.BankTransferRecord, error) {
	if input.ReviewerId <= 0 {
		return nil, ErrBankTransferStatusInvalid
	}
	var record model.BankTransferRecord
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", recordId).First(&record).Error; err != nil {
			return err
		}
		if record.ReviewStatus != model.BankTransferReviewPending {
			return ErrBankTransferStatusInvalid
		}
		var order model.PaymentOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", record.PaymentOrderId).First(&order).Error; err != nil {
			return err
		}
		if record.PaymentAccountId != order.PaymentAccountId {
			return ErrTenantPaymentBinding
		}
		if err := validateTenantBankReviewerTx(tx, &order, input); err != nil {
			return err
		}
		if input.Approved {
			if record.UserId != order.UserId || record.TenantId != order.TenantId || record.TransferAmount != order.Amount {
				return ErrPaymentInvalidAmount
			}
			if order.PaymentAccountId > 0 {
				reference := strings.TrimSpace(input.BankTransactionReference)
				record.BankTransactionReference = &reference
			}
			if _, err := s.confirmPaymentTx(ctx, tx, ConfirmPaymentInput{
				PaymentAccountId: order.PaymentAccountId, TenantId: receivingTenant(&order),
				MerchantId: order.MerchantIdSnapshot, AppId: order.AppIdSnapshot,
				Currency: order.Currency, AmountMinor: order.AmountMinor,
				OrderNo: order.OrderNo, Provider: model.PaymentProviderBankTransfer,
				EventType: "bank_transfer.approved", SignatureValid: true,
			}); err != nil {
				return err
			}
			record.ReviewStatus = model.BankTransferReviewApproved
		} else {
			record.ReviewStatus = model.BankTransferReviewRejected
			if input.FailedStatus {
				if err := tx.Model(&model.PaymentOrder{}).
					Where("id = ? AND status = ?", record.PaymentOrderId, model.PaymentOrderStatusPending).
					Update("status", model.PaymentOrderStatusFailed).Error; err != nil {
					return err
				}
			}
		}
		record.ReviewedBy = input.ReviewerId
		record.ReviewedAt = common.GetTimestamp()
		record.ReviewNote = input.ReviewNote
		return tx.Save(&record).Error
	})
	if err != nil {
		return nil, err
	}
	if input.Approved {
		if err := model.InvalidateUserCache(record.UserId); err != nil {
			common.SysError(fmt.Sprintf("bank transfer cache invalidation failed for user %d", record.UserId))
		}
	}
	return &record, nil
}

func (s *PaymentGatewayService) fulfillPaymentOrderTx(ctx context.Context, tx *gorm.DB, order *model.PaymentOrder) error {
	if order == nil {
		return errors.New("payment order is nil")
	}
	if order.FulfillmentStatus == model.PaymentFulfillmentSuccess {
		return nil
	}
	if order.Status != model.PaymentOrderStatusPaid || order.PricingVersion != paymentPricingVersion {
		return ErrPaymentUntrustedQuote
	}
	switch order.BusinessType {
	case model.PaymentBusinessTokenRecharge:
		return grantQuotedQuotaTx(tx.WithContext(ctx), order)
	case model.PaymentBusinessSubscriptionPurchase, model.PaymentBusinessSubscriptionRenewal:
		var plan model.SubscriptionPlan
		if order.PlanSnapshot == "" {
			return ErrPaymentUntrustedQuote
		}
		if err := common.UnmarshalJsonStr(order.PlanSnapshot, &plan); err != nil {
			return ErrPaymentUntrustedQuote
		}
		_, err := model.CreateUserSubscriptionFromPlanWithOwnershipTx(tx, order.UserId, &plan, "payment_order", ownershipFromPaymentOrder(order))
		return err
	default:
		return ErrPaymentInvalidBusinessType
	}
}

func validatePaymentBusiness(businessType string, businessId int) error {
	switch businessType {
	case model.PaymentBusinessTokenRecharge:
		return nil
	case model.PaymentBusinessSubscriptionPurchase, model.PaymentBusinessSubscriptionRenewal:
		if businessId <= 0 {
			return ErrPaymentInvalidBusinessId
		}
		return nil
	default:
		return ErrPaymentInvalidBusinessType
	}
}

func createPaymentCallbackLogTx(tx *gorm.DB, order *model.PaymentOrder, input ConfirmPaymentInput, status string, message string) error {
	log := model.PaymentCallbackLog{
		PaymentOrderId: order.Id,
		OrderNo:        order.OrderNo,
		Provider:       order.Provider,
		EventType:      input.EventType,
		RawPayload:     input.RawPayload,
		SignatureValid: input.SignatureValid,
		ProcessStatus:  status,
		ProcessMessage: strings.TrimSpace(message),
	}
	if input.ProcessMessage != "" {
		log.ProcessMessage = strings.TrimSpace(input.ProcessMessage + " " + message)
	}
	return tx.Create(&log).Error
}

func getPaymentSubscriptionPlanTx(tx *gorm.DB, planId int) (*model.SubscriptionPlan, error) {
	var plan model.SubscriptionPlan
	if err := tx.Where("id = ?", planId).First(&plan).Error; err != nil {
		return nil, err
	}
	plan.NormalizeAlphaFields()
	return &plan, nil
}

func ownershipFromPaymentOrder(order *model.PaymentOrder) model.OwnershipSnapshot {
	if order == nil {
		return model.NormalizeOwnership(model.OwnershipSnapshot{})
	}
	return model.NormalizeOwnership(model.OwnershipSnapshot{
		TenantId:              order.TenantId,
		OrganizationId:        order.OrganizationId,
		DepartmentId:          order.DepartmentId,
		DistributionChannelId: order.DistributionChannelId,
	})
}

func generatePaymentOrderNo(userId int) string {
	return fmt.Sprintf("PAY%d%s%s", userId, time.Now().Format("20060102150405"), strings.ToUpper(common.GetRandomString(8)))
}
