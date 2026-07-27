package common

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestQuotaFromFloatRoundRejectsNonFiniteAndNegative(t *testing.T) {
	t.Parallel()

	require.Equal(t, 0, QuotaFromFloatRound(math.NaN()))
	require.Equal(t, 0, QuotaFromFloatRound(math.Inf(1)))
	require.Equal(t, 0, QuotaFromFloatRound(-1))
}

func TestQuotaFromFloatRoundSaturatesToSafeQuota(t *testing.T) {
	t.Parallel()

	require.Equal(t, MaxSafeQuota, QuotaFromFloatRound(float64(MaxSafeQuota)*10))
}

func TestQuotaFromDecimalRoundSaturatesToSafeQuota(t *testing.T) {
	t.Parallel()

	value := decimal.NewFromInt(MaxSafeQuota).Mul(decimal.NewFromInt(10))

	require.Equal(t, MaxSafeQuota, QuotaFromDecimalRound(value))
}
