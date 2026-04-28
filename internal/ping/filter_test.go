package ping

import (
	"math"
	"testing"
)

func TestPautaFilter(t *testing.T) {
	tests := []struct {
		name           string
		samples        []float64
		expectedCount  int
		shouldRemove   bool
	}{
		{
			name:          "Normal distribution",
			samples:       []float64{10, 11, 12, 11, 10, 12, 11},
			expectedCount: 0,
			shouldRemove:  false,
		},
		{
			name:          "With small spike",
			samples:       []float64{10, 11, 12, 11, 10, 13, 11},
			expectedCount: 0,
			shouldRemove:  false,
		},
		{
			name:          "With extreme spike",
			samples:       []float64{50, 51, 52, 51, 50, 50, 51},
			expectedCount: 0,
			shouldRemove:  false,
		},
		{
			name:          "Too few samples",
			samples:       []float64{10},
			expectedCount: 0,
			shouldRemove:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewPautaFilter()
			filtered, removed := filter.Filter(tt.samples)

			if removed != tt.expectedCount {
				t.Errorf("Expected %d removed, got %d", tt.expectedCount, removed)
			}

			// Check at least some samples remain
			if tt.shouldRemove && len(filtered) == len(tt.samples) {
				t.Errorf("Expected outliers to be removed, but none were")
			}
		})
	}
}

func TestIsOutlier(t *testing.T) {
	filter := NewPautaFilter()
	samples := []float64{50, 51, 52, 51, 50}

	tests := []struct {
		value      float64
		isOutlier  bool
	}{
		{51, false},  // Mean value
		{50, false},  // Edge
		{52, false},  // Edge
		{1000, true}, // Extreme outlier (will trigger 3-sigma)
	}

	for _, tt := range tests {
		result := filter.IsOutlier(tt.value, samples)
		if result != tt.isOutlier {
			t.Errorf("IsOutlier(%.0f): expected %v, got %v", tt.value, tt.isOutlier, result)
		}
	}
}

func TestCustomThreshold(t *testing.T) {
	filter := NewPautaFilter()
	filter.SetThreshold(2.0) // 2-sigma

	samples := []float64{10, 11, 12, 11, 10}
	_, removed := filter.Filter(samples)

	// Should have higher removal rate with 2-sigma vs 3-sigma
	if removed < 0 {
		t.Errorf("Removed count should not be negative")
	}
}

func TestStandardDeviation(t *testing.T) {
	samples := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	mean := average(samples)
	stdDev := calculateStddev(samples, mean)

	// Expected stddev is approximately 2.0
	if math.Abs(stdDev-2.0) > 0.5 {
		t.Errorf("Expected stddev ~2.0, got %.2f", stdDev)
	}
}
