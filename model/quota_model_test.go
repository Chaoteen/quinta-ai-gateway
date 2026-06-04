package model

import (
	"strconv"
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/stretchr/testify/require"
)

func TestParseModelQuotaSnapshotEmptyIsUnrestricted(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		parsed, err := ParseModelQuotaSnapshot(raw)
		require.NoError(t, err)
		require.True(t, parsed.Unrestricted)
		require.Empty(t, parsed.Allow)
	}
}

func TestParseModelQuotaSnapshotAllowlist(t *testing.T) {
	parsed, err := ParseModelQuotaSnapshot(`{"allow":["gpt-4o"," gpt-4o-mini ","gpt-4o",""]}`)
	require.NoError(t, err)
	require.False(t, parsed.Unrestricted)
	require.Equal(t, []string{"gpt-4o", "gpt-4o-mini"}, parsed.Allow)
}

func TestParseModelQuotaSnapshotCleanupIsDeterministic(t *testing.T) {
	parsed, err := ParseModelQuotaSnapshot(`{"allow":[" gpt-4o ","gpt-4o","gpt-4o-mini"," gpt-4o-mini "]}`)
	require.NoError(t, err)
	require.False(t, parsed.Unrestricted)
	require.Equal(t, []string{"gpt-4o", "gpt-4o-mini"}, parsed.Allow)
}

func TestParseModelQuotaSnapshotInvalidJSON(t *testing.T) {
	_, err := ParseModelQuotaSnapshot(`{"allow":`)
	require.Error(t, err)
}

func TestQuotaReservationStatusesAndDefault(t *testing.T) {
	truncateTables(t)

	cases := []struct {
		name       string
		status     string
		wantStatus string
	}{
		{name: "default status", status: "", wantStatus: QuotaReservationStatusActive},
		{name: "active", status: QuotaReservationStatusActive, wantStatus: QuotaReservationStatusActive},
		{name: "committed", status: QuotaReservationStatusCommitted, wantStatus: QuotaReservationStatusCommitted},
		{name: "rolled back", status: QuotaReservationStatusRolledBack, wantStatus: QuotaReservationStatusRolledBack},
		{name: "expired", status: QuotaReservationStatusExpired, wantStatus: QuotaReservationStatusExpired},
	}

	for idx, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reservation := QuotaReservation{
				ReservationId:   "quota-reservation-status-" + strconv.Itoa(idx+1),
				RequestId:       "quota-request-status-" + strconv.Itoa(idx+1),
				UserId:          100 + idx,
				ModelName:       " gpt-4o ",
				Status:          tc.status,
				TokenReserved:   1000,
				RequestReserved: 1,
			}
			require.NoError(t, DB.Create(&reservation).Error)
			require.Equal(t, tc.wantStatus, reservation.Status)
			require.Equal(t, "gpt-4o", reservation.ModelName)
			require.NotZero(t, reservation.CreatedAt)
			require.NotZero(t, reservation.UpdatedAt)
		})
	}
}

func TestQuotaUsageRecordStatuses(t *testing.T) {
	truncateTables(t)

	cases := []struct {
		name      string
		dimension string
		status    string
	}{
		{name: "reserved", dimension: QuotaDimensionToken, status: QuotaUsageStatusReserved},
		{name: "committed", dimension: QuotaDimensionRequest, status: QuotaUsageStatusCommitted},
		{name: "rolled back", dimension: QuotaDimensionModel, status: QuotaUsageStatusRolledBack},
		{name: "reset", dimension: QuotaDimensionReset, status: QuotaUsageStatusReset},
	}

	for idx, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := QuotaUsageRecord{
				RequestId:      "quota-usage-request-" + strconv.Itoa(idx+1),
				ReservationId:  "quota-usage-reservation-" + strconv.Itoa(idx+1),
				UserId:         200 + idx,
				ModelName:      "gpt-4o",
				QuotaDimension: " " + tc.dimension + " ",
				TokenDelta:     100,
				RequestDelta:   1,
				Status:         " " + tc.status + " ",
			}
			require.NoError(t, DB.Create(&record).Error)
			require.Equal(t, tc.dimension, record.QuotaDimension)
			require.Equal(t, tc.status, record.Status)
			require.NotZero(t, record.OccurredAt)
			require.NotZero(t, record.CreatedAt)
			require.NotZero(t, record.UpdatedAt)
		})
	}
}

func TestQuotaEngineModelsAreMigrated(t *testing.T) {
	require.True(t, DB.Migrator().HasTable(&QuotaReservation{}))
	require.True(t, DB.Migrator().HasTable(&QuotaUsageRecord{}))
}

func TestQuotaFoundationRecordsDoNotMutateLegacyQuotaOrOrders(t *testing.T) {
	truncateTables(t)

	user := User{Id: 3101, Username: "quota-safety-user", Quota: 5000, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&user).Error)
	token := Token{Id: 3201, UserId: user.Id, Key: "quota-safety-token", Name: "quota safety", Status: common.TokenStatusEnabled, RemainQuota: 4000}
	require.NoError(t, DB.Create(&token).Error)
	order := SubscriptionOrder{Id: 3301, UserId: user.Id, TradeNo: "quota-safety-order", Status: common.TopUpStatusPending, Money: 12.34}
	require.NoError(t, DB.Create(&order).Error)

	require.NoError(t, DB.Create(&QuotaReservation{
		ReservationId:      "quota-safety-reservation",
		RequestId:          "quota-safety-request",
		UserId:             user.Id,
		UserSubscriptionId: 3401,
		ModelName:          "gpt-4o",
		TokenReserved:      100,
		RequestReserved:    1,
	}).Error)
	require.NoError(t, DB.Create(&QuotaUsageRecord{
		RequestId:          "quota-safety-request",
		ReservationId:      "quota-safety-reservation",
		UserId:             user.Id,
		UserSubscriptionId: 3401,
		ModelName:          "gpt-4o",
		QuotaDimension:     QuotaDimensionToken,
		TokenDelta:         100,
		RequestDelta:       1,
		Status:             QuotaUsageStatusReserved,
	}).Error)

	var gotUser User
	require.NoError(t, DB.Select("quota").Where("id = ?", user.Id).First(&gotUser).Error)
	require.Equal(t, 5000, gotUser.Quota)

	var gotToken Token
	require.NoError(t, DB.Select("remain_quota").Where("id = ?", token.Id).First(&gotToken).Error)
	require.Equal(t, 4000, gotToken.RemainQuota)

	var gotOrder SubscriptionOrder
	require.NoError(t, DB.Where("id = ?", order.Id).First(&gotOrder).Error)
	require.Equal(t, common.TopUpStatusPending, gotOrder.Status)
	require.Equal(t, 12.34, gotOrder.Money)
}
