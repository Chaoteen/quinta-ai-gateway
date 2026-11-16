package model

import (
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestSupplierMigrationPreservesExistingChannelCredentials(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:supplier-migration?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	require.NoError(t, db.Exec("CREATE TABLE channels (id integer PRIMARY KEY AUTOINCREMENT, name text, key text NOT NULL, type integer, status integer)").Error)
	require.NoError(t, db.Table("channels").Create(map[string]interface{}{"id": 1, "name": "existing", "key": "fixture-credential", "type": 17, "status": 1}).Error)
	require.NoError(t, autoMigrateCompatible(db, &Channel{}, &PlatformSupplier{}, &SupplierAccount{}, &SupplierPrice{}, &SupplierStatement{}, &SupplierPayment{}, &SupplierAudit{}, &SupplierRevision{}))
	var channel Channel
	require.NoError(t, db.First(&channel, 1).Error)
	require.Equal(t, "fixture-credential", channel.Key)
	require.Equal(t, 0, channel.SupplierAccountId)
	require.Equal(t, 1, channel.Status)
}
