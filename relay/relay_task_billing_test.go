package relay

import (
	"math"
	"testing"

	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	"github.com/Lorry-San/nbapi/types"
	"github.com/stretchr/testify/require"
)

func TestRecalcQuotaFromRatiosFiltersInvalidMultipliers(t *testing.T) {
	info := &relaycommon.RelayInfo{PriceData: types.PriceData{Quota: 100}}
	info.PriceData.AddOtherRatio("duration", 2)

	quota, ok := recalcQuotaFromRatios(info, map[string]float64{
		"duration": 3,
		"zero":     0,
		"negative": -1,
		"nan":      math.NaN(),
		"inf":      math.Inf(1),
	})

	require.True(t, ok)
	require.Equal(t, 150, quota)
}

func TestRecalcQuotaFromRatiosRejectsAllInvalidMultipliers(t *testing.T) {
	info := &relaycommon.RelayInfo{PriceData: types.PriceData{Quota: 100}}
	info.PriceData.AddOtherRatio("duration", 2)

	quota, ok := recalcQuotaFromRatios(info, map[string]float64{
		"zero": 0,
		"nan":  math.NaN(),
	})

	require.False(t, ok)
	require.Zero(t, quota)
	require.True(t, info.PriceData.HasOtherRatio("duration"))
}
