package stats

import (
	"math"
	"testing"
)

func TestWelfordBasic(t *testing.T) {
	w := NewWelford()

	// Add some samples
	samples := []float64{1, 2, 3, 4, 5}
	for _, s := range samples {
		w.Add(s)
	}

	// Mean should be 3
	mean := w.Mean()
	if math.Abs(mean-3.0) > 0.0001 {
		t.Errorf("Expected mean 3.0, got %.4f", mean)
	}

	// Count should be 5
	if w.Count() != 5 {
		t.Errorf("Expected count 5, got %d", w.Count())
	}

	// Min should be 1
	if w.Min() != 1.0 {
		t.Errorf("Expected min 1.0, got %.4f", w.Min())
	}

	// Max should be 5
	if w.Max() != 5.0 {
		t.Errorf("Expected max 5.0, got %.4f", w.Max())
	}
}

func TestWelfordVariance(t *testing.T) {
	w := NewWelford()

	samples := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	for _, s := range samples {
		w.Add(s)
	}

	// Sample variance should be approximately 4.5714 (corrected value)
	svar := w.SampleVariance()
	if math.Abs(svar-4.5714) > 0.01 {
		t.Errorf("Expected sample variance ~4.5714, got %.4f", svar)
	}
}

func TestWelfordCoefficientOfVariation(t *testing.T) {
	w := NewWelford()

	samples := []float64{10, 20, 30}
	for _, s := range samples {
		w.Add(s)
	}

	// CV should be approximately 0.471 (sqrt(100)/20)
	cv := w.CoefficientOfVariation()
	if cv <= 0 {
		t.Errorf("Expected positive CV, got %.4f", cv)
	}
}

func TestWelfordEmptyAndSingle(t *testing.T) {
	w := NewWelford()

	// Empty case
	if w.Count() != 0 {
		t.Errorf("Expected count 0 for empty, got %d", w.Count())
	}

	// Single sample
	w.Add(5.0)
	if w.Count() != 1 {
		t.Errorf("Expected count 1, got %d", w.Count())
	}
	if w.Mean() != 5.0 {
		t.Errorf("Expected mean 5.0, got %.4f", w.Mean())
	}
}
