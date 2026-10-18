package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
