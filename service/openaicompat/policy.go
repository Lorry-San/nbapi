package openaicompat

import (
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	if !isChatCompletionsToResponsesSupportedChannel(channelType) {
		return false
	}
	if !policy.IsChannelEnabled(channelID, channelType) {
		return false
	}
	return matchAnyRegex(policy.ModelPatterns, model)
}

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	return ShouldChatCompletionsUseResponsesPolicy(
		model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy,
		channelID,
		channelType,
		model,
	)
}

func isChatCompletionsToResponsesSupportedChannel(channelType int) bool {
	switch channelType {
	case constant.ChannelTypeOpenAI,
		constant.ChannelTypeAzure,
		constant.ChannelTypeAli,
		constant.ChannelTypePerplexity,
		constant.ChannelCloudflare,
		constant.ChannelTypeVolcEngine,
		constant.ChannelTypeXai,
		constant.ChannelTypeCodex,
		constant.ChannelTypeMoonshot:
		return true
	default:
		return false
	}
}
