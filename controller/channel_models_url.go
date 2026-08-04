package controller

import (
	"github.com/Lorry-San/nbapi/constant"
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
)

func buildChannelModelsURL(baseURL string, channelType int) string {
	switch channelType {
	case constant.ChannelTypeAli:
		return relaycommon.GetFullRequestURL(baseURL, "/compatible-mode/v1/models", channelType)
	case constant.ChannelTypeZhipu_v4:
		if plan, ok := constant.ChannelSpecialBases[baseURL]; ok && plan.OpenAIBaseURL != "" {
			return relaycommon.GetFullRequestURL(plan.OpenAIBaseURL, "/v1/models", channelType)
		}
		return relaycommon.GetFullRequestURL(baseURL, "/api/paas/v4/models", channelType)
	case constant.ChannelTypeVolcEngine, constant.ChannelTypeMoonshot:
		if plan, ok := constant.ChannelSpecialBases[baseURL]; ok && plan.OpenAIBaseURL != "" {
			return relaycommon.GetFullRequestURL(plan.OpenAIBaseURL, "/v1/models", channelType)
		}
		return relaycommon.GetFullRequestURL(baseURL, "/v1/models", channelType)
	default:
		return relaycommon.GetFullRequestURL(baseURL, "/v1/models", channelType)
	}
}
