package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCacheRatioClaudeOpus5FallbackAndAliases(t *testing.T) {
	savedCacheRatios := CacheRatio2JSONString()
	savedCreateCacheRatios := CreateCacheRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateCacheRatioByJSONString(savedCacheRatios))
		require.NoError(t, UpdateCreateCacheRatioByJSONString(savedCreateCacheRatios))
	})

	require.NoError(t, UpdateCacheRatioByJSONString(`{}`))
	require.NoError(t, UpdateCreateCacheRatioByJSONString(`{}`))

	for _, model := range []string{
		"claude-opus-5",
		"anthropic/claude-opus-5",
		"claude-opus-5-thinking",
	} {
		ratio, ok := GetCacheRatio(model)
		require.True(t, ok, model)
		assert.InDelta(t, 0.1, ratio, 1e-9, model)
	}

	ratio, ok := GetCacheRatio("unconfigured-model")
	assert.False(t, ok)
	assert.Equal(t, 1.0, ratio)

	createRatio, ok := GetCreateCacheRatio("claude-opus-5")
	require.True(t, ok)
	assert.InDelta(t, 1.25, createRatio, 1e-9)
}

func TestClaudeOpus5AliasesHonorRuntimeCacheRatioOverrides(t *testing.T) {
	savedCacheRatios := CacheRatio2JSONString()
	savedCreateCacheRatios := CreateCacheRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateCacheRatioByJSONString(savedCacheRatios))
		require.NoError(t, UpdateCreateCacheRatioByJSONString(savedCreateCacheRatios))
	})

	require.NoError(t, UpdateCacheRatioByJSONString(`{
		"claude-opus-5": 0.2,
		"claude-opus-5-thinking": 0.3
	}`))
	require.NoError(t, UpdateCreateCacheRatioByJSONString(`{"claude-opus-5": 1.5}`))

	ratio, ok := GetCacheRatio("anthropic/claude-opus-5")
	require.True(t, ok)
	assert.InDelta(t, 0.2, ratio, 1e-9)

	ratio, ok = GetCacheRatio("anthropic/claude-opus-5-thinking")
	require.True(t, ok)
	assert.InDelta(t, 0.3, ratio, 1e-9)

	ratio, ok = GetCacheRatio("claude-opus-5-max")
	require.True(t, ok)
	assert.InDelta(t, 0.2, ratio, 1e-9)

	createRatio, ok := GetCreateCacheRatio("anthropic/claude-opus-5-thinking")
	require.True(t, ok)
	assert.InDelta(t, 1.5, createRatio, 1e-9)
}
