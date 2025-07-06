package server

import (

	"github.com/atharvwasthere/Fastlane/internal/location"
	"github.com/atharvwasthere/Fastlane/types"
)

var (
	calculateDistance float64
)

type Scorer struct {
	UserLoc *location.Location
}

func NewScorer(userLoc *location.Location) *Scorer {
	return &Scorer{UserLoc: userLoc}
}

/* 
we will rank server based on their latency and distance, 
returning the best server and its ping result
for the score the lower the better 

the formula used will be 
=-=-=> Score = 0.7 * latency (ms) + 0.3 * distance (km)

we will be calculating the distance using Haversine distance formula 
*/

func (s *Scorer) RankServers(servers []types.Server, pingResults []*types.PingResult ) (*types.Server, *types.PingResult) {
	if s.UserLoc != nil {
		for i, srv := range servers {
			servers[i].Distance = calculateDistance(s.UserLoc.Lat, s.UserLoc.Lon, srv.Latitude , srv.Longitude)
		}
	}

	var bestServer *types.Server
	var bestPing *types.PingResult
	bestScore := float64(1<<63 - 1)

	for i, srv := range servers {
		if !pingResults[i].Success {
			continue
		}

		latencyMs := float64(srv.Latency.Milliseconds())
		distanceKm := srv.Distance
		if s.UserLoc == nil {
			distanceKm = 0
		}
		score := 0.7*latencyMs + 0.3*distanceKm
		if score < bestScore {
			bestScore = score
			bestServer = &servers[i]
			bestPing = pingResults[i]
		}

	}

	return bestServer, bestPing

}

