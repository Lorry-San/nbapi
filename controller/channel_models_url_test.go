package controller

import (
	"testing"

	"github.com/Lorry-San/nbapi/constant"
	"github.com/stretchr/testify/require"
)

func TestBuildChannelModelsURLUsesVersionedMoonshotBaseURL(t *testing.T) {
	t.Parallel()

	got := buildChannelModelsURL("https://api.moonshot.cn/v3", constant.ChannelTypeMoonshot)

	require.Equal(t, "https://api.moonshot.cn/v3/models", got)
}

func TestBuildChannelModelsURLKeepsDefaultVersionForPlainBaseURL(t *testing.T) {
	t.Parallel()

	got := buildChannelModelsURL("https://api.moonshot.cn", constant.ChannelTypeMoonshot)

	require.Equal(t, "https://api.moonshot.cn/v1/models", got)
}

func TestBuildChannelModelsURLUsesSpecialPlanVersionPath(t *testing.T) {
	t.Parallel()

	got := buildChannelModelsURL("doubao-coding-plan", constant.ChannelTypeVolcEngine)

	require.Equal(t, "https://ark.cn-beijing.volces.com/api/coding/v3/models", got)
}

func TestBuildChannelModelsURLUsesVersionedOpenRouterBaseURL(t *testing.T) {
	t.Parallel()

	got := buildChannelModelsURL("https://openrouter.ai/api/v1", constant.ChannelTypeOpenRouter)

	require.Equal(t, "https://openrouter.ai/api/v1/models", got)
}
