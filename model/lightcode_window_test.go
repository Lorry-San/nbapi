package model

import (
	"testing"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestLightcodeShortAndLongWindowsLimitIndependently(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&SubscriptionPreConsumeRecord{}, &LightcodeGroupModel{}))
	t.Cleanup(func() { DB.Where("pool = ?", SubscriptionPoolLightcode).Delete(&SubscriptionPreConsumeRecord{}) })
	user := User{Username: "lightcode-window", Group: "default", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&user).Error)
	plan := SubscriptionPlan{
		Title: "Windows", Pool: SubscriptionPoolLightcode, RoutingGroup: "g", AllowedModels: []string{"m"},
		DurationUnit: "month", DurationValue: 1,
		WindowShortSeconds: 5 * 3600, WindowShortAmount: 100,
		WindowLongSeconds: 7 * 86400, WindowLongAmount: 250,
	}
	require.NoError(t, DB.Create(&plan).Error)
	var sub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		sub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, &plan, "test")
		return err
	}))

	charge := func(id string, amount int64) error {
		_, err := PreConsumeSubscriptionPool(id, user.Id, "m", amount, SubscriptionPoolLightcode, sub.Id)
		return err
	}
	require.NoError(t, charge("w1", 60))
	require.NoError(t, charge("w2", 40))
	require.ErrorIs(t, charge("w3", 1), ErrLightcodeWindowExhausted, "the 5h window is full")

	// The short window lapses; the long window keeps its 100 of 250.
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).Update("window_short_start", time.Now().Unix()-5*3600-1).Error)
	require.NoError(t, charge("w4", 100))
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).Update("window_short_start", time.Now().Unix()-5*3600-1).Error)
	require.NoError(t, charge("w5", 50))
	require.ErrorIs(t, charge("w6", 1), ErrLightcodeWindowExhausted, "the 7d window is full at 250")

	// A refund returns room to both open windows.
	require.NoError(t, RefundSubscriptionPreConsume("w5"))
	require.NoError(t, charge("w7", 50))

	// Settlement records real usage beyond the cap; the next admission is refused.
	require.NoError(t, PostConsumeUserSubscriptionDelta(sub.Id, 30))
	require.NoError(t, DB.First(sub, sub.Id).Error)
	assert.EqualValues(t, 280, sub.WindowLongUsed)
	short, long := LightcodeWindowStatus(sub, time.Now().Unix())
	require.NotNil(t, short)
	require.NotNil(t, long)
	assert.EqualValues(t, 250, long.Limit)
	assert.Greater(t, long.ResetAt, time.Now().Unix())

	// Once the long window lapses, its usage is gone for the status view as well.
	_, lapsed := LightcodeWindowStatus(sub, time.Now().Unix()+7*86400+1)
	assert.Zero(t, lapsed.Used)
	assert.Zero(t, lapsed.ResetAt)
}
