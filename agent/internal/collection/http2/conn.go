package http2

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/thread_koder/mochi/agent/internal/collection/http1"
	"github.com/thread_koder/mochi/agent/internal/metrics"
	"github.com/thread_koder/mochi/agent/internal/span"
	"golang.org/x/net/http2/hpack"
)

const (
	DirRecv = 0
	DirSend = 1

	hpackTableSize = 4096
	hpackMaxString = 4096
)

type Request struct {
	StreamID     uint32
	Method       string
	Path         string
	Route        string
	GRPC         bool
	Key          metrics.SeriesKey
	Start        time.Time
	Traceparent  string
	GRPCTraceBin string
	TraceID      [span.TraceIDSize]byte
	SpanID       [span.SpanIDSize]byte
	ParentSpanID [span.SpanIDSize]byte
	Sampled      bool
}

type Completion struct {
	Key          metrics.SeriesKey
	Method       string
	Route        string
	StatusClass  string
	StatusCode   int
	GRPC         bool
	GRPCStatus   *int
	Start        time.Time
	End          time.Time
	TraceID      [span.TraceIDSize]byte
	SpanID       [span.SpanIDSize]byte
	ParentSpanID [span.SpanIDSize]byte
	Sampled      bool
}

type pendingBlock struct {
	streamID  uint32
	frameType uint8
	endStream bool
	frags     [][]byte
}

type Conn struct {
	sendBuf []byte
	recvBuf []byte

	sendDec *hpack.Decoder
	recvDec *hpack.Decoder

	block    *pendingBlock
	streams  map[uint32]*Request
	emitStop bool
}

func NewConn() *Conn {
	return &Conn{
		streams: make(map[uint32]*Request),
		sendDec: newDecoder(),
		recvDec: newDecoder(),
	}
}

func newDecoder() *hpack.Decoder {
	decoder := hpack.NewDecoder(hpackTableSize, nil)
	decoder.SetMaxStringLength(hpackMaxString)
	return decoder
}

// keepLeftover stores leftover in buf. When owned, leftover is a suffix of the
// same backing array so copy+reslice reuses capacity. Otherwise append into
// any retained cap ([:0] from a prior call) or grow.
func keepLeftover(buf *[]byte, leftover []byte, owned bool) {
	if len(leftover) > incompleteCap {
		*buf = nil
		return
	}
	if len(leftover) == 0 {
		if *buf != nil {
			*buf = (*buf)[:0]
		}
		return
	}
	if owned {
		copied := copy(*buf, leftover)
		*buf = (*buf)[:copied]
		return
	}
	*buf = append((*buf)[:0], leftover...)
}

// Feed parses one direction. truncated means the syscall may have been cut at
// 1024 bytes so incomplete frames are dropped, not leftover stitched.
// Returned requests have Method/Path/GRPC/StreamID/Traceparent/GRPCTraceBin/Start.
// Caller sets Route/Key. stop is true when DropAll ran (desync or truncated field block).
func (c *Conn) Feed(dir uint8, data []byte, truncated bool, now time.Time, outstanding *int) (reqs []Request, hops []Completion, stop bool) {
	if c.emitStop {
		return nil, nil, false
	}

	var buf *[]byte
	switch dir {
	case DirSend:
		buf = &c.sendBuf
	case DirRecv:
		buf = &c.recvBuf
	default:
		return nil, nil, false
	}

	owned := false
	if len(*buf) > 0 {
		*buf = append(*buf, data...)
		data = *buf
		owned = true
	}

	if dir == DirSend {
		switch {
		case bytes.HasPrefix(data, clientPreface):
			data = data[len(clientPreface):]
		case LooksLikePreface(data) && len(data) < len(clientPreface):
			if !truncated {
				keepLeftover(buf, data, owned)
			} else if owned {
				*buf = (*buf)[:0]
			}
			return nil, nil, false
		}
	}

	var dec *hpack.Decoder
	if dir == DirSend {
		dec = c.sendDec
	} else {
		dec = c.recvDec
	}

	pos := 0
	for pos < len(data) {
		if len(data)-pos < frameHeaderLen {
			c.cloneBlockFrags()
			if truncated {
				if owned {
					*buf = (*buf)[:0]
				}
				break
			}
			keepLeftover(buf, data[pos:], owned)
			break
		}
		hdr, ok := readFrameHeader(data[pos:])
		if !ok {
			*outstanding -= c.DropAll()
			return reqs, hops, true
		}
		total := frameHeaderLen + int(hdr.Length)
		if len(data)-pos < total {
			if truncated {
				if isFieldBlockType(hdr.Type) {
					*outstanding -= c.DropAll()
					return reqs, hops, true
				}
				c.cloneBlockFrags()
				if owned {
					*buf = (*buf)[:0]
				}
				break
			}
			leftover := data[pos:]
			if len(leftover) > incompleteCap {
				*outstanding -= c.DropAll()
				return reqs, hops, true
			}
			c.cloneBlockFrags()
			keepLeftover(buf, leftover, owned)
			break
		}
		hdr.Payload = data[pos+frameHeaderLen : pos+total]
		pos += total

		moreReqs, moreHops, frameStop := c.handleFrame(dir, dec, hdr, now, outstanding)
		reqs = append(reqs, moreReqs...)
		hops = append(hops, moreHops...)
		if frameStop {
			*outstanding -= c.DropAll()
			return reqs, hops, true
		}
	}
	if pos == len(data) {
		c.cloneBlockFrags()
		keepLeftover(buf, nil, owned)
	}
	return reqs, hops, false
}

func isFieldBlockType(t uint8) bool {
	return t == frameHeaders || t == framePushPromise || t == frameContinuation
}

// cloneBlockFrags copies pending field block fragments so they outlive Feed.
// Ringbuf data is invalid after the next Read. keepLeftover can overwrite an owned stitch buffer.
func (c *Conn) cloneBlockFrags() {
	if c.block == nil {
		return
	}
	for i, frag := range c.block.frags {
		c.block.frags[i] = bytes.Clone(frag)
	}
}

func (c *Conn) handleFrame(dir uint8, dec *hpack.Decoder, fr frame, now time.Time, outstanding *int) (reqs []Request, hops []Completion, stop bool) {
	switch fr.Type {
	case frameData, frameSettings, framePing, frameWindowUpdate, frameGoAway, framePriority:
		return nil, nil, false
	case frameRSTStream:
		if fr.StreamID != 0 {
			c.dropStream(fr.StreamID, outstanding)
		}
		return nil, nil, false
	case frameHeaders, framePushPromise:
		if c.block != nil {
			return nil, nil, true
		}
		endHeaders := fr.Flags&flagEndHeaders != 0
		endStream := fr.Flags&flagEndStream != 0
		frag, ok := headerBlockFragment(fr.Type, fr.Flags, fr.Payload)
		if !ok {
			return nil, nil, true
		}
		if endHeaders {
			return c.finishBlock(dir, dec, fr.Type, fr.StreamID, endStream, [][]byte{frag}, now, outstanding)
		}
		c.block = &pendingBlock{
			streamID:  fr.StreamID,
			frameType: fr.Type,
			endStream: endStream,
			frags:     [][]byte{frag},
		}
		return nil, nil, false
	case frameContinuation:
		if c.block == nil || c.block.streamID != fr.StreamID {
			return nil, nil, true
		}
		c.block.frags = append(c.block.frags, fr.Payload)
		if fr.Flags&flagEndHeaders == 0 {
			return nil, nil, false
		}
		block := c.block
		c.block = nil
		return c.finishBlock(dir, dec, block.frameType, block.streamID, block.endStream, block.frags, now, outstanding)
	default:
		return nil, nil, false
	}
}

func (c *Conn) finishBlock(
	dir uint8,
	dec *hpack.Decoder,
	frameType uint8,
	streamID uint32,
	endStream bool,
	frags [][]byte,
	now time.Time,
	outstanding *int,
) (reqs []Request, hops []Completion, stop bool) {
	fields, err := decodeBlock(dec, frags)
	if err != nil {
		return nil, nil, true
	}
	if frameType == framePushPromise {
		return nil, nil, false
	}

	headers := collectHeaders(fields)
	switch dir {
	case DirSend:
		req, ok := c.onRequestHeaders(streamID, headers, now)
		if ok {
			reqs = append(reqs, req)
		}
		return reqs, nil, false
	case DirRecv:
		hop, ok := c.onResponseHeaders(streamID, headers, endStream, now, outstanding)
		if ok {
			hops = append(hops, hop)
		}
		return nil, hops, false
	default:
		return nil, nil, false
	}
}

func decodeBlock(dec *hpack.Decoder, frags [][]byte) ([]hpack.HeaderField, error) {
	var fields []hpack.HeaderField
	dec.SetEmitFunc(func(field hpack.HeaderField) {
		fields = append(fields, field)
	})
	for _, frag := range frags {
		if _, err := dec.Write(frag); err != nil {
			return nil, err
		}
	}
	if err := dec.Close(); err != nil {
		return nil, err
	}
	return fields, nil
}

type headerSet struct {
	method       string
	path         string
	status       int
	contentType  string
	grpcStatus   int
	hasGRPCStat  bool
	traceparent  string
	grpcTraceBin string
}

func collectHeaders(fields []hpack.HeaderField) headerSet {
	var h headerSet
	h.status = -1
	h.grpcStatus = -1
	for _, f := range fields {
		switch f.Name {
		case ":method":
			h.method = f.Value
		case ":path":
			h.path = f.Value
		case ":status":
			if n, err := strconv.Atoi(f.Value); err == nil {
				h.status = n
			}
		case "content-type":
			h.contentType = strings.ToLower(f.Value)
		case "grpc-status":
			if n, err := strconv.Atoi(f.Value); err == nil {
				h.grpcStatus = n
				h.hasGRPCStat = true
			}
		case "traceparent":
			if h.traceparent == "" {
				h.traceparent = f.Value
			}
		case "grpc-trace-bin":
			if h.grpcTraceBin == "" {
				h.grpcTraceBin = f.Value
			}
		}
	}
	return h
}

func (c *Conn) onRequestHeaders(streamID uint32, h headerSet, now time.Time) (Request, bool) {
	if streamID == 0 || streamID%2 == 0 {
		return Request{}, false
	}
	if h.method == "" || h.path == "" {
		return Request{}, false
	}
	grpc := isGRPCContentType(h.contentType)
	if !grpc && !http1.EmitMethod(h.method) {
		return Request{}, false
	}
	if grpc && h.method != "POST" {
		return Request{}, false
	}
	return Request{
		StreamID:     streamID,
		Method:       h.method,
		Path:         h.path,
		GRPC:         grpc,
		Start:        now,
		Traceparent:  h.traceparent,
		GRPCTraceBin: h.grpcTraceBin,
	}, true
}

func (c *Conn) onResponseHeaders(streamID uint32, h headerSet, endStream bool, now time.Time, outstanding *int) (Completion, bool) {
	if streamID == 0 {
		return Completion{}, false
	}
	req := c.streams[streamID]
	if req == nil {
		return Completion{}, false
	}

	if isGRPCContentType(h.contentType) {
		req.GRPC = true
	}

	if req.GRPC {
		if h.hasGRPCStat {
			if !endStream {
				return Completion{}, false
			}
			statusCode := grpcStatusHTTP(h.grpcStatus)
			return c.finishHop(streamID, req, http1.StatusClass(statusCode), statusCode, &h.grpcStatus, now, outstanding)
		}
		if h.status >= 0 && h.status != 200 {
			return c.finishHop(streamID, req, http1.StatusClass(h.status), h.status, nil, now, outstanding)
		}
		if endStream {
			c.dropStream(streamID, outstanding)
			return Completion{}, false
		}
		return Completion{}, false
	}

	if h.status < 0 {
		return Completion{}, false
	}
	if h.status >= 100 && h.status < 200 {
		return Completion{}, false
	}
	return c.finishHop(streamID, req, http1.StatusClass(h.status), h.status, nil, now, outstanding)
}

func (c *Conn) finishHop(streamID uint32, req *Request, class string, statusCode int, grpcStatus *int, now time.Time, outstanding *int) (Completion, bool) {
	if class == "" {
		c.dropStream(streamID, outstanding)
		return Completion{}, false
	}
	c.dropStream(streamID, outstanding)
	return Completion{
		Key:          req.Key,
		Method:       req.Method,
		Route:        req.Route,
		StatusClass:  class,
		StatusCode:   statusCode,
		GRPC:         req.GRPC,
		GRPCStatus:   grpcStatus,
		Start:        req.Start,
		End:          now,
		TraceID:      req.TraceID,
		SpanID:       req.SpanID,
		ParentSpanID: req.ParentSpanID,
		Sampled:      req.Sampled,
	}, true
}

// Enqueue stores a pending stream request. replaceTLS replaces an existing
// pending hop for the same stream ID, keeping stamped IDs.
func (c *Conn) Enqueue(req Request, replaceTLS bool, outstanding *int, maxOutstanding int) {
	if existing := c.streams[req.StreamID]; existing != nil {
		if replaceTLS {
			req.TraceID = existing.TraceID
			req.SpanID = existing.SpanID
			req.ParentSpanID = existing.ParentSpanID
			req.Sampled = existing.Sampled
			c.streams[req.StreamID] = &req
		}
		return
	}
	if *outstanding >= maxOutstanding {
		return
	}
	c.streams[req.StreamID] = &req
	*outstanding++
}

func (c *Conn) dropStream(streamID uint32, outstanding *int) {
	if _, ok := c.streams[streamID]; !ok {
		return
	}
	*outstanding--
	delete(c.streams, streamID)
}

func (c *Conn) DropAll() int {
	outstanding := len(c.streams)
	clear(c.streams)
	c.block = nil
	c.sendBuf = nil
	c.recvBuf = nil
	c.emitStop = true
	return outstanding
}

func (c *Conn) GCExpired(now time.Time, idle time.Duration) int {
	dropped := 0
	for id, req := range c.streams {
		if now.Sub(req.Start) > idle {
			delete(c.streams, id)
			dropped++
		}
	}
	return dropped
}

func (c *Conn) Idle() bool {
	if c.emitStop {
		return true
	}
	return len(c.sendBuf) == 0 && len(c.recvBuf) == 0 && c.block == nil && len(c.streams) == 0
}
