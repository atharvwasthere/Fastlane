package utils

import (
	"encoding/json"
	"fmt"
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
	return fmt.Sprintf("%.1f ms", float64(d.Milliseconds())+float64(d.Microseconds()%1000)/1000.0)
}