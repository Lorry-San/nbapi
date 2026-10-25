package oaichat

import (
	"testing"

	"github.com/Lorry-San/nbapi/dto"
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIChatRequestToClaudeMessagesDefaultsInvalidToolSchemaType(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "claude-opus-5",
		Messages: []dto.Message{
			{Role: "user", Content: "hi"},
		},
		Tools: []dto.ToolCallRequest{
			{
				Type: "function",
				Function: dto.FunctionRequest{
					Name: "lookup",
					Parameters: map[string]any{
						"type":       []any{"object"},
						"properties": map[string]any{},
					},
				},
			},
		},
	}

	claudeRequest, err := OpenAIChatRequestToClaudeMessages(nil, request)
	require.NoError(t, err)

	tools, ok := claudeRequest.Tools.([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool, ok := tools[0].(*dto.Tool)
	require.True(t, ok)
	assert.Equal(t, "object", tool.InputSchema["type"])
	assert.Equal(t, map[string]any{}, tool.InputSchema["properties"])
}

func TestOpenAIChatRequestToClaudeMessagesAdaptiveChannelReflectsClientEffort(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model:           "claude-opus-4-7",
		ReasoningEffort: "high",
		Messages:        []dto.Message{{Role: "user", Content: "hi"}},
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
	info.ChannelOtherSettings.ClaudeThinkingMode = dto.ClaudeThinkingModeAdaptive

	claudeRequest, err := OpenAIChatRequestToClaudeMessagesWithInfo(nil, info, request)
	require.NoError(t, err)
	require.NotNil(t, claudeRequest.Thinking)
	require.Equal(t, "adaptive", claudeRequest.Thinking.Type)
	require.JSONEq(t, `{"effort":"high"}`, string(claudeRequest.OutputConfig))
	require.Equal(t, "claude-opus-4-7", claudeRequest.Model)
	require.Nil(t, claudeRequest.Temperature)
}

func TestOpenAIChatRequestToClaudeMessagesDisabledChannelRemovesClientThinking(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model:           "claude-opus-4-7",
		ReasoningEffort: "high",
		Messages:        []dto.Message{{Role: "user", Content: "hi"}},
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
	info.ChannelOtherSettings.ClaudeThinkingMode = dto.ClaudeThinkingModeDisabled

	claudeRequest, err := OpenAIChatRequestToClaudeMessagesWithInfo(nil, info, request)
	require.NoError(t, err)
	require.Nil(t, claudeRequest.Thinking)
	require.Empty(t, claudeRequest.OutputConfig)
}

func TestOpenAIChatRequestToClaudeMessagesAppliesChannelThinkingDisplay(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		display     string
		wantType    string
		wantDisplay string
	}{
		{name: "adaptive with summarized", mode: dto.ClaudeThinkingModeAdaptive, display: dto.ClaudeThinkingDisplaySummarized, wantType: "adaptive", wantDisplay: "summarized"},
		{name: "adaptive with omitted", mode: dto.ClaudeThinkingModeAdaptive, display: dto.ClaudeThinkingDisplayOmitted, wantType: "adaptive", wantDisplay: "omitted"},
		{name: "adaptive keeps model default when unset", mode: dto.ClaudeThinkingModeAdaptive, display: "", wantType: "adaptive", wantDisplay: ""},
		{name: "enabled budget with summarized", mode: dto.ClaudeThinkingModeEnabled, display: dto.ClaudeThinkingDisplaySummarized, wantType: "enabled", wantDisplay: "summarized"},
		{name: "disabled has nothing to display", mode: dto.ClaudeThinkingModeDisabled, display: dto.ClaudeThinkingDisplaySummarized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := dto.GeneralOpenAIRequest{
				Model:           "claude-sonnet-5.5",
				ReasoningEffort: "medium",
				Messages:        []dto.Message{{Role: "user", Content: "hi"}},
			}
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
			info.ChannelOtherSettings.ClaudeThinkingMode = tt.mode
			info.ChannelOtherSettings.ClaudeThinkingDisplay = tt.display

			claudeRequest, err := OpenAIChatRequestToClaudeMessagesWithInfo(nil, info, request)
			require.NoError(t, err)
			if tt.wantType == "" {
				assert.Nil(t, claudeRequest.Thinking)
				return
			}
			require.NotNil(t, claudeRequest.Thinking)
			assert.Equal(t, tt.wantType, claudeRequest.Thinking.Type)
			assert.Equal(t, tt.wantDisplay, claudeRequest.Thinking.Display)
		})
	}
}

func TestOpenAIChatRequestToClaudeMessagesPreservesValidToolSchemaType(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "claude-opus-5",
		Messages: []dto.Message{
			{Role: "user", Content: "hi"},
		},
		Tools: []dto.ToolCallRequest{
			{
				Type: "function",
				Function: dto.FunctionRequest{
					Name: "lookup",
					Parameters: map[string]any{
						"type":       "object",
						"properties": map[string]any{},
					},
				},
			},
		},
	}

	claudeRequest, err := OpenAIChatRequestToClaudeMessages(nil, request)
	require.NoError(t, err)

	tools, ok := claudeRequest.Tools.([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool, ok := tools[0].(*dto.Tool)
	require.True(t, ok)
	assert.Equal(t, "object", tool.InputSchema["type"])
}
