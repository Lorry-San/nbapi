package model

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newSyncPlan(t *testing.T, title, model string) *SubscriptionPlan {
	t.Helper()
	plan := SubscriptionPlan{
		Title: title, Pool: SubscriptionPoolLightcode, RoutingGroup: "sync-group",
		AllowedModels: []string{model},
		DurationUnit:  "month", DurationValue: 1,
		WindowLongSeconds: 7 * 86400, WindowLongAmount: 500,
	}
	require.NoError(t, DB.Create(&plan).Error)
	return &plan
}

func seedSyncSubscription(t *testing.T, username string, plan *SubscriptionPlan) *UserSubscription {
	t.Helper()
	user := User{Username: username, Group: "default", Status: common.UserStatusEnabled}
	user.AffCode = username
	require.NoError(t, DB.Create(&user).Error)
	var sub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		sub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "test")
		return err
	}))
	return sub
}

func reloadSyncSubscription(t *testing.T, subId int) UserSubscription {
	t.Helper()
	var sub UserSubscription
	require.NoError(t, DB.First(&sub, subId).Error)
	return sub
}

// A plan gaining a model must reach subscriptions that already exist; editing a plan alone
// leaves them on their purchase-time snapshot, which is what users see as
// "this subscription does not support that model".
func TestSyncLightcodeSubscriptionsToPlanAddsNewModels(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}))
	plan := newSyncPlan(t, "Sync Add", "claude-opus-5")
	sub := seedSyncSubscription(t, "sync-add", plan)
	require.Equal(t, []string{"claude-opus-5"}, reloadSyncSubscription(t, sub.Id).AllowedModels)

	plan.AllowedModels = []string{"claude-opus-5", "kimi-k3"}
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Update("allowed_models", common.GetJsonString(plan.AllowedModels)).Error)

	results, err := SyncLightcodeSubscriptionsToPlan(plan.Id)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 1, results[0].UpdatedCount)
	assert.Equal(t, []string{"kimi-k3"}, results[0].AddedModels)
	assert.Empty(t, results[0].RemovedModels)

	stored := reloadSyncSubscription(t, sub.Id)
	assert.Equal(t, []string{"claude-opus-5", "kimi-k3"}, stored.AllowedModels)
}

// Quota and window state must survive a sync: only entitlements move.
func TestSyncLightcodeSubscriptionsToPlanPreservesUsage(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}))
	plan := newSyncPlan(t, "Sync Usage", "claude-opus-5")
	sub := seedSyncSubscription(t, "sync-usage", plan)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).Updates(map[string]interface{}{
		"amount_used": 900, "window_long_start": 1000, "window_long_used": 250,
	}).Error)

	plan.AllowedModels = []string{"claude-opus-5", "qwen3.8-max"}
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Update("allowed_models", common.GetJsonString(plan.AllowedModels)).Error)

	_, err := SyncLightcodeSubscriptionsToPlan(plan.Id)
	require.NoError(t, err)

	stored := reloadSyncSubscription(t, sub.Id)
	assert.EqualValues(t, 900, stored.AmountUsed)
	assert.EqualValues(t, 1000, stored.WindowLongStart)
	assert.EqualValues(t, 250, stored.WindowLongUsed)
	assert.Contains(t, stored.AllowedModels, "qwen3.8-max")
}

// Models removed from the plan must disappear, which is the other half of the sync.
func TestSyncLightcodeSubscriptionsToPlanRemovesDroppedModels(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}))
	plan := newSyncPlan(t, "Sync Drop", "claude-opus-5")
	sub := seedSyncSubscription(t, "sync-drop", plan)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).
		Update("allowed_models", common.GetJsonString([]string{"claude-opus-5", "gpt-6-sol"})).Error)

	results, err := SyncLightcodeSubscriptionsToPlan(plan.Id)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, []string{"gpt-6-sol"}, results[0].RemovedModels)

	assert.Equal(t, []string{"claude-opus-5"}, reloadSyncSubscription(t, sub.Id).AllowedModels)
}

// Expired rows are history; they must not be rewritten.
func TestSyncLightcodeSubscriptionsToPlanSkipsExpired(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}))
	plan := newSyncPlan(t, "Sync Expired", "claude-opus-5")
	sub := seedSyncSubscription(t, "sync-expired", plan)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).
		Update("status", "expired").Error)

	results, err := SyncLightcodeSubscriptionsToPlan(plan.Id)
	require.NoError(t, err)
	assert.Empty(t, results)

	assert.Equal(t, []string{"claude-opus-5"}, reloadSyncSubscription(t, sub.Id).AllowedModels)
}
