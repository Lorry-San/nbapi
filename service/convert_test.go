package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestClaudeToOpenAIRequestPreservesImageURLSource(t *testing.T) {
	claudeReq := dto.ClaudeRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{
						Type: "image",
						Source: &dto.ClaudeMessageSource{
							Type: "url",
							Url:  "https://example.com/image.png",
						},
					},
				},
			},
		},
	}

	openAIReq, err := ClaudeToOpenAIRequest(claudeReq, nil)
	require.NoError(t, err)
	require.Len(t, openAIReq.Messages, 1)

	content := openAIReq.Messages[0].ParseContent()
	require.Len(t, content, 1)
	require.Equal(t, dto.ContentTypeImageURL, content[0].Type)

	image, ok := content[0].ImageUrl.(*dto.MessageImageUrl)
	require.True(t, ok)
	require.Equal(t, "https://example.com/image.png", image.Url)
}

func TestClaudeToOpenAIRequestSkipsImageWithoutSource(t *testing.T) {
	claudeReq := dto.ClaudeRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{
						Type: "image",
					},
				},
			},
		},
	}

	openAIReq, err := ClaudeToOpenAIRequest(claudeReq, nil)
	require.NoError(t, err)
	require.Empty(t, openAIReq.Messages)
}

func TestClaudeToOpenAIRequestConvertsWebSearchToolToOptions(t *testing.T) {
	claudeReq := dto.ClaudeRequest{
		Model: "claude-3-5-sonnet",
		Tools: []any{
			dto.Tool{
				Name:        "lookup",
				Description: "lookup docs",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
			dto.ClaudeWebSearchTool{
				Type:    "web_search_20250305",
				Name:    "web_search",
				MaxUses: 10,
				UserLocation: &dto.ClaudeWebSearchUserLocation{
					Type:     "approximate",
					Timezone: "Asia/Shanghai",
					Country:  "CN",
				},
			},
		},
		Messages: []dto.ClaudeMessage{
			{
				Role:    "user",
				Content: "search this",
			},
		},
	}

	openAIReq, err := ClaudeToOpenAIRequest(claudeReq, nil)
	require.NoError(t, err)
	require.Len(t, openAIReq.Tools, 1)
	require.Equal(t, "lookup", openAIReq.Tools[0].Function.Name)
	require.NotNil(t, openAIReq.WebSearchOptions)
	require.Equal(t, "high", openAIReq.WebSearchOptions.SearchContextSize)

	var location map[string]any
	require.NoError(t, common.Unmarshal(openAIReq.WebSearchOptions.UserLocation, &location))
	require.Equal(t, "approximate", location["type"])

	approximate, ok := location["approximate"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Asia/Shanghai", approximate["timezone"])
	require.Equal(t, "CN", approximate["country"])
}

func TestClaudeToOpenAIRequestPreservesAssistantTextWithToolUse(t *testing.T) {
	claudeReq := dto.ClaudeRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.ClaudeMessage{
			{
				Role: "assistant",
				Content: []dto.ClaudeMediaMessage{
					{
						Type: "text",
						Text: common.GetPointer("I will call a tool."),
					},
					{
						Type:  "tool_use",
						Id:    "toolu_123",
						Name:  "lookup",
						Input: map[string]any{"query": "nbapi"},
					},
				},
			},
		},
	}

	openAIReq, err := ClaudeToOpenAIRequest(claudeReq, nil)
	require.NoError(t, err)
	require.Len(t, openAIReq.Messages, 1)

	assistant := openAIReq.Messages[0]
	require.Equal(t, "assistant", assistant.Role)
	require.Equal(t, "I will call a tool.", assistant.StringContent())

	toolCalls := assistant.ParseToolCalls()
	require.Len(t, toolCalls, 1)
	require.Equal(t, "toolu_123", toolCalls[0].ID)
	require.Equal(t, "lookup", toolCalls[0].Function.Name)
	require.JSONEq(t, `{"query":"nbapi"}`, toolCalls[0].Function.Arguments)
}

func TestClaudeToOpenAIRequestPreservesStringToolResult(t *testing.T) {
	claudeReq := dto.ClaudeRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{
						Type:      "tool_result",
						ToolUseId: "toolu_123",
						Content:   "tool output",
					},
				},
			},
		},
	}

	openAIReq, err := ClaudeToOpenAIRequest(claudeReq, nil)
	require.NoError(t, err)
	require.Len(t, openAIReq.Messages, 1)
	require.Equal(t, "tool", openAIReq.Messages[0].Role)
	require.Equal(t, "toolu_123", openAIReq.Messages[0].ToolCallId)
	require.Equal(t, "tool output", openAIReq.Messages[0].StringContent())
}
