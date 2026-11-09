package model

import (
	"path/filepath"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSQLiteStartupMigrationCanRepeatWithoutChangingData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "restart.db")), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	previousDB, previousSQLite := DB, common.UsingSQLite
	DB, common.UsingSQLite = db, true
	t.Cleanup(func() { DB, common.UsingSQLite = previousDB, previousSQLite })
	require.NoError(t, migrateDB())
	channel := DistributionChannel{Name: "Migration preservation", Code: "restart-preserve", CommissionRate: 0.125}
	require.NoError(t, db.Create(&channel).Error)
	minor := int64(1)
	app := InvoiceApplication{ApplicationNo: "restart-invoice", UserId: 1, InvoiceProfileId: 1,
		Amount: 0.01, AmountMinor: &minor, SnapshotVersion: 1, BuyerSnapshot: "preserved snapshot", Currency: "CNY"}
	require.NoError(t, db.Create(&app).Error)
	for range 2 {
		require.NoError(t, migrateDB())
	}
	var after DistributionChannel
	require.NoError(t, db.First(&after, channel.Id).Error)
	require.Equal(t, channel.CommissionRate, after.CommissionRate)
	var reloaded InvoiceApplication
	require.NoError(t, db.First(&reloaded, app.Id).Error)
	require.Equal(t, int64(1), *reloaded.AmountMinor)
	require.Equal(t, "preserved snapshot", reloaded.BuyerSnapshot)
	var count int64
	require.NoError(t, db.Model(&DistributionChannel{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestSQLiteAdditiveMigrationPreservesLegacyColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	// This is deliberately SQLite-only legacy DDL for an upgrade test.
	require.NoError(t, db.Exec("CREATE TABLE distribution_channels (id integer PRIMARY KEY, name varchar(128), code varchar(64), legacy_marker text)").Error)
	require.NoError(t, db.Exec("INSERT INTO distribution_channels (id, name, code, legacy_marker) VALUES (?, ?, ?, ?)", 1, "Legacy", "legacy", "keep").Error)
	for range 2 {
		require.NoError(t, autoMigrateCompatible(db, &DistributionChannel{}))
	}
	var channel DistributionChannel
	require.NoError(t, db.First(&channel, 1).Error)
	require.Equal(t, 1, channel.TenantId)
	require.Equal(t, "Legacy", channel.Name)
	var marker string
	require.NoError(t, db.Raw("SELECT legacy_marker FROM distribution_channels WHERE id = ?", 1).Scan(&marker).Error)
	require.Equal(t, "keep", marker)
	require.True(t, db.Migrator().HasIndex(&DistributionChannel{}, "idx_distribution_channels_code"))
}
