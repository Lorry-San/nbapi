package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/constant"
	"github.com/Lorry-San/nbapi/dto"
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChatCompletionsToResponsesStreamHandlerPreservesTextAndToolCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	toolIndex := 0
	finishReason := "tool_calls"
	sse := strings.Join([]string{
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_123",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"role":    "assistant",
					"content": "I will call a tool.",
				},
			}},
		}),
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_123",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"tool_calls": []map[string]any{{
						"index": toolIndex,
						"id":    "call_123",
						"type":  "function",
						"function": map[string]any{
							"name":      "lookup",
							"arguments": `{"query"`,
						},
					}},
				},
			}},
		}),
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_123",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"tool_calls": []map[string]any{{
						"index": toolIndex,
						"function": map[string]any{
							"arguments": `:"nbapi"}`,
						},
					}},
				},
				"finish_reason": finishReason,
			}},
			"usage": map[string]any{
				"prompt_tokens":     1,
				"completion_tokens": 2,
				"total_tokens":      3,
			},
		}),
		"data: [DONE]\n",
	}, "")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(common.RequestIdKey, "test")
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(sse)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "kimi-k2.7"},
	}

	usage, err := ChatCompletionsToResponsesStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 3, usage.TotalTokens)

	events := parseResponsesSSEEvents(t, recorder.Body.String())
	var sawText bool
	var sawToolName bool
	var argsDone string
	var argsDoneArguments string
	var completed *dto.OpenAIResponsesResponse

	for _, event := range events {
		switch event.Type {
		case "response.output_text.delta":
			if event.Delta == "I will call a tool." {
				sawText = true
			}
		case dto.ResponsesOutputTypeItemAdded:
			if event.Item != nil && event.Item.Type == "function_call" && event.Item.Name == "lookup" {
				sawToolName = true
			}
		case "response.function_call_arguments.done":
			argsDone = event.Delta
			argsDoneArguments = event.Arguments
		case "response.completed":
			completed = event.Response
		}
	}

	require.True(t, sawText)
	require.True(t, sawToolName)
	require.JSONEq(t, `{"query":"nbapi"}`, argsDone)
	require.JSONEq(t, `{"query":"nbapi"}`, argsDoneArguments)
	require.NotNil(t, completed)
	require.Equal(t, "completed", rawStatusString(t, completed.Status))
	require.Len(t, completed.Output, 2)
	require.Equal(t, "message", completed.Output[0].Type)
	require.Equal(t, "function_call", completed.Output[1].Type)
	require.JSONEq(t, `{"query":"nbapi"}`, string(completed.Output[1].Arguments))
	body, err := common.Marshal(completed)
	require.NoError(t, err)
	require.Contains(t, string(body), `"arguments":"{\"query\":\"nbapi\"}"`)
}

func TestChatCompletionsToResponsesStreamHandlerHidesReasoningContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	sse := strings.Join([]string{
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_reasoning",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"reasoning_content": "hidden chain of thought",
				},
			}},
		}),
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_reasoning",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"content": "visible answer",
				},
			}},
		}),
		"data: [DONE]\n",
	}, "")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(common.RequestIdKey, "test")
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(sse)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "kimi-k2.7"},
	}

	usage, err := ChatCompletionsToResponsesStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)

	body := recorder.Body.String()
	require.NotContains(t, body, "hidden chain of thought")
	require.Contains(t, body, "visible answer")

	events := parseResponsesSSEEvents(t, body)
	var deltas []string
	var completed *dto.OpenAIResponsesResponse
	for _, event := range events {
		if event.Type == "response.output_text.delta" {
			deltas = append(deltas, event.Delta)
		}
		if event.Type == "response.completed" {
			completed = event.Response
		}
	}

	require.Equal(t, []string{"visible answer"}, deltas)
	require.NotNil(t, completed)
	require.Len(t, completed.Output, 1)
	require.Equal(t, "visible answer", completed.Output[0].Content[0].Text)
}

func TestChatCompletionsToResponsesStreamHandlerHonorsSerialToolCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	sse := strings.Join([]string{
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_serial_tools",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"tool_calls": []map[string]any{{
						"index": 0,
						"id":    "call_first",
						"type":  "function",
						"function": map[string]any{
							"name":      "shell_command",
							"arguments": `{"command":"Get-ComputerInfo"}`,
						},
					}},
				},
			}},
		}),
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_serial_tools",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"tool_calls": []map[string]any{{
						"index": 1,
						"id":    "call_second",
						"type":  "function",
						"function": map[string]any{
							"name":      "shell_command",
							"arguments": `{"command":"Get-CimInstance Win32_Processor"}`,
						},
					}},
				},
			}},
		}),
		"data: [DONE]\n",
	}, "")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(common.RequestIdKey, "test")
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(sse)),
	}
	info := &relaycommon.RelayInfo{
		Request: &dto.OpenAIResponsesRequest{
			ParallelToolCalls: jsonRaw(t, false),
		},
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "kimi-k2.7"},
	}

	usage, err := ChatCompletionsToResponsesStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)

	events := parseResponsesSSEEvents(t, recorder.Body.String())
	var completed *dto.OpenAIResponsesResponse
	var toolCallNames []string
	for _, event := range events {
		if event.Type == dto.ResponsesOutputTypeItemDone && event.Item != nil && event.Item.Type == "function_call" {
			toolCallNames = append(toolCallNames, event.Item.CallId)
		}
		if event.Type == "response.completed" {
			completed = event.Response
		}
	}

	require.Equal(t, []string{"call_first"}, toolCallNames)
	require.NotNil(t, completed)
	require.Len(t, completed.Output, 1)
	require.Equal(t, "call_first", completed.Output[0].CallId)
	require.JSONEq(t, `{"command":"Get-ComputerInfo"}`, string(completed.Output[0].Arguments))
}

func TestChatCompletionsToResponsesStreamHandlerShowsReasoningWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	sse := strings.Join([]string{
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_reasoning_visible",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"reasoning_content": "visible reasoning",
				},
			}},
		}),
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_reasoning_visible",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"content": "visible answer",
				},
			}},
		}),
		"data: [DONE]\n",
	}, "")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(common.RequestIdKey, "test")
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(sse)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.7",
			ChannelSetting:    dto.ChannelSettings{ThinkingToContent: true},
		},
	}

	usage, err := ChatCompletionsToResponsesStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)

	events := parseResponsesSSEEvents(t, recorder.Body.String())
	var deltas []string
	var completed *dto.OpenAIResponsesResponse
	for _, event := range events {
		if event.Type == "response.output_text.delta" {
			deltas = append(deltas, event.Delta)
		}
		if event.Type == "response.completed" {
			completed = event.Response
		}
	}

	require.Equal(t, []string{"<think>\n", "visible reasoning", "\n</think>\n", "visible answer"}, deltas)
	require.NotNil(t, completed)
	require.Len(t, completed.Output, 1)
	require.Equal(t, "<think>\nvisible reasoning\n</think>\nvisible answer", completed.Output[0].Content[0].Text)
}

func TestChatCompletionsToResponsesStreamHandlerHidesThinkTagsByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	sse := strings.Join([]string{
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_think_tags",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"content": "<thi",
				},
			}},
		}),
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_think_tags",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"content": "nk>hidden thought</thi",
				},
			}},
		}),
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_think_tags",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"content": "nk>visible answer",
				},
			}},
		}),
		"data: [DONE]\n",
	}, "")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(common.RequestIdKey, "test")
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(sse)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "kimi-k2.7"},
	}

	usage, err := ChatCompletionsToResponsesStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)

	body := recorder.Body.String()
	require.NotContains(t, body, "hidden thought")
	require.Contains(t, body, "visible answer")

	events := parseResponsesSSEEvents(t, body)
	var deltas []string
	var completed *dto.OpenAIResponsesResponse
	for _, event := range events {
		if event.Type == "response.output_text.delta" {
			deltas = append(deltas, event.Delta)
		}
		if event.Type == "response.completed" {
			completed = event.Response
		}
	}

	require.Equal(t, []string{"visible answer"}, deltas)
	require.NotNil(t, completed)
	require.Len(t, completed.Output, 1)
	require.Equal(t, "visible answer", completed.Output[0].Content[0].Text)
}

func TestChatCompletionsToResponsesStreamHandlerKeepsThinkTagsWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	sse := strings.Join([]string{
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_think_tags_visible",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"content": "<think>visible thought</think>visible answer",
				},
			}},
		}),
		"data: [DONE]\n",
	}, "")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(common.RequestIdKey, "test")
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(sse)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.7",
			ChannelSetting:    dto.ChannelSettings{ThinkingToContent: true},
		},
	}

	usage, err := ChatCompletionsToResponsesStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)

	body := recorder.Body.String()
	events := parseResponsesSSEEvents(t, body)
	var deltas []string
	var completed *dto.OpenAIResponsesResponse
	for _, event := range events {
		if event.Type == "response.output_text.delta" {
			deltas = append(deltas, event.Delta)
		}
		if event.Type == "response.completed" {
			completed = event.Response
		}
	}

	require.Equal(t, []string{"<think>visible thought</think>visible answer"}, deltas)
	require.NotNil(t, completed)
	require.Len(t, completed.Output, 1)
	require.Equal(t, "<think>visible thought</think>visible answer", completed.Output[0].Content[0].Text)
}

func TestChatCompletionsToResponsesStreamHandlerClosesReasoningOnlyOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	sse := strings.Join([]string{
		chatSSE(t, map[string]any{
			"id":      "chatcmpl_reasoning_only",
			"object":  "chat.completion.chunk",
			"created": 123,
			"model":   "kimi-k2.7",
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{
					"reasoning": "visible reasoning",
				},
			}},
		}),
		"data: [DONE]\n",
	}, "")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(common.RequestIdKey, "test")
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(sse)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.7",
			ChannelSetting:    dto.ChannelSettings{ThinkingToContent: true},
		},
	}

	usage, err := ChatCompletionsToResponsesStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)

	events := parseResponsesSSEEvents(t, recorder.Body.String())
	var completed *dto.OpenAIResponsesResponse
	for _, event := range events {
		if event.Type == "response.completed" {
			completed = event.Response
		}
	}

	require.NotNil(t, completed)
	require.Len(t, completed.Output, 1)
	require.Equal(t, "<think>\nvisible reasoning\n</think>\n", completed.Output[0].Content[0].Text)
}

func chatSSE(t *testing.T, event map[string]any) string {
	t.Helper()
	raw, err := common.Marshal(event)
	require.NoError(t, err)
	return "data: " + string(raw) + "\n"
}

func parseResponsesSSEEvents(t *testing.T, body string) []dto.ResponsesStreamResponse {
	t.Helper()

	events := make([]dto.ResponsesStreamResponse, 0)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if data == "" || data == "[DONE]" {
			continue
		}
		var event dto.ResponsesStreamResponse
		require.NoError(t, common.Unmarshal([]byte(data), &event))
		events = append(events, event)
	}
	return events
}

func rawStatusString(t *testing.T, raw []byte) string {
	t.Helper()
	var status string
	require.NoError(t, common.Unmarshal(raw, &status))
	return status
}

func jsonRaw(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := common.Marshal(value)
	require.NoError(t, err)
	return raw
}
