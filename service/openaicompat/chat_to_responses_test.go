package openaicompat

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestChatCompletionsRequestToResponsesRequestPreservesResponsesFields(t *testing.T) {
	stream := true
	includeUsage := true
	topLogprobs := 3
	parallelToolCalls := false

	req := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "hi",
			},
		},
		Stream:               &stream,
		StreamOptions:        &dto.StreamOptions{IncludeUsage: includeUsage},
		TopLogProbs:          &topLogprobs,
		ServiceTier:          json.RawMessage(`"flex"`),
		PromptCacheKey:       "tenant-a",
		PromptCacheRetention: json.RawMessage(`"24h"`),
		SafetyIdentifier:     json.RawMessage(`"user-123"`),
		ParallelTooCalls:     &parallelToolCalls,
	}

	out, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, "flex", out.ServiceTier)
	require.NotNil(t, out.StreamOptions)
	require.True(t, out.StreamOptions.IncludeUsage)
	require.Equal(t, &topLogprobs, out.TopLogProbs)
	require.JSONEq(t, `"tenant-a"`, string(out.PromptCacheKey))
	require.JSONEq(t, `"24h"`, string(out.PromptCacheRetention))
	require.JSONEq(t, `"user-123"`, string(out.SafetyIdentifier))
	require.JSONEq(t, `false`, string(out.ParallelToolCalls))
}

func TestChatCompletionsRequestToResponsesRequestConvertsLegacyFunctions(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "look it up",
			},
		},
		Functions: json.RawMessage(`[{"name":"lookup","description":"lookup docs","parameters":{"type":"object"}}]`),
		FunctionCall: json.RawMessage(`{
			"name": "lookup"
		}`),
	}

	out, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	require.NotNil(t, out)

	var tools []map[string]any
	require.NoError(t, common.Unmarshal(out.Tools, &tools))
	require.Len(t, tools, 1)
	require.Equal(t, "function", tools[0]["type"])
	require.Equal(t, "lookup", tools[0]["name"])
	require.Equal(t, "lookup docs", tools[0]["description"])

	var toolChoice map[string]any
	require.NoError(t, common.Unmarshal(out.ToolChoice, &toolChoice))
	require.Equal(t, "function", toolChoice["type"])
	require.Equal(t, "lookup", toolChoice["name"])
}
