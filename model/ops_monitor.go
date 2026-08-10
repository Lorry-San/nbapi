package model

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Lorry-San/nbapi/common"
)

const maxOpsMonitorLogs = 200000

type OpsMonitorOverview struct {
	RequestCount      int64   `json:"request_count"`
	SuccessCount      int64   `json:"success_count"`
	ErrorCount        int64   `json:"error_count"`
	ExcludedErrors    int64   `json:"excluded_errors"`
	PromptTokens      int64   `json:"prompt_tokens"`
	CompletionTokens  int64   `json:"completion_tokens"`
	TotalTokens       int64   `json:"total_tokens"`
	AvgQPS            float64 `json:"avg_qps"`
	AvgTPS            float64 `json:"avg_tps"`
	SLA               float64 `json:"sla"`
	ErrorRate         float64 `json:"error_rate"`
	UpstreamErrorRate float64 `json:"upstream_error_rate"`
	AvgLatencyMs      int64   `json:"avg_latency_ms"`
	AvgTTFTMs         int64   `json:"avg_ttft_ms"`
	P95LatencyMs      int64   `json:"p95_latency_ms"`
	P99LatencyMs      int64   `json:"p99_latency_ms"`
	P95TTFTMs         int64   `json:"p95_ttft_ms"`
	P99TTFTMs         int64   `json:"p99_ttft_ms"`
}

type OpsMonitorPoint struct {
	Ts        int64   `json:"ts"`
	Requests int64   `json:"requests"`
	Errors   int64   `json:"errors"`
	Tokens   int64   `json:"tokens"`
	QPS      float64 `json:"qps"`
	TPS      float64 `json:"tps"`
	SLA      float64 `json:"sla"`
	AvgTTFT  int64   `json:"avg_ttft_ms"`
}

type OpsMonitorChannel struct {
	ChannelId        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	Requests         int64   `json:"requests"`
	Errors           int64   `json:"errors"`
	Tokens           int64   `json:"tokens"`
	AvgLatencyMs     int64   `json:"avg_latency_ms"`
	AvgTTFTMs        int64   `json:"avg_ttft_ms"`
	SLA              float64 `json:"sla"`
	QueueDepth       int64   `json:"queue_depth"`
	ChannelSwitches  int64   `json:"channel_switches"`
	LatestError      string  `json:"latest_error,omitempty"`
	LatestErrorAt    int64   `json:"latest_error_at,omitempty"`
	LatestStatusCode int     `json:"latest_status_code,omitempty"`
}

type OpsMonitorError struct {
	Id           int    `json:"id"`
	CreatedAt    int64  `json:"created_at"`
	ChannelId    int    `json:"channel_id"`
	ChannelName  string `json:"channel_name"`
	ModelName    string `json:"model_name"`
	StatusCode   int    `json:"status_code"`
	ErrorCode    string `json:"error_code"`
	ErrorType    string `json:"error_type"`
	RequestId    string `json:"request_id"`
	UpstreamId   string `json:"upstream_request_id"`
	Content      string `json:"content"`
	RequestPath  string `json:"request_path"`
	BusinessLike bool   `json:"business_like"`
}

type OpsMonitorChannelSwitchPoint struct {
	Ts       int64 `json:"ts"`
	Switches int64 `json:"switches"`
}

type OpsMonitorResult struct {
	StartTimestamp int64                          `json:"start_timestamp"`
	EndTimestamp   int64                          `json:"end_timestamp"`
	BucketSeconds  int64                          `json:"bucket_seconds"`
	Overview       OpsMonitorOverview             `json:"overview"`
	Trend          []OpsMonitorPoint              `json:"trend"`
	Channels       []OpsMonitorChannel            `json:"channels"`
	ChannelSwitch  []OpsMonitorChannelSwitchPoint `json:"channel_switch_trend"`
	Errors         []OpsMonitorError              `json:"errors"`
	UpdatedAt      int64                          `json:"updated_at"`
	Truncated      bool                           `json:"truncated"`
}

type opsChannelAgg struct {
	requests        int64
	errors          int64
	excludedErrs    int64
	tokens          int64
	latencyMs       int64
	latencyCount    int64
	ttftMs          int64
	ttftCount       int64
	channelSwitches int64
	latestError     OpsMonitorError
}

type opsBucketAgg struct {
	requests     int64
	errors       int64
	excludedErrs int64
	tokens       int64
	ttftMs       int64
	ttftCount    int64
	switches     int64
}

func GetOpsMonitor(startTs int64, endTs int64, channelId int, modelName string, group string) (OpsMonitorResult, error) {
	now := time.Now().Unix()
	if endTs <= 0 {
		endTs = now
	}
	if startTs <= 0 || startTs >= endTs {
		startTs = endTs - 3600
	}
	if endTs-startTs > 30*24*3600 {
		startTs = endTs - 30*24*3600
	}
	bucketSeconds := chooseOpsBucketSeconds(endTs - startTs)
	bucketCount := int((endTs-startTs)/bucketSeconds) + 1
	if bucketCount < 1 {
		bucketCount = 1
	}

	query := LOG_DB.Model(&Log{}).
		Where("created_at >= ? AND created_at <= ? AND type IN ?", startTs, endTs, []int{LogTypeConsume, LogTypeError})
	if channelId > 0 {
		query = query.Where("channel_id = ?", channelId)
	}
	if modelName != "" {
		pattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return OpsMonitorResult{}, err
		}
		query = query.Where("model_name LIKE ? ESCAPE '!'", pattern)
	}
	if group != "" {
		query = query.Where(logGroupCol+" = ?", group)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return OpsMonitorResult{}, err
	}

	var logs []Log
	if err := query.
		Select("id, created_at, type, content, model_name, prompt_tokens, completion_tokens, use_time, channel_id, "+logGroupCol+", request_id, upstream_request_id, other").
		Order("created_at asc, id asc").
		Limit(maxOpsMonitorLogs).
		Find(&logs).Error; err != nil {
		return OpsMonitorResult{}, err
	}

	channelIds := map[int]struct{}{}
	for i := range logs {
		if logs[i].ChannelId > 0 {
			channelIds[logs[i].ChannelId] = struct{}{}
		}
	}
	channelNames := getOpsChannelNames(channelIds)
	channels := map[int]*opsChannelAgg{}
	buckets := map[int64]*opsBucketAgg{}
	latencies := make([]int64, 0)
	ttfts := make([]int64, 0)
	latestErrors := make([]OpsMonitorError, 0, 20)

	result := OpsMonitorResult{
		StartTimestamp: startTs,
		EndTimestamp:   endTs,
		BucketSeconds:  bucketSeconds,
		UpdatedAt:      now,
		Truncated:      total > int64(len(logs)),
	}

	for i := range logs {
		log := logs[i]
		bucketTs := startTs + ((log.CreatedAt - startTs) / bucketSeconds * bucketSeconds)
		if bucketTs < startTs {
			bucketTs = startTs
		}
		bucket := buckets[bucketTs]
		if bucket == nil {
			bucket = &opsBucketAgg{}
			buckets[bucketTs] = bucket
		}
		channel := channels[log.ChannelId]
		if channel == nil {
			channel = &opsChannelAgg{}
			channels[log.ChannelId] = channel
		}

		other, _ := common.StrToMap(log.Other)
		switch log.Type {
		case LogTypeConsume:
			tokens := int64(log.PromptTokens + log.CompletionTokens)
			result.Overview.SuccessCount++
			result.Overview.PromptTokens += int64(log.PromptTokens)
			result.Overview.CompletionTokens += int64(log.CompletionTokens)
			result.Overview.TotalTokens += tokens
			bucket.requests++
			bucket.tokens += tokens
			channel.requests++
			channel.tokens += tokens
			if log.UseTime > 0 {
				ms := int64(log.UseTime) * 1000
				channel.latencyMs += ms
				channel.latencyCount++
				latencies = append(latencies, ms)
			}
			if ttft, ok := opsFloat(other, "frt"); ok && ttft >= 0 {
				ms := int64(ttft)
				bucket.ttftMs += ms
				bucket.ttftCount++
				channel.ttftMs += ms
				channel.ttftCount++
				ttfts = append(ttfts, ms)
			}
			switches := opsChannelSwitchCount(other)
			if switches > 0 {
				bucket.switches += switches
				channel.channelSwitches += switches
			}
		case LogTypeError:
			statusCode := int(opsNumber(other, "status_code"))
			errItem := OpsMonitorError{
				Id:            log.Id,
				CreatedAt:     log.CreatedAt,
				ChannelId:     log.ChannelId,
				ChannelName:   channelNames[log.ChannelId],
				ModelName:     log.ModelName,
				StatusCode:    statusCode,
				ErrorCode:     opsString(other, "error_code"),
				ErrorType:     opsString(other, "error_type"),
				RequestId:     log.RequestId,
				UpstreamId:    log.UpstreamRequestId,
				Content:       log.Content,
				RequestPath:   opsString(other, "request_path"),
				BusinessLike:  isOpsBusinessLikeError(statusCode, opsString(other, "error_code")),
			}
			result.Overview.ErrorCount++
			bucket.errors++
			channel.errors++
			if errItem.BusinessLike {
				result.Overview.ExcludedErrors++
				bucket.excludedErrs++
				channel.excludedErrs++
			}
			if log.UseTime > 0 {
				ms := int64(log.UseTime) * 1000
				channel.latencyMs += ms
				channel.latencyCount++
				latencies = append(latencies, ms)
			}
			channel.latestError = errItem
			latestErrors = append(latestErrors, errItem)
		}
	}

	result.Overview.RequestCount = result.Overview.SuccessCount + result.Overview.ErrorCount
	seconds := float64(endTs - startTs)
	if seconds <= 0 {
		seconds = 1
	}
	result.Overview.AvgQPS = round2(float64(result.Overview.RequestCount) / seconds)
	result.Overview.AvgTPS = round2(float64(result.Overview.TotalTokens) / seconds)
	result.Overview.ErrorRate = errorPercent(result.Overview.ErrorCount, result.Overview.RequestCount)
	result.Overview.UpstreamErrorRate = errorPercent(result.Overview.ErrorCount-result.Overview.ExcludedErrors, result.Overview.RequestCount)
	result.Overview.SLA = opsSLA(result.Overview.SuccessCount, result.Overview.ErrorCount, result.Overview.ExcludedErrors)
	result.Overview.AvgLatencyMs = avgInt64(latencies)
	result.Overview.AvgTTFTMs = avgInt64(ttfts)
	result.Overview.P95LatencyMs = percentileInt64(latencies, 0.95)
	result.Overview.P99LatencyMs = percentileInt64(latencies, 0.99)
	result.Overview.P95TTFTMs = percentileInt64(ttfts, 0.95)
	result.Overview.P99TTFTMs = percentileInt64(ttfts, 0.99)

	result.Trend = make([]OpsMonitorPoint, 0, bucketCount)
	result.ChannelSwitch = make([]OpsMonitorChannelSwitchPoint, 0, bucketCount)
	for ts := startTs; ts <= endTs; ts += bucketSeconds {
		bucket := buckets[ts]
		if bucket == nil {
			bucket = &opsBucketAgg{}
		}
		result.Trend = append(result.Trend, OpsMonitorPoint{
			Ts:        ts,
			Requests: bucket.requests + bucket.errors,
			Errors:   bucket.errors,
			Tokens:   bucket.tokens,
			QPS:      round2(float64(bucket.requests+bucket.errors) / float64(bucketSeconds)),
			TPS:      round2(float64(bucket.tokens) / float64(bucketSeconds)),
			SLA:      opsSLA(bucket.requests, bucket.errors, bucket.excludedErrs),
			AvgTTFT:  avgPair(bucket.ttftMs, bucket.ttftCount),
		})
		result.ChannelSwitch = append(result.ChannelSwitch, OpsMonitorChannelSwitchPoint{
			Ts:       ts,
			Switches: bucket.switches,
		})
	}

	result.Channels = make([]OpsMonitorChannel, 0, len(channels))
	for id, agg := range channels {
		totalReq := agg.requests + agg.errors
		item := OpsMonitorChannel{
			ChannelId:       id,
			ChannelName:     channelNames[id],
			Requests:        totalReq,
			Errors:          agg.errors,
			Tokens:          agg.tokens,
			AvgLatencyMs:    avgPair(agg.latencyMs, agg.latencyCount),
			AvgTTFTMs:       avgPair(agg.ttftMs, agg.ttftCount),
			SLA:             opsSLA(agg.requests, agg.errors, agg.excludedErrs),
			QueueDepth:      0,
			ChannelSwitches: agg.channelSwitches,
		}
		if agg.latestError.Id != 0 {
			item.LatestError = agg.latestError.Content
			item.LatestErrorAt = agg.latestError.CreatedAt
			item.LatestStatusCode = agg.latestError.StatusCode
		}
		result.Channels = append(result.Channels, item)
	}
	sort.Slice(result.Channels, func(i, j int) bool {
		return result.Channels[i].Requests > result.Channels[j].Requests
	})

	sort.Slice(latestErrors, func(i, j int) bool {
		return latestErrors[i].CreatedAt > latestErrors[j].CreatedAt
	})
	if len(latestErrors) > 50 {
		latestErrors = latestErrors[:50]
	}
	result.Errors = latestErrors

	return result, nil
}

func chooseOpsBucketSeconds(rangeSeconds int64) int64 {
	switch {
	case rangeSeconds <= 3600:
		return 60
	case rangeSeconds <= 6*3600:
		return 300
	case rangeSeconds <= 24*3600:
		return 900
	case rangeSeconds <= 7*24*3600:
		return 3600
	default:
		return 6 * 3600
	}
}

func getOpsChannelNames(ids map[int]struct{}) map[int]string {
	names := map[int]string{0: "未分组"}
	if len(ids) == 0 {
		return names
	}
	idList := make([]int, 0, len(ids))
	for id := range ids {
		if id > 0 {
			idList = append(idList, id)
		}
	}
	if len(idList) == 0 {
		return names
	}
	var rows []struct {
		Id   int
		Name string
	}
	_ = DB.Table("channels").Select("id, name").Where("id IN ?", idList).Find(&rows).Error
	for _, row := range rows {
		names[row.Id] = row.Name
	}
	for _, id := range idList {
		if names[id] == "" {
			names[id] = "渠道 #" + strconv.Itoa(id)
		}
	}
	return names
}

func opsChannelSwitchCount(other map[string]interface{}) int64 {
	admin, ok := other["admin_info"].(map[string]interface{})
	if !ok {
		return 0
	}
	raw, ok := admin["use_channel"].([]interface{})
	if !ok || len(raw) <= 1 {
		return 0
	}
	return int64(len(raw) - 1)
}

func opsString(other map[string]interface{}, key string) string {
	if other == nil {
		return ""
	}
	if value, ok := other[key]; ok && value != nil {
		return strings.TrimSpace(common.Interface2String(value))
	}
	return ""
}

func opsNumber(other map[string]interface{}, key string) float64 {
	value, ok := opsFloat(other, key)
	if !ok {
		return 0
	}
	return value
}

func opsFloat(other map[string]interface{}, key string) (float64, bool) {
	if other == nil {
		return 0, false
	}
	switch value := other[key].(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case string:
		v, err := strconv.ParseFloat(value, 64)
		return v, err == nil
	default:
		return 0, false
	}
}

func isOpsBusinessLikeError(statusCode int, code string) bool {
	if statusCode == 429 || statusCode == 529 {
		return true
	}
	code = strings.ToLower(code)
	return strings.Contains(code, "insufficient") || strings.Contains(code, "quota") || strings.Contains(code, "rate_limit")
}

func opsSLA(success int64, errors int64, excludedErrors int64) float64 {
	eligibleErrors := errors - excludedErrors
	if eligibleErrors < 0 {
		eligibleErrors = 0
	}
	return percent(success, success+eligibleErrors)
}

func percent(part int64, total int64) float64 {
	if total <= 0 {
		return 100
	}
	return round3(float64(part) / float64(total) * 100)
}

func errorPercent(part int64, total int64) float64 {
	if total <= 0 {
		return 0
	}
	if part < 0 {
		part = 0
	}
	return round3(float64(part) / float64(total) * 100)
}

func avgInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	var sum int64
	for _, value := range values {
		sum += value
	}
	return sum / int64(len(values))
}

func avgPair(sum int64, count int64) int64 {
	if count <= 0 {
		return 0
	}
	return sum / count
}

func percentileInt64(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	idx := int(math.Ceil(float64(len(values))*p)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
