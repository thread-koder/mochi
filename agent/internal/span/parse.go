package span

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// TraceContext is extracted outbound request parent context.
// Deferred is true when OpenCensus options said the callee may decide.
type TraceContext struct {
	TraceID      [TraceIDSize]byte
	ParentSpanID [SpanIDSize]byte
	Sampled      bool
	Deferred     bool
}

// W3C traceparent version-00 layout: version-trace-id-parent-id-flags.
const (
	traceparentMinLen       = 55
	traceparentVersionEnd   = 2
	traceparentTraceIDStart = 3
	traceparentTraceIDEnd   = 35
	traceparentParentStart  = 36
	traceparentParentEnd    = 52
	traceparentFlagsStart   = 53
	traceparentFlagsEnd     = 55

	versionForbidden = 0xff
	sampledBit       = 0x01

	ocVersion      = 0
	ocFieldTraceID = 0
	ocFieldSpanID  = 1
	ocFieldOptions = 2
)

// ParseTraceparent parses a W3C Trace Context version-00 (or higher with
// ignored trailing fields) traceparent header value.
func ParseTraceparent(value string) (TraceContext, bool) {
	value = strings.TrimSpace(value)
	if len(value) < traceparentMinLen {
		return TraceContext{}, false
	}
	if value[traceparentVersionEnd] != '-' ||
		value[traceparentTraceIDEnd] != '-' ||
		value[traceparentParentEnd] != '-' {
		return TraceContext{}, false
	}
	if len(value) > traceparentMinLen && value[traceparentMinLen] != '-' {
		return TraceContext{}, false
	}

	var version [1]byte
	var versionSrc [2]byte
	copy(versionSrc[:], value[:traceparentVersionEnd])
	if _, err := hex.Decode(version[:], versionSrc[:]); err != nil || version[0] == versionForbidden {
		return TraceContext{}, false
	}

	var tc TraceContext
	var traceSrc [TraceIDSize * 2]byte
	copy(traceSrc[:], value[traceparentTraceIDStart:traceparentTraceIDEnd])
	if _, err := hex.Decode(tc.TraceID[:], traceSrc[:]); err != nil {
		return TraceContext{}, false
	}
	var parentSrc [SpanIDSize * 2]byte
	copy(parentSrc[:], value[traceparentParentStart:traceparentParentEnd])
	if _, err := hex.Decode(tc.ParentSpanID[:], parentSrc[:]); err != nil {
		return TraceContext{}, false
	}
	var flags [1]byte
	var flagsSrc [2]byte
	copy(flagsSrc[:], value[traceparentFlagsStart:traceparentFlagsEnd])
	if _, err := hex.Decode(flags[:], flagsSrc[:]); err != nil {
		return TraceContext{}, false
	}

	if tc.TraceID == zeroTraceID || tc.ParentSpanID == zeroSpanID {
		return TraceContext{}, false
	}
	tc.Sampled = flags[0]&sampledBit != 0
	return tc, true
}

// ParseGRPCTraceBin parses OpenCensus/gRPC binary span context from a
// header field value (version-0 fields and/or Base64, padded or unpadded).
func ParseGRPCTraceBin(value string) (TraceContext, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return TraceContext{}, false
	}
	for chunk := range strings.SplitSeq(value, ",") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		if tc, ok := parseGRPCTraceBinChunk(chunk); ok {
			return tc, true
		}
	}
	return TraceContext{}, false
}

func parseGRPCTraceBinChunk(chunk string) (TraceContext, bool) {
	raw := []byte(chunk)
	if tc, ok := parseOCBinary(raw); ok {
		return tc, true
	}
	decoded, err := base64.RawStdEncoding.DecodeString(chunk)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(chunk)
	}
	if err != nil {
		return TraceContext{}, false
	}
	return parseOCBinary(decoded)
}

func parseOCBinary(data []byte) (TraceContext, bool) {
	if len(data) < 1 || data[0] != ocVersion {
		return TraceContext{}, false
	}
	var (
		tc         TraceContext
		haveTrace  bool
		haveParent bool
		haveOpts   bool
	)
	i := 1
fields:
	for i < len(data) {
		fieldID := data[i]
		i++
		switch fieldID {
		case ocFieldTraceID:
			if i+TraceIDSize > len(data) {
				return TraceContext{}, false
			}
			copy(tc.TraceID[:], data[i:i+TraceIDSize])
			i += TraceIDSize
			haveTrace = true
		case ocFieldSpanID:
			if i+SpanIDSize > len(data) {
				return TraceContext{}, false
			}
			copy(tc.ParentSpanID[:], data[i:i+SpanIDSize])
			i += SpanIDSize
			haveParent = true
		case ocFieldOptions:
			if i+1 > len(data) {
				return TraceContext{}, false
			}
			opts := data[i]
			i++
			haveOpts = true
			if opts&sampledBit != 0 {
				tc.Sampled = true
			} else {
				tc.Deferred = true
			}
		default:
			break fields
		}
	}
	if !haveTrace || !haveParent {
		return TraceContext{}, false
	}
	if tc.TraceID == zeroTraceID || tc.ParentSpanID == zeroSpanID {
		return TraceContext{}, false
	}
	if !haveOpts {
		tc.Deferred = true
	}
	return tc, true
}
