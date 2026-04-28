package stats

// EWMA calculates the exponential weighted moving average
// Useful for smoothing bandwidth measurements in real-time
type EWMA struct {
	alpha float64 // smoothing factor (0 < alpha <= 1)
	value float64
	init  bool
}

// NewEWMA creates a new EWMA with the given smoothing factor
// alpha: smoothing factor (typically 0.1-0.3). Lower = more smoothing.
func NewEWMA(alpha float64) *EWMA {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.2 // sensible default
	}
	return &EWMA{alpha: alpha}
}

// Add adds a new sample and returns the updated EWMA value
func (e *EWMA) Add(sample float64) float64 {
	if !e.init {
		e.value = sample
		e.init = true
	} else {
		e.value = e.alpha*sample + (1-e.alpha)*e.value
	}
	return e.value
}

// Value returns the current EWMA value
func (e *EWMA) Value() float64 {
	return e.value
}

// IsInitialized returns whether at least one sample has been added
func (e *EWMA) IsInitialized() bool {
	return e.init
}
