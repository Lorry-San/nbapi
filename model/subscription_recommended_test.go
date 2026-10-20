package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func recommendedPlanIds(t *testing.T, pool string) []int {
	t.Helper()
	var ids []int
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("pool = ? AND recommended = ?", pool, true).Order("id").Pluck("id", &ids).Error)
	return ids
}

func TestRecommendingAPlanClearsOthersOfTheSamePoolOnly(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	first := newResetCardPlan(t, "Rec First")
	second := newResetCardPlan(t, "Rec Second")
	api := SubscriptionPlan{Title: "Rec Api", Pool: SubscriptionPoolAPI, Recommended: true, DurationUnit: "month", DurationValue: 1}
	require.NoError(t, DB.Create(&api).Error)
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", first.Id).Update("recommended", true).Error)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		require.NoError(t, tx.Model(&SubscriptionPlan{}).Where("id = ?", second.Id).Update("recommended", true).Error)
		return ClearOtherRecommendedPlans(tx, second.Id, SubscriptionPoolLightcode, true)
	}))

	assert.Equal(t, []int{second.Id}, recommendedPlanIds(t, SubscriptionPoolLightcode), "one badge per pool")
	assert.Equal(t, []int{api.Id}, recommendedPlanIds(t, SubscriptionPoolAPI), "another pool keeps its own")
}

func TestSavingAPlanWithoutTheFlagLeavesTheCurrentRecommendation(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	current := newResetCardPlan(t, "Rec Current")
	other := newResetCardPlan(t, "Rec Other")
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", current.Id).Update("recommended", true).Error)

	require.NoError(t, ClearOtherRecommendedPlans(DB, other.Id, SubscriptionPoolLightcode, false))

	assert.Equal(t, []int{current.Id}, recommendedPlanIds(t, SubscriptionPoolLightcode))
}
