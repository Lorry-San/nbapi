package model

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"gorm.io/gorm"
)

const (
	SubscriptionPoolAPI       = "api"
	SubscriptionPoolLightcode = "lightcode"
)

var ErrLightcodePendingSubscription = errors.New("a Lightcode renewal is already pending")

// ReplaceLightcodeGroupModels swaps a group's model list and drops removed models from the
// group's plans, which validation would otherwise reject on the next plan save. Existing
// subscriptions keep their purchase-time snapshot.
func ReplaceLightcodeGroupModels(group string, models []LightcodeGroupModel) error {
	var changed []int
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("routing_group = ?", group).Delete(&LightcodeGroupModel{}).Error; err != nil {
			return err
		}
		if len(models) > 0 {
			for i := range models {
				models[i].Id, models[i].RoutingGroup = 0, group
			}
			if err := tx.Create(&models).Error; err != nil {
				return err
			}
		}
		keep := make(map[string]bool, len(models))
		for _, m := range models {
			keep[m.Model] = true
		}
		var plans []SubscriptionPlan
		if err := tx.Where("pool = ? AND routing_group = ?", SubscriptionPoolLightcode, group).Find(&plans).Error; err != nil {
			return err
		}
		for _, plan := range plans {
			allowed := slices.DeleteFunc(slices.Clone(plan.AllowedModels), func(name string) bool { return !keep[name] })
			if len(allowed) == len(plan.AllowedModels) {
				continue
			}
			plan.AllowedModels = allowed
			if err := tx.Model(&plan).Select("allowed_models").Updates(&plan).Error; err != nil {
				return err
			}
			changed = append(changed, plan.Id)
		}
		return nil
	})
	if err == nil {
		for _, id := range changed {
			InvalidateSubscriptionPlanCache(id)
		}
	}
	return err
}

// LightcodeSubscriptionSyncResult reports how many live subscriptions picked up a plan's
// current model list.
type LightcodeSubscriptionSyncResult struct {
	PlanId          int      `json:"plan_id"`
	PlanTitle       string   `json:"plan_title"`
	UpdatedCount    int      `json:"updated_count"`
	AffectedUserIds []int    `json:"affected_user_ids"`
	AddedModels     []string `json:"added_models"`
	RemovedModels   []string `json:"removed_models"`
}

// SyncLightcodeSubscriptionsToPlan refreshes the frozen model list of live subscriptions
// (optionally limited to one plan) from their plan. Quotas, windows and reset times are
// untouched: only entitlements that the operator changed on the plan move forward.
func SyncLightcodeSubscriptionsToPlan(planId int) ([]LightcodeSubscriptionSyncResult, error) {
	var plans []SubscriptionPlan
	query := DB.Where("pool = ?", SubscriptionPoolLightcode)
	if planId > 0 {
		query = query.Where("id = ?", planId)
	}
	if err := query.Find(&plans).Error; err != nil {
		return nil, err
	}
	var results []LightcodeSubscriptionSyncResult
	err := DB.Transaction(func(tx *gorm.DB) error {
		for i := range plans {
			plan := &plans[i]
			if plan.RoutingGroup == "" || len(plan.AllowedModels) == 0 {
				continue
			}
			// Only subscriptions whose snapshot really differs need a write.
			var subs []UserSubscription
			if err := tx.Where("plan_id = ? AND pool = ? AND status IN ?", plan.Id, SubscriptionPoolLightcode, []string{"active", "pending"}).Find(&subs).Error; err != nil {
				return err
			}
			result := LightcodeSubscriptionSyncResult{PlanId: plan.Id, PlanTitle: plan.Title}
			encoded := common.GetJsonString(plan.AllowedModels)
			added := make(map[string]bool)
			removed := make(map[string]bool)
			for j := range subs {
				sub := &subs[j]
				if slices.Equal(sub.AllowedModels, plan.AllowedModels) {
					continue
				}
				// The pre-sync list is what the subscription advertises today, so the
				// delta reports what users gain and lose from this sync.
				for _, name := range plan.AllowedModels {
					if !slices.Contains(sub.AllowedModels, name) {
						added[name] = true
					}
				}
				for _, name := range sub.AllowedModels {
					if !slices.Contains(plan.AllowedModels, name) {
						removed[name] = true
					}
				}
				if err := tx.Model(&UserSubscription{}).Where("id = ?", sub.Id).
					UpdateColumn("allowed_models", encoded).Error; err != nil {
					return err
				}
				result.UpdatedCount++
				result.AffectedUserIds = append(result.AffectedUserIds, sub.UserId)
			}
			if result.UpdatedCount == 0 {
				continue
			}
			result.AddedModels = slices.Sorted(maps.Keys(added))
			result.RemovedModels = slices.Sorted(maps.Keys(removed))
			results = append(results, result)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// LightcodeGroupModel adds presentation and reasoning policy to a channel routing group.
type LightcodeGroupModel struct {
	Id              int      `json:"id"`
	RoutingGroup    string   `json:"routing_group" gorm:"type:varchar(64);uniqueIndex:idx_lc_group_model"`
	Model           string   `json:"model" gorm:"type:varchar(191);uniqueIndex:idx_lc_group_model"`
	DisplayName     string   `json:"display_name" gorm:"type:varchar(191)"`
	Efforts         []string `json:"efforts" gorm:"serializer:json;type:text"`
	DefaultEffort   string   `json:"default_effort" gorm:"type:varchar(32)"`
	ContextWindow   int      `json:"context_window"`
	MaxOutputTokens int      `json:"max_output_tokens"`
	Vision          bool     `json:"vision"`
	// Hidden keeps an internal route callable without exposing it in the Lightcode model selector.
	Hidden          bool     `json:"hidden"`
	// Multiplier is applied to Lightcode consumption only; API pricing is unaffected. 0 means 1.
	Multiplier float64 `json:"multiplier" gorm:"not null;default:1"`
}

func (m *LightcodeGroupModel) EffectiveMultiplier() float64 {
	if m.Multiplier <= 0 {
		return 1
	}
	return m.Multiplier
}

func ValidateLightcodePlan(plan *SubscriptionPlan) error {
	plan.NormalizeDefaults()
	if plan.Pool != SubscriptionPoolAPI && plan.Pool != SubscriptionPoolLightcode {
		return errors.New("invalid subscription pool")
	}
	if plan.Pool == SubscriptionPoolAPI {
		return nil
	}
	if plan.UpgradeGroup != "" || plan.DowngradeGroup != "" {
		return errors.New("Lightcode subscriptions cannot change the API user group")
	}
	plan.AllowWalletOverflow = common.GetPointer(false)
	if strings.TrimSpace(plan.RoutingGroup) == "" || plan.RoutingGroup == "auto" {
		return errors.New("Lightcode requires one explicit routing group")
	}
	if len(plan.AllowedModels) == 0 {
		return errors.New("select at least one Lightcode model")
	}
	for _, w := range [][2]int64{{plan.WindowShortSeconds, plan.WindowShortAmount}, {plan.WindowLongSeconds, plan.WindowLongAmount}} {
		if w[0] < 0 || w[1] < 0 || (w[0] == 0) != (w[1] == 0) {
			return errors.New("each usage window needs both a positive length and a positive quota, or neither")
		}
	}
	if plan.WindowShortSeconds > 0 && plan.WindowLongSeconds > 0 && plan.WindowShortSeconds >= plan.WindowLongSeconds {
		return errors.New("the short usage window must be shorter than the long one")
	}
	var models []LightcodeGroupModel
	if err := DB.Where("routing_group = ?", plan.RoutingGroup).Find(&models).Error; err != nil {
		return err
	}
	for _, name := range plan.AllowedModels {
		if !slices.ContainsFunc(models, func(m LightcodeGroupModel) bool { return m.Model == name }) {
			return fmt.Errorf("model %q is not configured in Lightcode group %q", name, plan.RoutingGroup)
		}
	}
	return nil
}

// LightcodeAllowedModels is the live entitlement list for a subscription: the plan's current
// model list, not a copy frozen at purchase. Editing a plan therefore reaches every active
// subscriber on the next request, which is what makes new models appear without a sync step.
// It reads the plan row directly rather than through the plan cache, so an entitlement change
// is never served stale. A missing plan falls back to the stored list so an orphaned
// subscription keeps working.
func LightcodeAllowedModels(sub *UserSubscription) []string {
	if sub == nil {
		return nil
	}
	var plan SubscriptionPlan
	if err := DB.Where("id = ?", sub.PlanId).First(&plan).Error; err != nil {
		return sub.AllowedModels
	}
	if len(plan.AllowedModels) == 0 {
		return sub.AllowedModels
	}
	return plan.AllowedModels
}

// LightcodeModelEntitlement is the shared model check for admission and catalog; reason is "" when allowed.
func LightcodeModelEntitlement(sub *UserSubscription, name string, policies []LightcodeGroupModel) (policy *LightcodeGroupModel, reason string) {
	if !slices.Contains(LightcodeAllowedModels(sub), name) {
		return nil, "subscription_model_unsupported"
	}
	for i := range policies {
		if policies[i].RoutingGroup == sub.RoutingGroup && policies[i].Model == name {
			return &policies[i], ""
		}
	}
	return nil, "model_unavailable"
}

// The write obtains SQLite's writer reservation; row locking covers MySQL and PostgreSQL.
func lockLightcodeUser(tx *gorm.DB, userId int) error {
	if err := tx.Model(&User{}).Where("id = ?", userId).UpdateColumn("id", gorm.Expr("id")).Error; err != nil {
		return err
	}
	var user User
	return lockForUpdate(tx).Select("id").First(&user, userId).Error
}

func advanceLightcodeSubscriptionsTx(tx *gorm.DB, userId int, now int64) error {
	if err := tx.Model(&UserSubscription{}).Where("user_id = ? AND pool = ? AND status IN ? AND end_time <= ?", userId, SubscriptionPoolLightcode, []string{"active", "pending"}, now).Updates(map[string]interface{}{"status": "expired", "updated_at": now}).Error; err != nil {
		return err
	}
	var active int64
	if err := tx.Model(&UserSubscription{}).Where("user_id = ? AND pool = ? AND status = ?", userId, SubscriptionPoolLightcode, "active").Count(&active).Error; err != nil {
		return err
	}
	if active > 0 {
		return nil
	}
	return tx.Model(&UserSubscription{}).Where("user_id = ? AND pool = ? AND status = ? AND start_time <= ? AND end_time > ?", userId, SubscriptionPoolLightcode, "pending", now, now).Updates(map[string]interface{}{"status": "active", "updated_at": now}).Error
}

func createLightcodeSubscriptionTx(tx *gorm.DB, userId int, plan *SubscriptionPlan, source string) (*UserSubscription, error) {
	if err := lockLightcodeUser(tx, userId); err != nil {
		return nil, err
	}
	now := getDBTimestampTx(tx)
	if err := advanceLightcodeSubscriptionsTx(tx, userId, now); err != nil {
		return nil, err
	}
	var current []UserSubscription
	if err := tx.Where("user_id = ? AND pool = ? AND status IN ?", userId, SubscriptionPoolLightcode, []string{"active", "pending"}).Find(&current).Error; err != nil {
		return nil, err
	}
	start, status := now, "active"
	for _, sub := range current {
		if sub.Status == "pending" {
			return nil, ErrLightcodePendingSubscription
		}
		start, status = sub.EndTime, "pending"
	}
	end, err := calcPlanEndTime(time.Unix(start, 0), plan)
	if err != nil {
		return nil, err
	}
	if plan.RoutingGroup == "" || len(plan.AllowedModels) == 0 || plan.UpgradeGroup != "" || plan.DowngradeGroup != "" {
		return nil, errors.New("invalid Lightcode subscription entitlements")
	}
	snapshot, err := common.Marshal(plan)
	if err != nil {
		return nil, err
	}
	sub := &UserSubscription{
		UserId: userId, PlanId: plan.Id, Pool: SubscriptionPoolLightcode,
		RoutingGroup: plan.RoutingGroup,
		// Entitlements are read from the plan, so the stored copy is only a fallback for a
		// subscription whose plan disappears later.
		AllowedModels: slices.Clone(plan.AllowedModels),
		PlanSnapshot: string(snapshot), AmountTotal: plan.TotalAmount, StartTime: start, EndTime: end,
		Status: status, Source: source, LastResetTime: start,
		NextResetTime: calcNextResetTime(time.Unix(start, 0), plan, end), AllowWalletOverflow: false,
	}
	return sub, tx.Create(sub).Error
}

// ErrLightcodeWindowExhausted reports that a usage window has no room left for the request.
var ErrLightcodeWindowExhausted = errors.New("Lightcode usage window exhausted")

// lightcodeWindows reads the window limits frozen into the subscription at purchase.
func lightcodeWindows(sub *UserSubscription) (shortSec, shortCap, longSec, longCap int64) {
	var plan SubscriptionPlan
	if sub.PlanSnapshot == "" || common.UnmarshalJsonStr(sub.PlanSnapshot, &plan) != nil {
		return 0, 0, 0, 0
	}
	return plan.WindowShortSeconds, plan.WindowShortAmount, plan.WindowLongSeconds, plan.WindowLongAmount
}

// rollLightcodeWindows clears windows whose period ended; it never opens one.
func rollLightcodeWindows(sub *UserSubscription, now int64) {
	shortSec, _, longSec, _ := lightcodeWindows(sub)
	if sub.WindowShortStart > 0 && (shortSec <= 0 || now >= sub.WindowShortStart+shortSec) {
		sub.WindowShortStart, sub.WindowShortUsed = 0, 0
	}
	if sub.WindowLongStart > 0 && (longSec <= 0 || now >= sub.WindowLongStart+longSec) {
		sub.WindowLongStart, sub.WindowLongUsed = 0, 0
	}
}

// applyLightcodeWindows charges delta (negative refunds) against both windows,
// opening a lapsed window at the first positive charge. Admission (enforce)
// rejects a charge that would overrun a window; settlement records the real
// usage even past the cap, because the response was already delivered, and the
// overrun blocks the next admission instead.
func applyLightcodeWindows(sub *UserSubscription, delta int64, now int64, enforce bool) error {
	if sub.Pool != SubscriptionPoolLightcode {
		return nil
	}
	rollLightcodeWindows(sub, now)
	shortSec, shortCap, longSec, longCap := lightcodeWindows(sub)
	if delta > 0 && enforce {
		if shortCap > 0 && sub.WindowShortUsed > shortCap-delta {
			return ErrLightcodeWindowExhausted
		}
		if longCap > 0 && sub.WindowLongUsed > longCap-delta {
			return ErrLightcodeWindowExhausted
		}
	}
	if delta > 0 {
		if shortSec > 0 && shortCap > 0 && sub.WindowShortStart == 0 {
			sub.WindowShortStart = now
		}
		if longSec > 0 && longCap > 0 && sub.WindowLongStart == 0 {
			sub.WindowLongStart = now
		}
	}
	if sub.WindowShortStart > 0 {
		sub.WindowShortUsed = max(sub.WindowShortUsed+delta, 0)
	}
	if sub.WindowLongStart > 0 {
		sub.WindowLongUsed = max(sub.WindowLongUsed+delta, 0)
	}
	return nil
}

// LightcodeWindowView is the client-facing state of one usage window.
type LightcodeWindowView struct {
	Limit       int64 `json:"-"`
	Used        int64 `json:"-"`
	UsedPercent int   `json:"used_percent"`
	ResetAt     int64 `json:"reset_at"`
}

// LightcodeWindowsView pairs the short and long windows; a nil side means the plan has no such window.
type LightcodeWindowsView struct {
	Short *LightcodeWindowView `json:"short"`
	Long  *LightcodeWindowView `json:"long"`
}

// lightcodeSettledWindowUsed excludes reservations for requests that have not
// written their consume log yet. Reservations still remain in Window*Used for
// admission, but they must not make the dashboard percentage jump while a long
// request is in flight.
func lightcodeSettledWindowUsed(sub *UserSubscription, start, used int64) int64 {
	if sub == nil || sub.Id <= 0 || sub.UserId <= 0 || start <= 0 || used <= 0 || DB == nil || LOG_DB == nil {
		return used
	}
	var pending []SubscriptionPreConsumeRecord
	if err := DB.Select("request_id, pre_consumed").Where(
		"pool = ? AND user_id = ? AND user_subscription_id = ? AND status = ? AND created_at >= ?",
		SubscriptionPoolLightcode, sub.UserId, sub.Id, "consumed", start,
	).Find(&pending).Error; err != nil || len(pending) == 0 {
		return used
	}
	ids := make([]string, 0, len(pending))
	amounts := make(map[string]int64, len(pending))
	for _, record := range pending {
		if record.RequestId == "" || record.PreConsumed <= 0 {
			continue
		}
		ids = append(ids, record.RequestId)
		amounts[record.RequestId] = record.PreConsumed
	}
	if len(ids) == 0 {
		return used
	}
	var settled []string
	if err := LOG_DB.Table("logs").Where("user_id = ? AND type = ? AND request_id IN ?", sub.UserId, LogTypeConsume, ids).Pluck("request_id", &settled).Error; err != nil {
		return used
	}
	settledSet := make(map[string]struct{}, len(settled))
	for _, requestID := range settled {
		settledSet[requestID] = struct{}{}
	}
	var inFlight int64
	for requestID, amount := range amounts {
		if _, ok := settledSet[requestID]; !ok {
			inFlight += amount
		}
	}
	return max(used-inFlight, 0)
}

// LightcodeWindowStatus projects both windows without mutating state; a lapsed window reads as empty.
func LightcodeWindowStatus(sub *UserSubscription, now int64) (short, long *LightcodeWindowView) {
	view := *sub
	rollLightcodeWindows(&view, now)
	shortSec, shortCap, longSec, longCap := lightcodeWindows(&view)
	if shortCap > 0 && shortSec > 0 {
		short = &LightcodeWindowView{Limit: shortCap, Used: lightcodeSettledWindowUsed(&view, view.WindowShortStart, view.WindowShortUsed)}
		if view.WindowShortStart > 0 {
			short.ResetAt = view.WindowShortStart + shortSec
		}
	}
	if longCap > 0 && longSec > 0 {
		long = &LightcodeWindowView{Limit: longCap, Used: lightcodeSettledWindowUsed(&view, view.WindowLongStart, view.WindowLongUsed)}
		if view.WindowLongStart > 0 {
			long.ResetAt = view.WindowLongStart + longSec
		}
	}
	for _, window := range []*LightcodeWindowView{short, long} {
		if window != nil {
			window.UsedPercent = int(min(100, max(0, math.Ceil(float64(window.Used)/float64(window.Limit)*100))))
		}
	}
	return short, long
}

// LightcodeSubscriptions advances due renewals before exposing current entitlements.
func LightcodeSubscriptions(userId int) ([]UserSubscription, error) {
	var subs []UserSubscription
	// Fast path: lock-free read; fall through to the locked path whenever any expiry, activation or reset is due.
	now := GetDBTimestamp()
	if err := DB.Where("user_id = ? AND pool = ? AND status IN ?", userId, SubscriptionPoolLightcode, []string{"active", "pending"}).Order("start_time asc").Find(&subs).Error; err != nil {
		return nil, err
	}
	if !lightcodeMaintenanceDue(subs, now) {
		return subs, nil
	}
	subs = nil
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockLightcodeUser(tx, userId); err != nil {
			return err
		}
		now := getDBTimestampTx(tx)
		if err := advanceLightcodeSubscriptionsTx(tx, userId, now); err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND pool = ? AND status IN ?", userId, SubscriptionPoolLightcode, []string{"active", "pending"}).Order("start_time asc").Find(&subs).Error; err != nil {
			return err
		}
		for i := range subs {
			if subs[i].Status != "active" {
				continue
			}
			var plan SubscriptionPlan
			if err := common.UnmarshalJsonStr(subs[i].PlanSnapshot, &plan); err != nil {
				return err
			}
			if err := maybeResetUserSubscriptionWithPlanTx(tx, &subs[i], &plan, now); err != nil {
				return err
			}
		}
		return nil
	})
	return subs, err
}

func lightcodeMaintenanceDue(subs []UserSubscription, now int64) bool {
	hasActive := false
	for _, s := range subs {
		if s.Status == "active" {
			hasActive = true
		}
	}
	for _, s := range subs {
		if s.EndTime <= now {
			return true
		}
		if s.Status == "pending" && !hasActive && s.StartTime <= now {
			return true
		}
		if s.Status != "active" {
			continue
		}
		var plan SubscriptionPlan
		if common.UnmarshalJsonStr(s.PlanSnapshot, &plan) != nil {
			return true
		}
		if NormalizeResetPeriod(plan.QuotaResetPeriod) != SubscriptionResetNever && (s.NextResetTime <= 0 || s.NextResetTime <= now) {
			return true
		}
	}
	return false
}

// ActivateDueLightcodeRenewals also handles pending renewals after a long outage.
func ActivateDueLightcodeRenewals(limit int) error {
	var users []int
	now := GetDBTimestamp()
	if err := DB.Model(&UserSubscription{}).Distinct("user_id").
		Where("pool = ? AND ((status = ? AND end_time <= ?) OR (status = ? AND start_time <= ?))", SubscriptionPoolLightcode, "active", now, "pending", now).
		Limit(limit).Pluck("user_id", &users).Error; err != nil {
		return err
	}
	for _, user := range users {
		if _, err := LightcodeSubscriptions(user); err != nil {
			return err
		}
	}
	return nil
}
