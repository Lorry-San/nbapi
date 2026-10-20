package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Reset card kinds. A short card clears only the 5-hour window; a full card clears both.
const (
	LightcodeResetCardShort = "short"
	LightcodeResetCardFull  = "full"
)

// Reset targets shared by gifting cards and scheduling resets.
const (
	LightcodeResetTargetAll  = "all"
	LightcodeResetTargetPlan = "plan"
	LightcodeResetTargetUser = "user"
)

// LightcodeResetCardValidity is how long a gifted card stays usable.
const LightcodeResetCardValidity = 30 * 24 * time.Hour

var (
	ErrResetCardNotFound      = errors.New("重置卡不存在")
	ErrResetCardUnavailable   = errors.New("重置卡已使用或已过期")
	ErrResetCardNoSubscription = errors.New("没有可重置的有效 Lightcode 订阅")
	ErrResetTargetInvalid     = errors.New("无效的重置对象")
	ErrResetKindInvalid       = errors.New("无效的重置卡类型")
	ErrResetScheduleInPast    = errors.New("定时重置时间必须晚于当前时间")
)

// LightcodeResetCard is one gifted card a user may spend on their own subscription.
type LightcodeResetCard struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id" gorm:"index"`
	Kind      string `json:"kind" gorm:"type:varchar(16)"`
	Source    string `json:"source" gorm:"type:varchar(16)"`
	ExpiresAt int64  `json:"expires_at" gorm:"bigint;index"`
	UsedAt    int64  `json:"used_at" gorm:"bigint;default:0"`
	// UsedSubscriptionId records which subscription the card cleared.
	UsedSubscriptionId int   `json:"used_subscription_id" gorm:"default:0"`
	CreatedAt          int64 `json:"created_at" gorm:"bigint"`
}

// Usable reports whether the card can still be spent at now.
func (card *LightcodeResetCard) Usable(now int64) bool {
	return card.UsedAt == 0 && card.ExpiresAt > now
}

// LightcodeScheduledReset clears subscription windows at a chosen future time.
type LightcodeScheduledReset struct {
	Id         int    `json:"id"`
	TargetType string `json:"target_type" gorm:"type:varchar(16);index"`
	// TargetId is the plan id or user id; zero for the all target.
	TargetId int   `json:"target_id" gorm:"index"`
	RunAt    int64 `json:"run_at" gorm:"bigint;index"`
	// AdvanceResetTime mirrors the immediate reset option.
	AdvanceResetTime bool   `json:"advance_reset_time"`
	Status           string `json:"status" gorm:"type:varchar(16);index"`
	ResetCount       int    `json:"reset_count"`
	Error            string `json:"error" gorm:"type:text"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint"`
	ExecutedAt       int64  `json:"executed_at" gorm:"bigint;default:0"`
}

// Scheduled reset lifecycle.
const (
	LightcodeScheduleStatusPending   = "pending"
	LightcodeScheduleStatusDone      = "done"
	LightcodeScheduleStatusFailed    = "failed"
	LightcodeScheduleStatusCancelled = "cancelled"
)

func validResetTarget(targetType string, targetId int) error {
	switch targetType {
	case LightcodeResetTargetAll:
		return nil
	case LightcodeResetTargetPlan, LightcodeResetTargetUser:
		if targetId <= 0 {
			return ErrResetTargetInvalid
		}
		return nil
	default:
		return ErrResetTargetInvalid
	}
}

// lightcodeResetTargetUsers resolves which users a target reaches, counting only users
// that hold an active Lightcode subscription so cards are not gifted to non-subscribers.
func lightcodeResetTargetUsers(tx *gorm.DB, targetType string, targetId int, now int64) ([]int, error) {
	query := tx.Model(&UserSubscription{}).
		Where("pool = ? AND status = ? AND end_time > ?", SubscriptionPoolLightcode, "active", now)
	switch targetType {
	case LightcodeResetTargetPlan:
		query = query.Where("plan_id = ?", targetId)
	case LightcodeResetTargetUser:
		query = query.Where("user_id = ?", targetId)
	}
	var userIds []int
	if err := query.Distinct("user_id").Pluck("user_id", &userIds).Error; err != nil {
		return nil, err
	}
	return userIds, nil
}

// GiftLightcodeResetCards gives count cards of kind to every user the target reaches and
// returns how many users received them.
func GiftLightcodeResetCards(targetType string, targetId int, kind string, count int) (int, error) {
	if err := validResetTarget(targetType, targetId); err != nil {
		return 0, err
	}
	if kind != LightcodeResetCardShort && kind != LightcodeResetCardFull {
		return 0, ErrResetKindInvalid
	}
	if count <= 0 || count > 100 {
		return 0, errors.New("每人赠送数量需在 1 到 100 之间")
	}
	now := GetDBTimestamp()
	expires := now + int64(LightcodeResetCardValidity/time.Second)
	var users int
	err := DB.Transaction(func(tx *gorm.DB) error {
		userIds, err := lightcodeResetTargetUsers(tx, targetType, targetId, now)
		if err != nil {
			return err
		}
		if len(userIds) == 0 {
			return ErrResetCardNoSubscription
		}
		cards := make([]LightcodeResetCard, 0, len(userIds)*count)
		for _, userId := range userIds {
			for i := 0; i < count; i++ {
				cards = append(cards, LightcodeResetCard{
					UserId: userId, Kind: kind, Source: targetType, ExpiresAt: expires, CreatedAt: now,
				})
			}
		}
		if err := tx.Create(&cards).Error; err != nil {
			return err
		}
		users = len(userIds)
		return nil
	})
	return users, err
}

// LightcodeUsableResetCards lists the cards a user can still spend, soonest expiry first.
func LightcodeUsableResetCards(userId int) ([]LightcodeResetCard, error) {
	var cards []LightcodeResetCard
	err := DB.Where("user_id = ? AND used_at = 0 AND expires_at > ?", userId, GetDBTimestamp()).
		Order("expires_at asc, id asc").Find(&cards).Error
	return cards, err
}

// UseLightcodeResetCard spends one of the user's cards on their active subscription.
// A short card clears the 5-hour window; a full card clears both windows.
func UseLightcodeResetCard(userId, cardId int) (*LightcodeResetCard, error) {
	var card LightcodeResetCard
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockLightcodeUser(tx, userId); err != nil {
			return err
		}
		if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", cardId, userId).First(&card).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrResetCardNotFound
			}
			return err
		}
		now := getDBTimestampTx(tx)
		if !card.Usable(now) {
			return ErrResetCardUnavailable
		}
		var sub UserSubscription
		if err := tx.Where("user_id = ? AND pool = ? AND status = ? AND end_time > ?", userId, SubscriptionPoolLightcode, "active", now).
			Order("end_time asc, id asc").First(&sub).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrResetCardNoSubscription
			}
			return err
		}
		updates := map[string]interface{}{"window_short_start": 0, "window_short_used": 0}
		if card.Kind == LightcodeResetCardFull {
			updates["window_long_start"] = 0
			updates["window_long_used"] = 0
		}
		if err := tx.Model(&UserSubscription{}).Where("id = ?", sub.Id).Updates(updates).Error; err != nil {
			return err
		}
		card.UsedAt, card.UsedSubscriptionId = now, sub.Id
		return tx.Model(&LightcodeResetCard{}).Where("id = ?", card.Id).
			Updates(map[string]interface{}{"used_at": now, "used_subscription_id": sub.Id}).Error
	})
	if err != nil {
		return nil, err
	}
	return &card, nil
}

// ScheduleLightcodeReset records a reset to run at runAt.
func ScheduleLightcodeReset(targetType string, targetId int, runAt int64, advanceResetTime bool) (*LightcodeScheduledReset, error) {
	if err := validResetTarget(targetType, targetId); err != nil {
		return nil, err
	}
	now := GetDBTimestamp()
	if runAt <= now {
		return nil, ErrResetScheduleInPast
	}
	if targetType == LightcodeResetTargetAll {
		targetId = 0
	}
	schedule := &LightcodeScheduledReset{
		TargetType: targetType, TargetId: targetId, RunAt: runAt, AdvanceResetTime: advanceResetTime,
		Status: LightcodeScheduleStatusPending, CreatedAt: now,
	}
	return schedule, DB.Create(schedule).Error
}

// CancelLightcodeScheduledReset withdraws a schedule that has not run yet.
func CancelLightcodeScheduledReset(id int) error {
	result := DB.Model(&LightcodeScheduledReset{}).
		Where("id = ? AND status = ?", id, LightcodeScheduleStatusPending).
		Update("status", LightcodeScheduleStatusCancelled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("定时重置不存在或已执行")
	}
	return nil
}

// ListLightcodeScheduledResets returns recent schedules, newest first.
func ListLightcodeScheduledResets(limit int) ([]LightcodeScheduledReset, error) {
	var schedules []LightcodeScheduledReset
	err := DB.Order("run_at desc, id desc").Limit(limit).Find(&schedules).Error
	return schedules, err
}

// LightcodePendingResetFor returns the soonest pending reset that reaches this subscription.
func LightcodePendingResetFor(sub *UserSubscription) (*LightcodeScheduledReset, error) {
	if sub == nil {
		return nil, nil
	}
	var schedule LightcodeScheduledReset
	err := DB.Where("status = ? AND (target_type = ? OR (target_type = ? AND target_id = ?) OR (target_type = ? AND target_id = ?))",
		LightcodeScheduleStatusPending,
		LightcodeResetTargetAll,
		LightcodeResetTargetPlan, sub.PlanId,
		LightcodeResetTargetUser, sub.UserId).
		Order("run_at asc, id asc").First(&schedule).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

// RunDueLightcodeScheduledResets executes every pending schedule whose time has come.
func RunDueLightcodeScheduledResets(limit int) (int, error) {
	now := GetDBTimestamp()
	var due []LightcodeScheduledReset
	if err := DB.Where("status = ? AND run_at <= ?", LightcodeScheduleStatusPending, now).
		Order("run_at asc, id asc").Limit(limit).Find(&due).Error; err != nil {
		return 0, err
	}
	for i := range due {
		schedule := &due[i]
		// Claim the row first so a concurrent tick never runs the same schedule twice.
		claim := DB.Model(&LightcodeScheduledReset{}).
			Where("id = ? AND status = ?", schedule.Id, LightcodeScheduleStatusPending).
			Update("status", "running")
		if claim.Error != nil || claim.RowsAffected == 0 {
			continue
		}
		count, err := executeLightcodeScheduledReset(schedule)
		updates := map[string]interface{}{"executed_at": GetDBTimestamp(), "reset_count": count, "status": LightcodeScheduleStatusDone}
		if err != nil {
			updates["status"] = LightcodeScheduleStatusFailed
			updates["error"] = err.Error()
		}
		DB.Model(&LightcodeScheduledReset{}).Where("id = ?", schedule.Id).Updates(updates)
	}
	return len(due), nil
}

func executeLightcodeScheduledReset(schedule *LightcodeScheduledReset) (int, error) {
	var (
		result *SubscriptionResetResult
		err    error
	)
	switch schedule.TargetType {
	case LightcodeResetTargetAll:
		result, err = AdminResetAllActiveSubscriptions(schedule.AdvanceResetTime)
	case LightcodeResetTargetPlan:
		result, err = AdminResetPlanSubscriptions(schedule.TargetId, schedule.AdvanceResetTime)
	case LightcodeResetTargetUser:
		result, err = adminResetLightcodeUser(schedule.TargetId, schedule.AdvanceResetTime)
	default:
		return 0, ErrResetTargetInvalid
	}
	if err != nil {
		return 0, err
	}
	return result.ResetCount, nil
}

// adminResetLightcodeUser clears every active Lightcode subscription a user holds.
func adminResetLightcodeUser(userId int, advanceResetTime bool) (*SubscriptionResetResult, error) {
	var result *SubscriptionResetResult
	now := GetDBTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		var subs []UserSubscription
		if err := lockForUpdate(tx).
			Where("user_id = ? AND pool = ? AND status = ? AND end_time > ?", userId, SubscriptionPoolLightcode, "active", now).
			Find(&subs).Error; err != nil {
			return err
		}
		if len(subs) == 0 {
			return ErrResetCardNoSubscription
		}
		for i := range subs {
			plan, err := freshSubscriptionPlanTx(tx, subs[i].PlanId)
			if err != nil {
				return err
			}
			if err := resetUserSubscriptionTx(tx, &subs[i], plan, now, advanceResetTime); err != nil {
				return err
			}
		}
		result = buildSubscriptionResetResult(&SubscriptionPlan{}, subs, advanceResetTime)
		return nil
	})
	return result, err
}
