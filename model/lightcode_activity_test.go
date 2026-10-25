package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// now is 2027-01-15 08:00:00 UTC.
const lightcodeActivityNow = int64(1800000000)

func lcActivityLog(user int, daysAgo int, model string, tokens int) Log {
	return Log{UserId: user, Type: LogTypeConsume, CreatedAt: lightcodeActivityNow - int64(daysAgo)*86400, ModelName: model, PromptTokens: tokens, CompletionTokens: tokens, Quota: 999, Other: `{"lightcode":true,"group_ratio":2}`}
}

func seedLightcodeActivity(t *testing.T, logs []Log) {
	truncateTables(t)
	require.NoError(t, LOG_DB.Exec("DELETE FROM logs").Error)
	require.NoError(t, DB.AutoMigrate(&LightcodeGroupModel{}))
	require.NoError(t, DB.Where("1 = 1").Delete(&LightcodeGroupModel{}).Error)
	t.Cleanup(func() { DB.Where("1 = 1").Delete(&LightcodeGroupModel{}) })
	for i := range logs {
		require.NoError(t, LOG_DB.Create(&logs[i]).Error)
	}
}

func TestGetLightcodeActivityShapeAndFiltering(t *testing.T) {
	seedLightcodeActivity(t, []Log{
		lcActivityLog(1, 0, "m-a", 50),
		{UserId: 1, Type: LogTypeConsume, CreatedAt: lightcodeActivityNow - 5, ModelName: "m-b", PromptTokens: 10, CompletionTokens: 5, TokenName: "Lightcode", Other: `{"billing_source":"subscription"}`},
		{UserId: 1, Type: LogTypeConsume, CreatedAt: lightcodeActivityNow - 5, ModelName: "m-a", PromptTokens: 7, TokenName: "x", Other: `{}`},
		{UserId: 1, Type: LogTypeError, CreatedAt: lightcodeActivityNow - 5, ModelName: "m-a", PromptTokens: 5, Other: `{"lightcode":true}`},
		lcActivityLog(2, 0, "m-a", 1000),
		lcActivityLog(1, 371, "m-a", 1000),
	})
	res, err := GetLightcodeActivity(1, 0, lightcodeActivityNow)
	require.NoError(t, err)
	require.Len(t, res.Days, LightcodeActivityDays)
	require.Equal(t, "2027-01-15", res.Days[370].Date)
	require.Equal(t, "2026-01-10", res.Days[0].Date)
	require.EqualValues(t, 2, res.Stats.Requests)
	require.EqualValues(t, 115, res.Stats.TotalTokens)
	require.EqualValues(t, 115, res.Days[370].TotalTokens)
	require.False(t, res.Truncated)
	require.Equal(t, []LightcodeActivityModel{{ModelID: "m-a", Model: "m-a", Requests: 1, TotalTokens: 100}, {ModelID: "m-b", Model: "m-b", Requests: 1, TotalTokens: 15}}, res.Models)
	for _, d := range res.Days[:370] {
		require.Zero(t, d.Requests)
	}

	raw, err := json.Marshal(res)
	require.NoError(t, err)
	body := strings.ToLower(string(raw))
	for _, banned := range []string{"quota", "price", "ratio", "money", "amount", "cost", "multiplier"} {
		require.NotContains(t, body, banned)
	}
}

func TestGetLightcodeActivityUsesTheAgentCatalogNames(t *testing.T) {
	seedLightcodeActivity(t, []Log{
		lcActivityLog(1, 0, "deepseek-v4.1-flash", 50),
		lcActivityLog(1, 0, "removed-model", 10),
	})
	require.NoError(t, DB.Create(&[]UserSubscription{
		{UserId: 1, Pool: SubscriptionPoolLightcode, RoutingGroup: "old", Status: "expired", StartTime: 1},
		{UserId: 1, Pool: SubscriptionPoolLightcode, RoutingGroup: "current", Status: "active", StartTime: 2},
	}).Error)
	require.NoError(t, DB.Create(&[]LightcodeGroupModel{
		{RoutingGroup: "unrelated", Model: "removed-model", DisplayName: "Other user's name"},
		{RoutingGroup: "old", Model: "deepseek-v4.1-flash", DisplayName: "Old name"},
		{RoutingGroup: "current", Model: "deepseek-v4.1-flash", DisplayName: "DeepSeek V4.1 Flash"},
	}).Error)
	res, err := GetLightcodeActivity(1, 0, lightcodeActivityNow)
	require.NoError(t, err)
	require.Equal(t, "DeepSeek V4.1 Flash", res.Models[0].Model)
	require.Equal(t, "removed-model", res.Models[1].Model)
	require.EqualValues(t, 120, res.Stats.TotalTokens)
	// An expired subscription still provides names for its historical usage.
	require.NoError(t, DB.Where("user_id = ? AND status = ?", 1, "active").Delete(&UserSubscription{}).Error)
	res, err = GetLightcodeActivity(1, 0, lightcodeActivityNow)
	require.NoError(t, err)
	require.Equal(t, "Old name", res.Models[0].Model)
}

func TestGetLightcodeActivityTimeZone(t *testing.T) {
	// 23:00 UTC yesterday: yesterday at UTC, today at UTC+1, still yesterday at UTC-10.
	seedLightcodeActivity(t, []Log{{UserId: 1, Type: LogTypeConsume, CreatedAt: lightcodeActivityNow - 9*3600, ModelName: "m", PromptTokens: 1, Other: `{"lightcode":true}`}})
	cases := []struct {
		tz       int
		today    string
		lastIdx  int
		tokensAt int
	}{
		{0, "2027-01-15", 370, 369},
		{60, "2027-01-15", 370, 370},
		{-600, "2027-01-14", 370, 370},
		{840, "2027-01-15", 370, 370},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.tz), func(t *testing.T) {
			res, err := GetLightcodeActivity(1, tc.tz, lightcodeActivityNow)
			require.NoError(t, err)
			require.Len(t, res.Days, LightcodeActivityDays)
			require.Equal(t, tc.today, res.Days[tc.lastIdx].Date)
			require.EqualValues(t, 1, res.Days[tc.tokensAt].TotalTokens)
		})
	}
}

func TestGetLightcodeActivityStreaks(t *testing.T) {
	cases := []struct {
		name    string
		daysAgo []int
		current int
		longest int
		active  int
	}{
		{"none", nil, 0, 0, 0},
		{"today only", []int{0}, 1, 1, 1},
		{"today empty yesterday active", []int{1, 2, 3}, 3, 3, 3},
		{"two days ago breaks", []int{2, 3}, 0, 2, 2},
		{"longest across gap", []int{0, 1, 5, 6, 7, 8}, 2, 4, 6},
		{"oldest day counts", []int{370, 369}, 0, 2, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var logs []Log
			for _, d := range tc.daysAgo {
				logs = append(logs, lcActivityLog(1, d, "m", 1))
			}
			seedLightcodeActivity(t, logs)
			res, err := GetLightcodeActivity(1, 0, lightcodeActivityNow)
			require.NoError(t, err)
			require.Equal(t, tc.current, res.Stats.CurrentStreakDays)
			require.Equal(t, tc.longest, res.Stats.LongestStreakDays)
			require.Equal(t, tc.active, res.Stats.ActiveDays)
		})
	}
}

func TestGetLightcodeActivityPeakAndTopModels(t *testing.T) {
	var logs []Log
	for i := 0; i < 12; i++ {
		logs = append(logs, lcActivityLog(1, 10, fmt.Sprintf("m-%02d", i), 10+i))
	}
	logs = append(logs, lcActivityLog(1, 3, "m-00", 5))
	seedLightcodeActivity(t, logs)
	res, err := GetLightcodeActivity(1, 0, lightcodeActivityNow)
	require.NoError(t, err)
	require.Equal(t, "2027-01-05", res.Stats.PeakDay)
	require.EqualValues(t, 2*(12*10+66), res.Stats.PeakDayTokens)
	require.Len(t, res.Models, 10)
	require.Equal(t, "m-11", res.Models[0].Model)
	// m-00 ties m-05 at 30 tokens and wins on name.
	require.Equal(t, "m-00", res.Models[6].Model)
	require.EqualValues(t, 30, res.Models[6].TotalTokens)
	require.Equal(t, "m-05", res.Models[7].Model)
	require.Equal(t, "m-03", res.Models[9].Model)

	seedLightcodeActivity(t, nil)
	empty, err := GetLightcodeActivity(1, 0, lightcodeActivityNow)
	require.NoError(t, err)
	require.Equal(t, "", empty.Stats.PeakDay)
	require.NotNil(t, empty.Models)
}

func TestGetLightcodeActivityCountsAnthropicCacheInTotal(t *testing.T) {
	truncateTables(t)
	now := int64(1800000000)
	logs := []Log{
		{UserId: 1, Type: LogTypeConsume, CreatedAt: now - 10, ModelName: "claude-x", PromptTokens: 100, CompletionTokens: 50, TokenName: "Lightcode", Other: `{"lightcode":true,"usage_semantic":"anthropic","cache_tokens":30,"cache_write_tokens":20}`},
		{UserId: 1, Type: LogTypeConsume, CreatedAt: now - 20, ModelName: "gpt-x", PromptTokens: 100, CompletionTokens: 50, TokenName: "Lightcode", Other: `{"lightcode":true,"cache_tokens":30}`},
	}
	for i := range logs {
		require.NoError(t, LOG_DB.Create(&logs[i]).Error)
	}

	res, err := GetLightcodeActivity(1, 0, now)
	require.NoError(t, err)
	byModel := map[string]int64{}
	for _, m := range res.Models {
		byModel[m.Model] = m.TotalTokens
	}
	require.EqualValues(t, 200, byModel["claude-x"])
	require.EqualValues(t, 150, byModel["gpt-x"])
}
