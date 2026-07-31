package common

import (
	"fmt"
	"strconv"
	"time"
)

func ParseTimeRange(timeRange string) (time.Duration, error) {
	if len(timeRange) > 0 && timeRange[len(timeRange)-1] == 'd' {
		daysPart := timeRange[:len(timeRange)-1]
		days, err := strconv.Atoi(daysPart)
		if err != nil {
			return 0, fmt.Errorf("failed to parse days: %w", err)
		}
		if days <= 0 {
			return 0, fmt.Errorf("days must be positive")
		}
		timeRange = fmt.Sprintf("%dh", days*24)
	}

	duration, err := time.ParseDuration(timeRange)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("time range must be positive")
	}

	return duration, nil
}
