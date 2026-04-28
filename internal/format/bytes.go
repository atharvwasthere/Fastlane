package format

import "fmt"

// Bytes converts a byte count to a human-readable string (B / KB / MB / GB).
func Bytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	kb := float64(b) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1f KB", kb)
	}
	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1f MB", mb)
	}
	gb := mb / 1024
	return fmt.Sprintf("%.1f GB", gb)
}

// Mbps formats a Mbps value with adaptive precision.
func Mbps(v float64) string {
	if v < 10 {
		return fmt.Sprintf("%.2f Mbps", v)
	}
	if v < 100 {
		return fmt.Sprintf("%.1f Mbps", v)
	}
	return fmt.Sprintf("%.0f Mbps", v)
}
