package server

import "testing"

func TestShortlistByDistance_NearestWins(t *testing.T) {
	servers := []Server{
		{ID: "tokyo", Latitude: 35.68, Longitude: 139.69, Host: "tk", Port: 1},
		{ID: "sydney", Latitude: -33.87, Longitude: 151.21, Host: "syd", Port: 1},
		{ID: "frankfurt", Latitude: 50.11, Longitude: 8.68, Host: "fra", Port: 1},
		{ID: "anycast", Latitude: 0, Longitude: 0, Host: "cf", Port: 443},
	}
	// User in Mumbai (19.08, 72.88). Frankfurt is closer than Tokyo or Sydney.
	out := shortlistByDistance(servers, 19.08, 72.88, 2)

	// First two slots should be the two nearest by haversine (frankfurt, tokyo).
	if out[0].ID != "frankfurt" {
		t.Errorf("out[0].ID = %q, want frankfurt", out[0].ID)
	}
	if out[1].ID != "tokyo" {
		t.Errorf("out[1].ID = %q, want tokyo", out[1].ID)
	}
	// Anycast always appended last as fallback.
	if out[len(out)-1].ID != "anycast" {
		t.Errorf("last entry = %q, want anycast", out[len(out)-1].ID)
	}
}

func TestShortlistByDistance_AllAnycast(t *testing.T) {
	servers := []Server{
		{ID: "cf", Host: "cf", Port: 443},
		{ID: "cachefly", Host: "cf2", Port: 443},
	}
	out := shortlistByDistance(servers, 19.0, 72.0, 5)
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
}
