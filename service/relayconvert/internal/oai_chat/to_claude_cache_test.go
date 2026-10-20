package oaichat

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCacheTestRequest(model string) dto.GeneralOpenAIRequest {
	return dto.GeneralOpenAIRequest{
		Model: model,
		Messages: []dto.Message{
			{Role: "system", Content: "you are Lightcode"},
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "reply"},
			{Role: "user", Content: "second"},
		},
		Tools: []dto.ToolCallRequest{
			{Type: "function", Function: dto.FunctionRequest{Name: "read_file", Parameters: map[string]any{"type": "object"}}},
			{Type: "function", Function: dto.FunctionRequest{Name: "write_file", Parameters: map[string]any{"type": "object"}}},
		},
	}
}

func convertForCacheTest(t *testing.T, request dto.GeneralOpenAIRequest) *dto.ClaudeRequest {
	t.Helper()
	converted, err := OpenAIChatRequestToClaudeMessages(nil, request)
	require.NoError(t, err)
	return converted
}

// Claude requests must carry cache_control so a multi-turn transcript is served from the
// upstream prompt cache instead of repaying every token of input.
func TestOpenAIChatToClaudeMarksCacheBreakpointsForClaudeModels(t *testing.T) {
	converted := convertForCacheTest(t, newCacheTestRequest("claude-sonnet-5.5"))

	system, ok := converted.System.([]dto.ClaudeMediaMessage)
	require.True(t, ok, "system prompt converts to blocks")
	require.NotEmpty(t, system)
	assert.JSONEq(t, `{"type":"ephemeral"}`, string(system[len(system)-1].CacheControl), "the system prefix anchors the cache")

	tools, ok := converted.Tools.([]any)
	require.True(t, ok)
	require.Len(t, tools, 2)
	first, ok := tools[0].(*dto.Tool)
	require.True(t, ok)
	last, ok := tools[1].(*dto.Tool)
	require.True(t, ok)
	assert.Empty(t, first.CacheControl, "earlier tools stay unmarked; Anthropic caches through the last marked one")
	assert.JSONEq(t, `{"type":"ephemeral"}`, string(last.CacheControl))

	tail := converted.Messages[len(converted.Messages)-1]
	blocks, ok := tail.Content.([]dto.ClaudeMediaMessage)
	require.True(t, ok, "a plain-text tail becomes a block so it can carry the marker")
	require.NotEmpty(t, blocks)
	assert.JSONEq(t, `{"type":"ephemeral"}`, string(blocks[len(blocks)-1].CacheControl))
}

// Other model families keep their own cache conventions; adding Anthropic markers to a
// non-Claude request would be rejected or ignored upstream.
func TestOpenAIChatToClaudeLeavesNonClaudeModelsUnmarked(t *testing.T) {
	converted := convertForCacheTest(t, newCacheTestRequest("gpt-6.1-sol"))

	system, ok := converted.System.([]dto.ClaudeMediaMessage)
	require.True(t, ok)
	for i := range system {
		assert.Empty(t, system[i].CacheControl)
	}
	tools, ok := converted.Tools.([]any)
	require.True(t, ok)
	for _, item := range tools {
		tool, ok := item.(*dto.Tool)
		require.True(t, ok)
		assert.Empty(t, tool.CacheControl)
	}
	for _, message := range converted.Messages {
		assert.False(t, claudeMessageHasCacheControl(message))
	}
}

// A caller that already lays out its own breakpoints stays authoritative.
func TestOpenAIChatToClaudeKeepsCallerBreakpoints(t *testing.T) {
	// Content arrives as decoded JSON, which is how a request reaches the converter.
	raw := []byte(`{
		"model": "claude-opus-5",
		"messages": [
			{"role": "system", "content": "you are Lightcode"},
			{"role": "user", "content": [
				{"type": "text", "text": "cached", "cache_control": {"type": "ephemeral"}}
			]},
			{"role": "assistant", "content": "reply"},
			{"role": "user", "content": "second"}
		]
	}`)
	var request dto.GeneralOpenAIRequest
	require.NoError(t, common.Unmarshal(raw, &request))

	converted := convertForCacheTest(t, request)

	system, ok := converted.System.([]dto.ClaudeMediaMessage)
	require.True(t, ok)
	assert.Empty(t, system[len(system)-1].CacheControl, "the gateway does not add a second layout")

	var marked int
	for _, message := range converted.Messages {
		if claudeMessageHasCacheControl(message) {
			marked++
		}
	}
	assert.Equal(t, 1, marked, "only the caller's own marker survives")
}

// Marking must not change request content between identical turns: the upstream hashes the
// serialized prefix, so instability would defeat the cache it is meant to enable.
func TestOpenAIChatToClaudeCacheMarkersAreStable(t *testing.T) {
	request := newCacheTestRequest("claude-sonnet-5")

	first := convertForCacheTest(t, request)
	second := convertForCacheTest(t, request)

	assert.Equal(t, common.GetJsonString(first), common.GetJsonString(second),
		"the same input converts to byte-identical JSON")
	assert.NotContains(t, common.GetJsonString(first), "ephemeral_", "no request-varying value enters the marker")
}

func TestApplyClaudeCacheControlHandlesRequestsWithoutPrefix(t *testing.T) {
	request := &dto.ClaudeRequest{Model: "claude-sonnet-5"}
	ApplyClaudeCacheControl(request)
	assert.Empty(t, request.Messages, "an empty request gains nothing")

	ApplyClaudeCacheControl(nil)
}
