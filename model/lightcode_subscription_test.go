package model

import (
	"testing"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestLightcodeRenewalAndIsolatedBilling(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&SubscriptionPreConsumeRecord{}, &LightcodeGroupModel{}))
	t.Cleanup(func() { DB.Where("pool = ?", SubscriptionPoolLightcode).Delete(&SubscriptionPreConsumeRecord{}) })
	user := User{Username: "lightcode-renewal", Group: "api-group", Quota: 5000, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&user).Error)
	plan := SubscriptionPlan{Title: "Lightcode Pro", Pool: SubscriptionPoolLightcode, RoutingGroup: "lc-pro", AllowedModels: []string{"model-a"}, TotalAmount: 1000, DurationUnit: "month", DurationValue: 1}
	require.NoError(t, DB.Create(&plan).Error)
	create := func() *UserSubscription {
		var sub *UserSubscription
		require.NoError(t, DB.Transaction(func(tx *gorm.DB) error { var err error; sub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, &plan, "test"); return err }))
		return sub
	}
	active, pending := create(), create()
	assert.Equal(t, "active", active.Status)
	assert.Equal(t, "pending", pending.Status)
	assert.Equal(t, active.EndTime, pending.StartTime)
	err := DB.Transaction(func(tx *gorm.DB) error { _, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, &plan, "test"); return err })
	require.ErrorIs(t, err, ErrLightcodePendingSubscription)
	hasAPI, err := HasActiveUserSubscription(user.Id)
	require.NoError(t, err); assert.False(t, hasAPI)
	_, err = PreConsumeUserSubscription("lc-api-denied", user.Id, "model-a", 0, 100)
	require.Error(t, err)
	_, err = PreConsumeSubscriptionPool("lc-wrong-model", user.Id, "model-b", 100, SubscriptionPoolLightcode, active.Id)
	require.Error(t, err)
	_, err = PreConsumeSubscriptionPool("lc-charge", user.Id, "model-a", 100, SubscriptionPoolLightcode, active.Id)
	require.NoError(t, err)
	_, err = PreConsumeSubscriptionPool("lc-charge", user.Id, "model-a", 100, SubscriptionPoolLightcode, active.Id)
	require.NoError(t, err, "a repeated request id is idempotent")
	require.NoError(t, DB.First(active, active.Id).Error); assert.EqualValues(t, 100, active.AmountUsed)
	require.NoError(t, RefundSubscriptionPreConsume("lc-charge"))
	require.NoError(t, RefundSubscriptionPreConsume("lc-charge"))
	require.NoError(t, DB.First(active, active.Id).Error); assert.Zero(t, active.AmountUsed)
	require.NoError(t, DB.First(&user, user.Id).Error)
	assert.Equal(t, "api-group", user.Group); assert.Equal(t, 5000, user.Quota)
	now := time.Now().Unix()
	require.NoError(t, DB.Model(active).Update("end_time", now-1).Error)
	require.NoError(t, DB.Model(pending).Update("start_time", now-1).Error)
	subs, err := LightcodeSubscriptions(user.Id)
	require.NoError(t, err); require.Len(t, subs, 1)
	assert.Equal(t, pending.Id, subs[0].Id); assert.Equal(t, "active", subs[0].Status)
}
