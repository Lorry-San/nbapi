package model

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Lorry-San/nbapi/common"
)

const (
	OpsAlertMetricErrorRate             = "error_rate"
	OpsAlertMetricUpstreamErrorRate     = "upstream_error_rate"
	OpsAlertMetricSuccessRate           = "success_rate"
	OpsAlertMetricSLA                   = "sla"
	OpsAlertMetricAvgQPS                = "avg_qps"
	OpsAlertMetricAvgTPS                = "avg_tps"
	OpsAlertMetricRequestCount          = "request_count"
	OpsAlertMetricErrorCount            = "error_count"
	OpsAlertMetricTotalTokens           = "total_tokens"
	OpsAlertMetricAvgLatencyMs          = "avg_latency_ms"
	OpsAlertMetricP95LatencyMs          = "p95_latency_ms"
	OpsAlertMetricP99LatencyMs          = "p99_latency_ms"
	OpsAlertMetricAvgTTFTMs             = "avg_ttft_ms"
	OpsAlertMetricP95TTFTMs             = "p95_ttft_ms"
	OpsAlertMetricP99TTFTMs             = "p99_ttft_ms"
	OpsAlertMetricChannelSwitches       = "channel_switches"
	OpsAlertMetricCPUUsagePercent       = "cpu_usage_percent"
	OpsAlertMetricMemoryUsagePercent    = "memory_usage_percent"
	OpsAlertMetricDiskUsagePercent      = "disk_usage_percent"
	OpsAlertMetricConcurrencyQueueDepth = "concurrency_queue_depth"

	OpsAlertComparatorGT  = ">"
	OpsAlertComparatorGTE = ">="
	OpsAlertComparatorLT  = "<"
	OpsAlertComparatorLTE = "<="

	OpsAlertStatusFiring   = "firing"
	OpsAlertStatusResolved = "resolved"
)

type OpsAlertRule struct {
	Id                 int     `json:"id" gorm:"primaryKey"`
	Name               string  `json:"name" gorm:"type:varchar(128);not null"`
	Description        string  `json:"description" gorm:"type:text"`
	Metric             string  `json:"metric" gorm:"type:varchar(64);not null;index"`
	Comparator         string  `json:"comparator" gorm:"type:varchar(8);not null;default:'>'"`
	Threshold          float64 `json:"threshold" gorm:"not null"`
	Level              string  `json:"level" gorm:"type:varchar(8);not null;default:'P2';index"`
	Enabled            bool    `json:"enabled" gorm:"not null;default:true;index"`
	WindowSeconds      int64   `json:"window_seconds" gorm:"not null;default:300"`
	DurationSeconds    int64   `json:"duration_seconds" gorm:"not null;default:60"`
	CooldownSeconds    int64   `json:"cooldown_seconds" gorm:"not null;default:1800"`
	NotifyEmail        bool    `json:"notify_email" gorm:"not null;default:true"`
	Scope              string  `json:"scope" gorm:"type:varchar(16);not null;default:'overall'"`
	ChannelId          int     `json:"channel_id" gorm:"index"`
	ModelName          string  `json:"model_name" gorm:"type:varchar(128);index"`
	Group              string  `json:"group" gorm:"type:varchar(64);column:group_name;index"`
	LastState          string  `json:"last_state" gorm:"type:varchar(16);not null;default:'resolved';index"`
	FirstTriggeredAt   int64   `json:"first_triggered_at" gorm:"index"`
	LastTriggeredAt    int64   `json:"last_triggered_at" gorm:"index"`
	LastRecoveredAt    int64   `json:"last_recovered_at" gorm:"index"`
	LastNotifiedAt     int64   `json:"last_notified_at" gorm:"index"`
	LastValue          float64 `json:"last_value"`
	LastMessage        string  `json:"last_message" gorm:"type:text"`
	CreatedAt          int64   `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt          int64   `json:"updated_at" gorm:"autoUpdateTime;index"`
}

type OpsAlertEvent struct {
	Id               int     `json:"id" gorm:"primaryKey"`
	RuleId           int     `json:"rule_id" gorm:"index;not null"`
	RuleName         string  `json:"rule_name" gorm:"type:varchar(128);not null"`
	Title            string  `json:"title" gorm:"type:varchar(256);not null"`
	Message          string  `json:"message" gorm:"type:text"`
	Metric           string  `json:"metric" gorm:"type:varchar(64);not null;index"`
	Comparator       string  `json:"comparator" gorm:"type:varchar(8);not null"`
	Threshold        float64 `json:"threshold" gorm:"not null"`
	CurrentValue     float64 `json:"current_value"`
	Level            string  `json:"level" gorm:"type:varchar(8);not null;index"`
	Status           string  `json:"status" gorm:"type:varchar(16);not null;index"`
	Scope            string  `json:"scope" gorm:"type:varchar(16);not null;default:'overall'"`
	ChannelId        int     `json:"channel_id" gorm:"index"`
	ChannelName      string  `json:"channel_name" gorm:"type:varchar(128)"`
	ModelName        string  `json:"model_name" gorm:"type:varchar(128);index"`
	Group            string  `json:"group" gorm:"type:varchar(64);column:group_name;index"`
	WindowSeconds    int64   `json:"window_seconds"`
	DurationSeconds  int64   `json:"duration_seconds"`
	TriggeredAt      int64   `json:"triggered_at" gorm:"index"`
	ResolvedAt       int64   `json:"resolved_at" gorm:"index"`
	EmailSent        bool    `json:"email_sent" gorm:"not null;default:false;index"`
	EmailError       string  `json:"email_error" gorm:"type:text"`
	EmailRecipient   string  `json:"email_recipient" gorm:"type:text"`
	NotificationType string  `json:"notification_type" gorm:"type:varchar(16);not null;default:'email'"`
	CreatedAt        int64   `json:"created_at" gorm:"autoCreateTime;index"`
}

type OpsAlertRuleInput struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Metric          string  `json:"metric"`
	Comparator      string  `json:"comparator"`
	Threshold       float64 `json:"threshold"`
	Level           string  `json:"level"`
	Enabled         bool    `json:"enabled"`
	WindowSeconds   int64   `json:"window_seconds"`
	DurationSeconds int64   `json:"duration_seconds"`
	CooldownSeconds int64   `json:"cooldown_seconds"`
	NotifyEmail     bool    `json:"notify_email"`
	Scope           string  `json:"scope"`
	ChannelId       int     `json:"channel_id"`
	ModelName       string  `json:"model_name"`
	Group           string  `json:"group"`
}

type OpsAlertEventQuery struct {
	StartTs int64
	EndTs   int64
	Level   string
	Status  string
	RuleId  int
	Limit   int
}

type OpsAlertEvaluation struct {
	Value       float64
	Matched     bool
	Message     string
	ChannelName string
}

type OpsAlertNotificationRecipient struct {
	Email string
	Label string
}

type OpsAlertMetricsMetadata struct {
	Key         string `json:"key"`
	Label      string `json:"label"`
	Unit       string `json:"unit"`
	Comparator string `json:"default_comparator"`
}

var defaultOpsAlertRules = []OpsAlertRuleInput{
	{Name: "错误率极�?, Description: "当错误率超过 20% 且持�?1 分钟时触发告警（服务严重异常�?, Metric: OpsAlertMetricErrorRate, Comparator: OpsAlertComparatorGT, Threshold: 20, Level: "P0", Enabled: true, WindowSeconds: 60, DurationSeconds: 60, CooldownSeconds: 1800, NotifyEmail: true, Scope: "overall"},
	{Name: "错误率过�?, Description: "当错误率超过 5% 且持�?5 分钟时触发告�?, Metric: OpsAlertMetricErrorRate, Comparator: OpsAlertComparatorGT, Threshold: 5, Level: "P1", Enabled: true, WindowSeconds: 300, DurationSeconds: 300, CooldownSeconds: 1800, NotifyEmail: true, Scope: "overall"},
	{Name: "成功率过�?, Description: "当成功率低于 95% 且持�?5 分钟时触发告�?, Metric: OpsAlertMetricSuccessRate, Comparator: OpsAlertComparatorLT, Threshold: 95, Level: "P0", Enabled: true, WindowSeconds: 300, DurationSeconds: 300, CooldownSeconds: 1800, NotifyEmail: true, Scope: "overall"},
	{Name: "P95延迟过高", Description: "�?P95 延迟超过 2000ms 且持�?10 分钟时触发告�?, Metric: OpsAlertMetricP95LatencyMs, Comparator: OpsAlertComparatorGT, Threshold: 2000, Level: "P2", Enabled: true, WindowSeconds: 600, DurationSeconds: 600, CooldownSeconds: 1800, NotifyEmail: true, Scope: "overall"},
	{Name: "P99延迟过高", Description: "�?P99 延迟超过 3000ms 且持�?10 分钟时触发告�?, Metric: OpsAlertMetricP99LatencyMs, Comparator: OpsAlertComparatorGT, Threshold: 3000, Level: "P2", Enabled: true, WindowSeconds: 600, DurationSeconds: 600, CooldownSeconds: 1800, NotifyEmail: true, Scope: "overall"},
	{Name: "内存使用率过�?, Description: "当内存使用率超过 90% 且持�?10 分钟时触发告警（可能导致 OOM�?, Metric: OpsAlertMetricMemoryUsagePercent, Comparator: OpsAlertComparatorGT, Threshold: 90, Level: "P1", Enabled: true, WindowSeconds: 600, DurationSeconds: 600, CooldownSeconds: 1800, NotifyEmail: true, Scope: "overall"},
	{Name: "CPU使用率过�?, Description: "�?CPU 使用率超�?85% 且持�?10 分钟时触发告�?, Metric: OpsAlertMetricCPUUsagePercent, Comparator: OpsAlertComparatorGT, Threshold: 85, Level: "P2", Enabled: true, WindowSeconds: 600, DurationSeconds: 600, CooldownSeconds: 1800, NotifyEmail: true, Scope: "overall"},
}

var opsAlertSeedMu sync.Mutex

func GetOpsAlertMetricsMetadata() []OpsAlertMetricsMetadata {
	return []OpsAlertMetricsMetadata{
		{Key: OpsAlertMetricErrorRate, Label: "错误�?, Unit: "%", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricUpstreamErrorRate, Label: "上游错误�?, Unit: "%", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricSuccessRate, Label: "成功�?, Unit: "%", Comparator: OpsAlertComparatorLT},
		{Key: OpsAlertMetricSLA, Label: "SLA", Unit: "%", Comparator: OpsAlertComparatorLT},
		{Key: OpsAlertMetricAvgQPS, Label: "平均 QPS", Unit: "", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricAvgTPS, Label: "平均 TPS", Unit: "", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricRequestCount, Label: "请求数量", Unit: "", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricErrorCount, Label: "错误数量", Unit: "", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricTotalTokens, Label: "Token 数量", Unit: "", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricAvgLatencyMs, Label: "平均请求时长", Unit: "ms", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricP95LatencyMs, Label: "P95 请求时长", Unit: "ms", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricP99LatencyMs, Label: "P99 请求时长", Unit: "ms", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricAvgTTFTMs, Label: "平均�?Token", Unit: "ms", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricP95TTFTMs, Label: "P95 �?Token", Unit: "ms", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricP99TTFTMs, Label: "P99 �?Token", Unit: "ms", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricChannelSwitches, Label: "渠道切换次数", Unit: "", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricCPUUsagePercent, Label: "CPU 使用�?, Unit: "%", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricMemoryUsagePercent, Label: "内存使用�?, Unit: "%", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricDiskUsagePercent, Label: "磁盘使用�?, Unit: "%", Comparator: OpsAlertComparatorGT},
		{Key: OpsAlertMetricConcurrencyQueueDepth, Label: "并发/排队深度", Unit: "", Comparator: OpsAlertComparatorGT},
	}
}

func SeedDefaultOpsAlertRules() error {
	opsAlertSeedMu.Lock()
	defer opsAlertSeedMu.Unlock()

	var count int64
	if err := DB.Model(&OpsAlertRule{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for _, input := range defaultOpsAlertRules {
		rule, err := buildOpsAlertRuleFromInput(input, nil)
		if err != nil {
			return err
		}
		if err := DB.Create(rule).Error; err != nil {
			return err
		}
	}
	return nil
}

func ListOpsAlertRules() ([]OpsAlertRule, error) {
	var rules []OpsAlertRule
	err := DB.Order("enabled desc, level asc, id asc").Find(&rules).Error
	return rules, err
}

func CreateOpsAlertRule(input OpsAlertRuleInput) (*OpsAlertRule, error) {
	rule, err := buildOpsAlertRuleFromInput(input, nil)
	if err != nil {
		return nil, err
	}
	err = DB.Select(
		"Name",
		"Description",
		"Metric",
		"Comparator",
		"Threshold",
		"Level",
		"Enabled",
		"WindowSeconds",
		"DurationSeconds",
		"CooldownSeconds",
		"NotifyEmail",
		"Scope",
		"ChannelId",
		"ModelName",
		"Group",
		"LastState",
	).Create(rule).Error
	return rule, err
}

func UpdateOpsAlertRule(id int, input OpsAlertRuleInput) (*OpsAlertRule, error) {
	if id <= 0 {
		return nil, errors.New("invalid rule id")
	}
	var existing OpsAlertRule
	if err := DB.First(&existing, id).Error; err != nil {
		return nil, err
	}
	rule, err := buildOpsAlertRuleFromInput(input, &existing)
	if err != nil {
		return nil, err
	}
	rule.Id = id
	rule.LastState = existing.LastState
	rule.FirstTriggeredAt = existing.FirstTriggeredAt
	rule.LastTriggeredAt = existing.LastTriggeredAt
	rule.LastRecoveredAt = existing.LastRecoveredAt
	rule.LastNotifiedAt = existing.LastNotifiedAt
	rule.LastValue = existing.LastValue
	rule.LastMessage = existing.LastMessage
	err = DB.Model(&existing).Updates(map[string]interface{}{
		"name":              rule.Name,
		"description":       rule.Description,
		"metric":            rule.Metric,
		"comparator":        rule.Comparator,
		"threshold":         rule.Threshold,
		"level":             rule.Level,
		"enabled":           rule.Enabled,
		"window_seconds":    rule.WindowSeconds,
		"duration_seconds":  rule.DurationSeconds,
		"cooldown_seconds":  rule.CooldownSeconds,
		"notify_email":      rule.NotifyEmail,
		"scope":             rule.Scope,
		"channel_id":        rule.ChannelId,
		"model_name":        rule.ModelName,
		"group_name":        rule.Group,
		"updated_at":        time.Now().Unix(),
	}).Error
	return rule, err
}

func DeleteOpsAlertRule(id int) error {
	if id <= 0 {
		return errors.New("invalid rule id")
	}
	return DB.Delete(&OpsAlertRule{}, id).Error
}

func ListOpsAlertEvents(query OpsAlertEventQuery) ([]OpsAlertEvent, error) {
	db := DB.Model(&OpsAlertEvent{})
	if query.StartTs > 0 {
		db = db.Where("created_at >= ?", query.StartTs)
	}
	if query.EndTs > 0 {
		db = db.Where("created_at <= ?", query.EndTs)
	}
	if query.Level != "" && query.Level != "all" {
		db = db.Where("level = ?", strings.ToUpper(query.Level))
	}
	if query.Status != "" && query.Status != "all" {
		db = db.Where("status = ?", query.Status)
	}
	if query.RuleId > 0 {
		db = db.Where("rule_id = ?", query.RuleId)
	}
	limit := query.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var events []OpsAlertEvent
	err := db.Order("created_at desc, id desc").Limit(limit).Find(&events).Error
	return events, err
}

func GetEnabledOpsAlertRules() ([]OpsAlertRule, error) {
	var rules []OpsAlertRule
	err := DB.Where("enabled = ?", true).Order("level asc, id asc").Find(&rules).Error
	return rules, err
}

func EvaluateOpsAlertRule(rule OpsAlertRule, now int64) (OpsAlertEvaluation, error) {
	if now <= 0 {
		now = time.Now().Unix()
	}
	window := normalizedOpsAlertWindow(rule.WindowSeconds)
	start := now - window
	metric := strings.TrimSpace(rule.Metric)

	if isSystemOpsAlertMetric(metric) {
		status := common.GetSystemStatus()
		value := 0.0
		switch metric {
		case OpsAlertMetricCPUUsagePercent:
			value = status.CPUUsage
		case OpsAlertMetricMemoryUsagePercent:
			value = status.MemoryUsage
		case OpsAlertMetricDiskUsagePercent:
			value = status.DiskUsage
		default:
			return OpsAlertEvaluation{}, fmt.Errorf("unsupported system alert metric: %s", metric)
		}
		return OpsAlertEvaluation{
			Value:   roundOpsAlertValue(value),
			Matched: compareOpsAlertValue(value, rule.Comparator, rule.Threshold),
			Message: buildOpsAlertMessage(rule, value, window, "overall"),
		}, nil
	}

	result, err := GetOpsMonitor(start, now, rule.ChannelId, rule.ModelName, rule.Group)
	if err != nil {
		return OpsAlertEvaluation{}, err
	}
	value, err := extractOpsAlertMetricValue(metric, result)
	if err != nil {
		return OpsAlertEvaluation{}, err
	}
	scopeLabel := "overall"
	channelName := ""
	if rule.ChannelId > 0 {
		scopeLabel = fmt.Sprintf("channel:%d", rule.ChannelId)
		for _, channel := range result.Channels {
			if channel.ChannelId == rule.ChannelId {
				channelName = channel.ChannelName
				if channelName != "" {
					scopeLabel = channelName
				}
				break
			}
		}
	}
	if rule.ModelName != "" {
		scopeLabel = scopeLabel + " model:" + rule.ModelName
	}
	if rule.Group != "" {
		scopeLabel = scopeLabel + " group:" + rule.Group
	}

	return OpsAlertEvaluation{
		Value:       roundOpsAlertValue(value),
		Matched:     compareOpsAlertValue(value, rule.Comparator, rule.Threshold),
		Message:     buildOpsAlertMessage(rule, value, window, scopeLabel),
		ChannelName: channelName,
	}, nil
}

func HandleOpsAlertEvaluation(rule OpsAlertRule, evaluation OpsAlertEvaluation, now int64, notify func(OpsAlertEvent) (bool, string, string)) error {
	if now <= 0 {
		now = time.Now().Unix()
	}
	duration := normalizedOpsAlertDuration(rule.DurationSeconds)
	updates := map[string]interface{}{
		"last_value":   evaluation.Value,
		"last_message": evaluation.Message,
	}

	if evaluation.Matched {
		firstTriggeredAt := rule.FirstTriggeredAt
		if firstTriggeredAt <= 0 || rule.LastState == OpsAlertStatusResolved {
			firstTriggeredAt = now
		}
		updates["first_triggered_at"] = firstTriggeredAt
		updates["last_triggered_at"] = now
		if now-firstTriggeredAt >= duration {
			if rule.LastState != OpsAlertStatusFiring || shouldSendOpsAlertAgain(rule, now) {
				event := buildOpsAlertEvent(rule, evaluation, OpsAlertStatusFiring, now)
				emailSent, emailError, recipients := false, "", ""
				if rule.NotifyEmail && notify != nil {
					emailSent, emailError, recipients = notify(event)
				}
				event.EmailSent = emailSent
				event.EmailError = emailError
				event.EmailRecipient = recipients
				if err := DB.Create(&event).Error; err != nil {
					return err
				}
				updates["last_notified_at"] = now
			}
			updates["last_state"] = OpsAlertStatusFiring
		}
		return DB.Model(&OpsAlertRule{}).Where("id = ?", rule.Id).Updates(updates).Error
	}

	updates["first_triggered_at"] = int64(0)
	if rule.LastState == OpsAlertStatusFiring {
		event := buildOpsAlertEvent(rule, evaluation, OpsAlertStatusResolved, now)
		emailSent, emailError, recipients := false, "", ""
		if rule.NotifyEmail && notify != nil {
			emailSent, emailError, recipients = notify(event)
		}
		event.EmailSent = emailSent
		event.EmailError = emailError
		event.EmailRecipient = recipients
		event.ResolvedAt = now
		if err := DB.Create(&event).Error; err != nil {
			return err
		}
		updates["last_recovered_at"] = now
		updates["last_notified_at"] = now
	}
	updates["last_state"] = OpsAlertStatusResolved
	return DB.Model(&OpsAlertRule{}).Where("id = ?", rule.Id).Updates(updates).Error
}

func GetOpsAlertEmailRecipients() ([]OpsAlertNotificationRecipient, error) {
	var users []User
	if err := DB.Select("id", "username", "display_name", "email", "role", "status").
		Where("status = ? AND role >= ? AND email <> ?", common.UserStatusEnabled, common.RoleAdminUser, "").
		Find(&users).Error; err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	recipients := make([]OpsAlertNotificationRecipient, 0, len(users))
	for _, user := range users {
		email := strings.TrimSpace(user.Email)
		if email == "" {
			continue
		}
		key := strings.ToLower(email)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		label := user.DisplayName
		if label == "" {
			label = user.Username
		}
		recipients = append(recipients, OpsAlertNotificationRecipient{Email: email, Label: label})
	}
	sort.Slice(recipients, func(i, j int) bool {
		return recipients[i].Email < recipients[j].Email
	})
	return recipients, nil
}

func buildOpsAlertRuleFromInput(input OpsAlertRuleInput, existing *OpsAlertRule) (*OpsAlertRule, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("rule name is required")
	}
	metric := strings.TrimSpace(input.Metric)
	if !isSupportedOpsAlertMetric(metric) {
		return nil, fmt.Errorf("unsupported metric: %s", metric)
	}
	comparator := strings.TrimSpace(input.Comparator)
	if comparator == "" {
		comparator = defaultComparatorForOpsAlertMetric(metric)
	}
	if !isSupportedOpsAlertComparator(comparator) {
		return nil, fmt.Errorf("unsupported comparator: %s", comparator)
	}
	level := strings.ToUpper(strings.TrimSpace(input.Level))
	if level == "" {
		level = "P2"
	}
	if !isSupportedOpsAlertLevel(level) {
		return nil, fmt.Errorf("unsupported level: %s", level)
	}
	scope := strings.TrimSpace(input.Scope)
	if scope == "" {
		scope = "overall"
	}
	if scope != "overall" && scope != "channel" && scope != "model" && scope != "group" {
		return nil, fmt.Errorf("unsupported scope: %s", scope)
	}
	rule := &OpsAlertRule{
		Name:            name,
		Description:     strings.TrimSpace(input.Description),
		Metric:          metric,
		Comparator:      comparator,
		Threshold:       input.Threshold,
		Level:           level,
		Enabled:         input.Enabled,
		WindowSeconds:   normalizedOpsAlertWindow(input.WindowSeconds),
		DurationSeconds: normalizedOpsAlertDuration(input.DurationSeconds),
		CooldownSeconds: normalizedOpsAlertCooldown(input.CooldownSeconds),
		NotifyEmail:     input.NotifyEmail,
		Scope:           scope,
		ChannelId:       input.ChannelId,
		ModelName:       strings.TrimSpace(input.ModelName),
		Group:           strings.TrimSpace(input.Group),
		LastState:       OpsAlertStatusResolved,
	}
	if existing != nil {
		rule.CreatedAt = existing.CreatedAt
		rule.UpdatedAt = time.Now().Unix()
	}
	return rule, nil
}

func extractOpsAlertMetricValue(metric string, result OpsMonitorResult) (float64, error) {
	overview := result.Overview
	switch metric {
	case OpsAlertMetricErrorRate:
		return overview.ErrorRate, nil
	case OpsAlertMetricUpstreamErrorRate:
		return overview.UpstreamErrorRate, nil
	case OpsAlertMetricSuccessRate:
		return percent(overview.SuccessCount, overview.RequestCount), nil
	case OpsAlertMetricSLA:
		return overview.SLA, nil
	case OpsAlertMetricAvgQPS:
		return overview.AvgQPS, nil
	case OpsAlertMetricAvgTPS:
		return overview.AvgTPS, nil
	case OpsAlertMetricRequestCount:
		return float64(overview.RequestCount), nil
	case OpsAlertMetricErrorCount:
		return float64(overview.ErrorCount), nil
	case OpsAlertMetricTotalTokens:
		return float64(overview.TotalTokens), nil
	case OpsAlertMetricAvgLatencyMs:
		return float64(overview.AvgLatencyMs), nil
	case OpsAlertMetricP95LatencyMs:
		return float64(overview.P95LatencyMs), nil
	case OpsAlertMetricP99LatencyMs:
		return float64(overview.P99LatencyMs), nil
	case OpsAlertMetricAvgTTFTMs:
		return float64(overview.AvgTTFTMs), nil
	case OpsAlertMetricP95TTFTMs:
		return float64(overview.P95TTFTMs), nil
	case OpsAlertMetricP99TTFTMs:
		return float64(overview.P99TTFTMs), nil
	case OpsAlertMetricChannelSwitches:
		var switches int64
		for _, point := range result.ChannelSwitch {
			switches += point.Switches
		}
		return float64(switches), nil
	case OpsAlertMetricConcurrencyQueueDepth:
		var maxDepth int64
		for _, channel := range result.Channels {
			if channel.QueueDepth > maxDepth {
				maxDepth = channel.QueueDepth
			}
		}
		return float64(maxDepth), nil
	default:
		return 0, fmt.Errorf("unsupported metric: %s", metric)
	}
}

func buildOpsAlertEvent(rule OpsAlertRule, evaluation OpsAlertEvaluation, status string, now int64) OpsAlertEvent {
	titlePrefix := rule.Level
	if status == OpsAlertStatusResolved {
		titlePrefix = "恢复 " + rule.Level
	}
	title := fmt.Sprintf("%s: %s", titlePrefix, rule.Name)
	triggeredAt := rule.FirstTriggeredAt
	if triggeredAt <= 0 {
		triggeredAt = now
	}
	return OpsAlertEvent{
		RuleId:           rule.Id,
		RuleName:         rule.Name,
		Title:            title,
		Message:          evaluation.Message,
		Metric:           rule.Metric,
		Comparator:       rule.Comparator,
		Threshold:        rule.Threshold,
		CurrentValue:     evaluation.Value,
		Level:            rule.Level,
		Status:           status,
		Scope:            rule.Scope,
		ChannelId:        rule.ChannelId,
		ChannelName:      evaluation.ChannelName,
		ModelName:        rule.ModelName,
		Group:            rule.Group,
		WindowSeconds:    rule.WindowSeconds,
		DurationSeconds:  rule.DurationSeconds,
		TriggeredAt:      triggeredAt,
		NotificationType: "email",
	}
}

func buildOpsAlertMessage(rule OpsAlertRule, value float64, windowSeconds int64, scope string) string {
	return fmt.Sprintf("%s %s %.2f (current %.2f) over last %s (%s)",
		rule.Metric,
		rule.Comparator,
		rule.Threshold,
		roundOpsAlertValue(value),
		formatOpsAlertDuration(windowSeconds),
		scope,
	)
}

func shouldSendOpsAlertAgain(rule OpsAlertRule, now int64) bool {
	cooldown := normalizedOpsAlertCooldown(rule.CooldownSeconds)
	return rule.LastNotifiedAt <= 0 || now-rule.LastNotifiedAt >= cooldown
}

func compareOpsAlertValue(value float64, comparator string, threshold float64) bool {
	switch comparator {
	case OpsAlertComparatorGT:
		return value > threshold
	case OpsAlertComparatorGTE:
		return value >= threshold
	case OpsAlertComparatorLT:
		return value < threshold
	case OpsAlertComparatorLTE:
		return value <= threshold
	default:
		return false
	}
}

func isSystemOpsAlertMetric(metric string) bool {
	return metric == OpsAlertMetricCPUUsagePercent || metric == OpsAlertMetricMemoryUsagePercent || metric == OpsAlertMetricDiskUsagePercent
}

func isSupportedOpsAlertMetric(metric string) bool {
	for _, item := range GetOpsAlertMetricsMetadata() {
		if item.Key == metric {
			return true
		}
	}
	return false
}

func isSupportedOpsAlertComparator(comparator string) bool {
	return comparator == OpsAlertComparatorGT || comparator == OpsAlertComparatorGTE || comparator == OpsAlertComparatorLT || comparator == OpsAlertComparatorLTE
}

func isSupportedOpsAlertLevel(level string) bool {
	return level == "P0" || level == "P1" || level == "P2" || level == "P3"
}

func defaultComparatorForOpsAlertMetric(metric string) string {
	for _, item := range GetOpsAlertMetricsMetadata() {
		if item.Key == metric && item.Comparator != "" {
			return item.Comparator
		}
	}
	return OpsAlertComparatorGT
}

func normalizedOpsAlertWindow(value int64) int64 {
	if value <= 0 {
		return 300
	}
	if value < 60 {
		return 60
	}
	if value > 24*3600 {
		return 24 * 3600
	}
	return value
}

func normalizedOpsAlertDuration(value int64) int64 {
	if value <= 0 {
		return 60
	}
	if value < 60 {
		return 60
	}
	if value > 24*3600 {
		return 24 * 3600
	}
	return value
}

func normalizedOpsAlertCooldown(value int64) int64 {
	if value <= 0 {
		return 1800
	}
	if value < 60 {
		return 60
	}
	if value > 7*24*3600 {
		return 7 * 24 * 3600
	}
	return value
}

func roundOpsAlertValue(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Round(value*100) / 100
}

func formatOpsAlertDuration(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}
	if seconds%3600 == 0 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	if seconds%60 == 0 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%ds", seconds)
}
