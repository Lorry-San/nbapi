package common

import (
	"math"

	"github.com/shopspring/decimal"
)

const (
	MaxSafeQuota          = MaxQuota
	MaxRequestImageCount = 128
	MaxRequestDuration   = 3600
	MaxRequestTokens     = math.MaxInt32 / 2
)

func QuotaFromFloatRound(value float64) int {
	if !isFinitePositiveFloat(value) {
		return 0
	}
	rounded := math.Round(value)
	if rounded > float64(MaxSafeQuota) {
		return MaxSafeQuota
	}
	return int(rounded)
}

func QuotaFromFloatTrunc(value float64) int {
	if !isFinitePositiveFloat(value) {
		return 0
	}
	if value > float64(MaxSafeQuota) {
		return MaxSafeQuota
	}
	return int(value)
}

func QuotaFromDecimalRound(value decimal.Decimal) int {
	if value.LessThanOrEqual(decimal.Zero) {
		return 0
	}
	maxQuota := decimal.NewFromInt(MaxSafeQuota)
	if value.GreaterThan(maxQuota) {
		return MaxSafeQuota
	}
	return QuotaFromDecimal(value)
}

func QuotaFromInt64(value int64) int {
	if value <= 0 {
		return 0
	}
	if value > MaxSafeQuota {
		return MaxSafeQuota
	}
	return int(value)
}

func AddQuotaSafe(left, right int) int {
	if left <= 0 {
		return QuotaFromInt64(int64(right))
	}
	if right <= 0 {
		return QuotaFromInt64(int64(left))
	}
	if left > MaxSafeQuota-right {
		return MaxSafeQuota
	}
	return left + right
}

func BoundedRequestTokens(value uint) int {
	if value == 0 {
		return 0
	}
	if value > uint(MaxRequestTokens) {
		return MaxRequestTokens
	}
	return int(value)
}

func isFinitePositiveFloat(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
