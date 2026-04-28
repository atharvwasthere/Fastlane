package ping

import "math"

// PautaFilter applies 3-sigma Pauta filtering to remove outliers
type PautaFilter struct {
	threshold float64 // Number of standard deviations (default 3.0)
}

// NewPautaFilter creates a new Pauta filter with default 3-sigma threshold
func NewPautaFilter() *PautaFilter {
	return &PautaFilter{threshold: 3.0}
}

// Filter removes outliers from samples using Pauta method
// Returns filtered samples and count of removed outliers
func (pf *PautaFilter) Filter(samples []float64) ([]float64, int) {
	if len(samples) < 3 {
		return samples, 0
	}

	mean := average(samples)
	stdDev := calculateStddev(samples, mean)

	// If std dev is 0, all values are the same, no filtering needed
	if stdDev == 0 {
		return samples, 0
	}

	filtered := make([]float64, 0)
	removed := 0

	// 3-sigma rule: keep values within mean ± threshold*stddev
	lower := mean - pf.threshold*stdDev
	upper := mean + pf.threshold*stdDev

	for _, val := range samples {
		if val >= lower && val <= upper {
			filtered = append(filtered, val)
		} else {
			removed++
		}
	}

	// If all samples were filtered out, keep at least the median
	if len(filtered) == 0 {
		// Keep values closest to mean
		for _, val := range samples {
			if val == mean {
				filtered = append(filtered, val)
				removed--
			}
		}
		// If no exact matches, keep at least half
		if len(filtered) == 0 && len(samples) > 1 {
			return samples[:len(samples)/2], len(samples) - len(samples)/2
		}
	}

	return filtered, removed
}

// SetThreshold sets custom sigma threshold (typically 2.0-3.5)
func (pf *PautaFilter) SetThreshold(sigma float64) {
	if sigma > 0 {
		pf.threshold = sigma
	}
}

// calculateStddev computes standard deviation given mean
func calculateStddev(vals []float64, mean float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range vals {
		diff := v - mean
		sumSq += diff * diff
	}
	variance := sumSq / float64(len(vals)-1)
	return math.Sqrt(variance)
}

// IsOutlier checks if a single value is an outlier given samples
func (pf *PautaFilter) IsOutlier(val float64, samples []float64) bool {
	if len(samples) < 3 {
		return false
	}

	mean := average(samples)
	stdDev := calculateStddev(samples, mean)

	lower := mean - pf.threshold*stdDev
	upper := mean + pf.threshold*stdDev

	return val < lower || val > upper
}
