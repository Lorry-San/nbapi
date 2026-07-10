package billingexpr

import "math"

// QuotaRound converts a float64 quota value to int using half-away-from-zero
// rounding. Every tiered billing path (pre-consume, settlement, breakdown
// validation, log fields) MUST use this function to avoid +-1 discrepancies.
func QuotaRound(f float64) int {
	rounded := math.Round(f)
	if math.IsNaN(rounded) {
		return 0
	}
	if rounded >= math.MaxInt32 {
		return math.MaxInt32
	}
	if rounded <= math.MinInt32 {
		return math.MinInt32
	}
	return int(rounded)
}
