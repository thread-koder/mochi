package span

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

const (
	TraceIDSize = 16
	SpanIDSize  = 8
)

var (
	zeroTraceID [TraceIDSize]byte
	zeroSpanID  [SpanIDSize]byte
)

func IsZeroTraceID(id [TraceIDSize]byte) bool { return id == zeroTraceID }
func IsZeroSpanID(id [SpanIDSize]byte) bool   { return id == zeroSpanID }

// NewIDs mints a W3C trace_id and span_id, rejecting all-zero.
func NewIDs() (traceID [TraceIDSize]byte, spanID [SpanIDSize]byte, err error) {
	if err = fillNonZero(traceID[:]); err != nil {
		return zeroTraceID, zeroSpanID, fmt.Errorf("failed to generate trace_id: %w", err)
	}
	if err = fillNonZero(spanID[:]); err != nil {
		return zeroTraceID, zeroSpanID, fmt.Errorf("failed to generate span_id: %w", err)
	}
	return traceID, spanID, nil
}

// NewSpanID mints a span id, rejecting all-zero.
func NewSpanID() ([SpanIDSize]byte, error) {
	var spanID [SpanIDSize]byte
	if err := fillNonZero(spanID[:]); err != nil {
		return zeroSpanID, fmt.Errorf("failed to generate span_id: %w", err)
	}
	return spanID, nil
}

func fillNonZero(b []byte) error {
	for {
		if _, err := rand.Read(b); err != nil {
			return err
		}
		for _, v := range b {
			if v != 0 {
				return nil
			}
		}
	}
}

// Sampled matches otel-go TraceIDRatioBased: lower 8 bytes, 63-bit compare.
func Sampled(traceID [TraceIDSize]byte, rate float64) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	threshold := uint64(rate * (1 << 63))
	return binary.BigEndian.Uint64(traceID[TraceIDSize-SpanIDSize:])>>1 < threshold
}
