package compute

import (
	"fmt"
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
