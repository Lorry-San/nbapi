package model

import "gorm.io/gorm"

// ClearOtherRecommendedPlans keeps at most one recommended plan per pool: when the plan just
// saved is recommended, every other plan of the same pool loses the flag.
func ClearOtherRecommendedPlans(tx *gorm.DB, keepId int, pool string, recommended bool) error {
	if !recommended {
		return nil
	}
	result := tx.Model(&SubscriptionPlan{}).
		Where("pool = ? AND id <> ? AND recommended = ?", pool, keepId, true).
		Update("recommended", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		// Cached plans carry the old flag; drop them so the purchase page shows one badge.
		getSubscriptionPlanCache().Purge()
		_ = getSubscriptionPlanInfoCache().Purge()
	}
	return nil
}
