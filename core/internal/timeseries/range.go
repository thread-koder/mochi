package timeseries

import "time"

// Prometheus max allowed points per query.
const maxRangePoints = 11000

// RangeStepForTimeRange returns an appropriate Prometheus range-query step for the given time range.
// Shorter time ranges keep fine resolution for rightsizing, longer time ranges coarsen
// to limit query cost.
func RangeStepForTimeRange(timeRange time.Duration) time.Duration {
	const day = 24 * time.Hour

	var step time.Duration
	switch {
	case timeRange <= 2*day:
		step = time.Minute
	case timeRange <= 14*day:
		step = 5 * time.Minute
	case timeRange <= 45*day:
		step = 15 * time.Minute
	default:
		step = 30 * time.Minute
	}

	return coarsenStepToPointCap(timeRange, step)
}

func coarsenStepToPointCap(timeRange, step time.Duration) time.Duration {
	if step <= 0 {
		step = time.Minute
	}
	for timeRange/step > maxRangePoints {
		switch {
		case step < 5*time.Minute:
			step = 5 * time.Minute
		case step < 15*time.Minute:
			step = 15 * time.Minute
		case step < 30*time.Minute:
			step = 30 * time.Minute
		case step < time.Hour:
			step = time.Hour
		case step < 4*time.Hour:
			step = 4 * time.Hour
		default:
			step = 6 * time.Hour
		}
	}
	return step
}
