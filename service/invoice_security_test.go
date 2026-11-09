package service

import (
	"context"
	"math"
	"sync"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func invoiceSecurityFixture(t *testing.T) (*InvoiceService, InvoiceActor, model.PaymentOrder, *model.InvoiceProfile, InvoiceActor) {
	t.Helper()
	resetInvoiceTables(t)
	t.Setenv("PLATFORM_TENANT_ID", "900")
	user := seedInvoiceUser(t, 18001, 801)
	financeUser := seedInvoiceUser(t, 18002, 900)
	finance := InvoiceActor{UserId: financeUser.Id, Scope: model.AccessScope{TenantId: 900, RoleKey: common.RoleKeyFinance}}
	order := seedInvoicePayment(t, 18001, user, 100, model.PaymentOrderStatusPaid)
	svc := NewInvoiceService()
	actor := invoiceUserActor(user)
	profile, err := svc.CreateInvoiceProfile(context.Background(), actor, CreateInvoiceProfileInput{Title: "Original Buyer", TaxNo: "ORIGINAL-TAX"})
	require.NoError(t, err)
	return svc, actor, order, profile, finance
}

func invoiceSecurityApply(t *testing.T, svc *InvoiceService, actor InvoiceActor, order model.PaymentOrder, profile *model.InvoiceProfile, amount float64) *model.InvoiceApplication {
	t.Helper()
	app, err := svc.CreateInvoiceApplication(context.Background(), actor, CreateInvoiceApplicationInput{
		InvoiceProfileId: profile.Id, Amount: amount, SourceType: model.InvoiceSourcePaymentOrder, SourceId: order.Id,
	})
	require.NoError(t, err)
	return app
}

func TestSecurityInvoiceRejectsInvalidMoneyAndCurrency(t *testing.T) {
	svc, actor, order, profile, _ := invoiceSecurityFixture(t)
	for _, amount := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, -1, 0.001, 9999999} {
		_, err := svc.CreateInvoiceApplication(context.Background(), actor, CreateInvoiceApplicationInput{InvoiceProfileId: profile.Id, Amount: amount, SourceId: order.Id})
		require.ErrorIs(t, err, ErrInvoiceApplicationInvalid)
	}
	for _, field := range []string{"currency", "invoice_type", "source_type"} {
		in := CreateInvoiceApplicationInput{InvoiceProfileId: profile.Id, Amount: 1, SourceId: order.Id}
		switch field {
		case "currency":
			in.Currency = "USD"
		case "invoice_type":
			in.InvoiceType = "INVALID"
		case "source_type":
			in.SourceType = "INVALID"
		}
		_, err := svc.CreateInvoiceApplication(context.Background(), actor, in)
		require.Error(t, err)
	}
	require.NoError(t, model.DB.Model(&order).Update("currency", "USD").Error)
	_, err := svc.CreateInvoiceApplication(context.Background(), actor, CreateInvoiceApplicationInput{InvoiceProfileId: profile.Id, Amount: 1, SourceId: order.Id})
	require.ErrorIs(t, err, ErrInvoiceSourceInvalid)
}

func TestSecurityInvoiceBuyerSnapshotAndSellerAuthority(t *testing.T) {
	svc, actor, order, profile, finance := invoiceSecurityFixture(t)
	app := invoiceSecurityApply(t, svc, actor, order, profile, 12.34)
	require.Equal(t, int64(1234), *app.AmountMinor)
	require.NoError(t, model.DB.Model(profile).Updates(map[string]interface{}{"title": "Changed Buyer", "tax_no": "CHANGED-TAX"}).Error)
	var snapshot model.InvoiceProfile
	require.NoError(t, common.UnmarshalJsonStr(app.BuyerSnapshot, &snapshot))
	require.Equal(t, "Original Buyer", snapshot.Title)
	require.Equal(t, "ORIGINAL-TAX", snapshot.TaxNo)

	buyerAdmin := actor
	buyerAdmin.Scope.RoleKey = common.RoleKeyTenantAdmin
	_, err := svc.ReviewInvoiceApplication(context.Background(), buyerAdmin, app.Id, ReviewInvoiceApplicationInput{ReviewerId: actor.UserId, Approved: true})
	require.ErrorIs(t, err, ErrInvoiceOutOfScope)
	foreignFinance := finance
	foreignFinance.Scope.TenantId = actor.Scope.TenantId
	_, err = svc.ReviewInvoiceApplication(context.Background(), foreignFinance, app.Id, ReviewInvoiceApplicationInput{ReviewerId: finance.UserId, Approved: true})
	require.ErrorIs(t, err, ErrInvoiceOutOfScope)
	_, err = svc.ReviewInvoiceApplication(context.Background(), finance, app.Id, ReviewInvoiceApplicationInput{ReviewerId: 999, Approved: true})
	require.ErrorIs(t, err, ErrInvoiceOutOfScope)
	// Even root cannot approve its own purchase application.
	selfRoot := actor
	selfRoot.Scope.IsRoot = true
	_, err = svc.ReviewInvoiceApplication(context.Background(), selfRoot, app.Id, ReviewInvoiceApplicationInput{ReviewerId: actor.UserId, Approved: true})
	require.ErrorIs(t, err, ErrInvoiceOutOfScope)
	approved, err := svc.ReviewInvoiceApplication(context.Background(), finance, app.Id, ReviewInvoiceApplicationInput{ReviewerId: finance.UserId, Approved: true})
	require.NoError(t, err)
	require.Equal(t, model.InvoiceStatusApproved, approved.Status)
}

func TestSecurityInvoiceDuplicateRegistrationRollsBack(t *testing.T) {
	svc, actor, order, profile, finance := invoiceSecurityFixture(t)
	first := invoiceSecurityApply(t, svc, actor, order, profile, 10)
	second := invoiceSecurityApply(t, svc, actor, order, profile, 10)
	for _, app := range []*model.InvoiceApplication{first, second} {
		_, err := svc.ReviewInvoiceApplication(context.Background(), finance, app.Id, ReviewInvoiceApplicationInput{ReviewerId: finance.UserId, Approved: true})
		require.NoError(t, err)
	}
	issue := IssueInvoiceInput{UploadedBy: finance.UserId, InvoiceNo: " 12345678901234567890 ", FileName: "invoice.pdf", FileUrl: "https://files.example.com/invoice.pdf", FileType: model.InvoiceFileTypePDF}
	_, err := svc.IssueInvoice(context.Background(), finance, first.Id, issue)
	require.NoError(t, err)
	_, err = svc.IssueInvoice(context.Background(), finance, first.Id, issue)
	require.ErrorIs(t, err, ErrInvoiceStatusInvalid)
	_, err = svc.IssueInvoice(context.Background(), finance, second.Id, issue)
	require.ErrorIs(t, err, ErrInvoiceDuplicateNumber)
	var reloaded model.InvoiceApplication
	require.NoError(t, model.DB.First(&reloaded, second.Id).Error)
	require.Equal(t, model.InvoiceStatusApproved, reloaded.Status)
	require.Nil(t, reloaded.IssuedIdentity)
	var count int64
	require.NoError(t, model.DB.Model(&model.InvoiceFile{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
	// A javascript URL must not become a customer-facing download link.
	issue.InvoiceNo, issue.FileUrl = "22345678901234567890", "javascript:alert(1)"
	_, err = svc.IssueInvoice(context.Background(), finance, second.Id, issue)
	require.ErrorIs(t, err, ErrInvoiceApplicationInvalid)
}

func TestSecurityInvoiceMissingSnapshotRequiresReapplication(t *testing.T) {
	svc, actor, order, profile, finance := invoiceSecurityFixture(t)
	app := invoiceSecurityApply(t, svc, actor, order, profile, 10)
	require.NoError(t, model.DB.Model(app).Update("buyer_snapshot", "").Error)
	_, err := svc.ReviewInvoiceApplication(context.Background(), finance, app.Id, ReviewInvoiceApplicationInput{ReviewerId: finance.UserId, Approved: true})
	require.ErrorIs(t, err, ErrInvoiceSnapshotInvalid)
	// Invalid applications can be rejected to release their reservation.
	_, err = svc.ReviewInvoiceApplication(context.Background(), finance, app.Id, ReviewInvoiceApplicationInput{ReviewerId: finance.UserId, Approved: false})
	require.NoError(t, err)
}

func TestSecurityInvoicePersonalCannotRequestSpecialVAT(t *testing.T) {
	svc, actor, order, _, _ := invoiceSecurityFixture(t)
	profile, err := svc.CreateInvoiceProfile(context.Background(), actor, CreateInvoiceProfileInput{Title: "Personal Buyer", ProfileType: model.InvoiceProfileTypePersonal})
	require.NoError(t, err)
	_, err = svc.CreateInvoiceApplication(context.Background(), actor, CreateInvoiceApplicationInput{InvoiceProfileId: profile.Id, Amount: 10, SourceId: order.Id, InvoiceType: model.InvoiceTypeVATSpecial})
	require.ErrorIs(t, err, ErrInvoiceProfileInvalid)
}

func TestSecurityInvoiceConcurrentAmountReservation(t *testing.T) {
	svc, actor, order, profile, _ := invoiceSecurityFixture(t)
	require.NoError(t, model.DB.Model(&order).Update("amount", 0.03).Error)
	var wg sync.WaitGroup
	results := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.CreateInvoiceApplication(context.Background(), actor, CreateInvoiceApplicationInput{InvoiceProfileId: profile.Id, Amount: 0.01, SourceId: order.Id})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	var succeeded int
	for err := range results {
		if err == nil {
			succeeded++
		} else {
			require.ErrorIs(t, err, ErrInvoiceAmountExceeded)
		}
	}
	require.Equal(t, 3, succeeded)
}

func TestSecurityInvoiceAuditReadOnlyAndScope(t *testing.T) {
	svc, actor, order, profile, _ := invoiceSecurityFixture(t)
	app := invoiceSecurityApply(t, svc, actor, order, profile, 10)
	auditor := InvoiceActor{UserId: 18002, Scope: model.AccessScope{TenantId: 900, RoleKey: common.RoleKeyAuditor}}
	page, err := svc.ListInvoiceApplications(context.Background(), auditor, InvoiceListInput{})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	_, err = svc.DisableInvoiceProfile(context.Background(), auditor, profile.Id)
	require.ErrorIs(t, err, ErrInvoiceOutOfScope)
	_, err = svc.CreateInvoiceApplication(context.Background(), auditor, CreateInvoiceApplicationInput{InvoiceProfileId: profile.Id, Amount: 1, SourceId: order.Id})
	require.ErrorIs(t, err, ErrInvoiceOutOfScope)
	_, err = svc.ReviewInvoiceApplication(context.Background(), auditor, app.Id, ReviewInvoiceApplicationInput{ReviewerId: auditor.UserId, Approved: true})
	require.ErrorIs(t, err, ErrInvoiceOutOfScope)
	auditor.Scope.TenantId = 901
	page, err = svc.ListInvoiceApplications(context.Background(), auditor, InvoiceListInput{})
	require.NoError(t, err)
	require.Zero(t, page.Total)
}
