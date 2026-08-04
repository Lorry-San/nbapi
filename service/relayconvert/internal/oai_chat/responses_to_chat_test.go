package openaicompat

import (
	"encoding/json"
	"testing"

	"github.com/Lorry-San/nbapi/dto"
	"github.com/stretchr/testify/require"
)

func TestResponsesResponseToChatCompletionsResponsePreservesTextAndToolCalls(t *testing.T) {
	resp := &dto.OpenAIResponsesResponse{
		ID:        "resp_123",
		CreatedAt: 123,
		Model:     "gpt-5",
		Output: []dto.ResponsesOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{
						Type: "output_text",
						Text: "I will call a tool.",
					},
				},
			},
			{
				Type:      "function_call",
				ID:        "fc_123",
				CallId:    "call_123",
				Name:      "lookup",
				Arguments: json.RawMessage(`{"query":"nbapi"}`),
			},
		},
	}

	chatResp, usage, err := ResponsesResponseToChatCompletionsResponse(resp, "chatcmpl_test")
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Len(t, chatResp.Choices, 1)

	choice := chatResp.Choices[0]
	require.Equal(t, "tool_calls", choice.FinishReason)
	require.Equal(t, "I will call a tool.", choice.Message.StringContent())

	toolCalls := choice.Message.ParseToolCalls()
	require.Len(t, toolCalls, 1)
	require.Equal(t, "call_123", toolCalls[0].ID)
	require.Equal(t, "function", toolCalls[0].Type)
	require.Equal(t, "lookup", toolCalls[0].Function.Name)
	require.JSONEq(t, `{"query":"nbapi"}`, toolCalls[0].Function.Arguments)
}
