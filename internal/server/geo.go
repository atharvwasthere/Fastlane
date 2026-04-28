package server

import "math"

const earthRadiusKM = 6371.0

// HaversineDistance calculates the great-circle distance between two points
// using the Haversine formula. Returns distance in kilometers.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Convert degrees to radians
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	lat1Rad := toRad(lat1)
	lat2Rad := toRad(lat2)

	// Haversine formula
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Asin(math.Sqrt(a))
	return earthRadiusKM * c
}

// toRad converts degrees to radians
func toRad(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

// NormalizeLatency normalizes latency value between 0 and 1
// where 0 = 0ms and 1 = 100ms (baseline)
func NormalizeLatency(latencyMS float64) float64 {
	const baseline = 100.0
	normalized := latencyMS / baseline
	if normalized > 1.0 {
		return 1.0
	}
	if normalized < 0 {
		return 0
	}
	return normalized
}

// NormalizeThroughput normalizes throughput value between 0 and 1
// where 0 = 0Mbps and 1 = 1000Mbps (baseline)
// Lower is better for scoring (inverted)
func NormalizeThroughput(throughputMbps float64) float64 {
	const baseline = 1000.0
	normalized := throughputMbps / baseline
	if normalized > 1.0 {
		return 0.0 // Cap at 1.0, then invert
	}
	if normalized < 0 {
		normalized = 0
	}
	return 1.0 - normalized // Invert so higher throughput = lower score
}

// CalculateScore computes final server score
// score = 0.7 * latency_norm + 0.3 * (1 - throughput_norm)
// Lower score is better
func CalculateScore(latencyMS, throughputMbps float64) float64 {
	latencyNorm := NormalizeLatency(latencyMS)
	throughputNorm := NormalizeThroughput(throughputMbps)

	return 0.7*latencyNorm + 0.3*throughputNorm
}
