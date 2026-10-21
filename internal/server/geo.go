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
