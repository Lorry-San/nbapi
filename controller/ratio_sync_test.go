package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertOpenRouterToRatioDataIncludesCacheWrite(t *testing.T) {
	payload := `{
		"data": [
			{
				"id": "openai/gpt-test",
				"pricing": {
					"prompt": "0.000001",
					"completion": "0.000006",
					"input_cache_read": "0.0000001",
					"input_cache_write": "0.00000125"
				}
			},
			{
				"id": "example/free-model",
				"pricing": {
					"prompt": "0",
					"completion": "0"
				}
			},
			{
				"id": "openrouter/auto",
				"pricing": {
					"prompt": "-1",
					"completion": "-1",
					"input_cache_write": "0.000001"
				}
			}
		]
	}`

	converted, err := convertOpenRouterToRatioData(strings.NewReader(payload))
	require.NoError(t, err)

	modelRatios := converted["model_ratio"].(map[string]any)
	require.Equal(t, 0.5, modelRatios["openai/gpt-test"])
	require.Equal(t, 0.0, modelRatios["example/free-model"])
	require.NotContains(t, modelRatios, "openrouter/auto")

	completionRatios := converted["completion_ratio"].(map[string]any)
	require.Equal(t, 6.0, completionRatios["openai/gpt-test"])

	cacheRatios := converted["cache_ratio"].(map[string]any)
	require.Equal(t, 0.1, cacheRatios["openai/gpt-test"])

	createCacheRatios := converted["create_cache_ratio"].(map[string]any)
	require.Equal(t, 1.25, createCacheRatios["openai/gpt-test"])
	require.NotContains(t, createCacheRatios, "openrouter/auto")
}
