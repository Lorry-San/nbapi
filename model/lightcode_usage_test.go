package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLightcodeUsageCountsAnthropicCacheInTotal(t *testing.T) {
	truncateTables(t)
	now := int64(1800000000)
	logs := []Log{
		// Anthropic: input_tokens excludes cache, so total = input + output + cache read + cache write.
		{UserId: 1, Type: LogTypeConsume, CreatedAt: now - 10, ModelName: "claude-x", PromptTokens: 100, CompletionTokens: 50, TokenName: "Lightcode", Other: `{"lightcode":true,"usage_semantic":"anthropic","cache_tokens":30,"cache_write_tokens":20}`},
		// OpenAI-style: cache is already inside prompt_tokens, so total must not add it again.
		{UserId: 1, Type: LogTypeConsume, CreatedAt: now - 20, ModelName: "gpt-x", PromptTokens: 100, CompletionTokens: 50, TokenName: "Lightcode", Other: `{"lightcode":true,"cache_tokens":30}`},
	}
	for i := range logs {
		require.NoError(t, LOG_DB.Create(&logs[i]).Error)
	}

	res, err := GetLightcodeUsage(1, 7, 0, now)
	require.NoError(t, err)
	require.Len(t, res.Models, 2)
	byModel := map[string]int64{}
	for _, m := range res.Models {
		byModel[m.Model] = m.TotalTokens
	}
	require.EqualValues(t, 200, byModel["claude-x"])
	require.EqualValues(t, 150, byModel["gpt-x"])
}

func TestGetLightcodeUsageAggregatesOnlyOwnLightcodeLogs(t *testing.T) {
	truncateTables(t)
	now := int64(1800000000)
	logs := []Log{
		{UserId: 1, Type: LogTypeConsume, CreatedAt: now - 10, ModelName: "m-a", PromptTokens: 100, CompletionTokens: 50, Quota: 999, Other: `{"lightcode":true,"cache_tokens":30,"group_ratio":2}`},
		{UserId: 1, Type: LogTypeConsume, CreatedAt: now - 20, ModelName: "m-b", PromptTokens: 10, CompletionTokens: 5, TokenName: "Lightcode", Other: `{"billing_source":"subscription"}`},
		{UserId: 1, Type: LogTypeConsume, CreatedAt: now - 30, ModelName: "m-a", PromptTokens: 7, CompletionTokens: 3, TokenName: "x", Other: `{"billing_source":"subscription"}`},
		{UserId: 2, Type: LogTypeConsume, CreatedAt: now - 10, ModelName: "m-a", PromptTokens: 1000, CompletionTokens: 1000, Other: `{"lightcode":true}`},
		{UserId: 1, Type: LogTypeError, CreatedAt: now - 10, ModelName: "m-a", PromptTokens: 5, Other: `{"lightcode":true}`},
		{UserId: 1, Type: LogTypeConsume, CreatedAt: now - 100*86400, ModelName: "m-a", PromptTokens: 5, Other: `{"lightcode":true}`},
	}
	for i := range logs {
		require.NoError(t, LOG_DB.Create(&logs[i]).Error)
	}

	res, err := GetLightcodeUsage(1, 7, 0, now)
	require.NoError(t, err)
	require.Len(t, res.Days, 7)
	require.EqualValues(t, 2, res.Totals.Requests)
	require.EqualValues(t, 110, res.Totals.InputTokens)
	require.EqualValues(t, 55, res.Totals.OutputTokens)
	require.EqualValues(t, 30, res.Totals.CacheTokens)
	require.EqualValues(t, 165, res.Totals.TotalTokens)
	require.Len(t, res.Models, 2)
	require.Equal(t, "m-a", res.Models[0].Model)
	require.EqualValues(t, 150, res.Models[0].TotalTokens)

	raw, err := json.Marshal(res)
	require.NoError(t, err)
	body := strings.ToLower(string(raw))
	for _, banned := range []string{"quota", "price", "ratio", "money", "amount", "cost"} {
		require.NotContains(t, body, banned)
	}
}
