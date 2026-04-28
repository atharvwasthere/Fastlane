package stats

import "math"

// Welford computes running mean and variance using Welford's algorithm
// This is numerically stable and does not require storing all samples
type Welford struct {
	count   int64
	mean    float64
	M2      float64 // sum of squares of differences
	min     float64
	max     float64
	isEmpty bool
}

// NewWelford creates a new Welford accumulator
func NewWelford() *Welford {
	return &Welford{
		isEmpty: true,
	}
}

// Add adds a new sample to the accumulator
func (w *Welford) Add(x float64) {
	w.count++
	delta := x - w.mean
	w.mean = w.mean + delta/float64(w.count)
	delta2 := x - w.mean
	w.M2 = w.M2 + delta*delta2

	if w.isEmpty {
		w.min = x
		w.max = x
		w.isEmpty = false
	} else {
		if x < w.min {
			w.min = x
		}
		if x > w.max {
			w.max = x
		}
	}
}

// Mean returns the current mean
func (w *Welford) Mean() float64 {
	return w.mean
}

// Variance returns the current variance (population variance)
func (w *Welford) Variance() float64 {
	if w.count < 2 {
		return 0
	}
	return w.M2 / float64(w.count)
}

// SampleVariance returns the sample variance (unbiased estimator)
func (w *Welford) SampleVariance() float64 {
	if w.count < 2 {
		return 0
	}
	return w.M2 / float64(w.count-1)
}

// StdDev returns the standard deviation (population)
func (w *Welford) StdDev() float64 {
	return math.Sqrt(w.Variance())
}

// SampleStdDev returns the sample standard deviation
func (w *Welford) SampleStdDev() float64 {
	return math.Sqrt(w.SampleVariance())
}

// Min returns the minimum value seen
func (w *Welford) Min() float64 {
	return w.min
}

// Max returns the maximum value seen
func (w *Welford) Max() float64 {
	return w.max
}

// Count returns the number of samples
func (w *Welford) Count() int64 {
	return w.count
}

// CoefficientOfVariation returns the coefficient of variation (sample stddev/mean)
// Returns 0 if mean is 0 to avoid division by zero
func (w *Welford) CoefficientOfVariation() float64 {
	if w.mean == 0 || w.count < 2 {
		return 0
	}
	return w.SampleStdDev() / math.Abs(w.mean)
}
