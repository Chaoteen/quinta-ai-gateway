package model

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRecoveryCodeConcurrentConsumptionIsSingleUse(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{SkipDefaultTransaction: true})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	oldDB := DB
	DB = db
	t.Cleanup(func() { DB = oldDB; _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&TwoFABackupCode{}))
	require.NoError(t, CreateBackupCodes(1, []string{"ABCD-EFGH"}))
	// All contenders read the same unused code before any update executes.
	const contenders = 6
	var arrived sync.WaitGroup
	arrived.Add(contenders)
	release := make(chan struct{})
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:recovery-barrier", func(tx *gorm.DB) {
		arrived.Done()
		<-release
	}))
	var finished sync.WaitGroup
	var successes atomic.Int32
	errors := make(chan error, contenders)
	for i := 0; i < contenders; i++ {
		finished.Add(1)
		go func() {
			defer finished.Done()
			ok, err := ValidateBackupCode(1, "ABCD-EFGH")
			if err != nil {
				errors <- err
			}
			if ok {
				successes.Add(1)
			}
		}()
	}
	arrived.Wait()
	close(release)
	finished.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, successes.Load())
	require.NoError(t, db.Callback().Update().Remove("test:recovery-barrier"))
	ok, err := ValidateBackupCode(1, common.NormalizeBackupCode("ABCD-EFGH"))
	require.NoError(t, err)
	require.False(t, ok)
	count, err := GetUnusedBackupCodeCount(1)
	require.NoError(t, err)
	require.Zero(t, count)
}
