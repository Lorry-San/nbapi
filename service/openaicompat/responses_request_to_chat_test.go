package openaicompat

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestResponsesRequestToChatCompletionsRequestPreservesCoreFields(t *testing.T) {
	stream := true
	includeUsage := true
	maxOutputTokens := uint(512)
	temperature := 0.2
	topP := 0.9

	req := &dto.OpenAIResponsesRequest{
		Model: "kimi-k2.7",
		Input: json.RawMessage(`[
			{"role":"user","content":[
				{"type":"input_text","text":"look at this"},
				{"type":"input_image","image_url":"https://example.com/a.png"}
			]},
			{"type":"function_call_output","call_id":"call_1","output":"done"}
		]`),
		Instructions:         json.RawMessage(`"be concise"`),
		Stream:               &stream,
		StreamOptions:        &dto.StreamOptions{IncludeUsage: includeUsage},
		MaxOutputTokens:      &maxOutputTokens,
		Temperature:          &temperature,
		TopP:                 &topP,
		Tools:                json.RawMessage(`[{"type":"function","name":"lookup","description":"lookup docs","parameters":{"type":"object"}}]`),
		ToolChoice:           json.RawMessage(`{"type":"function","name":"lookup"}`),
		ParallelToolCalls:    json.RawMessage(`false`),
		PromptCacheKey:       json.RawMessage(`"tenant-a"`),
		SafetyIdentifier:     json.RawMessage(`"user-123"`),
		PromptCacheRetention: json.RawMessage(`"24h"`),
	}

	out, err := ResponsesRequestToChatCompletionsRequest(req)
	require.NoError(t, err)
	require.Equal(t, "kimi-k2.7", out.Model)
	require.True(t, *out.Stream)
	require.NotNil(t, out.StreamOptions)
	require.True(t, out.StreamOptions.IncludeUsage)
	require.Equal(t, maxOutputTokens, *out.MaxCompletionTokens)
	require.Equal(t, temperature, *out.Temperature)
	require.Equal(t, topP, *out.TopP)
	require.Equal(t, "tenant-a", out.PromptCacheKey)
	require.JSONEq(t, `"24h"`, string(out.PromptCacheRetention))
	require.JSONEq(t, `"user-123"`, string(out.SafetyIdentifier))
	require.NotNil(t, out.ParallelTooCalls)
	require.False(t, *out.ParallelTooCalls)

	require.Len(t, out.Messages, 3)
	require.Equal(t, "system", out.Messages[0].Role)
	require.Equal(t, "be concise", out.Messages[0].StringContent())
	require.Equal(t, "user", out.Messages[1].Role)
	require.Equal(t, "tool", out.Messages[2].Role)
	require.Equal(t, "call_1", out.Messages[2].ToolCallId)

	parts := out.Messages[1].ParseContent()
	require.Len(t, parts, 2)
	require.Equal(t, dto.ContentTypeText, parts[0].Type)
	require.Equal(t, dto.ContentTypeImageURL, parts[1].Type)

	require.Len(t, out.Tools, 1)
	require.Equal(t, "lookup", out.Tools[0].Function.Name)

	choice, ok := out.ToolChoice.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "function", choice["type"])
	fn, ok := choice["function"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "lookup", fn["name"])
}

func TestResponsesRequestToChatCompletionsRequestRejectsPreviousResponseID(t *testing.T) {
	_, err := ResponsesRequestToChatCompletionsRequest(&dto.OpenAIResponsesRequest{
		Model:              "kimi-k2.7",
		PreviousResponseID: "resp_previous",
		Input:              json.RawMessage(`"hi"`),
	})
	require.Error(t, err)
}

func TestChatCompletionsResponseToResponsesResponsePreservesToolCalls(t *testing.T) {
	msg := dto.Message{Role: "assistant", Content: "need a lookup"}
	msg.SetToolCalls([]dto.ToolCallRequest{{
		ID:   "call_1",
		Type: "function",
		Function: dto.FunctionRequest{
			Name:      "lookup",
			Arguments: `{"q":"nbapi"}`,
		},
	}})

	resp := &dto.OpenAITextResponse{
		Id:      "chatcmpl_123",
		Model:   "kimi-k2.7",
		Created: float64(123),
		Choices: []dto.OpenAITextResponseChoice{{
			Message:      msg,
			FinishReason: "tool_calls",
		}},
		Usage: dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}

	out, usage, err := ChatCompletionsResponseToResponsesResponse(resp, "resp_123")
	require.NoError(t, err)
	require.Equal(t, "resp_123", out.ID)
	require.Equal(t, 123, out.CreatedAt)
	require.Equal(t, 15, usage.TotalTokens)
	require.Len(t, out.Output, 2)
	require.Equal(t, "message", out.Output[0].Type)
	require.Equal(t, "function_call", out.Output[1].Type)
	require.Equal(t, "call_1", out.Output[1].CallId)
	require.Equal(t, "lookup", out.Output[1].Name)
	require.JSONEq(t, `{"q":"nbapi"}`, string(out.Output[1].Arguments))

	body, err := common.Marshal(out)
	require.NoError(t, err)
	require.JSONEq(t, `"completed"`, string(out.Status))
	require.Contains(t, string(body), `"object":"response"`)
}

func TestChatCompletionsResponseToResponsesResponseThinkingToContentOption(t *testing.T) {
	reasoning := "visible reasoning"
	resp := &dto.OpenAITextResponse{
		Id:      "chatcmpl_reasoning",
		Model:   "kimi-k2.7",
		Created: float64(123),
		Choices: []dto.OpenAITextResponseChoice{{
			Message: dto.Message{
				Role:             "assistant",
				Content:          "visible answer",
				ReasoningContent: &reasoning,
			},
			FinishReason: "stop",
		}},
		Usage: dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}

	hidden, _, err := ChatCompletionsResponseToResponsesResponse(resp, "resp_hidden")
	require.NoError(t, err)
	require.Len(t, hidden.Output, 1)
	require.Equal(t, "visible answer", hidden.Output[0].Content[0].Text)

	visible, _, err := ChatCompletionsResponseToResponsesResponseWithOptions(resp, "resp_visible", ChatCompletionsToResponsesOptions{
		ThinkingToContent: true,
	})
	require.NoError(t, err)
	require.Len(t, visible.Output, 1)
	require.Equal(t, "<think>\nvisible reasoning\n</think>\nvisible answer", visible.Output[0].Content[0].Text)
}
