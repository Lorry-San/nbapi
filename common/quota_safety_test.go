package common

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestQuotaFromFloatRejectsNonFiniteAndNegative(t *testing.T) {
	t.Parallel()

	require.Equal(t, 0, QuotaFromFloat(math.NaN()))
	require.Equal(t, 0, QuotaFromFloat(math.Inf(1)))
	require.Equal(t, 0, QuotaFromFloat(-1))
}

func TestQuotaFromFloatSaturatesToSafeQuota(t *testing.T) {
	t.Parallel()

	require.Equal(t, MaxSafeQuota, QuotaFromFloat(float64(MaxSafeQuota)*10))
}

func TestQuotaFromDecimalRoundSaturatesToSafeQuota(t *testing.T) {
	t.Parallel()

	value := decimal.NewFromInt(MaxSafeQuota).Mul(decimal.NewFromInt(10))

	require.Equal(t, MaxSafeQuota, QuotaFromDecimalRound(value))
}
