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

func TestLightcodeWindowStatusExcludesInFlightPreConsumes(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&SubscriptionPreConsumeRecord{}, &LightcodeGroupModel{}))
	user := User{Username: "lightcode-in-flight", Group: "default", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&user).Error)
	plan := SubscriptionPlan{
		Title: "In-flight", Pool: SubscriptionPoolLightcode, RoutingGroup: "g", AllowedModels: []string{"m"},
		WindowLongSeconds: 7 * 86400, WindowLongAmount: 200,
	}
	planJSON, err := common.Marshal(plan)
	require.NoError(t, err)
	sub := UserSubscription{UserId: user.Id, PlanId: plan.Id, Pool: SubscriptionPoolLightcode,
		PlanSnapshot: string(planJSON), Status: "active", WindowLongStart: 100, WindowLongUsed: 200}
	require.NoError(t, DB.Create(&sub).Error)
	require.NoError(t, DB.Create(&[]SubscriptionPreConsumeRecord{
		{Pool: SubscriptionPoolLightcode, RequestId: "settled-request", UserId: user.Id, UserSubscriptionId: sub.Id, PreConsumed: 100, Status: "consumed", CreatedAt: 100},
		{Pool: SubscriptionPoolLightcode, RequestId: "in-flight-request", UserId: user.Id, UserSubscriptionId: sub.Id, PreConsumed: 100, Status: "consumed", CreatedAt: 100},
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{UserId: user.Id, Type: LogTypeConsume, CreatedAt: 101, RequestId: "settled-request"}).Error)
	_, long := LightcodeWindowStatus(&sub, 101)
	require.NotNil(t, long)
	assert.EqualValues(t, 100, long.Used, "the in-flight reservation must not appear in the dashboard")
	assert.Equal(t, 50, long.UsedPercent)
}

func TestLightcodeWindowStatusPublishesPercentageWithoutRawQuota(t *testing.T) {
	plan := SubscriptionPlan{WindowShortSeconds: 18000, WindowShortAmount: 100, WindowLongSeconds: 604800, WindowLongAmount: 200}
	data, err := common.Marshal(plan)
	require.NoError(t, err)
	sub := UserSubscription{Pool: SubscriptionPoolLightcode, PlanSnapshot: string(data),
		WindowShortStart: 100, WindowShortUsed: 27, WindowLongStart: 100, WindowLongUsed: 450}
	short, long := LightcodeWindowStatus(&sub, 101)
	require.NotNil(t, short)
	require.NotNil(t, long)
	assert.Equal(t, 27, short.UsedPercent)
	assert.Equal(t, 100, long.UsedPercent, "settlement overruns stop at 100 percent in the UI")
	encoded, err := common.Marshal(short)
	require.NoError(t, err)
	assert.JSONEq(t, `{"used_percent":27,"reset_at":18100}`, string(encoded))
	short, _ = LightcodeWindowStatus(&sub, 18101)
	assert.Equal(t, 0, short.UsedPercent, "an expired window has no usage")
}
