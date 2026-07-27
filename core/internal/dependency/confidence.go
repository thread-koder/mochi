package dependency

import (
	"math"
)

const (
	defaultVolumeNorm     = 100.0
	mochiEBPFSourceWeight = 1.0
)

func Confidence(connects float64, recency float64, volumeNorm float64) float32 {
	if volumeNorm <= 0 {
		volumeNorm = defaultVolumeNorm
	}
	if recency < 0 {
		recency = 0
	}
	if recency > 1 {
		recency = 1
	}

	volume := math.Min(1.0, math.Log1p(connects)/math.Log1p(volumeNorm))
	return float32(recency * volume * mochiEBPFSourceWeight)
}
