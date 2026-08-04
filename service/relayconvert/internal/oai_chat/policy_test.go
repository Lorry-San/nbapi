package openaicompat

import (
	"testing"

	"github.com/Lorry-San/nbapi/constant"
	"github.com/Lorry-San/nbapi/setting/model_setting"
	"github.com/stretchr/testify/require"
)

func TestShouldChatCompletionsUseResponsesPolicyRequiresSupportedChannel(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		AllChannels:   true,
		ModelPatterns: []string{".*"},
	}

	require.False(t, ShouldChatCompletionsUseResponsesPolicy(
		policy,
		1,
		constant.ChannelTypeAnthropic,
		"claude-3-5-sonnet",
	))
	require.True(t, ShouldChatCompletionsUseResponsesPolicy(
		policy,
		1,
		constant.ChannelTypeCodex,
		"gpt-5-codex",
	))
	require.True(t, ShouldChatCompletionsUseResponsesPolicy(
		policy,
		1,
		constant.ChannelTypeAli,
		"qwen3-coder-plus",
	))
	require.True(t, ShouldChatCompletionsUseResponsesPolicy(
		policy,
		1,
		constant.ChannelCloudflare,
		"@cf/openai/gpt-oss-120b",
	))
	require.True(t, ShouldChatCompletionsUseResponsesPolicy(
		policy,
		1,
		constant.ChannelTypePerplexity,
		"sonar",
	))
	require.True(t, ShouldChatCompletionsUseResponsesPolicy(
		policy,
		1,
		constant.ChannelTypeVolcEngine,
		"doubao-seed-1-6",
	))
	require.True(t, ShouldChatCompletionsUseResponsesPolicy(
		policy,
		1,
		constant.ChannelTypeMoonshot,
		"kimi-k2.7",
	))
}
