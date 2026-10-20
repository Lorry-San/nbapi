package model

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedResetWindowSubscription creates one active Lightcode subscription holding
// used amounts in both windows, so a reset has something to clear.
func seedResetWindowSubscription(t *testing.T, username string, plan *SubscriptionPlan) *UserSubscription {
	t.Helper()
	user := User{Username: username, Group: "default", Status: common.UserStatusEnabled}
	// aff_code is unique, so each fixture user needs its own value.
	user.AffCode = username
	require.NoError(t, DB.Create(&user).Error)
	var sub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		sub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "test")
		return err
	}))
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).Updates(map[string]interface{}{
		"window_short_start": 1000, "window_short_used": 70,
		"window_long_start": 1000, "window_long_used": 400,
	}).Error)
	return sub
}

func reloadResetWindowSubscription(t *testing.T, subId int) UserSubscription {
	t.Helper()
	var sub UserSubscription
	require.NoError(t, DB.First(&sub, subId).Error)
	return sub
}

func newResetWindowPlan(t *testing.T, title string) *SubscriptionPlan {
	t.Helper()
	plan := SubscriptionPlan{
		Title: title, Pool: SubscriptionPoolLightcode, RoutingGroup: "reset-group", AllowedModels: []string{"m"},
		DurationUnit: "month", DurationValue: 1,
		WindowShortSeconds: 5 * 3600, WindowShortAmount: 100,
		WindowLongSeconds: 7 * 86400, WindowLongAmount: 500,
	}
	require.NoError(t, DB.Create(&plan).Error)
	return &plan
}

func TestAdminResetUserSubscriptionClearsBothWindows(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}))
	plan := newResetWindowPlan(t, "Reset One")
	sub := seedResetWindowSubscription(t, "reset-one", plan)

	result, err := AdminResetUserSubscription(sub.Id, true)
	require.NoError(t, err)
	require.Equal(t, 1, result.ResetCount)

	stored := reloadResetWindowSubscription(t, sub.Id)
	assert.Zero(t, stored.WindowShortStart, "the 5h window reopens for the next request")
	assert.Zero(t, stored.WindowShortUsed)
	assert.Zero(t, stored.WindowLongStart, "the 7d window reopens for the next request")
	assert.Zero(t, stored.WindowLongUsed)
	assert.Zero(t, stored.AmountUsed)
}

func TestAdminResetUserSubscriptionLeavesOtherSubscriptionsUntouched(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}))
	plan := newResetWindowPlan(t, "Reset Scoped")
	target := seedResetWindowSubscription(t, "reset-target", plan)
	other := seedResetWindowSubscription(t, "reset-other", plan)

	_, err := AdminResetUserSubscription(target.Id, true)
	require.NoError(t, err)

	untouched := reloadResetWindowSubscription(t, other.Id)
	assert.EqualValues(t, 70, untouched.WindowShortUsed)
	assert.EqualValues(t, 400, untouched.WindowLongUsed)
}

func TestAdminResetAllActiveSubscriptionsClearsEveryWindow(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}))
	first := newResetWindowPlan(t, "Reset All A")
	second := newResetWindowPlan(t, "Reset All B")
	a := seedResetWindowSubscription(t, "reset-all-a", first)
	b := seedResetWindowSubscription(t, "reset-all-b", second)

	result, err := AdminResetAllActiveSubscriptions(true)
	require.NoError(t, err)
	require.Equal(t, 2, result.ResetCount)
	assert.Equal(t, 2, result.UserCount)

	for _, subId := range []int{a.Id, b.Id} {
		stored := reloadResetWindowSubscription(t, subId)
		assert.Zero(t, stored.WindowShortUsed, "subscription %d 5h window", subId)
		assert.Zero(t, stored.WindowLongUsed, "subscription %d 7d window", subId)
		assert.Zero(t, stored.WindowShortStart)
		assert.Zero(t, stored.WindowLongStart)
	}
}

func TestAdminResetAllActiveSubscriptionsSkipsExpiredRows(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}))
	plan := newResetWindowPlan(t, "Reset All Expired")
	sub := seedResetWindowSubscription(t, "reset-expired", plan)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).
		Update("status", "expired").Error)

	_, err := AdminResetAllActiveSubscriptions(true)
	assert.Error(t, err, "an expired subscription is not an active one")

	stored := reloadResetWindowSubscription(t, sub.Id)
	assert.EqualValues(t, 70, stored.WindowShortUsed)
}
