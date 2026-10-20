package model

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func migrateResetCardTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}, &LightcodeResetCard{}, &LightcodeScheduledReset{}, &UserAvatar{}))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM lightcode_reset_cards")
		DB.Exec("DELETE FROM lightcode_scheduled_resets")
		DB.Exec("DELETE FROM user_avatars")
	})
}

func newResetCardPlan(t *testing.T, title string) *SubscriptionPlan {
	t.Helper()
	plan := SubscriptionPlan{
		Title: title, Pool: SubscriptionPoolLightcode, RoutingGroup: "card-group", AllowedModels: []string{"m"},
		DurationUnit: "month", DurationValue: 1,
		WindowShortSeconds: 5 * 3600, WindowShortAmount: 100,
		WindowLongSeconds: 7 * 86400, WindowLongAmount: 500,
	}
	require.NoError(t, DB.Create(&plan).Error)
	return &plan
}

// seedUsedSubscription creates a subscription that has spent part of both windows.
func seedUsedSubscription(t *testing.T, username string, plan *SubscriptionPlan) *UserSubscription {
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
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).Updates(map[string]interface{}{
		"window_short_start": 1000, "window_short_used": 70,
		"window_long_start": 1000, "window_long_used": 400,
	}).Error)
	return sub
}

func reloadCardSubscription(t *testing.T, id int) UserSubscription {
	t.Helper()
	var sub UserSubscription
	require.NoError(t, DB.First(&sub, id).Error)
	return sub
}

func TestGiftResetCardsToOneUserCreatesUsableCards(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	plan := newResetCardPlan(t, "Gift One")
	sub := seedUsedSubscription(t, "gift-one", plan)

	users, err := GiftLightcodeResetCards(LightcodeResetTargetUser, sub.UserId, LightcodeResetCardShort, 3)

	require.NoError(t, err)
	assert.Equal(t, 1, users)
	cards, err := LightcodeUsableResetCards(sub.UserId)
	require.NoError(t, err)
	require.Len(t, cards, 3, "a user can hold several cards")
	for _, card := range cards {
		assert.Equal(t, LightcodeResetCardShort, card.Kind)
		assert.InDelta(t, 30*24*3600, card.ExpiresAt-card.CreatedAt, 1, "cards last one month")
	}
}

func TestGiftResetCardsByPlanReachesOnlyThatPlan(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	planA := newResetCardPlan(t, "Plan A")
	planB := newResetCardPlan(t, "Plan B")
	subA := seedUsedSubscription(t, "plan-a-user", planA)
	subB := seedUsedSubscription(t, "plan-b-user", planB)

	users, err := GiftLightcodeResetCards(LightcodeResetTargetPlan, planA.Id, LightcodeResetCardFull, 1)

	require.NoError(t, err)
	assert.Equal(t, 1, users)
	cardsA, _ := LightcodeUsableResetCards(subA.UserId)
	cardsB, _ := LightcodeUsableResetCards(subB.UserId)
	assert.Len(t, cardsA, 1)
	assert.Empty(t, cardsB)
}

func TestGiftResetCardsToAllReachesEverySubscriber(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	plan := newResetCardPlan(t, "Gift All")
	seedUsedSubscription(t, "all-a", plan)
	seedUsedSubscription(t, "all-b", plan)

	users, err := GiftLightcodeResetCards(LightcodeResetTargetAll, 0, LightcodeResetCardShort, 2)

	require.NoError(t, err)
	assert.Equal(t, 2, users)
	var total int64
	DB.Model(&LightcodeResetCard{}).Count(&total)
	assert.EqualValues(t, 4, total)
}

func TestGiftResetCardsRejectsInvalidInput(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	_, err := GiftLightcodeResetCards(LightcodeResetTargetUser, 0, LightcodeResetCardShort, 1)
	assert.ErrorIs(t, err, ErrResetTargetInvalid)
	_, err = GiftLightcodeResetCards(LightcodeResetTargetAll, 0, "weekly", 1)
	assert.ErrorIs(t, err, ErrResetKindInvalid)
	_, err = GiftLightcodeResetCards(LightcodeResetTargetAll, 0, LightcodeResetCardShort, 0)
	assert.Error(t, err)
	_, err = GiftLightcodeResetCards(LightcodeResetTargetAll, 0, LightcodeResetCardShort, 1)
	assert.ErrorIs(t, err, ErrResetCardNoSubscription, "no subscriber means nothing to gift")
}

func TestUseShortResetCardClearsOnlyTheFiveHourWindow(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	plan := newResetCardPlan(t, "Use Short")
	sub := seedUsedSubscription(t, "use-short", plan)
	_, err := GiftLightcodeResetCards(LightcodeResetTargetUser, sub.UserId, LightcodeResetCardShort, 1)
	require.NoError(t, err)
	cards, _ := LightcodeUsableResetCards(sub.UserId)

	used, err := UseLightcodeResetCard(sub.UserId, cards[0].Id)

	require.NoError(t, err)
	assert.Equal(t, sub.Id, used.UsedSubscriptionId)
	stored := reloadCardSubscription(t, sub.Id)
	assert.Zero(t, stored.WindowShortUsed)
	assert.EqualValues(t, 1000, stored.WindowShortStart, "a card never moves the window start")
	assert.EqualValues(t, 400, stored.WindowLongUsed, "a short card leaves the 7-day window alone")
}

func TestUseFullResetCardClearsBothWindows(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	plan := newResetCardPlan(t, "Use Full")
	sub := seedUsedSubscription(t, "use-full", plan)
	_, err := GiftLightcodeResetCards(LightcodeResetTargetUser, sub.UserId, LightcodeResetCardFull, 1)
	require.NoError(t, err)
	cards, _ := LightcodeUsableResetCards(sub.UserId)

	_, err = UseLightcodeResetCard(sub.UserId, cards[0].Id)

	require.NoError(t, err)
	stored := reloadCardSubscription(t, sub.Id)
	assert.Zero(t, stored.WindowShortUsed)
	assert.Zero(t, stored.WindowLongUsed)
	assert.EqualValues(t, 1000, stored.WindowLongStart, "a card never moves the window start")
	assert.EqualValues(t, 1000, stored.WindowShortStart)
	assert.Equal(t, sub.EndTime, stored.EndTime, "a card never moves the subscription end date")
	assert.Equal(t, sub.NextResetTime, stored.NextResetTime)
}

func TestResetCardCannotBeSpentTwiceOrByAnotherUser(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	plan := newResetCardPlan(t, "Use Once")
	owner := seedUsedSubscription(t, "card-owner", plan)
	other := seedUsedSubscription(t, "card-other", plan)
	_, err := GiftLightcodeResetCards(LightcodeResetTargetUser, owner.UserId, LightcodeResetCardShort, 1)
	require.NoError(t, err)
	cards, _ := LightcodeUsableResetCards(owner.UserId)

	_, err = UseLightcodeResetCard(other.UserId, cards[0].Id)
	assert.ErrorIs(t, err, ErrResetCardNotFound, "another user cannot spend this card")

	_, err = UseLightcodeResetCard(owner.UserId, cards[0].Id)
	require.NoError(t, err)
	_, err = UseLightcodeResetCard(owner.UserId, cards[0].Id)
	assert.ErrorIs(t, err, ErrResetCardUnavailable)
	remaining, _ := LightcodeUsableResetCards(owner.UserId)
	assert.Empty(t, remaining)
}

func TestExpiredResetCardIsNotUsable(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	plan := newResetCardPlan(t, "Expired")
	sub := seedUsedSubscription(t, "card-expired", plan)
	require.NoError(t, DB.Create(&LightcodeResetCard{UserId: sub.UserId, Kind: LightcodeResetCardShort, ExpiresAt: 1, CreatedAt: 0}).Error)
	var card LightcodeResetCard
	require.NoError(t, DB.Where("user_id = ?", sub.UserId).First(&card).Error)

	listed, _ := LightcodeUsableResetCards(sub.UserId)
	assert.Empty(t, listed)
	_, err := UseLightcodeResetCard(sub.UserId, card.Id)
	assert.ErrorIs(t, err, ErrResetCardUnavailable)
	assert.EqualValues(t, 70, reloadCardSubscription(t, sub.Id).WindowShortUsed)
}

func TestScheduledResetRejectsPastTimes(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	_, err := ScheduleLightcodeReset(LightcodeResetTargetAll, 0, GetDBTimestamp()-1, true)
	assert.ErrorIs(t, err, ErrResetScheduleInPast)
}

func TestScheduledResetIsVisibleToTheSubscriptionItReaches(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	plan := newResetCardPlan(t, "Visible")
	sub := seedUsedSubscription(t, "schedule-visible", plan)
	runAt := GetDBTimestamp() + 3600
	_, err := ScheduleLightcodeReset(LightcodeResetTargetPlan, plan.Id, runAt, true)
	require.NoError(t, err)

	pending, err := LightcodePendingResetFor(sub)

	require.NoError(t, err)
	require.NotNil(t, pending)
	assert.Equal(t, runAt, pending.RunAt)
}

func TestScheduledResetRunsOnlyOnceItIsDue(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	plan := newResetCardPlan(t, "Due")
	sub := seedUsedSubscription(t, "schedule-due", plan)
	schedule, err := ScheduleLightcodeReset(LightcodeResetTargetUser, sub.UserId, GetDBTimestamp()+3600, true)
	require.NoError(t, err)

	ran, err := RunDueLightcodeScheduledResets(10)
	require.NoError(t, err)
	assert.Zero(t, ran, "a future schedule waits")
	assert.EqualValues(t, 70, reloadCardSubscription(t, sub.Id).WindowShortUsed)

	require.NoError(t, DB.Model(&LightcodeScheduledReset{}).Where("id = ?", schedule.Id).Update("run_at", GetDBTimestamp()-1).Error)
	ran, err = RunDueLightcodeScheduledResets(10)
	require.NoError(t, err)
	assert.Equal(t, 1, ran)

	stored := reloadCardSubscription(t, sub.Id)
	assert.Zero(t, stored.WindowShortUsed)
	assert.Zero(t, stored.WindowLongUsed)
	var done LightcodeScheduledReset
	require.NoError(t, DB.First(&done, schedule.Id).Error)
	assert.Equal(t, LightcodeScheduleStatusDone, done.Status)
	assert.Equal(t, 1, done.ResetCount)

	ran, err = RunDueLightcodeScheduledResets(10)
	require.NoError(t, err)
	assert.Zero(t, ran, "a finished schedule never runs again")
}

func TestCancelledScheduledResetDoesNotRun(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	plan := newResetCardPlan(t, "Cancel")
	sub := seedUsedSubscription(t, "schedule-cancel", plan)
	schedule, err := ScheduleLightcodeReset(LightcodeResetTargetAll, 0, GetDBTimestamp()+3600, true)
	require.NoError(t, err)

	require.NoError(t, CancelLightcodeScheduledReset(schedule.Id))
	require.NoError(t, DB.Model(&LightcodeScheduledReset{}).Where("id = ?", schedule.Id).Update("run_at", GetDBTimestamp()-1).Error)
	ran, err := RunDueLightcodeScheduledResets(10)

	require.NoError(t, err)
	assert.Zero(t, ran)
	assert.EqualValues(t, 70, reloadCardSubscription(t, sub.Id).WindowShortUsed)
	pending, _ := LightcodePendingResetFor(sub)
	assert.Nil(t, pending, "a cancelled schedule is no longer shown to the user")
	assert.Error(t, CancelLightcodeScheduledReset(schedule.Id), "cancelling twice reports the schedule is gone")
}

func TestUpdateUserDisplayNameValidatesLength(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	user := User{Username: "rename-me", Group: "default", Status: common.UserStatusEnabled, AffCode: "rename-me"}
	require.NoError(t, DB.Create(&user).Error)

	name, err := UpdateUserDisplayName(user.Id, "  春日野穹  ")
	require.NoError(t, err)
	assert.Equal(t, "春日野穹", name, "surrounding spaces are trimmed")

	_, err = UpdateUserDisplayName(user.Id, "   ")
	assert.ErrorIs(t, err, ErrDisplayNameInvalid)
	_, err = UpdateUserDisplayName(user.Id, "一二三四五六七八九十一二三四五六七八九十一")
	assert.ErrorIs(t, err, ErrDisplayNameInvalid, "21 characters exceeds the limit")
}

func TestUserAvatarRoundTripAndSizeLimit(t *testing.T) {
	truncateTables(t)
	migrateResetCardTables(t)
	assert.Zero(t, UserAvatarVersion(42))

	version, err := SaveUserAvatar(42, "image/png", []byte{0x89, 'P', 'N', 'G'})
	require.NoError(t, err)
	assert.Equal(t, version, UserAvatarVersion(42))
	avatar, err := GetUserAvatar(42)
	require.NoError(t, err)
	require.NotNil(t, avatar)
	assert.Equal(t, "image/png", avatar.ContentType)

	_, err = SaveUserAvatar(42, "image/png", make([]byte, LightcodeAvatarMaxBytes+1))
	assert.Error(t, err)
	require.NoError(t, DeleteUserAvatar(42))
	missing, err := GetUserAvatar(42)
	require.NoError(t, err)
	assert.Nil(t, missing)
}
