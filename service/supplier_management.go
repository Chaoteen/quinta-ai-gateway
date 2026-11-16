package service

import (
	"errors"
	"fmt"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"regexp"
	"strings"
)

var ErrSupplierConflict = errors.New("record changed; reload before saving")
var supplierSecretPattern = regexp.MustCompile(`(?i)(sk-[a-z0-9_-]{12,}|-----BEGIN|bearer\s+[a-z0-9._-]{12,})`)

func supplierAudit(tx *gorm.DB, actor int, action string, id int) error {
	return tx.Create(&model.SupplierAudit{ActorId: actor, Action: action, ResourceId: id, CreatedAt: common.GetTimestamp()}).Error
}
func supplierText(v string, max int) bool {
	return len([]rune(v)) <= max && !supplierSecretPattern.MatchString(v)
}

func SavePlatformSupplier(input model.PlatformSupplier, actor int, review bool) (model.PlatformSupplier, error) {
	if _, err := supplierProvider(input.Provider); err != nil {
		return input, err
	}
	if strings.TrimSpace(input.Name) == "" || !supplierText(input.Name, 128) || !supplierText(input.LegalEntity, 200) || !supplierText(input.Contact, 200) || !supplierText(input.AgreementRef, 200) {
		return input, errors.New("invalid supplier metadata; do not enter API secrets")
	}
	if input.Mode != "integration" && input.Mode != "resale" {
		return input, errors.New("invalid business mode")
	}
	if input.Status != "draft" && input.Status != "active" && input.Status != "suspended" {
		return input, errors.New("invalid supplier status")
	}
	if input.Currency != "CNY" && input.Currency != "USD" {
		return input, errors.New("unsupported procurement currency")
	}
	now := common.GetTimestamp()
	if input.Id == 0 {
		input.Version = 0
	}
	if input.Status == "active" && (!review || strings.TrimSpace(input.AgreementRef) == "" || strings.TrimSpace(input.LegalEntity) == "" || input.AgreementExpiresAt <= now) {
		return input, errors.New("activation requires an explicit contract review and future expiry")
	}
	var saved model.PlatformSupplier
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if input.Id != 0 {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&saved, input.Id).Error; err != nil {
				return err
			}
			if input.Version != saved.Version {
				return ErrSupplierConflict
			}
			var count int64
			if err := tx.Model(&model.SupplierAccount{}).Where("supplier_id = ?", input.Id).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 && (input.Provider != saved.Provider || input.Currency != saved.Currency) {
				return errors.New("provider and currency are immutable after account binding")
			}
			input.CreatedAt = saved.CreatedAt
		} else {
			input.CreatedAt = now
		}
		input.ReviewedAt = 0
		input.ReviewedBy = 0
		if input.Status == "active" {
			input.ReviewedAt = now
			input.ReviewedBy = actor
		}
		oldVersion := input.Version
		input.Version++
		input.UpdatedAt = now
		if input.Id == 0 {
			if err := tx.Create(&input).Error; err != nil {
				return err
			}
		} else {
			result := tx.Model(&model.PlatformSupplier{}).Where("id = ? AND version = ?", input.Id, oldVersion).Select("provider", "name", "legal_entity", "contact", "mode", "status", "currency", "agreement_ref", "agreement_expires_at", "reviewed_at", "reviewed_by", "version", "updated_at").Updates(&input)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrSupplierConflict
			}
		}
		revision := model.SupplierRevision{SupplierId: input.Id, Version: input.Version, Name: input.Name, LegalEntity: input.LegalEntity, Mode: input.Mode, Currency: input.Currency, Status: input.Status, AgreementRef: input.AgreementRef, AgreementExpiresAt: input.AgreementExpiresAt, ActorId: actor, CreatedAt: now}
		if err := tx.Create(&revision).Error; err != nil {
			return err
		}
		saved = input
		return supplierAudit(tx, actor, "supplier.save."+input.Status, input.Id)
	})
	return saved, err
}

func CreateSupplierAccount(input model.SupplierAccount, actor int) (model.SupplierAccount, error) {
	input.Id = 0
	if input.SupplierId < 1 || input.ChannelId < 1 || strings.TrimSpace(input.Name) == "" || !supplierText(input.Name, 128) || !supplierText(input.AccountRef, 128) || !supplierText(input.KeyRef, 128) || !supplierText(input.Region, 64) {
		return input, errors.New("invalid account metadata; store Key resource ID, never the secret")
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var supplier model.PlatformSupplier
		if err := tx.First(&supplier, input.SupplierId).Error; err != nil {
			return err
		}
		if supplier.Status != "active" || supplier.AgreementExpiresAt <= common.GetTimestamp() {
			return errors.New("review and activate supplier contract first")
		}
		var channel model.Channel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&channel, input.ChannelId).Error; err != nil {
			return err
		}
		if channel.SupplierAccountId != 0 {
			return errors.New("channel is already bound")
		}
		if err := validateSupplierChannel(supplier.Provider, &channel); err != nil {
			return err
		}
		input.Endpoint = constant.ChannelBaseURLs[channel.Type]
		if channel.BaseURL != nil && *channel.BaseURL != "" {
			input.Endpoint = *channel.BaseURL
		}
		input.Protocol = channel.Type
		input.Status = "active"
		input.CreatedAt = common.GetTimestamp()
		if err := tx.Create(&input).Error; err != nil {
			return err
		}
		result := tx.Model(&model.Channel{}).Where("id = ? AND supplier_account_id = ?", channel.Id, 0).Update("supplier_account_id", input.Id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrSupplierConflict
		}
		return supplierAudit(tx, actor, "account.bind", input.Id)
	})
	if err == nil {
		model.InitChannelCache()
	}
	return input, err
}

func SetSupplierAccountStatus(id int, status string, actor int) error {
	if status != "active" && status != "paused" {
		return errors.New("invalid account status")
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var account model.SupplierAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, id).Error; err != nil {
			return err
		}
		if account.Status == status {
			return nil
		}
		result := tx.Model(&model.SupplierAccount{}).Where("id = ?", id).Update("status", status)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return supplierAudit(tx, actor, "account."+status, id)
	})
}

func SupplierDecimalAmount(value string, scale int32) (int64, error) {
	if len(value) > 30 {
		return 0, errors.New("invalid decimal amount")
	}
	d, err := decimal.NewFromString(value)
	if err != nil || d.IsNegative() || d.GreaterThan(decimal.NewFromInt(1000000000)) {
		return 0, errors.New("amount must be between 0 and 1000000000")
	}
	units := d.Shift(scale)
	if !units.Equal(units.Truncate(0)) {
		return 0, errors.New("amount has too many decimal places")
	}
	return units.IntPart(), nil
}

type SupplierPriceInput struct {
	Unit        string `json:"unit"`
	UnitPrice   string `json:"unit_price"`
	AccountId   int    `json:"account_id"`
	Model       string `json:"model"`
	EffectiveAt int64  `json:"effective_at"`
	Input       string `json:"input"`
	Output      string `json:"output"`
	CacheRead   string `json:"cache_read"`
	CacheWrite  string `json:"cache_write"`
}

func CreateSupplierPrice(input SupplierPriceInput, actor int) (model.SupplierPrice, error) {
	row := model.SupplierPrice{AccountId: input.AccountId, Model: input.Model, EffectiveAt: input.EffectiveAt, CreatedAt: common.GetTimestamp()}
	if input.Unit == "" {
		input.Unit = "million_tokens"
	}
	switch input.Unit {
	case "million_tokens", "second", "image", "request", "million_characters":
	default:
		return row, errors.New("unsupported price unit")
	}
	row.Unit = input.Unit
	if input.UnitPrice == "" {
		input.UnitPrice = "0"
	}
	var unitErr error
	row.UnitPriceMicros, unitErr = SupplierDecimalAmount(input.UnitPrice, 6)
	if unitErr != nil {
		return row, unitErr
	}

	if strings.TrimSpace(input.Model) == "" || !supplierText(input.Model, 128) || input.EffectiveAt < 1 {
		return row, errors.New("model and effective date required")
	}
	prices := []string{input.Input, input.Output, input.CacheRead, input.CacheWrite}
	out := []*int64{&row.InputMicros, &row.OutputMicros, &row.CacheReadMicros, &row.CacheWriteMicros}
	for i, value := range prices {
		parsed, err := SupplierDecimalAmount(value, 6)
		if err != nil {
			return row, err
		}
		*out[i] = parsed
	}
	if row.Unit == "million_tokens" && row.UnitPriceMicros != 0 {
		return row, errors.New("token rates use input/output/cache prices; leave unit price at zero")
	}
	if row.Unit != "million_tokens" && (row.InputMicros != 0 || row.OutputMicros != 0 || row.CacheReadMicros != 0 || row.CacheWriteMicros != 0) {
		return row, errors.New("non-token rates use unit price; leave token prices at zero")
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var account model.SupplierAccount
		if err := tx.First(&account, input.AccountId).Error; err != nil {
			return err
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return supplierAudit(tx, actor, "price.create", row.Id)
	})
	return row, err
}

type SupplierStatementInput struct {
	AccountId   int    `json:"account_id"`
	ExternalRef string `json:"external_ref"`
	InvoiceRef  string `json:"invoice_ref"`
	StartAt     int64  `json:"start_at"`
	EndAt       int64  `json:"end_at"`
	Amount      string `json:"amount"`
}

func CreateSupplierStatement(input SupplierStatementInput, actor int) (model.SupplierStatement, error) {
	row := model.SupplierStatement{AccountId: input.AccountId, ExternalRef: input.ExternalRef, InvoiceRef: input.InvoiceRef, StartAt: input.StartAt, EndAt: input.EndAt, Status: "pending_review", CreatedAt: common.GetTimestamp()}
	if strings.TrimSpace(input.ExternalRef) == "" || !supplierText(input.ExternalRef, 128) || !supplierText(input.InvoiceRef, 128) || input.StartAt < 1 || input.EndAt <= input.StartAt || input.EndAt > common.GetTimestamp()+60 || input.EndAt-input.StartAt > 366*86400 {
		return row, errors.New("invalid statement reference or time range (maximum 366 days)")
	}
	amount, err := SupplierDecimalAmount(input.Amount, 2)
	if err != nil {
		return row, err
	}
	row.AmountMinor = amount
	var account model.SupplierAccount
	if err = model.DB.First(&account, input.AccountId).Error; err != nil {
		return row, err
	}
	var supplier model.PlatformSupplier
	if err = model.DB.First(&supplier, account.SupplierId).Error; err != nil {
		return row, err
	}
	row.Currency = supplier.Currency
	// Logs may live in a separate database; read a dated snapshot, not a cross-DB join.
	var usage struct {
		Requests int64
		Input    int64
		Output   int64
	}
	err = model.LOG_DB.Model(&model.Log{}).Select("COUNT(*) AS requests, COALESCE(SUM(prompt_tokens),0) AS input, COALESCE(SUM(completion_tokens),0) AS output").Where("channel_id = ? AND type = ? AND created_at >= ? AND created_at < ?", account.ChannelId, model.LogTypeConsume, input.StartAt, input.EndAt).Scan(&usage).Error
	if err != nil {
		return row, err
	}
	row.PlatformInput = usage.Input
	row.PlatformOutput = usage.Output
	row.PlatformRequests = usage.Requests
	row.SnapshotAt = common.GetTimestamp()
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return supplierAudit(tx, actor, "statement.create", row.Id)
	})
	return row, err
}

func ReviewSupplierStatement(id int, note string, actor int) error {
	if strings.TrimSpace(note) == "" || !supplierText(note, 500) {
		return errors.New("reconciliation note required; explain discrepancies and provider invoice checks")
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.SupplierStatement{}).Where("id = ? AND status = ?", id, "pending_review").Updates(map[string]interface{}{"status": "reviewed", "review_note": note, "reviewed_by": actor, "reviewed_at": common.GetTimestamp()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrSupplierConflict
		}
		if err := tx.Model(&model.SupplierStatement{}).Where("id = ? AND amount_minor = 0", id).Update("status", "settled").Error; err != nil {
			return err
		}
		return supplierAudit(tx, actor, "statement.review", id)
	})
}

func RecordSupplierPayment(statementId int, reference, amount string, actor int) (model.SupplierPayment, error) {
	row := model.SupplierPayment{StatementId: statementId, Reference: reference, CreatedBy: actor, CreatedAt: common.GetTimestamp()}
	if strings.TrimSpace(reference) == "" || !supplierText(reference, 128) {
		return row, errors.New("external bank payment reference required")
	}
	minor, err := SupplierDecimalAmount(amount, 2)
	if err != nil || minor <= 0 {
		return row, errors.New("payment must be a positive amount with at most two decimals")
	}
	row.AmountMinor = minor
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		var prior model.SupplierPayment
		err := tx.Where("reference = ?", reference).First(&prior).Error
		if err == nil {
			if prior.StatementId != statementId || prior.AmountMinor != minor {
				return errors.New("payment reference already used with another payload")
			}
			row = prior
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		result := tx.Model(&model.SupplierStatement{}).Where("id = ? AND status = ? AND amount_minor - paid_minor >= ?", statementId, "reviewed", minor).Update("paid_minor", gorm.Expr("paid_minor + ?", minor))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("statement is not reviewed or payment exceeds outstanding amount")
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.SupplierStatement{}).Where("id = ? AND paid_minor = amount_minor", statementId).Update("status", "settled").Error; err != nil {
			return err
		}
		return supplierAudit(tx, actor, "payment.record", row.Id)
	})
	return row, err
}

func SupplierSummary() (map[string]interface{}, error) {
	data := map[string]interface{}{"providers": SupplierProviders}
	queries := []struct {
		key  string
		dest interface{}
	}{
		{"suppliers", &[]model.PlatformSupplier{}}, {"accounts", &[]model.SupplierAccount{}}, {"prices", &[]model.SupplierPrice{}}, {"statements", &[]model.SupplierStatement{}}, {"payments", &[]model.SupplierPayment{}}, {"audit", &[]model.SupplierAudit{}}, {"revisions", &[]model.SupplierRevision{}},
	}
	for _, q := range queries {
		if err := model.DB.Order("id desc").Limit(500).Find(q.dest).Error; err != nil {
			return nil, fmt.Errorf("supplier records unavailable: %w", err)
		}
		data[q.key] = q.dest
	}
	return data, nil
}
