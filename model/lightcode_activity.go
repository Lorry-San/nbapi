package model

import (
	"sort"
	"time"

	"github.com/Lorry-San/nbapi/common"
)

const (
	LightcodeActivityDays      = 371
	lightcodeActivityMaxModels = 10
)

// Activity structs carry token and request counts only: no quota, price or
// ratio fields may ever be added here.
type LightcodeActivityDay struct {
	Date        string `json:"date"`
	Requests    int64  `json:"requests"`
	TotalTokens int64  `json:"total_tokens"`
}

type LightcodeActivityStats struct {
	TotalTokens       int64  `json:"total_tokens"`
	Requests          int64  `json:"requests"`
	ActiveDays        int    `json:"active_days"`
	PeakDayTokens     int64  `json:"peak_day_tokens"`
	PeakDay           string `json:"peak_day"`
	LongestStreakDays int    `json:"longest_streak_days"`
	CurrentStreakDays int    `json:"current_streak_days"`
}

type LightcodeActivity struct {
	Stats     LightcodeActivityStats `json:"stats"`
	Days      []LightcodeActivityDay `json:"days"`
	Models    []LightcodeUsageModel  `json:"models"`
	Truncated bool                   `json:"truncated"`
}

// GetLightcodeActivity aggregates the user's Lightcode consume logs over the
// last 53 weeks (ending today in the tzOffsetMinutes zone, minutes east of UTC).
func GetLightcodeActivity(userId int, tzOffsetMinutes int, now int64) (*LightcodeActivity, error) {
	days := LightcodeActivityDays
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
		Order("created_at desc").Limit(lightcodeUsageMaxRows).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	res := &LightcodeActivity{Days: make([]LightcodeActivityDay, days), Models: []LightcodeUsageModel{}, Truncated: len(rows) >= lightcodeUsageMaxRows}
	for i := 0; i < days; i++ {
		res.Days[i].Date = time.Unix(startLocal+int64(i)*86400, 0).UTC().Format("2006-01-02")
	}
	models := map[string]*LightcodeUsageModel{}
	for _, r := range rows {
		otherMap, _ := common.StrToMap(r.Other)
		if !isLightcodeLogOther(&Log{TokenName: r.TokenName}, otherMap) {
			continue
		}
		idx := int((r.CreatedAt + off - startLocal) / 86400)
		if r.CreatedAt+off < startLocal || idx >= days {
			continue
		}
		total := int64(r.PromptTokens) + int64(r.CompletionTokens)
		res.Days[idx].Requests++
		res.Days[idx].TotalTokens += total
		res.Stats.Requests++
		res.Stats.TotalTokens += total
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
	}

	run := 0
	for _, d := range res.Days {
		if d.TotalTokens <= 0 {
			run = 0
			continue
		}
		res.Stats.ActiveDays++
		run++
		if run > res.Stats.LongestStreakDays {
			res.Stats.LongestStreakDays = run
		}
		if d.TotalTokens > res.Stats.PeakDayTokens {
			res.Stats.PeakDayTokens = d.TotalTokens
			res.Stats.PeakDay = d.Date
		}
	}
	i := days - 1
	if res.Days[i].TotalTokens <= 0 {
		i--
	}
	for ; i >= 0 && res.Days[i].TotalTokens > 0; i-- {
		res.Stats.CurrentStreakDays++
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
	if len(res.Models) > lightcodeActivityMaxModels {
		res.Models = res.Models[:lightcodeActivityMaxModels]
	}
	return res, nil
}

// GetUserDisplayName reads only the display_name column; the user cache does
// not carry it.
func GetUserDisplayName(userId int) (string, error) {
	var name string
	err := DB.Model(&User{}).Select("display_name").Where("id = ?", userId).Limit(1).Scan(&name).Error
	return name, err
}
