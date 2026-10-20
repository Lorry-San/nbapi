package model

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newLiveEntitlementPlan(t *testing.T, title string, models ...string) *SubscriptionPlan {
	t.Helper()
	plan := SubscriptionPlan{
		Title: title, Pool: SubscriptionPoolLightcode, RoutingGroup: "live-group", AllowedModels: models,
		DurationUnit: "month", DurationValue: 1,
		WindowLongSeconds: 7 * 86400, WindowLongAmount: 500,
	}
	require.NoError(t, DB.Create(&plan).Error)
	return &plan
}

func seedLiveEntitlementSubscription(t *testing.T, username string, plan *SubscriptionPlan) *UserSubscription {
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

func setPlanModels(t *testing.T, planId int, models ...string) {
	t.Helper()
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", planId).
		Update("allowed_models", common.GetJsonString(models)).Error)
}

// A model added to a plan must work for existing subscribers immediately; the subscription
// row keeps a copy, but entitlement reads the plan, so no sync step is required.
func TestLightcodeEntitlementFollowsPlanAdditions(t *testing.T) {
	truncateTables(t)
	plan := newLiveEntitlementPlan(t, "Live Add", "claude-opus-5")
	sub := seedLiveEntitlementSubscription(t, "live-add", plan)
	policies := []LightcodeGroupModel{
		{RoutingGroup: "live-group", Model: "claude-opus-5"},
		{RoutingGroup: "live-group", Model: "gpt-6-luna"},
	}

	_, reason := LightcodeModelEntitlement(sub, "gpt-6-luna", policies)
	assert.Equal(t, "subscription_model_unsupported", reason)

	setPlanModels(t, plan.Id, "claude-opus-5", "gpt-6-luna")

	_, reason = LightcodeModelEntitlement(sub, "gpt-6-luna", policies)
	assert.Empty(t, reason, "the plan edit reaches the running subscriber")
}

// Removal is the other half of live entitlement: a model dropped from the plan stops working.
func TestLightcodeEntitlementFollowsPlanRemovals(t *testing.T) {
	truncateTables(t)
	plan := newLiveEntitlementPlan(t, "Live Remove", "claude-opus-5", "gpt-6-sol")
	sub := seedLiveEntitlementSubscription(t, "live-remove", plan)
	policies := []LightcodeGroupModel{
		{RoutingGroup: "live-group", Model: "claude-opus-5"},
		{RoutingGroup: "live-group", Model: "gpt-6-sol"},
	}

	_, reason := LightcodeModelEntitlement(sub, "gpt-6-sol", policies)
	assert.Empty(t, reason)

	setPlanModels(t, plan.Id, "claude-opus-5")

	_, reason = LightcodeModelEntitlement(sub, "gpt-6-sol", policies)
	assert.Equal(t, "subscription_model_unsupported", reason)
}

// A subscription whose plan row vanished must keep working off its stored list instead of
// losing every model at once.
func TestLightcodeEntitlementFallsBackWhenPlanIsMissing(t *testing.T) {
	truncateTables(t)
	assert.Nil(t, LightcodeAllowedModels(nil))

	sub := &UserSubscription{PlanId: 999999, RoutingGroup: "live-group", AllowedModels: []string{"claude-sonnet-5"}}
	assert.Equal(t, []string{"claude-sonnet-5"}, LightcodeAllowedModels(sub))

	policies := []LightcodeGroupModel{{RoutingGroup: "live-group", Model: "claude-sonnet-5"}}
	_, reason := LightcodeModelEntitlement(sub, "claude-sonnet-5", policies)
	assert.Empty(t, reason)
}

// Order changes must not change entitlement: a plan reorder keeps every model allowed.
func TestLightcodeEntitlementIgnoresPlanOrder(t *testing.T) {
	truncateTables(t)
	plan := newLiveEntitlementPlan(t, "Live Order", "claude-opus-5", "gemini-3-flash")
	sub := seedLiveEntitlementSubscription(t, "live-order", plan)
	policies := []LightcodeGroupModel{
		{RoutingGroup: "live-group", Model: "claude-opus-5"},
		{RoutingGroup: "live-group", Model: "gemini-3-flash"},
	}

	setPlanModels(t, plan.Id, "gemini-3-flash", "claude-opus-5")

	for _, name := range []string{"claude-opus-5", "gemini-3-flash"} {
		_, reason := LightcodeModelEntitlement(sub, name, policies)
		assert.Empty(t, reason, name)
	}
}
