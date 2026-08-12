package common

import (
	"testing"

	"github.com/Lorry-San/nbapi/dto"
	"github.com/Lorry-San/nbapi/types"
	"github.com/stretchr/testify/require"
)

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoResetResponsesConversionState(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:                         types.RelayFormatOpenAIResponses,
		RequestConversionChain:              []types.RelayFormat{types.RelayFormatOpenAIResponses, types.RelayFormatOpenAI},
		FinalRequestRelayFormat:             types.RelayFormatOpenAI,
		UpstreamResponsesViaChatCompletions: true,
		ResponsesToolMappings: map[string]dto.ResponsesToolMapping{
			"shell_command__run": {
				Kind:      dto.ResponsesToolKindNamespace,
				Name:      "run",
				Namespace: "shell_command",
			},
		},
	}

	info.ResetResponsesConversionState()

	require.Empty(t, info.FinalRequestRelayFormat)
	require.False(t, info.UpstreamResponsesViaChatCompletions)
	require.Nil(t, info.ResponsesToolMappings)
	require.Equal(t, []types.RelayFormat{types.RelayFormatOpenAIResponses}, info.RequestConversionChain)
}
