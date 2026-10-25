package model

import (
	"sort"
	"time"

	"github.com/Lorry-San/nbapi/common"
)

const (
	LightcodeUsageDefaultDays = 14
	LightcodeUsageMaxDays     = 90
	lightcodeUsageMaxRows     = 300000
	lightcodeUsageMaxModels   = 50
)

// Usage structs deliberately carry token and request counts only: no quota,
// price or ratio fields may ever be added here.
type LightcodeUsageDay struct {
	Date         string `json:"date"`
	Requests     int64  `json:"requests"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	CacheTokens  int64  `json:"cache_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
}

type LightcodeUsageModelDay struct {
	Date        string `json:"date"`
	Model       string `json:"model"`
	TotalTokens int64  `json:"total_tokens"`
}

type LightcodeUsageModel struct {
	Model       string `json:"model"`
	Requests    int64  `json:"requests"`
	TotalTokens int64  `json:"total_tokens"`
}

type LightcodeUsageTotals struct {
	Requests     int64 `json:"requests"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	CacheTokens  int64 `json:"cache_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

type LightcodeUsageResult struct {
	Days      []LightcodeUsageDay      `json:"days"`
	ModelDays []LightcodeUsageModelDay `json:"model_days"`
	Models    []LightcodeUsageModel    `json:"models"`
	Totals    LightcodeUsageTotals     `json:"totals"`
}

type lightcodeUsageRow struct {
	CreatedAt        int64
	ModelName        string
	TokenName        string
	PromptTokens     int
	CompletionTokens int
	Other            string
}

// GetLightcodeUsage aggregates the user's Lightcode consume logs per local day
// (tzOffsetMinutes is minutes east of UTC) and per model for the last `days` days.
func GetLightcodeUsage(userId int, days int, tzOffsetMinutes int, now int64) (*LightcodeUsageResult, error) {
	if days <= 0 {
		days = LightcodeUsageDefaultDays
	}
	if days > LightcodeUsageMaxDays {
		days = LightcodeUsageMaxDays
	}
	off := int64(tzOffsetMinutes) * 60
	nowLocal := now + off
	todayLocal := nowLocal - ((nowLocal%86400)+86400)%86400
	startLocal := todayLocal - int64(days-1)*86400
	start := startLocal - off
	end := todayLocal + 86400 - off

	var rows []lightcodeUsageRow
	err := LOG_DB.Table("logs").
		Select("created_at, model_name, token_name, prompt_tokens, completion_tokens, other").
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at < ?", userId, LogTypeConsume, start, end).
		Where("(other LIKE ? OR token_name = ?)", "%lightcode%", "Lightcode").
		Order("created_at asc").Limit(lightcodeUsageMaxRows).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	res := &LightcodeUsageResult{Days: make([]LightcodeUsageDay, days), ModelDays: []LightcodeUsageModelDay{}, Models: []LightcodeUsageModel{}}
	for i := 0; i < days; i++ {
		res.Days[i].Date = time.Unix(startLocal+int64(i)*86400, 0).UTC().Format("2006-01-02")
	}
	models := map[string]*LightcodeUsageModel{}
	type mdKey struct {
		day   int
		model string
	}
	modelDays := map[mdKey]int64{}

	for _, r := range rows {
		otherMap, _ := common.StrToMap(r.Other)
		if !isLightcodeLogOther(&Log{TokenName: r.TokenName}, otherMap) {
			continue
		}
		idx := int((r.CreatedAt + off - startLocal) / 86400)
		if idx < 0 || idx >= days {
			continue
		}
		var cache int64
		switch v := otherMap["cache_tokens"].(type) {
		case float64:
			cache = int64(v)
		case int:
			cache = int64(v)
		}
		in, out := int64(r.PromptTokens), int64(r.CompletionTokens)
		total := in + out
		// Anthropic /v1/messages reports input_tokens without cache read/write,
		// so total input adds them back. Other protocols already include cache
		// in prompt_tokens and must not be counted twice.
		if semantic, _ := otherMap["usage_semantic"].(string); semantic == "anthropic" {
			var cacheWrite int64
			switch v := otherMap["cache_write_tokens"].(type) {
			case float64:
				cacheWrite = int64(v)
			case int:
				cacheWrite = int64(v)
			}
			total += cache + cacheWrite
		}
		d := &res.Days[idx]
		d.Requests++
		d.InputTokens += in
		d.OutputTokens += out
		d.CacheTokens += cache
		d.TotalTokens += total
		name := r.ModelName
		if name == "" {
			name = "unknown"
		}
		m := models[name]
		if m == nil {
			m = &LightcodeUsageModel{Model: name}
			models[name] = m
		}
		m.Requests++
		m.TotalTokens += total
		modelDays[mdKey{idx, name}] += total
		t := &res.Totals
		t.Requests++
		t.InputTokens += in
		t.OutputTokens += out
		t.CacheTokens += cache
		t.TotalTokens += total
	}

	for _, m := range models {
		res.Models = append(res.Models, *m)
	}
	sort.Slice(res.Models, func(i, j int) bool {
		if res.Models[i].TotalTokens != res.Models[j].TotalTokens {
			return res.Models[i].TotalTokens > res.Models[j].TotalTokens
		}
		return res.Models[i].Model < res.Models[j].Model
	})
	if len(res.Models) > lightcodeUsageMaxModels {
		res.Models = res.Models[:lightcodeUsageMaxModels]
	}
	keep := map[string]bool{}
	for _, m := range res.Models {
		keep[m.Model] = true
	}
	for k, v := range modelDays {
		if keep[k.model] {
			res.ModelDays = append(res.ModelDays, LightcodeUsageModelDay{Date: res.Days[k.day].Date, Model: k.model, TotalTokens: v})
		}
	}
	sort.Slice(res.ModelDays, func(i, j int) bool {
		if res.ModelDays[i].Date != res.ModelDays[j].Date {
			return res.ModelDays[i].Date < res.ModelDays[j].Date
		}
		return res.ModelDays[i].Model < res.ModelDays[j].Model
	})
	return res, nil
}
