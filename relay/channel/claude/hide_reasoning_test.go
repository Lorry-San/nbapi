package claude

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/dto"
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hideReasoningInfo(hide bool) *relaycommon.RelayInfo {
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
	info.ChannelOtherSettings.HideReasoningContent = hide
	return info
}

func streamChunk(t *testing.T, delta dto.ChatCompletionsStreamResponseChoiceDelta, finish *string) *dto.ChatCompletionsStreamResponse {
	t.Helper()
	return &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{Delta: delta, FinishReason: finish}},
	}
}

func TestShouldHideReasoningFollowsChannelSetting(t *testing.T) {
	assert.True(t, shouldHideReasoning(hideReasoningInfo(true)))
	assert.False(t, shouldHideReasoning(hideReasoningInfo(false)))
	assert.False(t, shouldHideReasoning(nil))
	assert.False(t, shouldHideReasoning(&relaycommon.RelayInfo{}), "a request with no channel metadata hides nothing")
}

// A chunk made only of thinking tokens has nothing left to deliver once the thinking is
// removed, so it must be dropped instead of reaching the client as an empty delta.
func TestStripStreamReasoningDropsChunksThatCarriedOnlyThinking(t *testing.T) {
	thinking := "weighing the options"
	chunk := streamChunk(t, dto.ChatCompletionsStreamResponseChoiceDelta{ReasoningContent: &thinking}, nil)

	keep := stripOpenAIStreamReasoning(chunk)

	assert.False(t, keep)
	assert.Nil(t, chunk.Choices[0].Delta.ReasoningContent)
}

func TestStripStreamReasoningKeepsAnswerAndRemovesThinkingFromMixedChunk(t *testing.T) {
	thinking, answer := "private", "visible"
	chunk := streamChunk(t, dto.ChatCompletionsStreamResponseChoiceDelta{ReasoningContent: &thinking, Content: &answer}, nil)

	keep := stripOpenAIStreamReasoning(chunk)

	require.True(t, keep)
	assert.Nil(t, chunk.Choices[0].Delta.ReasoningContent)
	assert.Nil(t, chunk.Choices[0].Delta.Reasoning)
	assert.Equal(t, "visible", chunk.Choices[0].Delta.GetContentString())
	assert.NotContains(t, common.GetJsonString(chunk), "private")
}

func TestStripStreamReasoningKeepsStructuralChunks(t *testing.T) {
	stop := "stop"
	cases := map[string]*dto.ChatCompletionsStreamResponse{
		"role opener":  streamChunk(t, dto.ChatCompletionsStreamResponseChoiceDelta{Role: "assistant"}, nil),
		"finish":       streamChunk(t, dto.ChatCompletionsStreamResponseChoiceDelta{}, &stop),
		"tool call":    streamChunk(t, dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{{ID: "call_1"}}}, nil),
		"usage only":   {Usage: &dto.Usage{TotalTokens: 5}},
		"empty choice": {},
	}
	for name, chunk := range cases {
		assert.True(t, stripOpenAIStreamReasoning(chunk), name)
	}
}

func TestStripStreamReasoningToleratesNil(t *testing.T) {
	assert.False(t, stripOpenAIStreamReasoning(nil))
}

func TestStripResponseReasoningRemovesThinkingFromNonStreamingReply(t *testing.T) {
	thinking := "chain of thought"
	response := &dto.OpenAITextResponse{Choices: []dto.OpenAITextResponseChoice{{Message: dto.Message{Role: "assistant"}}}}
	response.Choices[0].SetStringContent("final answer")
	response.Choices[0].ReasoningContent = &thinking

	stripOpenAIResponseReasoning(response)

	body := common.GetJsonString(response)
	assert.NotContains(t, body, "chain of thought")
	assert.NotContains(t, body, "reasoning_content")
	assert.Contains(t, body, "final answer")
	stripOpenAIResponseReasoning(nil)
}

// Hiding reasoning is a presentation choice. The conversion that feeds billing runs first
// and still sees the thinking, so usage text is unchanged by the setting.
func TestHidingReasoningDoesNotChangeWhatBillingAccumulates(t *testing.T) {
	thinking := "pondering"
	claudeInfo := &ClaudeResponseInfo{Usage: &dto.Usage{}}
	response := &dto.ClaudeResponse{Type: "content_block_delta", Delta: &dto.ClaudeMediaMessage{Type: "thinking_delta", Thinking: &thinking}}
	chunk := StreamResponseClaude2OpenAI(response)

	require.True(t, FormatClaudeResponseInfo(response, chunk, claudeInfo))
	assert.Contains(t, claudeInfo.ResponseText.String(), "pondering", "billing saw the thinking before it was stripped")

	assert.False(t, stripOpenAIStreamReasoning(chunk))
}
