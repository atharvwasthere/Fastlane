package location

import (
	"fmt"
	"net"
	"os"

	"github.com/oschwald/geoip2-golang"
)

type UserLocation struct {
	City    string
	Country string
}

func GetUserLocation() *UserLocation {
	db, err := geoip2.Open("assets/GeoLite2-City.mmdb")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to open GeoLite2-City.mmdb: %v\n", err)
		return nil
	}
	defer db.Close()

	// Use a public IP (e.g., from an API or local interface)
	ip := net.ParseIP("182.72.39.9") // Replace with actual user IP lookup
	record, err := db.City(ip)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: GeoIP lookup failed: %v\n", err)
		return nil
	}

	return &UserLocation{
		City:    record.City.Names["en"],
		Country: record.Country.IsoCode,
	}
}