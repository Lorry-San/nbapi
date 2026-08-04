package types

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestPriceDataOtherRatiosAreValidatedAndCopied(t *testing.T) {
	priceData := PriceData{}
	priceData.AddOtherRatio("zero", 0)
	priceData.AddOtherRatio("negative", -1)
	priceData.AddOtherRatio("nan", math.NaN())
	priceData.AddOtherRatio("inf", math.Inf(1))
	priceData.AddOtherRatio("duration", 2.5)

	ratios := priceData.OtherRatios()
	require.Equal(t, map[string]float64{"duration": 2.5}, ratios)

	ratios["duration"] = 99
	ratios["new"] = 3
	require.Equal(t, map[string]float64{"duration": 2.5}, priceData.OtherRatios())
}

func TestPriceDataAppliesOnlyValidOtherRatios(t *testing.T) {
	priceData := PriceData{}
	require.True(t, priceData.ReplaceOtherRatios(map[string]float64{
		"duration": 2,
		"size":     1.5,
		"invalid":  math.NaN(),
	}))

	require.Equal(t, 3.0, priceData.OtherRatioMultiplier())
	require.Equal(t, 30.0, priceData.ApplyOtherRatiosToFloat(10))
	require.Equal(t, 10.0, priceData.RemoveOtherRatiosFromFloat(30))
	require.True(t, decimal.NewFromInt(30).Equal(priceData.ApplyOtherRatiosToDecimal(decimal.NewFromInt(10))))

	require.False(t, priceData.ReplaceOtherRatios(map[string]float64{"invalid": math.Inf(1)}))
	require.Nil(t, priceData.OtherRatios())
}

func TestPriceDataFloatMultiplicationSaturates(t *testing.T) {
	priceData := PriceData{}
	priceData.AddOtherRatio("first", math.MaxInt32)
	priceData.AddOtherRatio("second", math.MaxInt32)

	require.Equal(t, float64(math.MaxInt32), priceData.ApplyOtherRatiosToFloat(100))
}
