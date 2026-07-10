package system_setting

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/stretchr/testify/require"
)

func preserveThemeState(t *testing.T) {
	t.Helper()
	frontend := themeSettings.Frontend
	active := common.GetTheme()
	t.Cleanup(func() {
		themeSettings.Frontend = frontend
		common.SetTheme(active)
	})
}

func TestFrontendThemeDefaultsToDefault(t *testing.T) {
	preserveThemeState(t)

	require.Equal(t, "default", themeSettings.Frontend)
	UpdateAndSyncTheme()
	require.Equal(t, "default", common.GetTheme())
}

func TestFrontendThemeKeepsClassicCompatibility(t *testing.T) {
	preserveThemeState(t)

	themeSettings.Frontend = "classic"
	UpdateAndSyncTheme()
	require.Equal(t, "classic", common.GetTheme())
}
