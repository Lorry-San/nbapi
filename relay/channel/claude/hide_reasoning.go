package claude

import (
	"github.com/Lorry-San/nbapi/dto"
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
)

// shouldHideReasoning reports whether this channel withholds the model's reasoning from the
// client. The upstream still reasons and is still billed for it; only the reply is trimmed.
func shouldHideReasoning(info *relaycommon.RelayInfo) bool {
	return info != nil && info.ChannelMeta != nil && info.ChannelOtherSettings.HideReasoningContent
}

// stripOpenAIStreamReasoning removes reasoning from one OpenAI-format stream chunk. It reports
// whether anything remains worth sending: a chunk that carried only reasoning has no role,
// content, tool call, finish reason, or usage left, and forwarding it would emit an empty
// delta per thinking token.
func stripOpenAIStreamReasoning(response *dto.ChatCompletionsStreamResponse) bool {
	if response == nil {
		return false
	}
	for i := range response.Choices {
		response.Choices[i].Delta.ReasoningContent = nil
		response.Choices[i].Delta.Reasoning = nil
	}
	if response.Usage != nil {
		return true
	}
	for i := range response.Choices {
		choice := response.Choices[i]
		if choice.FinishReason != nil || choice.Delta.Role != "" ||
			choice.Delta.GetContentString() != "" || len(choice.Delta.ToolCalls) > 0 {
			return true
		}
	}
	return len(response.Choices) == 0
}

// stripOpenAIResponseReasoning removes reasoning from a non-streaming OpenAI-format reply.
func stripOpenAIResponseReasoning(response *dto.OpenAITextResponse) {
	if response == nil {
		return
	}
	for i := range response.Choices {
		// Message is embedded, so these are the same fields the client reads as message.*.
		response.Choices[i].ReasoningContent = nil
		response.Choices[i].Reasoning = nil
	}
}
