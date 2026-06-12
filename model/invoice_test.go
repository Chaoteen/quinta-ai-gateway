package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvoiceModelsAreMigrated(t *testing.T) {
	require.True(t, DB.Migrator().HasTable(&InvoiceProfile{}))
	require.True(t, DB.Migrator().HasTable(&InvoiceApplication{}))
	require.True(t, DB.Migrator().HasTable(&InvoiceFile{}))
	for _, column := range []string{"tenant_id", "user_id", "profile_type", "title", "tax_no", "is_default", "status"} {
		require.True(t, DB.Migrator().HasColumn(&InvoiceProfile{}, column), "missing invoice_profiles.%s", column)
	}
	for _, column := range []string{"application_no", "source_type", "source_id", "invoice_no", "invoice_date", "issued_at"} {
		require.True(t, DB.Migrator().HasColumn(&InvoiceApplication{}, column), "missing invoice_applications.%s", column)
	}
}

func TestInvoiceModelNormalization(t *testing.T) {
	profile := InvoiceProfile{ProfileType: "personal", Status: "disabled", Title: "  Alice  "}
	profile.Normalize()
	require.Equal(t, InvoiceProfileTypePersonal, profile.ProfileType)
	require.Equal(t, InvoiceProfileStatusDisabled, profile.Status)
	require.Equal(t, "Alice", profile.Title)

	app := InvoiceApplication{InvoiceType: "vat_special", Status: "approved", SourceType: "manual", Currency: "usd"}
	app.Normalize()
	require.Equal(t, InvoiceTypeVATSpecial, app.InvoiceType)
	require.Equal(t, InvoiceStatusApproved, app.Status)
	require.Equal(t, InvoiceSourceManual, app.SourceType)
	require.Equal(t, "USD", app.Currency)
}
