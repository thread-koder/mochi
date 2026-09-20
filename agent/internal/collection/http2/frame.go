package http2

import (
	"bytes"
	"encoding/binary"
)

// RFC 9113 frame header, frame type, and flag wire values.
const (
	frameHeaderLen = 9

	frameData         = 0x0
	frameHeaders      = 0x1
	framePriority     = 0x2
	frameRSTStream    = 0x3
	frameSettings     = 0x4
	framePushPromise  = 0x5
	framePing         = 0x6
	frameGoAway       = 0x7
	frameWindowUpdate = 0x8
	frameContinuation = 0x9

	flagEndStream  = 0x1
	flagEndHeaders = 0x4
	flagPadded     = 0x8
	flagPriority   = 0x20

	streamIDReservedBit = 0x80
	streamIDMask        = 0x7fffffff

	settingsParamLen   = 6
	pingPayloadLen     = 8
	windowUpdateLen    = 4
	rstStreamLen       = 4
	priorityPayloadLen = 5
	goAwayMinLen       = 8
	padLengthSize      = 1
	promisedIDLen      = 4

	classifyFrameWalk = 8
	incompleteCap     = 8 << 10
)

// clientPreface is the 24 byte HTTP/2 connection preface (RFC 9113).
var clientPreface = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")

type frame struct {
	Length   uint32
	Type     uint8
	Flags    uint8
	StreamID uint32
	Payload  []byte
}

func readFrameHeader(buf []byte) (frame, bool) {
	if len(buf) < frameHeaderLen {
		return frame{}, false
	}
	if buf[5]&streamIDReservedBit != 0 {
		return frame{}, false
	}
	length := uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])
	return frame{
		Length:   length,
		Type:     buf[3],
		Flags:    buf[4],
		StreamID: binary.BigEndian.Uint32(buf[5:9]) & streamIDMask,
	}, true
}

func LooksLikePreface(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	return bytes.HasPrefix(data, clientPreface) || bytes.HasPrefix(clientPreface, data)
}

// LooksLikeFrames reports whether data begins with plausible HTTP/2 frames
// that include SETTINGS or HEADERS (used to classify a new flow).
func LooksLikeFrames(data []byte) bool {
	pos := 0
	for range classifyFrameWalk {
		if pos > len(data)-frameHeaderLen {
			return false
		}
		hdr, ok := readFrameHeader(data[pos:])
		if !ok {
			return false
		}
		if !plausibleFrame(hdr) {
			return false
		}
		switch hdr.Type {
		case frameHeaders, frameSettings:
			return true
		}
		next := pos + frameHeaderLen + int(hdr.Length)
		if next > len(data) {
			return false
		}
		pos = next
	}
	return false
}

func plausibleFrame(hdr frame) bool {
	switch hdr.Type {
	case frameData, frameHeaders, framePriority, frameRSTStream,
		frameSettings, framePushPromise, framePing, frameGoAway,
		frameWindowUpdate, frameContinuation:
	default:
		// Unknown types are ignored once classified. Detection requires known.
		return false
	}
	switch hdr.Type {
	case frameSettings:
		if hdr.StreamID != 0 || hdr.Length%settingsParamLen != 0 {
			return false
		}
	case framePing:
		if hdr.StreamID != 0 || hdr.Length != pingPayloadLen {
			return false
		}
	case frameWindowUpdate:
		if hdr.Length != windowUpdateLen {
			return false
		}
	case frameRSTStream:
		if hdr.StreamID == 0 || hdr.Length != rstStreamLen {
			return false
		}
	case framePriority:
		if hdr.StreamID == 0 || hdr.Length != priorityPayloadLen {
			return false
		}
	case frameGoAway:
		if hdr.StreamID != 0 || hdr.Length < goAwayMinLen {
			return false
		}
	case frameHeaders, frameData, framePushPromise, frameContinuation:
		if hdr.Type == frameContinuation && hdr.StreamID == 0 {
			return false
		}
		if hdr.Type == framePushPromise && hdr.StreamID == 0 {
			return false
		}
	}
	return true
}

// headerBlockFragment strips PADDED/PRIORITY from a HEADERS or PUSH_PROMISE
// payload and returns the HPACK fragment. ok is false if the payload is short.
func headerBlockFragment(frameType uint8, flags uint8, payload []byte) (frag []byte, ok bool) {
	rest := payload
	if flags&flagPadded != 0 {
		if len(rest) < padLengthSize {
			return nil, false
		}
		pad := int(rest[0])
		rest = rest[padLengthSize:]
		if pad > len(rest) {
			return nil, false
		}
		rest = rest[:len(rest)-pad]
	}
	switch frameType {
	case frameHeaders:
		if flags&flagPriority != 0 {
			if len(rest) < priorityPayloadLen {
				return nil, false
			}
			rest = rest[priorityPayloadLen:]
		}
	case framePushPromise:
		if len(rest) < promisedIDLen {
			return nil, false
		}
		rest = rest[promisedIDLen:]
	}
	return rest, true
}
