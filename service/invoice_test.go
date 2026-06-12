package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func resetInvoiceTables(t *testing.T) {
	t.Helper()
	cleanup := func() {
		model.DB.Exec("DELETE FROM invoice_files")
		model.DB.Exec("DELETE FROM invoice_applications")
		model.DB.Exec("DELETE FROM invoice_profiles")
		model.DB.Exec("DELETE FROM payment_orders")
		model.DB.Exec("DELETE FROM users")
	}
	cleanup()
	t.Cleanup(cleanup)
}

func seedInvoiceUser(t *testing.T, id int, tenantId int) model.User {
	t.Helper()
	user := model.User{
		Id:       id,
		TenantId: tenantId,
		Username: fmt.Sprintf("invoice-user-%d-%d", id, time.Now().UnixNano()),
		Role:     common.RoleCommonUser,
		RoleKey:  common.RoleKeyUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  fmt.Sprintf("invoice-aff-%d", id),
	}
	require.NoError(t, model.DB.Create(&user).Error)
	return user
}

func seedInvoicePayment(t *testing.T, id int, user model.User, amount float64, status string) model.PaymentOrder {
	t.Helper()
	order := model.PaymentOrder{
		Id:           id,
		OrderNo:      fmt.Sprintf("INV-PAY-%d-%d", id, time.Now().UnixNano()),
		TenantId:     user.TenantId,
		UserId:       user.Id,
		Provider:     model.PaymentProviderMock,
		BusinessType: model.PaymentBusinessTokenRecharge,
		Amount:       amount,
		Currency:     "USD",
		Status:       status,
		Subject:      "invoice payment",
	}
	require.NoError(t, model.DB.Create(&order).Error)
	return order
}

func invoiceUserActor(user model.User) InvoiceActor {
	return InvoiceActor{UserId: user.Id, Scope: model.AccessScope{TenantId: user.TenantId, RoleKey: common.RoleKeyUser}}
}

func invoiceAdminActor(user model.User) InvoiceActor {
	return InvoiceActor{UserId: user.Id, Scope: model.AccessScope{TenantId: user.TenantId, RoleKey: common.RoleKeyTenantAdmin}}
}

func TestInvoiceProfileDefaultMutualExclusion(t *testing.T) {
	resetInvoiceTables(t)
	user := seedInvoiceUser(t, 11001, 111)
	svc := NewInvoiceService()

	first, err := svc.CreateInvoiceProfile(context.Background(), invoiceUserActor(user), CreateInvoiceProfileInput{
		ProfileType: model.InvoiceProfileTypeCompany,
		Title:       "Company A",
		TaxNo:       "TAX-A",
		IsDefault:   true,
	})
	require.NoError(t, err)
	second, err := svc.CreateInvoiceProfile(context.Background(), invoiceUserActor(user), CreateInvoiceProfileInput{
		ProfileType: model.InvoiceProfileTypeCompany,
		Title:       "Company B",
		TaxNo:       "TAX-B",
		IsDefault:   true,
	})
	require.NoError(t, err)

	var reloaded model.InvoiceProfile
	require.NoError(t, model.DB.Where("id = ?", first.Id).First(&reloaded).Error)
	require.False(t, reloaded.IsDefault)
	require.True(t, second.IsDefault)
}

func TestInvoiceApplicationPaymentValidationAndAmountLimit(t *testing.T) {
	resetInvoiceTables(t)
	user := seedInvoiceUser(t, 11002, 112)
	paid := seedInvoicePayment(t, 11101, user, 100, model.PaymentOrderStatusPaid)
	unpaid := seedInvoicePayment(t, 11102, user, 50, model.PaymentOrderStatusPending)
	svc := NewInvoiceService()
	profile, err := svc.CreateInvoiceProfile(context.Background(), invoiceUserActor(user), CreateInvoiceProfileInput{Title: "Company", TaxNo: "TAX"})
	require.NoError(t, err)

	_, err = svc.CreateInvoiceApplication(context.Background(), invoiceUserActor(user), CreateInvoiceApplicationInput{
		InvoiceProfileId: profile.Id,
		Amount:           10,
		InvoiceType:      model.InvoiceTypeVATNormal,
		SourceType:       model.InvoiceSourcePaymentOrder,
		SourceId:         unpaid.Id,
	})
	require.ErrorIs(t, err, ErrInvoiceSourceInvalid)

	first, err := svc.CreateInvoiceApplication(context.Background(), invoiceUserActor(user), CreateInvoiceApplicationInput{
		InvoiceProfileId: profile.Id,
		Amount:           70,
		InvoiceType:      model.InvoiceTypeVATNormal,
		SourceType:       model.InvoiceSourcePaymentOrder,
		SourceId:         paid.Id,
	})
	require.NoError(t, err)
	require.Equal(t, model.InvoiceStatusPending, first.Status)

	_, err = svc.CreateInvoiceApplication(context.Background(), invoiceUserActor(user), CreateInvoiceApplicationInput{
		InvoiceProfileId: profile.Id,
		Amount:           40,
		InvoiceType:      model.InvoiceTypeVATNormal,
		SourceType:       model.InvoiceSourcePaymentOrder,
		SourceId:         paid.Id,
	})
	require.ErrorIs(t, err, ErrInvoiceAmountExceeded)
}

func TestInvoiceReviewIssueAndFile(t *testing.T) {
	resetInvoiceTables(t)
	user := seedInvoiceUser(t, 11003, 113)
	admin := seedInvoiceUser(t, 11004, 113)
	order := seedInvoicePayment(t, 11201, user, 120, model.PaymentOrderStatusPaid)
	svc := NewInvoiceService()
	profile, err := svc.CreateInvoiceProfile(context.Background(), invoiceUserActor(user), CreateInvoiceProfileInput{Title: "Company", TaxNo: "TAX"})
	require.NoError(t, err)
	app, err := svc.CreateInvoiceApplication(context.Background(), invoiceUserActor(user), CreateInvoiceApplicationInput{
		InvoiceProfileId: profile.Id,
		Amount:           120,
		InvoiceType:      model.InvoiceTypeVATSpecial,
		SourceType:       model.InvoiceSourcePaymentOrder,
		SourceId:         order.Id,
	})
	require.NoError(t, err)

	approved, err := svc.ReviewInvoiceApplication(context.Background(), invoiceAdminActor(admin), app.Id, ReviewInvoiceApplicationInput{ReviewerId: admin.Id, Approved: true, ReviewNote: "ok"})
	require.NoError(t, err)
	require.Equal(t, model.InvoiceStatusApproved, approved.Status)

	issued, err := svc.IssueInvoice(context.Background(), invoiceAdminActor(admin), app.Id, IssueInvoiceInput{
		UploadedBy: admin.Id,
		InvoiceNo:  "INV-001",
		FileName:   "invoice.pdf",
		FileUrl:    "https://example.com/invoice.pdf",
		FileType:   model.InvoiceFileTypePDF,
	})
	require.NoError(t, err)
	require.Equal(t, model.InvoiceStatusIssued, issued.Status)
	require.Equal(t, "INV-001", issued.InvoiceNo)
	require.NotZero(t, issued.IssuedAt)

	files, err := svc.ListInvoiceFiles(context.Background(), invoiceUserActor(user), InvoiceListInput{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), files.Total)
	require.Equal(t, "invoice.pdf", files.Items[0].FileName)
}

func TestInvoiceOwnershipIsolation(t *testing.T) {
	resetInvoiceTables(t)
	userA := seedInvoiceUser(t, 11005, 114)
	userB := seedInvoiceUser(t, 11006, 115)
	orderA := seedInvoicePayment(t, 11301, userA, 80, model.PaymentOrderStatusPaid)
	orderB := seedInvoicePayment(t, 11302, userB, 90, model.PaymentOrderStatusPaid)
	svc := NewInvoiceService()
	profileA, err := svc.CreateInvoiceProfile(context.Background(), invoiceUserActor(userA), CreateInvoiceProfileInput{Title: "A", TaxNo: "TAX-A"})
	require.NoError(t, err)
	profileB, err := svc.CreateInvoiceProfile(context.Background(), invoiceUserActor(userB), CreateInvoiceProfileInput{Title: "B", TaxNo: "TAX-B"})
	require.NoError(t, err)
	_, err = svc.CreateInvoiceApplication(context.Background(), invoiceUserActor(userA), CreateInvoiceApplicationInput{InvoiceProfileId: profileA.Id, Amount: 80, SourceType: model.InvoiceSourcePaymentOrder, SourceId: orderA.Id})
	require.NoError(t, err)
	_, err = svc.CreateInvoiceApplication(context.Background(), invoiceUserActor(userB), CreateInvoiceApplicationInput{InvoiceProfileId: profileB.Id, Amount: 90, SourceType: model.InvoiceSourcePaymentOrder, SourceId: orderB.Id})
	require.NoError(t, err)

	page, err := svc.ListInvoiceApplications(context.Background(), invoiceAdminActor(userA), InvoiceListInput{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Equal(t, userA.Id, page.Items[0].UserId)
}
