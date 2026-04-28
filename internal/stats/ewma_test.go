package stats

import (
	"math"
	"testing"
)

func TestEWMABasic(t *testing.T) {
	ewma := NewEWMA(0.2)

	if ewma.IsInitialized() {
		t.Error("Expected EWMA to not be initialized before adding data")
	}

	// Add first sample
	value := ewma.Add(100.0)
	if !ewma.IsInitialized() {
		t.Error("Expected EWMA to be initialized after first sample")
	}
	if value != 100.0 {
		t.Errorf("Expected first value to be 100.0, got %.4f", value)
	}

	// Add second sample
	value = ewma.Add(80.0)
	expected := 0.2*80.0 + 0.8*100.0 // = 96.0
	if math.Abs(value-expected) > 0.0001 {
		t.Errorf("Expected value %.4f, got %.4f", expected, value)
	}
}

func TestEWMASmoothing(t *testing.T) {
	// Lower alpha = more smoothing
	ewma := NewEWMA(0.1)

	samples := []float64{100, 90, 110, 95, 105}
	for _, s := range samples {
		ewma.Add(s)
	}

	// Value should be between min and max
	value := ewma.Value()
	if value < 90 || value > 110 {
		t.Errorf("Expected smoothed value between 90-110, got %.4f", value)
	}
}

func TestEWMADefaultAlpha(t *testing.T) {
	// Invalid alpha should default to 0.2
	ewma := NewEWMA(-1.0)
	ewma.Add(100.0)
	value := ewma.Add(80.0)

	expected := 0.2*80.0 + 0.8*100.0
	if math.Abs(value-expected) > 0.0001 {
		t.Errorf("Expected default alpha 0.2, got different behavior")
	}
}
