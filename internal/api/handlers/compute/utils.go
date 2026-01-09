package compute

import (
	"fmt"
	"strconv"
	"time"
)

// Parses time range from query parameter (e.g., "24h", "7d", "1h30m")
func parseTimeRange(timeRangeStr string) (time.Duration, error) {
	duration, err := time.ParseDuration(timeRangeStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse time range: %w", err)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("time range must be positive")
	}

	return duration, nil
}

// Helper function to parse integer from string
func parseInt(s string) (int, error) {
	result, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer: %w", err)
	}
	return result, nil
}

// Helper function to parse int64 from string
func parseInt64(s string) (int64, error) {
	result, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int64: %w", err)
	}
	return result, nil
}
