package moonshot

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIResponsesRequestFallsBackToChatCompletions(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RelayMode:      relayconstant.RelayModeResponses,
		RelayFormat:    types.RelayFormatOpenAIResponses,
		RequestURLPath: "/v1/responses",
	}

	adaptor := &Adaptor{}
	maxOutputTokens := uint(256)
	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model:           "kimi-k2.7",
		Input:           json.RawMessage(`"hello"`),
		Instructions:    json.RawMessage(`"answer briefly"`),
		MaxOutputTokens: &maxOutputTokens,
	})

	require.NoError(t, err)
	chatReq, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "kimi-k2.7", chatReq.Model)
	require.Len(t, chatReq.Messages, 2)
	require.Equal(t, "system", chatReq.Messages[0].Role)
	require.Equal(t, "user", chatReq.Messages[1].Role)
	require.Equal(t, "hello", chatReq.Messages[1].StringContent())
	require.Equal(t, maxOutputTokens, *chatReq.MaxTokens)
	require.Nil(t, chatReq.MaxCompletionTokens)

	require.True(t, info.UpstreamResponsesViaChatCompletions)
	require.Equal(t, relayconstant.RelayModeChatCompletions, info.RelayMode)
	require.Equal(t, "/v1/chat/completions", info.RequestURLPath)
	require.Equal(t, types.RelayFormatOpenAI, info.FinalRequestRelayFormat)
}

func TestGetRequestURLForSpecialBaseUsesChatCompletionsWhenResponsesFallback(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RelayMode:                            relayconstant.RelayModeChatCompletions,
		RelayFormat:                          types.RelayFormatOpenAIResponses,
		ChannelBaseUrl:                       "kimi-coding-plan",
		UpstreamResponsesViaChatCompletions: true,
	}

	url, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://api.kimi.com/coding/v1/chat/completions", url)
}
