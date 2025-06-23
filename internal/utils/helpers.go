package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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