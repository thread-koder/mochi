package span

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// NewIDs mints a W3C trace_id (16) and span_id (8), rejecting all-zero.
func NewIDs() (traceID [16]byte, spanID [8]byte, err error) {
	for {
		if _, err = rand.Read(traceID[:]); err != nil {
			return [16]byte{}, [8]byte{}, fmt.Errorf("failed to generate trace_id: %w", err)
		}
		if _, err = rand.Read(spanID[:]); err != nil {
			return [16]byte{}, [8]byte{}, fmt.Errorf("failed to generate span_id: %w", err)
		}
		if traceID != [16]byte{} && spanID != [8]byte{} {
			return traceID, spanID, nil
		}
	}
}

// Sampled matches otel-go TraceIDRatioBased: lower 8 bytes, 63-bit compare.
func Sampled(traceID [16]byte, rate float64) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	threshold := uint64(rate * (1 << 63))
	return binary.BigEndian.Uint64(traceID[8:16])>>1 < threshold
}
