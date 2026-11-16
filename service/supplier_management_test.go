package service

import (
	"fmt"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"sync"
	"testing"
)

func supplierTestDB(t *testing.T) {
	t.Helper()
	old, oldLog, oldCache := model.DB, model.LOG_DB, common.MemoryCacheEnabled
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB, common.MemoryCacheEnabled = db, db, false
	require.NoError(t, db.AutoMigrate(&model.PlatformSupplier{}, &model.SupplierAccount{}, &model.SupplierPrice{}, &model.SupplierStatement{}, &model.SupplierPayment{}, &model.SupplierAudit{}, &model.SupplierRevision{}, &model.Channel{}, &model.Log{}))
	t.Cleanup(func() {
		model.DB, model.LOG_DB, common.MemoryCacheEnabled = old, oldLog, oldCache
		s, _ := db.DB()
		s.Close()
	})
}
func TestSupplierProcurementLifecycle(t *testing.T) {
	supplierTestDB(t)
	now := common.GetTimestamp()
	supplier := model.PlatformSupplier{Name: "supplier", Provider: "deepseek", LegalEntity: "Company", Mode: "resale", Status: "active", Currency: "CNY", AgreementRef: "contract-123", AgreementExpiresAt: now + 86400}
	_, err := SavePlatformSupplier(supplier, 1, false)
	require.Error(t, err)
	supplier, err = SavePlatformSupplier(supplier, 1, true)
	require.NoError(t, err)
	stale := supplier
	stale.Version = 0
	_, err = SavePlatformSupplier(stale, 1, true)
	require.Error(t, err)
	base := "https://api.deepseek.com"
	ch := model.Channel{Name: "DS", Type: constant.ChannelTypeDeepSeek, Key: "fixture", BaseURL: &base, Status: 1}
	require.NoError(t, model.DB.Create(&ch).Error)
	account, err := CreateSupplierAccount(model.SupplierAccount{SupplierId: supplier.Id, ChannelId: ch.Id, Name: "prod", KeyRef: "resource-id"}, 1)
	require.NoError(t, err)
	ch.SupplierAccountId = account.Id
	require.NoError(t, model.CheckSupplierChannelAccess(&ch))
	ch.Type = 100000
	require.Error(t, model.CheckSupplierChannelAccess(&ch))
	ch.Type = constant.ChannelTypeDeepSeek
	require.NoError(t, SetSupplierAccountStatus(account.Id, "paused", 1))
	require.Error(t, model.CheckSupplierChannelAccess(&ch))
	require.NoError(t, SetSupplierAccountStatus(account.Id, "active", 1))
	_, err = CreateSupplierPrice(SupplierPriceInput{AccountId: account.Id, Model: "model", EffectiveAt: now, Input: "0.123456", Output: "1.2", CacheRead: "0", CacheWrite: "0"}, 1)
	require.NoError(t, err)
	require.NoError(t, model.LOG_DB.Create(&model.Log{ChannelId: ch.Id, CreatedAt: now - 10, Type: model.LogTypeConsume, PromptTokens: 100, CompletionTokens: 20}).Error)
	statement, err := CreateSupplierStatement(SupplierStatementInput{AccountId: account.Id, ExternalRef: "monthly-1", StartAt: now - 3600, EndAt: now, Amount: "10.00"}, 1)
	require.NoError(t, err)
	require.EqualValues(t, 100, statement.PlatformInput)
	_, err = RecordSupplierPayment(statement.Id, "payment-1", "10.00", 1)
	require.Error(t, err)
	require.NoError(t, ReviewSupplierStatement(statement.Id, "Compared provider bill and resolved differences", 1))
	payment, err := RecordSupplierPayment(statement.Id, "payment-1", "6.00", 1)
	require.NoError(t, err)
	repeated, err := RecordSupplierPayment(statement.Id, "payment-1", "6.00", 1)
	require.NoError(t, err)
	require.Equal(t, payment.Id, repeated.Id)
	_, err = RecordSupplierPayment(statement.Id, "payment-2", "5.00", 1)
	require.Error(t, err)
	_, err = RecordSupplierPayment(statement.Id, "payment-2", "4.00", 1)
	require.NoError(t, err)
	require.NoError(t, model.DB.First(&statement, statement.Id).Error)
	require.Equal(t, "settled", statement.Status)
	require.EqualValues(t, 1000, statement.PaidMinor)
	supplier.Status = "suspended"
	_, err = SavePlatformSupplier(supplier, 1, false)
	require.NoError(t, err)
	require.Error(t, model.CheckSupplierChannelAccess(&ch))
	var audits int64
	require.NoError(t, model.DB.Model(&model.SupplierAudit{}).Count(&audits).Error)
	require.Greater(t, audits, int64(6))
}
func TestSupplierRejectsUnofficialEndpointsAndCredentials(t *testing.T) {
	for _, endpoint := range []string{"http://api.deepseek.com", "https://api.deepseek.com.evil.example", "https://169.254.169.254", "https://user:pass@api.deepseek.com", "https://api.deepseek.com/coding/v1"} {
		ch := model.Channel{Type: constant.ChannelTypeDeepSeek, BaseURL: &endpoint}
		require.Error(t, validateSupplierChannel("deepseek", &ch))
	}
	for _, s := range []string{"-1", "NaN", "0.0000001", "1000000001"} {
		_, err := SupplierDecimalAmount(s, 6)
		require.Error(t, err)
	}
	require.False(t, supplierText("sk-testcredentials1234567890", 128))
}

func TestSupplierConcurrentPaymentsCannotExceedStatement(t *testing.T) {
	supplierTestDB(t)
	statement := model.SupplierStatement{AccountId: 1, ExternalRef: "concurrency", Currency: "CNY", AmountMinor: 1000, Status: "reviewed"}
	require.NoError(t, model.DB.Create(&statement).Error)
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	start := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_, err := RecordSupplierPayment(statement.Id, fmt.Sprintf("bank-%d", index), "1.00", 1)
			if err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}(i)
	}
	close(start)
	wg.Wait()
	require.Greater(t, success, 0)
	require.LessOrEqual(t, success, 10)
	require.NoError(t, model.DB.First(&statement, statement.Id).Error)
	require.EqualValues(t, success*100, statement.PaidMinor)
	var payments int64
	require.NoError(t, model.DB.Model(&model.SupplierPayment{}).Count(&payments).Error)
	require.EqualValues(t, success, payments)
}
