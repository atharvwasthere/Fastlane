package server

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		expectedKM             float64
		tolerance              float64
	}{
		{
			name: "Same point",
			lat1: 40.7128, lon1: -74.0060,
			lat2: 40.7128, lon2: -74.0060,
			expectedKM: 0,
			tolerance:  0.1,
		},
		{
			name: "New York to Los Angeles",
			lat1: 40.7128, lon1: -74.0060,
			lat2: 34.0522, lon2: -118.2437,
			expectedKM: 3944, // approximately
			tolerance:  100,
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
