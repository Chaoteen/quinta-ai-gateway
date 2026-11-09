package model

import (
	"testing"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/stretchr/testify/require"
)

func TestSecurityCheckinClaimRollsBackForMissingUser(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Checkin{}))
	entry := &Checkin{UserId: 199991, CheckinDate: "2026-11-08", QuotaAwarded: 100, CreatedAt: time.Now().Unix()}
	_, err := userCheckinWithTransaction(entry, entry.UserId, 100)
	require.Error(t, err)
	var count int64
	require.NoError(t, DB.Model(&Checkin{}).Where("user_id = ?", entry.UserId).Count(&count).Error)
	require.Zero(t, count)
}

func TestSecurityCheckinBalanceOverflowRollsBackClaim(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Checkin{}))
	user := &User{Id: 199992, Username: "security-checkin-overflow", Password: "test-password", Quota: int(^uint(0) >> 1), Status: common.UserStatusEnabled, AffCode: "checkin-overflow"}
	require.NoError(t, DB.Create(user).Error)
	t.Cleanup(func() { DB.Exec("DELETE FROM users WHERE id = ?", user.Id) })
	entry := &Checkin{UserId: user.Id, CheckinDate: "2026-11-08", QuotaAwarded: 100, CreatedAt: time.Now().Unix()}
	_, err := userCheckinWithTransaction(entry, user.Id, 100)
	require.Error(t, err)
	var count int64
	require.NoError(t, DB.Model(&Checkin{}).Where("user_id = ?", user.Id).Count(&count).Error)
	require.Zero(t, count)
}
