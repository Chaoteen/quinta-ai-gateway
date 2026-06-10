package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"github.com/stretchr/testify/require"
)

func seedVoucherUser(t *testing.T, id int, tenantId int, organizationId int, quota int) model.User {
	t.Helper()
	user := model.User{
		Id:             id,
		TenantId:       tenantId,
		OrganizationId: organizationId,
		Username:       fmt.Sprintf("voucher-user-%d-%d", id, time.Now().UnixNano()),
		Password:       "password123",
		Role:           common.RoleCommonUser,
		RoleKey:        common.RoleKeyUser,
		Status:         common.UserStatusEnabled,
		Group:          "default",
		Quota:          quota,
		AffCode:        fmt.Sprintf("voucher-aff-%d", id),
	}
	require.NoError(t, model.DB.Create(&user).Error)
	return user
}

func seedVoucherPlan(t *testing.T, code string) model.SubscriptionPlan {
	t.Helper()
	plan := model.SubscriptionPlan{
		Code:          fmt.Sprintf("%s-%d", code, time.Now().UnixNano()),
		Name:          code,
		Title:         code,
		PriceAmount:   0,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TokenQuota:    1000,
	}
	require.NoError(t, model.DB.Create(&plan).Error)
	return plan
}

func voucherTenantActor(user model.User) VoucherActor {
	return VoucherActor{
		UserId: user.Id,
		Scope:  model.AccessScope{TenantId: user.TenantId, RoleKey: common.RoleKeyTenantAdmin},
	}
}

func TestCreateVoucherBatchAndGenerateCodes(t *testing.T) {
	truncate(t)
	admin := seedVoucherUser(t, 9601, 61, 0, 0)
	svc := NewVoucherService()

	batch, err := svc.CreateVoucherBatch(context.Background(), voucherTenantActor(admin), CreateVoucherBatchInput{
		Name:        "Token Campaign",
		VoucherType: model.VoucherTypeToken,
		CreatedBy:   admin.Id,
	})
	require.NoError(t, err)
	require.NotZero(t, batch.Id)
	require.NotEmpty(t, batch.BatchNo)
	require.Equal(t, model.VoucherBatchStatusActive, batch.Status)

	vouchers, err := svc.GenerateVouchers(context.Background(), voucherTenantActor(admin), GenerateVouchersInput{
		BatchId:     batch.Id,
		Quantity:    3,
		QuotaAmount: 500,
	})
	require.NoError(t, err)
	require.Len(t, vouchers, 3)
	require.NotEmpty(t, vouchers[0].VoucherCode)

	var reloaded model.VoucherBatch
	require.NoError(t, model.DB.Where("id = ?", batch.Id).First(&reloaded).Error)
	require.Equal(t, 3, reloaded.Quantity)
}

func TestGenerateVoucherRejectsDuplicateCode(t *testing.T) {
	truncate(t)
	admin := seedVoucherUser(t, 9602, 61, 0, 0)
	svc := NewVoucherService()
	batch, err := svc.CreateVoucherBatch(context.Background(), voucherTenantActor(admin), CreateVoucherBatchInput{
		Name:        "Duplicate Campaign",
		VoucherType: model.VoucherTypeToken,
		CreatedBy:   admin.Id,
	})
	require.NoError(t, err)
	_, err = svc.GenerateVouchers(context.Background(), voucherTenantActor(admin), GenerateVouchersInput{
		BatchId:     batch.Id,
		QuotaAmount: 100,
		Codes:       []string{"DUP-CODE", "DUP-CODE"},
	})
	require.ErrorIs(t, err, ErrVoucherAlreadyRedeemed)
}

func TestRedeemTokenVoucherIsIdempotent(t *testing.T) {
	truncate(t)
	admin := seedVoucherUser(t, 9603, 62, 0, 0)
	user := seedVoucherUser(t, 9604, 62, 0, 100)
	svc := NewVoucherService()
	batch, err := svc.CreateVoucherBatch(context.Background(), voucherTenantActor(admin), CreateVoucherBatchInput{
		Name:        "Token Redeem",
		VoucherType: model.VoucherTypeToken,
		CreatedBy:   admin.Id,
	})
	require.NoError(t, err)
	vouchers, err := svc.GenerateVouchers(context.Background(), voucherTenantActor(admin), GenerateVouchersInput{
		BatchId:     batch.Id,
		QuotaAmount: 900,
		Codes:       []string{"TOKEN-ONCE"},
	})
	require.NoError(t, err)

	first, err := svc.RedeemVoucher(context.Background(), user.Id, vouchers[0].VoucherCode)
	require.NoError(t, err)
	second, err := svc.RedeemVoucher(context.Background(), user.Id, vouchers[0].VoucherCode)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)

	var reloaded model.User
	require.NoError(t, model.DB.Select("quota").Where("id = ?", user.Id).First(&reloaded).Error)
	require.Equal(t, 1000, reloaded.Quota)

	var count int64
	require.NoError(t, model.DB.Model(&model.VoucherRedemption{}).Where("voucher_id = ?", vouchers[0].Id).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestRedeemSubscriptionVoucherCreatesSubscription(t *testing.T) {
	truncate(t)
	admin := seedVoucherUser(t, 9605, 63, 0, 0)
	user := seedVoucherUser(t, 9606, 63, 0, 0)
	plan := seedVoucherPlan(t, "voucher-plan")
	svc := NewVoucherService()
	batch, err := svc.CreateVoucherBatch(context.Background(), voucherTenantActor(admin), CreateVoucherBatchInput{
		Name:        "Subscription Voucher",
		VoucherType: model.VoucherTypeSubscription,
		CreatedBy:   admin.Id,
	})
	require.NoError(t, err)
	vouchers, err := svc.GenerateVouchers(context.Background(), voucherTenantActor(admin), GenerateVouchersInput{
		BatchId:            batch.Id,
		SubscriptionPlanId: plan.Id,
		Codes:              []string{"SUB-ONCE"},
	})
	require.NoError(t, err)
	_, err = svc.RedeemVoucher(context.Background(), user.Id, vouchers[0].VoucherCode)
	require.NoError(t, err)

	var count int64
	require.NoError(t, model.DB.Model(&model.UserSubscription{}).Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestRedeemVoucherRejectsExpiredAndDisabled(t *testing.T) {
	truncate(t)
	admin := seedVoucherUser(t, 9607, 64, 0, 0)
	user := seedVoucherUser(t, 9608, 64, 0, 0)
	svc := NewVoucherService()
	batch, err := svc.CreateVoucherBatch(context.Background(), voucherTenantActor(admin), CreateVoucherBatchInput{
		Name:        "Invalid Voucher",
		VoucherType: model.VoucherTypeToken,
		CreatedBy:   admin.Id,
	})
	require.NoError(t, err)
	expired, err := svc.GenerateVouchers(context.Background(), voucherTenantActor(admin), GenerateVouchersInput{
		BatchId:     batch.Id,
		QuotaAmount: 100,
		ExpiredAt:   common.GetTimestamp() - 1,
		Codes:       []string{"EXPIRED-CODE"},
	})
	require.NoError(t, err)
	_, err = svc.RedeemVoucher(context.Background(), user.Id, expired[0].VoucherCode)
	require.ErrorIs(t, err, ErrVoucherExpired)

	disabled, err := svc.GenerateVouchers(context.Background(), voucherTenantActor(admin), GenerateVouchersInput{
		BatchId:     batch.Id,
		QuotaAmount: 100,
		Codes:       []string{"DISABLED-CODE"},
	})
	require.NoError(t, err)
	_, err = svc.DisableVoucher(context.Background(), voucherTenantActor(admin), disabled[0].Id)
	require.NoError(t, err)
	_, err = svc.RedeemVoucher(context.Background(), user.Id, disabled[0].VoucherCode)
	require.ErrorIs(t, err, ErrVoucherDisabled)
}

func TestVoucherOwnershipIsolation(t *testing.T) {
	truncate(t)
	tenantA := seedVoucherUser(t, 9609, 65, 0, 0)
	tenantB := seedVoucherUser(t, 9610, 66, 0, 0)
	svc := NewVoucherService()
	batchA, err := svc.CreateVoucherBatch(context.Background(), voucherTenantActor(tenantA), CreateVoucherBatchInput{
		Name:        "Tenant A",
		VoucherType: model.VoucherTypeToken,
		CreatedBy:   tenantA.Id,
	})
	require.NoError(t, err)
	_, err = svc.CreateVoucherBatch(context.Background(), voucherTenantActor(tenantB), CreateVoucherBatchInput{
		Name:        "Tenant B",
		VoucherType: model.VoucherTypeToken,
		CreatedBy:   tenantB.Id,
	})
	require.NoError(t, err)

	page, err := svc.ListBatches(context.Background(), voucherTenantActor(tenantA), VoucherListInput{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Equal(t, batchA.Id, page.Items[0].Id)

	_, err = svc.GenerateVouchers(context.Background(), voucherTenantActor(tenantB), GenerateVouchersInput{
		BatchId:     batchA.Id,
		Quantity:    1,
		QuotaAmount: 100,
	})
	require.True(t, errors.Is(err, ErrVoucherOutOfScope), "got %v", err)
}
