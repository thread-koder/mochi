package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ParseTimeRange parses time range values like "24h", "7d", or "1h30m".
func ParseTimeRange(timeRangeStr string) (time.Duration, error) {
	if len(timeRangeStr) > 0 && timeRangeStr[len(timeRangeStr)-1] == 'd' {
		daysStr := timeRangeStr[:len(timeRangeStr)-1]
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse days: %w", err)
		}
		if days <= 0 {
			return 0, fmt.Errorf("days must be positive")
		}
		hours := days * 24
		timeRangeStr = fmt.Sprintf("%dh", hours)
	}

	duration, err := time.ParseDuration(timeRangeStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("time range must be positive")
	}

	return duration, nil
}

// IsNotFoundError reports whether err represents a missing resource.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "not found")
}
