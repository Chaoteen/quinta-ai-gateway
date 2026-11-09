package model

import (
	"fmt"

	"gorm.io/gorm"
)

// autoMigrateCompatible keeps SQLite startup migrations additive. Its current
// driver cannot reliably reconstruct existing DECIMAL(p,s) table definitions
// during implicit ALTER COLUMN operations. Existing column/type changes require
// an explicit migration; startup must not rebuild or drop customer tables.
// MySQL and PostgreSQL continue to use the regular GORM migrator.
func autoMigrateCompatible(db *gorm.DB, values ...interface{}) error {
	if db.Dialector.Name() != "sqlite" {
		return db.AutoMigrate(values...)
	}
	for _, value := range values {
		statement := &gorm.Statement{DB: db}
		if err := statement.Parse(value); err != nil {
			return err
		}
		migrator := db.Migrator()
		if !migrator.HasTable(value) {
			if err := migrator.CreateTable(value); err != nil {
				return fmt.Errorf("create SQLite table %s: %w", statement.Table, err)
			}
			continue
		}
		for _, name := range statement.Schema.DBNames {
			field := statement.Schema.FieldsByDBName[name]
			if field.IgnoreMigration || migrator.HasColumn(value, name) {
				continue
			}
			if field.PrimaryKey {
				return fmt.Errorf("SQLite table %s requires an explicit primary-key migration", statement.Table)
			}
			if err := migrator.AddColumn(value, name); err != nil {
				return fmt.Errorf("add SQLite column %s.%s: %w", statement.Table, name, err)
			}
		}
		for name := range statement.Schema.ParseIndexes() {
			if !migrator.HasIndex(value, name) {
				if err := migrator.CreateIndex(value, name); err != nil {
					return fmt.Errorf("create SQLite index %s: %w", name, err)
				}
			}
		}
	}
	return nil
}
