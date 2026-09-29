package oaichat

import (
	"testing"

	"github.com/Lorry-San/nbapi/dto"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestChatCompletionsRequestToResponsesRequestInstructionsAndTools(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Model: "gpt-test",
		N:     lo.ToPtr(1),
		Messages: []dto.Message{
			{Role: "system", Content: "system rules"},
			{Role: "developer", Content: "developer rules"},
			{Role: "user", Content: []any{
				map[string]any{"type": "text", "text": "look"},
				map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://example.test/a.png"}},
			}},
			assistantMessageWithTool("partial text", "call_1", "lookup", `{"q":"x"}`),
			{Role: "tool", ToolCallId: "call_1", Content: "tool result"},
		},
	}

	got, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)

	assert.Equal(t, "gpt-test", got.Model)
	assert.Equal(t, `"system rules\n\ndeveloper rules"`, string(got.Instructions))
	assert.Equal(t, "input_image", gjson.GetBytes(got.Input, "0.content.1.type").String())
	assert.Equal(t, "function_call", gjson.GetBytes(got.Input, "2.type").String())
	assert.Equal(t, "call_1", gjson.GetBytes(got.Input, "2.call_id").String())
	assert.Equal(t, "function_call_output", gjson.GetBytes(got.Input, "3.type").String())
}

func TestChatCompletionsRequestToResponsesRequestRejectsMultipleChoices(t *testing.T) {
	_, err := ChatCompletionsRequestToResponsesRequest(&dto.GeneralOpenAIRequest{
		Model: "gpt-test",
		N:     lo.ToPtr(2),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "n>1")
}

func assistantMessageWithTool(content string, id string, name string, args string) dto.Message {
	msg := dto.Message{Role: "assistant", Content: content}
	msg.SetToolCalls([]dto.ToolCallRequest{
		{
			ID:   id,
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      name,
				Arguments: args,
			},
		},
	})
	return msg
}

func TestChatCompletionsRequestToResponsesRejectsMissingToolCallID(t *testing.T) {
	_, err := ChatCompletionsRequestToResponsesRequest(&dto.GeneralOpenAIRequest{
		Model:    "gpt-test",
		Messages: []dto.Message{{Role: "tool", Content: "untrusted tool output"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool_call_id")
}

func TestChatCompletionsRequestToResponsesRejectsMalformedAssistantToolCall(t *testing.T) {
	message := dto.Message{Role: "assistant"}
	message.SetToolCalls([]dto.ToolCallRequest{{
		Type:     "function",
		Function: dto.FunctionRequest{Name: "lookup", Arguments: `{}`},
	}})

	_, err := ChatCompletionsRequestToResponsesRequest(&dto.GeneralOpenAIRequest{
		Model:    "gpt-test",
		Messages: []dto.Message{message},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing id")
}

func TestChatCompletionsRequestToResponsesPreservesNonLeadingInstructions(t *testing.T) {
	got, err := ChatCompletionsRequestToResponsesRequest(&dto.GeneralOpenAIRequest{
		Model: "gpt-test",
		Messages: []dto.Message{
			{Role: "user", Content: "before"},
			{Role: "system", Content: "middle constraint"},
			{Role: "user", Content: "after"},
		},
	})
	require.NoError(t, err)
	assert.Empty(t, got.Instructions)
	assert.Equal(t, "system", gjson.GetBytes(got.Input, "1.role").String())
	assert.Equal(t, "middle constraint", gjson.GetBytes(got.Input, "1.content").String())
}

func TestChatCompletionsRequestToResponsesDoesNotForwardChatStreamOptions(t *testing.T) {
	includeUsage := true
	got, err := ChatCompletionsRequestToResponsesRequest(&dto.GeneralOpenAIRequest{
		Model:         "gpt-test",
		Messages:      []dto.Message{{Role: "user", Content: "hello"}},
		StreamOptions: &dto.StreamOptions{IncludeUsage: includeUsage},
	})
	require.NoError(t, err)
	assert.Nil(t, got.StreamOptions)
}
