package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOaiResponsesToChatStreamHandlerPreservesTextAndToolCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	sse := strings.Join([]string{
		responsesSSE(t, map[string]any{
			"type": "response.created",
			"response": map[string]any{
				"model":      "gpt-5",
				"created_at": 123,
			},
		}),
		responsesSSE(t, map[string]any{
			"type":  "response.output_text.delta",
			"delta": "I will call a tool.",
		}),
		responsesSSE(t, map[string]any{
			"type": "response.output_item.added",
			"item": map[string]any{
				"type":    "function_call",
				"id":      "fc_123",
				"call_id": "call_123",
				"name":    "lookup",
			},
		}),
		responsesSSE(t, map[string]any{
			"type":    "response.function_call_arguments.delta",
			"item_id": "fc_123",
			"delta":   `{"query"`,
		}),
		responsesSSE(t, map[string]any{
			"type":    "response.function_call_arguments.delta",
			"item_id": "fc_123",
			"delta":   `:"nbapi"}`,
		}),
		responsesSSE(t, map[string]any{
			"type": "response.completed",
			"response": map[string]any{
				"model":      "gpt-5",
				"created_at": 123,
				"usage": map[string]any{
					"input_tokens":  1,
					"output_tokens": 2,
					"total_tokens":  3,
				},
			},
		}),
		"data: [DONE]\n",
	}, "")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(common.RequestIdKey, "test")
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(sse)),
	}
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5"},
	}

	usage, err := OaiResponsesToChatStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 3, usage.TotalTokens)

	chunks := parseChatSSEChunks(t, recorder.Body.String())
	var sawText bool
	var sawToolName bool
	var toolArgs strings.Builder
	var finishReason string

	for _, chunk := range chunks {
		require.Len(t, chunk.Choices, 1)
		choice := chunk.Choices[0]
		if choice.Delta.Content != nil && *choice.Delta.Content == "I will call a tool." {
			sawText = true
		}
		for _, toolCall := range choice.Delta.ToolCalls {
			require.Equal(t, "call_123", toolCall.ID)
			if toolCall.Function.Name == "lookup" {
				sawToolName = true
			}
			toolArgs.WriteString(toolCall.Function.Arguments)
		}
		if choice.FinishReason != nil {
			finishReason = *choice.FinishReason
		}
	}

	require.True(t, sawText)
	require.True(t, sawToolName)
	require.JSONEq(t, `{"query":"nbapi"}`, toolArgs.String())
	require.Equal(t, "tool_calls", finishReason)
}

func responsesSSE(t *testing.T, event map[string]any) string {
	t.Helper()
	raw, err := common.Marshal(event)
	require.NoError(t, err)
	return "data: " + string(raw) + "\n"
}

func parseChatSSEChunks(t *testing.T, body string) []dto.ChatCompletionsStreamResponse {
	t.Helper()

	chunks := make([]dto.ChatCompletionsStreamResponse, 0)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if data == "" || data == "[DONE]" {
			continue
		}
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.Unmarshal([]byte(data), &chunk))
		if len(chunk.Choices) == 0 {
			continue
		}
		chunks = append(chunks, chunk)
	}
	return chunks
}
