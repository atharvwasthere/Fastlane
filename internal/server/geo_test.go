package server

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name     string
		lat1, lon1, lat2, lon2 float64
		expectedKM float64
		tolerance float64
	}{
		{
			name:       "Same point",
			lat1:       40.7128, lon1: -74.0060,
			lat2:       40.7128, lon2: -74.0060,
			expectedKM: 0,
			tolerance: 0.1,
		},
		{
			name:       "New York to Los Angeles",
			lat1:       40.7128, lon1: -74.0060,
			lat2:       34.0522, lon2: -118.2437,
			expectedKM: 3944, // approximately
			tolerance: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HaversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(result-tt.expectedKM) > tt.tolerance {
				t.Errorf("Expected ~%.0f km, got %.0f km", tt.expectedKM, result)
			}
		})
	}
}

func TestNormalizeLatency(t *testing.T) {
	tests := []struct {
		latency  float64
		expected float64
	}{
		{0, 0},
		{50, 0.5},
		{100, 1.0},
		{150, 1.0}, // Capped at 1.0
		{-10, 0},   // Negative clamped to 0
	}

	for _, tt := range tests {
		result := NormalizeLatency(tt.latency)
		if math.Abs(result-tt.expected) > 0.0001 {
			t.Errorf("NormalizeLatency(%.1f): expected %.2f, got %.2f", tt.latency, tt.expected, result)
		}
	}
}

func TestNormalizeThroughput(t *testing.T) {
	tests := []struct {
		throughput float64
		expected   float64
	}{
		{0, 1.0},           // 0 Mbps -> score 1.0 (worst)
		{500, 0.5},         // 500 Mbps -> score 0.5
		{1000, 0.0},        // 1000 Mbps -> score 0.0 (best)
		{2000, 0.0},        // Capped at 1.0 throughput, then inverted
		{-100, 1.0},        // Negative throughput -> score 1.0
	}

	for _, tt := range tests {
		result := NormalizeThroughput(tt.throughput)
		if math.Abs(result-tt.expected) > 0.0001 {
			t.Errorf("NormalizeThroughput(%.1f): expected %.2f, got %.2f", tt.throughput, tt.expected, result)
		}
	}
}

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		latency    float64
		throughput float64
		desc       string
	}{
		{50, 500, "medium latency, medium throughput"},
		{10, 950, "low latency, high throughput"},
		{100, 0, "high latency, no throughput"},
	}

	for _, tt := range tests {
		score := CalculateScore(tt.latency, tt.throughput)
		if score < 0 || score > 1 {
			t.Errorf("CalculateScore(%s): expected 0-1, got %.2f", tt.desc, score)
		}
	}
}
