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
	OrderNo        string
	Provider       string
	EventType      string
	RawPayload     string
	SignatureValid bool
	ProcessMessage string
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
	ReviewerId   int
	Approved     bool
	ReviewNote   string
	RawPayload   string
	EventType    string
	FailedStatus bool
}

func (s *PaymentGatewayService) CreatePaymentOrder(ctx context.Context, input CreatePaymentOrderInput) (*model.PaymentOrder, error) {
	provider := model.NormalizePaymentProvider(input.Provider)
	if provider != model.PaymentProviderMock && provider != model.PaymentProviderBankTransfer {
		return nil, ErrPaymentInvalidProvider
	}
	businessType := model.NormalizePaymentBusinessType(input.BusinessType)
	if err := validatePaymentBusiness(businessType, input.BusinessId); err != nil {
		return nil, err
	}
	if input.UserId <= 0 {
		return nil, errors.New("invalid user id")
	}
	if input.Amount <= 0 {
		return nil, ErrPaymentInvalidAmount
	}
	ownership := model.NormalizeOwnership(model.OwnershipSnapshot{
		TenantId:              input.TenantId,
		OrganizationId:        input.OrganizationId,
		DepartmentId:          input.DepartmentId,
		DistributionChannelId: input.DistributionChannelId,
	})
	if input.TenantId == 0 {
		var err error
		ownership, err = model.RequiredOwnershipByUserId(input.UserId)
		if err != nil {
			return nil, err
		}
	}
	if err := model.ValidateOwnershipHierarchy(ownership); err != nil {
		return nil, err
	}
	if input.ExpiredAt == 0 {
		input.ExpiredAt = common.GetTimestamp() + 30*24*60*60
	}
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
	if err := model.DB.WithContext(ctx).Create(order).Error; err != nil {
		return nil, err
	}
	return order, nil
}

func (s *PaymentGatewayService) ConfirmPayment(ctx context.Context, input ConfirmPaymentInput) (*model.PaymentOrder, error) {
	if strings.TrimSpace(input.OrderNo) == "" {
		return nil, ErrPaymentOrderNotFound
	}
	provider := model.NormalizePaymentProvider(input.Provider)
	if input.EventType == "" {
		input.EventType = "payment.confirmed"
	}
	var out model.PaymentOrder
	var processErr error
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("order_no = ?", strings.TrimSpace(input.OrderNo)).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPaymentOrderNotFound
			}
			return err
		}
		if provider != "" && provider != order.Provider {
			processErr = ErrPaymentInvalidProvider
			return createPaymentCallbackLogTx(tx, &order, input, model.PaymentCallbackProcessFailed, processErr.Error())
		}
		if order.Status == model.PaymentOrderStatusPaid {
			out = order
			return createPaymentCallbackLogTx(tx, &order, input, model.PaymentCallbackProcessIgnored, "payment order already paid")
		}
		if order.Status != model.PaymentOrderStatusPending {
			processErr = ErrPaymentOrderStatusInvalid
			return createPaymentCallbackLogTx(tx, &order, input, model.PaymentCallbackProcessFailed, processErr.Error())
		}
		if order.ExpiredAt > 0 && order.ExpiredAt < common.GetTimestamp() {
			order.Status = model.PaymentOrderStatusExpired
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			processErr = ErrPaymentOrderStatusInvalid
			return createPaymentCallbackLogTx(tx, &order, input, model.PaymentCallbackProcessFailed, "payment order expired")
		}
		if err := s.fulfillPaymentOrderTx(ctx, tx, &order); err != nil {
			order.FulfillmentStatus = model.PaymentFulfillmentFailed
			order.FulfillmentMessage = err.Error()
			if saveErr := tx.Save(&order).Error; saveErr != nil {
				return saveErr
			}
			processErr = err
			return createPaymentCallbackLogTx(tx, &order, input, model.PaymentCallbackProcessFailed, err.Error())
		}
		now := common.GetTimestamp()
		order.Status = model.PaymentOrderStatusPaid
		order.PaidAt = now
		order.FulfillmentStatus = model.PaymentFulfillmentSuccess
		order.FulfilledAt = now
		order.FulfillmentMessage = "fulfilled"
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		out = order
		return createPaymentCallbackLogTx(tx, &order, input, model.PaymentCallbackProcessSuccess, "payment fulfilled")
	})
	if err != nil {
		return nil, err
	}
	if processErr != nil {
		return &out, processErr
	}
	return &out, nil
}

func (s *PaymentGatewayService) FulfillPaymentOrder(ctx context.Context, orderId int) (*model.PaymentOrder, error) {
	var out model.PaymentOrder
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", orderId).First(&order).Error; err != nil {
			return err
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
	if input.TransferAmount <= 0 {
		return nil, ErrPaymentInvalidAmount
	}
	var order model.PaymentOrder
	if err := model.DB.WithContext(ctx).Where("id = ? AND user_id = ?", input.PaymentOrderId, input.UserId).First(&order).Error; err != nil {
		return nil, err
	}
	if order.Provider != model.PaymentProviderBankTransfer {
		return nil, ErrPaymentInvalidProvider
	}
	record := &model.BankTransferRecord{
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
	var record model.BankTransferRecord
	var orderNo string
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", recordId).First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBankTransferNotFound
			}
			return err
		}
		if record.ReviewStatus != model.BankTransferReviewPending {
			return ErrBankTransferStatusInvalid
		}
		record.ReviewedBy = input.ReviewerId
		record.ReviewedAt = common.GetTimestamp()
		record.ReviewNote = input.ReviewNote
		if input.Approved {
			record.ReviewStatus = model.BankTransferReviewApproved
		} else {
			record.ReviewStatus = model.BankTransferReviewRejected
		}
		if err := tx.Save(&record).Error; err != nil {
			return err
		}
		if input.Approved {
			var order model.PaymentOrder
			if err := tx.Where("id = ?", record.PaymentOrderId).First(&order).Error; err != nil {
				return err
			}
			orderNo = order.OrderNo
			return nil
		}
		if input.FailedStatus {
			if err := tx.Model(&model.PaymentOrder{}).
				Where("id = ? AND status = ?", record.PaymentOrderId, model.PaymentOrderStatusPending).
				Update("status", model.PaymentOrderStatusFailed).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if input.Approved {
		eventType := input.EventType
		if eventType == "" {
			eventType = "bank_transfer.approved"
		}
		_, err = s.ConfirmPayment(ctx, ConfirmPaymentInput{
			OrderNo:        orderNo,
			Provider:       model.PaymentProviderBankTransfer,
			EventType:      eventType,
			RawPayload:     input.RawPayload,
			SignatureValid: true,
		})
		if err != nil {
			return nil, err
		}
		if err := model.DB.WithContext(ctx).Where("id = ?", recordId).First(&record).Error; err != nil {
			return nil, err
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
	switch order.BusinessType {
	case model.PaymentBusinessTokenRecharge:
		quotaToAdd := order.BusinessId
		if quotaToAdd <= 0 {
			quotaToAdd = int(decimal.NewFromFloat(order.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
		}
		if quotaToAdd <= 0 {
			return ErrPaymentInvalidBusinessId
		}
		if err := tx.WithContext(ctx).Model(&model.User{}).Where("id = ?", order.UserId).
			Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}
		return nil
	case model.PaymentBusinessSubscriptionPurchase:
		if order.BusinessId <= 0 {
			return ErrPaymentInvalidBusinessId
		}
		plan, err := getPaymentSubscriptionPlanTx(tx, order.BusinessId)
		if err != nil {
			return err
		}
		_, err = model.CreateUserSubscriptionFromPlanWithOwnershipTx(tx, order.UserId, plan, "payment_order", ownershipFromPaymentOrder(order))
		return err
	case model.PaymentBusinessSubscriptionRenewal:
		if order.BusinessId <= 0 {
			return ErrPaymentInvalidBusinessId
		}
		var sub model.UserSubscription
		if err := tx.WithContext(ctx).Where("id = ? AND user_id = ?", order.BusinessId, order.UserId).First(&sub).Error; err != nil {
			return err
		}
		plan, err := getPaymentSubscriptionPlanTx(tx, sub.PlanId)
		if err != nil {
			return err
		}
		_, err = model.CreateUserSubscriptionFromPlanWithOwnershipTx(tx, sub.UserId, plan, "payment_renewal", ownershipFromPaymentOrder(order))
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
