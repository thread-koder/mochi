package compute

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Parses time range from query parameter (e.g., "24h", "7d", "1h30m")
func parseTimeRange(timeRangeStr string) (time.Duration, error) {
	if len(timeRangeStr) > 0 && timeRangeStr[len(timeRangeStr)-1] == 'd' {
		// Extract the number part
		daysStr := timeRangeStr[:len(timeRangeStr)-1]
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse days: %w", err)
		}
		if days <= 0 {
			return 0, fmt.Errorf("days must be positive")
		}
		// Convert days to hours
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

// Helper function to check if an error is a "not found" error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "not found")
}
