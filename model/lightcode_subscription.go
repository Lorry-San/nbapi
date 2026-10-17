package model

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"gorm.io/gorm"
)

const (
	SubscriptionPoolAPI = "api"
	SubscriptionPoolLightcode = "lightcode"
)

var ErrLightcodePendingSubscription = errors.New("a Lightcode renewal is already pending")

func ReplaceLightcodeGroupModels(group string, models []LightcodeGroupModel) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("routing_group = ?", group).Delete(&LightcodeGroupModel{}).Error; err != nil { return err }
		if len(models) == 0 { return nil }
		return tx.Create(&models).Error
	})
}

// LightcodeGroupModel adds presentation and reasoning policy to a channel routing group.
type LightcodeGroupModel struct {
	Id int `json:"id"`
	RoutingGroup string `json:"routing_group" gorm:"type:varchar(64);uniqueIndex:idx_lc_group_model"`
	Model string `json:"model" gorm:"type:varchar(191);uniqueIndex:idx_lc_group_model"`
	DisplayName string `json:"display_name" gorm:"type:varchar(191)"`
	Efforts []string `json:"efforts" gorm:"serializer:json;type:text"`
	DefaultEffort string `json:"default_effort" gorm:"type:varchar(32)"`
	ContextWindow int `json:"context_window"`
	MaxOutputTokens int `json:"max_output_tokens"`
	Vision bool `json:"vision"`
}

func ValidateLightcodePlan(plan *SubscriptionPlan) error {
	plan.NormalizeDefaults()
	if plan.Pool != SubscriptionPoolAPI && plan.Pool != SubscriptionPoolLightcode { return errors.New("invalid subscription pool") }
	if plan.Pool == SubscriptionPoolAPI { return nil }
	if plan.UpgradeGroup != "" || plan.DowngradeGroup != "" { return errors.New("Lightcode subscriptions cannot change the API user group") }
	plan.AllowWalletOverflow = common.GetPointer(false)
	if strings.TrimSpace(plan.RoutingGroup) == "" || plan.RoutingGroup == "auto" { return errors.New("Lightcode requires one explicit routing group") }
	if len(plan.AllowedModels) == 0 { return errors.New("select at least one Lightcode model") }
	var models []LightcodeGroupModel
	if err := DB.Where("routing_group = ?", plan.RoutingGroup).Find(&models).Error; err != nil { return err }
	for _, name := range plan.AllowedModels {
		if !slices.ContainsFunc(models, func(m LightcodeGroupModel) bool { return m.Model == name }) { return fmt.Errorf("model %q is not configured in Lightcode group %q", name, plan.RoutingGroup) }
	}
	return nil
}

// The write obtains SQLite's writer reservation; row locking covers MySQL and PostgreSQL.
func lockLightcodeUser(tx *gorm.DB, userId int) error {
	if err := tx.Model(&User{}).Where("id = ?", userId).UpdateColumn("id", gorm.Expr("id")).Error; err != nil { return err }
	var user User
	return lockForUpdate(tx).Select("id").First(&user, userId).Error
}

func advanceLightcodeSubscriptionsTx(tx *gorm.DB, userId int, now int64) error {
	if err := tx.Model(&UserSubscription{}).Where("user_id = ? AND pool = ? AND status IN ? AND end_time <= ?", userId, SubscriptionPoolLightcode, []string{"active", "pending"}, now).Updates(map[string]interface{}{"status": "expired", "updated_at": now}).Error; err != nil { return err }
	var active int64
	if err := tx.Model(&UserSubscription{}).Where("user_id = ? AND pool = ? AND status = ?", userId, SubscriptionPoolLightcode, "active").Count(&active).Error; err != nil { return err }
	if active > 0 { return nil }
	return tx.Model(&UserSubscription{}).Where("user_id = ? AND pool = ? AND status = ? AND start_time <= ? AND end_time > ?", userId, SubscriptionPoolLightcode, "pending", now, now).Updates(map[string]interface{}{"status": "active", "updated_at": now}).Error
}

func createLightcodeSubscriptionTx(tx *gorm.DB, userId int, plan *SubscriptionPlan, source string) (*UserSubscription, error) {
	if err := lockLightcodeUser(tx, userId); err != nil { return nil, err }
	now := getDBTimestampTx(tx)
	if err := advanceLightcodeSubscriptionsTx(tx, userId, now); err != nil { return nil, err }
	var current []UserSubscription
	if err := tx.Where("user_id = ? AND pool = ? AND status IN ?", userId, SubscriptionPoolLightcode, []string{"active", "pending"}).Find(&current).Error; err != nil { return nil, err }
	start, status := now, "active"
	for _, sub := range current {
		if sub.Status == "pending" { return nil, ErrLightcodePendingSubscription }
		start, status = sub.EndTime, "pending"
	}
	end, err := calcPlanEndTime(time.Unix(start, 0), plan)
	if err != nil { return nil, err }
	if plan.RoutingGroup == "" || len(plan.AllowedModels) == 0 || plan.UpgradeGroup != "" || plan.DowngradeGroup != "" { return nil, errors.New("invalid Lightcode subscription entitlements") }
	snapshot, err := common.Marshal(plan)
	if err != nil { return nil, err }
	sub := &UserSubscription{
		UserId: userId, PlanId: plan.Id, Pool: SubscriptionPoolLightcode,
		RoutingGroup: plan.RoutingGroup, AllowedModels: slices.Clone(plan.AllowedModels),
		PlanSnapshot: string(snapshot), AmountTotal: plan.TotalAmount, StartTime: start, EndTime: end,
		Status: status, Source: source, LastResetTime: start,
		NextResetTime: calcNextResetTime(time.Unix(start, 0), plan, end), AllowWalletOverflow: false,
	}
	return sub, tx.Create(sub).Error
}

// LightcodeSubscriptions advances due renewals before exposing current entitlements.
func LightcodeSubscriptions(userId int) ([]UserSubscription, error) {
	var subs []UserSubscription
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockLightcodeUser(tx, userId); err != nil { return err }
		now := getDBTimestampTx(tx)
		if err := advanceLightcodeSubscriptionsTx(tx, userId, now); err != nil { return err }
		if err := tx.Where("user_id = ? AND pool = ? AND status IN ?", userId, SubscriptionPoolLightcode, []string{"active", "pending"}).Order("start_time asc").Find(&subs).Error; err != nil { return err }
		for i := range subs {
			if subs[i].Status != "active" { continue }
			var plan SubscriptionPlan
			if err := common.UnmarshalJsonStr(subs[i].PlanSnapshot, &plan); err != nil { return err }
			if err := maybeResetUserSubscriptionWithPlanTx(tx, &subs[i], &plan, now); err != nil { return err }
		}
		return nil
	})
	return subs, err
}

// ActivateDueLightcodeRenewals also handles pending renewals after a long outage.
func ActivateDueLightcodeRenewals(limit int) error {
	var users []int
	now := GetDBTimestamp()
	if err := DB.Model(&UserSubscription{}).Distinct("user_id").
		Where("pool = ? AND ((status = ? AND end_time <= ?) OR (status = ? AND start_time <= ?))", SubscriptionPoolLightcode, "active", now, "pending", now).
		Limit(limit).Pluck("user_id", &users).Error; err != nil { return err }
	for _, user := range users { if _, err := LightcodeSubscriptions(user); err != nil { return err } }
	return nil
}
