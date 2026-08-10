package oaichat

import (
	"testing"

	"github.com/Lorry-San/nbapi/dto"
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
