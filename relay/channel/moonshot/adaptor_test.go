package moonshot

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	channelconstant "github.com/Lorry-San/nbapi/constant"
	"github.com/Lorry-San/nbapi/dto"
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	relayconstant "github.com/Lorry-San/nbapi/relay/constant"
	"github.com/Lorry-San/nbapi/types"
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
		ChannelBaseUrl: "https://api.moonshot.cn/v3",
		ChannelType:    channelconstant.ChannelTypeMoonshot,
		RelayMode:      relayconstant.RelayModeChatCompletions,
	}

	got, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.moonshot.cn/v3/chat/completions", got)
}

func TestGetRequestURLUsesVersionedBaseURLForResponses(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelBaseUrl: "https://api.moonshot.cn/v3",
		ChannelType:    channelconstant.ChannelTypeMoonshot,
		RelayFormat:    types.RelayFormatOpenAIResponses,
		RelayMode:      relayconstant.RelayModeResponses,
		RequestURLPath: "/v1/responses",
	}

	got, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.moonshot.cn/v3/responses", got)
}

func TestGetRequestURLUsesVersionedBaseURLForChatToResponsesConversion(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelBaseUrl: "https://api.moonshot.cn/v3",
		ChannelType:    channelconstant.ChannelTypeMoonshot,
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
		ChannelBaseUrl: "doubao-coding-plan",
		ChannelType:    channelconstant.ChannelTypeMoonshot,
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
				"name":    "shell_command",
				"input":   "whoami",
			},
		}),
		Tools: mustMoonshotRawMessage(t, []map[string]any{
			{
				"type":        "namespace",
				"name":        "shell_command",
				"description": "must be dropped for moonshot chat",
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
				"type": "plugin",
				"name": "browser",
			},
		}),
		ToolChoice: mustMoonshotRawMessage(t, map[string]any{
			"type": "namespace",
			"name": "shell_command",
		}),
		Store:                mustMoonshotRawMessage(t, false),
		Metadata:             mustMoonshotRawMessage(t, map[string]any{"trace": "drop"}),
		PromptCacheKey:       mustMoonshotRawMessage(t, "drop"),
		PromptCacheRetention: mustMoonshotRawMessage(t, "drop"),
		SafetyIdentifier:     mustMoonshotRawMessage(t, "drop"),
		EnableThinking:       mustMoonshotRawMessage(t, false),
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
	require.Equal(t, relayconstant.RelayModeChatCompletions, info.RelayMode)
	require.Equal(t, "/v1/chat/completions", info.RequestURLPath)
	require.Equal(t, types.RelayFormatOpenAI, info.GetFinalRequestRelayFormat())
	require.Len(t, chatReq.Messages, 3)
	require.Equal(t, "system", chatReq.Messages[0].Role)
	require.Equal(t, "system rules", chatReq.Messages[0].StringContent())
	require.Equal(t, "user", chatReq.Messages[1].Role)
	require.Equal(t, "hello", chatReq.Messages[1].StringContent())
	require.Equal(t, "user", chatReq.Messages[2].Role)
	require.Empty(t, chatReq.Messages[2].ParseToolCalls())
	require.Len(t, chatReq.Tools, 1)
	require.Equal(t, "function", chatReq.Tools[0].Type)
	require.Equal(t, "lookup", chatReq.Tools[0].Function.Name)
	require.Nil(t, chatReq.ToolChoice)
	require.Empty(t, chatReq.Store)
	require.Empty(t, chatReq.Metadata)
	require.Empty(t, chatReq.PromptCacheKey)
	require.Empty(t, chatReq.PromptCacheRetention)
	require.Empty(t, chatReq.SafetyIdentifier)
	require.Empty(t, chatReq.EnableThinking)
}

func TestConvertOpenAIResponsesRequestKeepsValidFunctionHistoryOnly(t *testing.T) {
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
	require.Len(t, chatReq.Messages, 3)
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

func mustMoonshotRawMessage(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := common.Marshal(value)
	require.NoError(t, err)
	return raw
}
