package moonshot

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	channelconstant "github.com/Lorry-San/nbapi/constant"
	"github.com/Lorry-San/nbapi/dto"
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	relayconstant "github.com/Lorry-San/nbapi/relay/constant"
	"github.com/Lorry-San/nbapi/types"
	"github.com/moonshotai/walle"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIRequestKimiK26UsesOnlyAllowedTemperature(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:       "kimi-k2.6",
		Temperature: common.GetPointer[float64](0.7),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, convertedRequest.Temperature)
	require.Equal(t, 1.0, *convertedRequest.Temperature)
}

func TestConvertOpenAIRequestKimiK26KeepsOmittedTemperatureOmitted(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model: "kimi-k2.6",
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Nil(t, convertedRequest.Temperature)
}

func TestConvertOpenAIRequestOtherMoonshotModelKeepsTemperature(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:       "kimi-k2.5",
		Temperature: common.GetPointer[float64](0.7),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.5",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, convertedRequest.Temperature)
	require.Equal(t, 0.7, *convertedRequest.Temperature)
}

func TestGetRequestURLUsesVersionedBaseURL(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.moonshot.cn/v3",
			ChannelType:    channelconstant.ChannelTypeMoonshot,
		},
		RelayMode: relayconstant.RelayModeChatCompletions,
	}

	got, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.moonshot.cn/v3/chat/completions", got)
}

func TestGetRequestURLUsesVersionedBaseURLForResponses(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.moonshot.cn/v3",
			ChannelType:    channelconstant.ChannelTypeMoonshot,
		},
		RelayFormat:    types.RelayFormatOpenAIResponses,
		RelayMode:      relayconstant.RelayModeResponses,
		RequestURLPath: "/v1/responses",
	}

	got, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.moonshot.cn/v3/responses", got)
}

func TestGetRequestURLUsesVersionedChatURLForConvertedResponses(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.moonshot.cn/v3",
			ChannelType:    channelconstant.ChannelTypeMoonshot,
		},
		RelayFormat:                         types.RelayFormatOpenAIResponses,
		RelayMode:                           relayconstant.RelayModeResponses,
		RequestURLPath:                      "/v1/responses",
		UpstreamResponsesViaChatCompletions: true,
	}

	got, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.moonshot.cn/v3/chat/completions", got)
	require.Equal(t, relayconstant.RelayModeResponses, info.RelayMode)
}

func TestGetRequestURLUsesVersionedBaseURLForChatToResponsesConversion(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.moonshot.cn/v3",
			ChannelType:    channelconstant.ChannelTypeMoonshot,
		},
		RelayFormat:    types.RelayFormatOpenAI,
		RelayMode:      relayconstant.RelayModeResponses,
		RequestURLPath: "/v1/responses",
	}

	got, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.moonshot.cn/v3/responses", got)
}

func TestGetRequestURLUsesVersionedSpecialPlanForResponses(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "doubao-coding-plan",
			ChannelType:    channelconstant.ChannelTypeMoonshot,
		},
		RelayFormat:    types.RelayFormatOpenAIResponses,
		RelayMode:      relayconstant.RelayModeResponses,
		RequestURLPath: "/v1/responses",
	}

	got, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://ark.cn-beijing.volces.com/api/coding/v3/responses", got)
}

func TestConvertOpenAIResponsesRequestKimiK26UsesOnlyAllowedTemperature(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model:       "kimi-k2.6",
		Temperature: common.GetPointer[float64](0.7),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, convertedRequest.Temperature)
	require.Equal(t, 1.0, *convertedRequest.Temperature)
}

func TestConvertOpenAIResponsesRequestUsesMoonshotChatConversion(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model:        "kimi-k2.7",
		Instructions: mustMoonshotRawMessage(t, "system rules"),
		Input: mustMoonshotRawMessage(t, []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": "hello"},
				},
			},
			{
				"type":    "custom_tool_call",
				"call_id": "custom_1",
				"name":    "apply_patch",
				"input":   "whoami",
			},
			{
				"type":    "custom_tool_call_output",
				"call_id": "custom_1",
				"output":  "done",
			},
		}),
		Tools: mustMoonshotRawMessage(t, []map[string]any{
			{
				"type": "namespace",
				"name": "shell_command",
				"tools": []map[string]any{
					{
						"type":        "function",
						"name":        "run",
						"description": "Run a command",
						"parameters": map[string]any{
							"type": "object",
						},
					},
				},
			},
			{
				"type":        "function",
				"name":        "lookup",
				"description": "Lookup data",
				"parameters": map[string]any{
					"type": "object",
				},
			},
			{
				"type":        "custom",
				"name":        "apply_patch",
				"description": "Apply a patch",
			},
			{
				"type": "tool_search",
			},
		}),
		ToolChoice: mustMoonshotRawMessage(t, map[string]any{
			"type":      "function",
			"namespace": "shell_command",
			"name":      "run",
		}),
		Store:                mustMoonshotRawMessage(t, false),
		Metadata:             mustMoonshotRawMessage(t, map[string]any{"trace": "drop"}),
		PromptCacheKey:       mustMoonshotRawMessage(t, "drop"),
		PromptCacheRetention: mustMoonshotRawMessage(t, "drop"),
		SafetyIdentifier:     mustMoonshotRawMessage(t, "drop"),
		EnableThinking:       mustMoonshotRawMessage(t, false),
	}
	info := &relaycommon.RelayInfo{
		RelayFormat:    types.RelayFormatOpenAIResponses,
		RelayMode:      relayconstant.RelayModeResponses,
		RequestURLPath: "/v1/responses",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.7",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	chatReq, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, relayconstant.RelayModeResponses, info.RelayMode)
	require.Equal(t, "/v1/responses", info.RequestURLPath)
	require.True(t, info.UpstreamResponsesViaChatCompletions)
	require.Equal(t, types.RelayFormatOpenAI, info.GetFinalRequestRelayFormat())
	require.Len(t, chatReq.Messages, 4)
	require.Equal(t, "system", chatReq.Messages[0].Role)
	require.Equal(t, "system rules", chatReq.Messages[0].StringContent())
	require.Equal(t, "user", chatReq.Messages[1].Role)
	require.Equal(t, "hello", chatReq.Messages[1].StringContent())
	require.Equal(t, "assistant", chatReq.Messages[2].Role)
	historyCalls := chatReq.Messages[2].ParseToolCalls()
	require.Len(t, historyCalls, 1)
	require.Equal(t, "apply_patch", historyCalls[0].Function.Name)
	require.JSONEq(t, `{"input":"whoami"}`, historyCalls[0].Function.Arguments)
	require.Equal(t, "tool", chatReq.Messages[3].Role)
	require.Equal(t, "custom_1", chatReq.Messages[3].ToolCallId)
	require.Len(t, chatReq.Tools, 4)
	require.Equal(t, "function", chatReq.Tools[0].Type)
	require.Equal(t, "shell_command__run", chatReq.Tools[0].Function.Name)
	require.Equal(t, "lookup", chatReq.Tools[1].Function.Name)
	require.Equal(t, "apply_patch", chatReq.Tools[2].Function.Name)
	require.Equal(t, "tool_search", chatReq.Tools[3].Function.Name)
	require.Equal(t, map[string]any{
		"type": "function",
		"function": map[string]any{
			"name": "shell_command__run",
		},
	}, chatReq.ToolChoice)
	require.Equal(t, dto.ResponsesToolMapping{
		Kind:      dto.ResponsesToolKindNamespace,
		Name:      "run",
		Namespace: "shell_command",
	}, info.ResponsesToolMappings["shell_command__run"])
	require.Equal(t, dto.ResponsesToolKindCustom, info.ResponsesToolMappings["apply_patch"].Kind)
	require.Equal(t, dto.ResponsesToolKindToolSearch, info.ResponsesToolMappings["tool_search"].Kind)
	require.Empty(t, chatReq.Store)
	require.Empty(t, chatReq.Metadata)
	require.Empty(t, chatReq.PromptCacheKey)
	require.Empty(t, chatReq.PromptCacheRetention)
	require.Empty(t, chatReq.SafetyIdentifier)
	require.Empty(t, chatReq.EnableThinking)
}

func TestConvertOpenAIResponsesRequestPreservesFunctionHistoryWithSafeNames(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "kimi-k2.7",
		Input: mustMoonshotRawMessage(t, []map[string]any{
			{
				"type":      "function_call",
				"call_id":   "call_1",
				"name":      "lookup",
				"arguments": map[string]any{"q": "x"},
			},
			{
				"type":    "function_call_output",
				"call_id": "call_1",
				"output":  map[string]any{"ok": true},
			},
			{
				"type":    "function_call_output",
				"call_id": "missing_call",
				"output":  "orphan",
			},
			{
				"type":      "function_call",
				"call_id":   "call_bad",
				"name":      "bad.namespace",
				"arguments": "{}",
			},
		}),
	}
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAIResponses,
		RelayMode:   relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.7",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	chatReq, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, chatReq.Messages, 4)
	require.Equal(t, "assistant", chatReq.Messages[0].Role)
	toolCalls := chatReq.Messages[0].ParseToolCalls()
	require.Len(t, toolCalls, 1)
	require.Equal(t, "function", toolCalls[0].Type)
	require.Equal(t, "call_1", toolCalls[0].ID)
	require.Equal(t, "lookup", toolCalls[0].Function.Name)
	require.Equal(t, "tool", chatReq.Messages[1].Role)
	require.Equal(t, "call_1", chatReq.Messages[1].ToolCallId)
	require.Equal(t, "user", chatReq.Messages[2].Role)
	require.Empty(t, chatReq.Messages[2].ToolCallId)
	require.Equal(t, "assistant", chatReq.Messages[3].Role)
	invalidNameCalls := chatReq.Messages[3].ParseToolCalls()
	require.Len(t, invalidNameCalls, 1)
	require.Equal(t, "call_bad", invalidNameCalls[0].ID)
	require.NotEqual(t, "bad.namespace", invalidNameCalls[0].Function.Name)
	require.True(t, moonshotValidChatFunctionName(invalidNameCalls[0].Function.Name))
	require.Equal(t, "bad.namespace", info.ResponsesToolMappings[invalidNameCalls[0].Function.Name].Name)
}

func TestConvertOpenAIResponsesRequestKeepsToolSearchCatalogInHistory(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "kimi-k2.7",
		Tools: mustMoonshotRawMessage(t, []map[string]any{
			{"type": "tool_search"},
		}),
		Input: mustMoonshotRawMessage(t, []map[string]any{
			{
				"type":      "tool_search_call",
				"call_id":   "search_1",
				"arguments": map[string]any{"query": "load shell tools"},
			},
			{
				"type":    "tool_search_output",
				"call_id": "search_1",
				"tools": []map[string]any{
					{
						"type": "namespace",
						"name": "shell_command",
						"tools": []map[string]any{
							{
								"type":       "function",
								"name":       "run",
								"parameters": map[string]any{"type": "object"},
							},
						},
					},
				},
			},
			{"type": "message", "role": "user", "content": "run pwd"},
		}),
	}
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAIResponses,
		RelayMode:   relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "kimi-k2.7"},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	chatReq, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, chatReq.Tools, 2)
	require.Equal(t, "tool_search", chatReq.Tools[0].Function.Name)
	require.Equal(t, "shell_command__run", chatReq.Tools[1].Function.Name)
	require.Len(t, chatReq.Messages, 3)
	require.Equal(t, "tool", chatReq.Messages[1].Role)
	require.Contains(t, chatReq.Messages[1].StringContent(), "shell_command")
	require.Equal(t, dto.ResponsesToolMapping{
		Kind:      dto.ResponsesToolKindNamespace,
		Name:      "run",
		Namespace: "shell_command",
	}, info.ResponsesToolMappings["shell_command__run"])
}

func TestConvertOpenAIResponsesRequestSeparatesSameNameToolKinds(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "kimi-k2.7",
		Tools: mustMoonshotRawMessage(t, []map[string]any{
			{
				"type":       "function",
				"name":       "apply_patch",
				"parameters": map[string]any{"type": "object"},
			},
			{
				"type": "custom",
				"name": "apply_patch",
			},
		}),
		ToolChoice: mustMoonshotRawMessage(t, map[string]any{
			"type": "custom",
			"name": "apply_patch",
		}),
	}
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAIResponses,
		RelayMode:   relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "kimi-k2.7"},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	chatReq, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, chatReq.Tools, 2)
	functionName := chatReq.Tools[0].Function.Name
	customName := chatReq.Tools[1].Function.Name
	require.Equal(t, "apply_patch", functionName)
	require.NotEqual(t, functionName, customName)
	require.True(t, moonshotValidChatFunctionName(customName))
	require.Equal(t, dto.ResponsesToolKindFunction, info.ResponsesToolMappings[functionName].Kind)
	require.Equal(t, dto.ResponsesToolKindCustom, info.ResponsesToolMappings[customName].Kind)
	require.Equal(t, map[string]any{
		"type": "function",
		"function": map[string]any{
			"name": customName,
		},
	}, chatReq.ToolChoice)
}

func TestConvertOpenAIResponsesRequestKeepsInternalChatToResponsesPathNative(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "kimi-k2.7",
		Input: mustMoonshotRawMessage(t, "hello"),
	}
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAI,
		RelayMode:   relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.7",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	_, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.Equal(t, relayconstant.RelayModeResponses, info.RelayMode)
}

func TestConvertOpenAIResponsesRequestSkipsEmptyDeveloperSystemMessage(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "kimi-k3",
		Input: mustMoonshotRawMessage(t, []map[string]any{
			{
				"role":    "developer",
				"content": "",
			},
			{
				"role":    "user",
				"content": "hello",
			},
		}),
	}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k3",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	for _, message := range chatRequest.Messages {
		require.NotEqual(t, "system", message.Role, "developer 空消息不应转换为 system 消息")
	}
	require.Len(t, chatRequest.Messages, 1)
	require.Equal(t, "user", chatRequest.Messages[0].Role)
}

func TestConvertOpenAIResponsesRequestDropsReasoningItems(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "kimi-k3",
		Input: mustMoonshotRawMessage(t, []map[string]any{
			{
				"role":    "user",
				"content": "hello",
			},
			{
				"type":    "reasoning",
				"summary": []map[string]any{},
			},
			{
				"role":    "assistant",
				"content": "answer",
			},
		}),
	}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k3",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, chatRequest.Messages, 2)
	for _, message := range chatRequest.Messages {
		require.NotContains(t, message.StringContent(), "reasoning")
	}
}

func TestConvertOpenAIResponsesRequestKeepsAssistantTextAndToolCallsInOneMessage(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "kimi-k3",
		Input: mustMoonshotRawMessage(t, []map[string]any{
			{
				"type":    "message",
				"role":    "assistant",
				"content": "Check the handoff notes, then run the tests.",
			},
			{
				"type":    "reasoning",
				"summary": []map[string]any{},
			},
			{
				"type":      "function_call",
				"call_id":   "call_1",
				"name":      "shell_command",
				"arguments": map[string]any{"command": "go version"},
			},
			{
				"type":    "reasoning",
				"summary": []map[string]any{},
			},
			{
				"type":      "function_call",
				"call_id":   "call_2",
				"name":      "shell_command",
				"arguments": map[string]any{"command": "go test ./relay/channel/moonshot"},
			},
			{
				"type":    "function_call_output",
				"call_id": "call_1",
				"output":  "go version go1.24.0",
			},
			{
				"type":    "function_call_output",
				"call_id": "call_2",
				"output":  "ok",
			},
		}),
	}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k3",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, chatRequest.Messages, 3)
	require.Equal(t, "assistant", chatRequest.Messages[0].Role)
	require.Equal(t, "Check the handoff notes, then run the tests.", chatRequest.Messages[0].StringContent())
	toolCalls := chatRequest.Messages[0].ParseToolCalls()
	require.Len(t, toolCalls, 2)
	require.Equal(t, "call_1", toolCalls[0].ID)
	require.Equal(t, "shell_command", toolCalls[0].Function.Name)
	require.JSONEq(t, `{"command":"go version"}`, toolCalls[0].Function.Arguments)
	require.Equal(t, "call_2", toolCalls[1].ID)
	require.Equal(t, "shell_command", toolCalls[1].Function.Name)
	require.JSONEq(t, `{"command":"go test ./relay/channel/moonshot"}`, toolCalls[1].Function.Arguments)
	require.Equal(t, "tool", chatRequest.Messages[1].Role)
	require.Equal(t, "call_1", chatRequest.Messages[1].ToolCallId)
	require.Equal(t, "go version go1.24.0", chatRequest.Messages[1].StringContent())
	require.Equal(t, "tool", chatRequest.Messages[2].Role)
	require.Equal(t, "call_2", chatRequest.Messages[2].ToolCallId)
	require.Equal(t, "ok", chatRequest.Messages[2].StringContent())
	for i := 1; i < len(chatRequest.Messages); i++ {
		require.False(t, chatRequest.Messages[i-1].Role == "assistant" && chatRequest.Messages[i].Role == "assistant")
	}
}

func TestConvertOpenAIResponsesRequestForcesStreamUsage(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model:  "kimi-k3",
		Input:  mustMoonshotRawMessage(t, "hello"),
		Stream: common.GetPointer(true),
	}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
		IsStream:    true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName:    "kimi-k3",
			SupportStreamOptions: true,
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, chatRequest.StreamOptions)
	require.True(t, chatRequest.StreamOptions.IncludeUsage)
}

func TestConvertOpenAIResponsesRequestForcesStreamUsagePreservesOptions(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model:  "kimi-k3",
		Input:  mustMoonshotRawMessage(t, "hello"),
		Stream: common.GetPointer(true),
		StreamOptions: &dto.StreamOptions{
			IncludeObfuscation: true,
		},
	}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
		IsStream:    true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName:    "kimi-k3",
			SupportStreamOptions: true,
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, chatRequest.StreamOptions)
	require.True(t, chatRequest.StreamOptions.IncludeUsage)
	require.True(t, chatRequest.StreamOptions.IncludeObfuscation)
}

func TestMoonshotNormalizeFunctionParametersCanonicalizesCodexSchema(t *testing.T) {
	parameters := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"mode": map[string]any{
				"$ref": "#/$defs/__schema20",
			},
		},
		"$defs": map[string]any{
			"__schema20": map[string]any{
				"type":        "string",
				"minLength":   1,
				"format":      "uuid",
				"description": "Target thread UUID for heartbeat automations.",
				"$ref":        "#/$defs/__schema2",
			},
			"__schema2": map[string]any{
				"type":      "string",
				"minLength": 1,
			},
		},
	}

	normalized := moonshotNormalizeFunctionParameters(parameters)
	raw, err := common.Marshal(normalized)
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"$schema"`)
	normalizedMap, ok := normalized.(map[string]any)
	require.True(t, ok)
	defs, ok := normalizedMap["$defs"].(map[string]any)
	require.True(t, ok)
	schema20, ok := defs["__schema20"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, schema20, "$ref")
	require.Equal(t, "string", schema20["type"])
	require.EqualValues(t, 1, schema20["minLength"])
	require.NotContains(t, schema20, "format")
	require.Equal(t, "Target thread UUID for heartbeat automations.", schema20["description"])

	schema, err := walle.ParseSchema(string(raw))
	require.NoError(t, err)
	require.NoError(t, schema.Validate(walle.WithValidateLevel(walle.ValidateLevelUltra)))
}

func TestConvertOpenAIResponsesRequestCanonicalizesCodexToolSchema(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "kimi-k3",
		Input: mustMoonshotRawMessage(t, "hello"),
		Tools: mustMoonshotRawMessage(t, []map[string]any{
			{
				"type": "function",
				"name": "automation_update",
				"parameters": map[string]any{
					"$schema": "https://json-schema.org/draft/2020-12/schema",
					"type":    "object",
					"properties": map[string]any{
						"mode": map[string]any{
							"$ref": "#/$defs/__schema20",
						},
					},
					"$defs": map[string]any{
						"__schema20": map[string]any{
							"type":        "string",
							"minLength":   1,
							"format":      "uuid",
							"description": "Target thread UUID for heartbeat automations.",
							"$ref":        "#/$defs/__schema2",
						},
						"__schema2": map[string]any{
							"type":      "string",
							"minLength": 1,
						},
					},
				},
			},
		}),
	}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "kimi-k3"},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, chatRequest.Tools, 1)

	raw, err := common.Marshal(chatRequest.Tools[0].Function.Parameters)
	require.NoError(t, err)
	schema, err := walle.ParseSchema(string(raw))
	require.NoError(t, err)
	require.NoError(t, schema.Validate(walle.WithValidateLevel(walle.ValidateLevelUltra)))
}

func mustMoonshotRawMessage(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := common.Marshal(value)
	require.NoError(t, err)
	return raw
}
