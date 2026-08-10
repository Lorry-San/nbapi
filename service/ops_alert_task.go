package service

import (
	"fmt"
	"html"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/logger"
	"github.com/Lorry-San/nbapi/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const opsAlertEvaluationInterval = time.Minute

var (
	opsAlertTaskOnce    sync.Once
	opsAlertTaskRunning atomic.Bool
)

func StartOpsAlertEvaluationTask() {
	opsAlertTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			common.SysLog(fmt.Sprintf("ops alert evaluation task started: tick=%s", opsAlertEvaluationInterval))
			_ = RunOpsAlertEvaluationOnce()
			ticker := time.NewTicker(opsAlertEvaluationInterval)
			defer ticker.Stop()
			for range ticker.C {
				if err := RunOpsAlertEvaluationOnce(); err != nil {
					common.SysLog(fmt.Sprintf("ops alert evaluation failed: %v", err))
				}
			}
		})
	})
}

func RunOpsAlertEvaluationOnce() error {
	if !opsAlertTaskRunning.CompareAndSwap(false, true) {
		return nil
	}
	defer opsAlertTaskRunning.Store(false)

	if err := model.SeedDefaultOpsAlertRules(); err != nil {
		return err
	}
	rules, err := model.GetEnabledOpsAlertRules()
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, rule := range rules {
		evaluation, err := model.EvaluateOpsAlertRule(rule, now)
		if err != nil {
			logger.LogWarn(nil, fmt.Sprintf("ops alert rule %d evaluate failed: %v", rule.Id, err))
			continue
		}
		if err := model.HandleOpsAlertEvaluation(rule, evaluation, now, sendOpsAlertEmail); err != nil {
			logger.LogWarn(nil, fmt.Sprintf("ops alert rule %d update failed: %v", rule.Id, err))
		}
	}
	return nil
}

func sendOpsAlertEmail(event model.OpsAlertEvent) (bool, string, string) {
	recipients, err := model.GetOpsAlertEmailRecipients()
	if err != nil {
		return false, err.Error(), ""
	}
	if len(recipients) == 0 {
		return false, "no admin email recipients", ""
	}

	emails := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		emails = append(emails, recipient.Email)
	}
	receiver := strings.Join(emails, ";")
	subject := fmt.Sprintf("[NBAPI %s] %s", event.Level, event.Title)
	if event.Status == model.OpsAlertStatusResolved {
		subject = fmt.Sprintf("[NBAPI 恢复] %s", event.Title)
	}
	content := buildOpsAlertEmailContent(event)
	if err := common.SendEmail(subject, receiver, content); err != nil {
		return false, err.Error(), receiver
	}
	return true, "", receiver
}

func buildOpsAlertEmailContent(event model.OpsAlertEvent) string {
	statusText := "触发"
	if event.Status == model.OpsAlertStatusResolved {
		statusText = "恢复"
	}
	occurredAt := event.CreatedAt
	if occurredAt <= 0 {
		occurredAt = time.Now().Unix()
	}
	rows := []struct {
		Key   string
		Value string
	}{
		{"状态", statusText},
		{"级别", event.Level},
		{"规则", event.RuleName},
		{"指标", event.Metric},
		{"条件", fmt.Sprintf("%s %.2f", event.Comparator, event.Threshold)},
		{"当前值", fmt.Sprintf("%.2f", event.CurrentValue)},
		{"范围", formatOpsAlertEmailScope(event)},
		{"持续时间", formatOpsAlertEmailDuration(event.DurationSeconds)},
		{"发生时间", time.Unix(occurredAt, 0).Format("2006-01-02 15:04:05")},
	}
	var builder strings.Builder
	builder.WriteString("<div style=\"font-family:Arial,'Microsoft YaHei',sans-serif;color:#1f2937;line-height:1.7\">")
	builder.WriteString("<h2 style=\"margin:0 0 12px\">NBAPI 运维告警")
	builder.WriteString(html.EscapeString(" - " + statusText))
	builder.WriteString("</h2>")
	builder.WriteString("<p style=\"margin:0 0 16px;color:#4b5563\">")
	builder.WriteString(html.EscapeString(event.Message))
	builder.WriteString("</p>")
	builder.WriteString("<table style=\"border-collapse:collapse;width:100%;max-width:720px\">")
	for _, row := range rows {
		builder.WriteString("<tr>")
		builder.WriteString("<td style=\"border:1px solid #e5e7eb;background:#f9fafb;padding:8px 10px;width:120px;font-weight:600\">")
		builder.WriteString(html.EscapeString(row.Key))
		builder.WriteString("</td><td style=\"border:1px solid #e5e7eb;padding:8px 10px\">")
		builder.WriteString(html.EscapeString(row.Value))
		builder.WriteString("</td></tr>")
	}
	builder.WriteString("</table>")
	builder.WriteString("<p style=\"margin-top:16px;color:#6b7280;font-size:12px\">此邮件发送给已启用告警通知的管理员和超级管理员账号。</p>")
	builder.WriteString("</div>")
	return builder.String()
}

func formatOpsAlertEmailScope(event model.OpsAlertEvent) string {
	parts := []string{event.Scope}
	if event.ChannelName != "" {
		parts = append(parts, event.ChannelName)
	} else if event.ChannelId > 0 {
		parts = append(parts, fmt.Sprintf("channel:%d", event.ChannelId))
	}
	if event.ModelName != "" {
		parts = append(parts, "model:"+event.ModelName)
	}
	if event.Group != "" {
		parts = append(parts, "group:"+event.Group)
	}
	return strings.Join(parts, " ")
}

func formatOpsAlertEmailDuration(seconds int64) string {
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
