package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"math"
)

func ToJSON(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fµs", float64(d.Microseconds()))
	} else if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1e6)
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const radius = 6378.137 // Earth radius in km
	phi1 := lat1 * (3.141592653589793 / 180.0)
	phi2 := lat2 * (3.141592653589793 / 180.0)
	deltaPhi := (lat2 - lat1) * (3.141592653589793 / 180.0)
	deltaLambda := (lon2 - lon1) * (3.141592653589793 / 180.0)
	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) + math.Cos(phi1)*math.Cos(phi2)*math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return radius * c
}


func SaveReport(report interface{}, server string) (string, error) {
	// get the home directory 
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}

	// reports directory
	reportsDir := filepath.Join(home, ".fastlane", "reports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create reports directory: %v", err)
	}

	// timestamped filename (e.g., FL-21JUN-0411.json)
	now := time.Now()
	timestamp := now.Format("02Jan-1504")
	filename := fmt.Sprintf("FL-%s.json", strings.ToUpper(timestamp))
	filepath := filepath.Join(reportsDir, filename)

	// report structure
	reportStruct := struct {
		Timestamp string      `json:"timestamp"`
		Server    string      `json:"server"`
		Results   interface{} `json:"results"`
	}{
		Timestamp: now.Format(time.RFC3339),
		Server:    server,
		Results:   report,
	}

	data, err := json.MarshalIndent(reportStruct, "","  ")
		if err != nil {
		return "", fmt.Errorf("failed to serialize report: %v", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save report: %v", err)
	}

	return filepath, nil

}