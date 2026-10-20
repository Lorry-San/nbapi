package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceLightcodeGroupModelsPrunesPlanModels(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}))
	t.Cleanup(func() { DB.Where("1 = 1").Delete(&LightcodeGroupModel{}) })

	row := func(name string) LightcodeGroupModel {
		return LightcodeGroupModel{Model: name, DisplayName: name, ContextWindow: 1000, MaxOutputTokens: 100, Multiplier: 1}
	}
	hidden := row("c")
	hidden.Hidden = true
	require.NoError(t, ReplaceLightcodeGroupModels("g", []LightcodeGroupModel{row("a"), row("b"), hidden}))
	require.NoError(t, ReplaceLightcodeGroupModels("other", []LightcodeGroupModel{row("a")}))

	plan := SubscriptionPlan{Title: "p", Pool: SubscriptionPoolLightcode, RoutingGroup: "g", AllowedModels: []string{"a", "b", "c"}, DurationUnit: "month", DurationValue: 1}
	otherPlan := SubscriptionPlan{Title: "o", Pool: SubscriptionPoolLightcode, RoutingGroup: "other", AllowedModels: []string{"a"}, DurationUnit: "month", DurationValue: 1}
	require.NoError(t, DB.Create(&plan).Error)
	require.NoError(t, DB.Create(&otherPlan).Error)

	require.NoError(t, ReplaceLightcodeGroupModels("g", []LightcodeGroupModel{row("a"), hidden}))

	var storedHidden LightcodeGroupModel
	require.NoError(t, DB.Where("routing_group = ? AND model = ?", "g", "c").First(&storedHidden).Error)
	require.True(t, storedHidden.Hidden)

	var got, untouched SubscriptionPlan
	require.NoError(t, DB.First(&got, plan.Id).Error)
	require.NoError(t, DB.First(&untouched, otherPlan.Id).Error)
	require.Equal(t, []string{"a", "c"}, got.AllowedModels)
	require.Equal(t, []string{"a"}, untouched.AllowedModels)

	cached, err := getSubscriptionPlanByIdTx(nil, plan.Id)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "c"}, cached.AllowedModels)
	require.NoError(t, ValidateLightcodePlan(cached))
}
