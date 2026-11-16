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

func TestSupplierStaleUnboundCacheStillEnforcesLivePolicy(t *testing.T) {
	supplierTestDB(t)
	now := common.GetTimestamp()
	base := "https://api.deepseek.com"
	supplier, err := SavePlatformSupplier(model.PlatformSupplier{Provider: "deepseek", Name: "DS", LegalEntity: "Company", Mode: "integration", Status: "active", Currency: "CNY", AgreementRef: "contract", AgreementExpiresAt: now + 3600}, 1, true)
	require.NoError(t, err)
	channel := model.Channel{Name: "live", Type: constant.ChannelTypeDeepSeek, Key: "fixture", BaseURL: &base, Status: 1}
	require.NoError(t, model.DB.Create(&channel).Error)
	cached := channel // fetched before another instance binds the resource
	account, err := CreateSupplierAccount(model.SupplierAccount{SupplierId: supplier.Id, ChannelId: channel.Id, Name: "account"}, 1)
	require.NoError(t, err)
	require.Equal(t, 0, cached.SupplierAccountId)
	require.NoError(t, model.CheckSupplierChannelAccess(&cached))
	require.NoError(t, SetSupplierAccountStatus(account.Id, "paused", 1))
	require.Error(t, model.CheckSupplierChannelAccess(&cached))
	require.NoError(t, SetSupplierAccountStatus(account.Id, "active", 1))
	require.NoError(t, model.DB.Model(&model.PlatformSupplier{}).Where("id = ?", supplier.Id).Update("agreement_expires_at", now-1).Error)
	require.Error(t, model.CheckSupplierChannelAccess(&cached))
	require.NoError(t, model.DB.Model(&model.PlatformSupplier{}).Where("id = ?", supplier.Id).Update("agreement_expires_at", now+3600).Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("base_url", "https://example.com").Error)
	require.Error(t, model.CheckSupplierChannelAccess(&cached))
}

func TestSupplierBoundChannelDeletePathsAreAtomic(t *testing.T) {
	for _, operation := range []string{"single", "batch", "disabled", "status"} {
		t.Run(operation, func(t *testing.T) {
			supplierTestDB(t)
			require.NoError(t, model.DB.AutoMigrate(&model.Ability{}))
			first := model.Channel{Name: "unbound", Type: 43, Key: "fixture", Status: 2}
			bound := model.Channel{Name: "bound", Type: 43, Key: "fixture", Status: 2, SupplierAccountId: 99}
			require.NoError(t, model.DB.Create(&first).Error)
			require.NoError(t, model.DB.Create(&bound).Error)
			require.NoError(t, model.DB.Create(&model.Ability{ChannelId: first.Id, Model: "fixture", Group: "default", Enabled: true}).Error)
			var err error
			switch operation {
			case "single":
				err = bound.Delete()
			case "batch":
				err = model.BatchDeleteChannels([]int{first.Id, bound.Id})
			case "disabled":
				_, err = model.DeleteDisabledChannel()
			case "status":
				_, err = model.DeleteChannelByStatus(2)
			}
			require.Error(t, err)
			var count int64
			require.NoError(t, model.DB.Model(&model.Channel{}).Count(&count).Error)
			require.EqualValues(t, 2, count)
			require.NoError(t, first.Delete())
			require.NoError(t, model.DB.Model(&model.Ability{}).Where("channel_id = ?", first.Id).Count(&count).Error)
			require.Zero(t, count)
			require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", bound.Id).Update("supplier_account_id", 0).Error)
			require.NoError(t, model.DB.Create(&model.SupplierAccount{ChannelId: bound.Id, SupplierId: 88, Name: "live-binding"}).Error)
			require.Error(t, bound.Delete())
		})
	}
}

func TestSupplierCleanupRespectsTenantScope(t *testing.T) {
	supplierTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Ability{}))
	own := model.Channel{TenantId: 1, Name: "own", Key: "fixture", Type: 43, Status: 2}
	other := model.Channel{TenantId: 2, Name: "other", Key: "fixture", Type: 43, Status: 2, SupplierAccountId: 99}
	require.NoError(t, model.DB.Create(&own).Error)
	require.NoError(t, model.DB.Create(&other).Error)
	deleted, err := model.DeleteDisabledChannel(model.TenantScope{TenantId: 1})
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	var remaining model.Channel
	require.NoError(t, model.DB.First(&remaining, other.Id).Error)
	require.Equal(t, 99, remaining.SupplierAccountId)
}
