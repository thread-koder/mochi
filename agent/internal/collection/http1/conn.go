package http1

import (
	"time"

	"github.com/thread_koder/mochi/agent/internal/metrics"
	"github.com/thread_koder/mochi/agent/internal/span"
)

const (
	DirRecv = 0
	DirSend = 1
)

type Request struct {
	Method       string
	Path         string
	Route        string
	Key          metrics.SeriesKey
	Start        time.Time
	Traceparent  string
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
	Start        time.Time
	End          time.Time
	TraceID      [span.TraceIDSize]byte
	SpanID       [span.SpanIDSize]byte
	ParentSpanID [span.SpanIDSize]byte
	Sampled      bool
}

type Conn struct {
	sendBuf     []byte
	recvBuf     []byte
	outstanding []Request
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

// Feed parses one direction of a prefix. opaque stops HTTP/1 on this flow.
// outstanding is decremented when a queued request is consumed (including 101).
// Returned requests have Method/Path/Traceparent/Start. Caller templates Route and Enqueues.
func (c *Conn) Feed(dir uint8, data []byte, now time.Time, outstanding *int) (reqs []Request, hops []Completion, opaque bool) {
	var buf *[]byte
	switch dir {
	case DirSend:
		buf = &c.sendBuf
	case DirRecv:
		buf = &c.recvBuf
	default:
		return nil, nil, false
	}

	if len(*buf) == 0 && !LooksLikeStart(data) {
		return nil, nil, false
	}

	owned := false
	if len(*buf) > 0 {
		*buf = append(*buf, data...)
		data = *buf
		owned = true
	}

	msgs, leftover, stop := ParsePrefix(data)
	keepLeftover(buf, leftover, owned)
	if stop {
		opaque = true
	}

	switch dir {
	case DirSend:
		for _, msg := range msgs {
			if !msg.Request {
				continue
			}
			if msg.Opaque || !EmitMethod(msg.Method) {
				if msg.Opaque {
					opaque = true
				}
				continue
			}
			reqs = append(reqs, Request{
				Method:      msg.Method,
				Path:        msg.Path,
				Traceparent: msg.Traceparent,
				Start:       now,
			})
		}
	case DirRecv:
		for _, msg := range msgs {
			if msg.Request {
				continue
			}
			if hop, ok := c.pairResponse(msg, now, outstanding); ok {
				hops = append(hops, hop)
			}
			if msg.Opaque {
				opaque = true
			}
		}
	}

	if opaque {
		c.sendBuf = nil
		c.recvBuf = nil
	}
	return reqs, hops, opaque
}

// Enqueue appends a pending request. replaceTLS replaces the last matching
// method+route when a TLS kind revisits a socket queued hop, keeping stamped IDs.
func (c *Conn) Enqueue(req Request, replaceTLS bool, outstanding *int, maxOutstanding int) {
	if replaceTLS {
		if n := len(c.outstanding); n > 0 {
			last := &c.outstanding[n-1]
			if last.Method == req.Method && last.Route == req.Route {
				traceID, spanID, parentID, sampled := last.TraceID, last.SpanID, last.ParentSpanID, last.Sampled
				*last = req
				last.TraceID = traceID
				last.SpanID = spanID
				last.ParentSpanID = parentID
				last.Sampled = sampled
				return
			}
		}
	}
	if *outstanding >= maxOutstanding {
		return
	}
	c.outstanding = append(c.outstanding, req)
	*outstanding++
}

func (c *Conn) pairResponse(msg Message, now time.Time, outstanding *int) (Completion, bool) {
	if len(c.outstanding) == 0 {
		return Completion{}, false
	}
	if msg.Status >= 100 && msg.Status < 200 && msg.Status != 101 {
		return Completion{}, false
	}

	req := c.outstanding[0]
	c.outstanding = c.outstanding[1:]
	*outstanding--

	if msg.Status == 101 {
		return Completion{}, false
	}
	class := StatusClass(msg.Status)
	if class == "" {
		return Completion{}, false
	}
	return Completion{
		Key:          req.Key,
		Method:       req.Method,
		Route:        req.Route,
		StatusClass:  class,
		StatusCode:   msg.Status,
		Start:        req.Start,
		End:          now,
		TraceID:      req.TraceID,
		SpanID:       req.SpanID,
		ParentSpanID: req.ParentSpanID,
		Sampled:      req.Sampled,
	}, true
}

func (c *Conn) DropOutstanding() int {
	n := len(c.outstanding)
	c.outstanding = nil
	c.sendBuf = nil
	c.recvBuf = nil
	return n
}

func (c *Conn) GCExpired(now time.Time, idle time.Duration) int {
	kept := c.outstanding[:0]
	dropped := 0
	for _, req := range c.outstanding {
		if now.Sub(req.Start) > idle {
			dropped++
			continue
		}
		kept = append(kept, req)
	}
	c.outstanding = kept
	return dropped
}

func (c *Conn) Idle() bool {
	return len(c.outstanding) == 0 && len(c.sendBuf) == 0 && len(c.recvBuf) == 0
}
