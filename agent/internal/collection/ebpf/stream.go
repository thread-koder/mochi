package ebpf

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cilium/ebpf/ringbuf"
	"github.com/rs/zerolog"
	"github.com/thread_koder/mochi/agent/internal/collection/conntrack"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/thread_koder/mochi/agent/internal/metrics"
)

// Must match STREAM_CAP in bpf/tcp_stream.c.
const streamPayloadMax = 1024

const (
	streamDirRecv = 0
	streamDirSend = 1

	// Must match STREAM_KIND_* in bpf/stream_hdr.h.
	streamKindSocket  = 0
	streamKindOpenSSL = 1
	streamKindGoTLS   = 2
)

// streamWireEvent matches struct stream_event in bpf/stream_hdr.h.
// Kind is 0 on the socket path. TLS sets openssl or gotls.
type streamWireEvent struct {
	Pid      uint32
	Len      uint32
	CgroupID uint64
	Family   uint16
	Sport    uint16
	Dport    uint16
	Dir      uint8
	Kind     uint8
	Saddr    [16]byte
	Daddr    [16]byte
	Data     [streamPayloadMax]byte
}

var streamWireEventSize = binary.Size(streamWireEvent{})

var http1Prefixes = [][]byte{
	[]byte("GET "),
	[]byte("POST "),
	[]byte("PUT "),
	[]byte("HEAD "),
	[]byte("DELETE "),
	[]byte("PATCH "),
	[]byte("OPTIONS "),
	[]byte("CONNECT "),
	[]byte("HTTP/1"),
}

func (c *Collector) runStream(ctx context.Context) {
	log := logger.WithComponent("ebpf-stream")
	go func() {
		<-ctx.Done()
		_ = c.streamEvents.Close()
	}()

	for {
		record, err := c.streamEvents.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				return
			}
			log.Error().Err(err).Msg("Failed to read TCP stream ringbuf")
			continue
		}
		c.handleStreamRecord(log, record.RawSample)
	}
}

func (c *Collector) handleStreamRecord(log zerolog.Logger, raw []byte) {
	if zerolog.GlobalLevel() > zerolog.DebugLevel {
		return
	}

	event, err := parseStreamWireEvent(raw)
	if err != nil {
		return
	}
	c.dumpHTTP1(log, "TCP stream", "", event)
}

func (c *Collector) dumpHTTP1(log zerolog.Logger, msg, via string, event streamWireEvent) {
	if int(event.Len) > len(event.Data) {
		return
	}
	payload := event.Data[:event.Len]
	if !http1Prefix(payload) {
		return
	}

	src, ok := addrFromEvent(event.Family, event.Saddr[:])
	if !ok {
		return
	}
	dst, ok := addrFromEvent(event.Family, event.Daddr[:])
	if !ok {
		return
	}
	if !src.IsValid() || src.IsUnspecified() || !dst.IsValid() || dst.IsUnspecified() {
		return
	}
	if dst.Unmap().IsLoopback() {
		return
	}
	if event.Pid == 0 && event.CgroupID == 0 {
		return
	}
	pod, ok := c.resolver.Resolve(event.Pid, event.CgroupID)
	if !ok {
		return
	}

	actualAddr, actualPort := c.conntrackClient.ActualDst(
		conntrack.IPProtocol(metrics.ProtocolTCP),
		conntrack.Endpoint{Addr: src, Port: event.Sport},
		conntrack.Endpoint{Addr: dst, Port: event.Dport},
	)

	dir := "recv"
	if event.Dir == streamDirSend {
		dir = "send"
	}
	eventLog := log.Debug().
		Str("src_pod_uid", pod.UID).
		Str("src_namespace", pod.Namespace).
		Str("src_pod", pod.Name).
		Str("src", src.String()).
		Uint16("sport", event.Sport).
		Str("dst", dst.String()).
		Uint16("dport", event.Dport).
		Str("actual_dst", actualAddr.String()).
		Uint16("actual_dport", actualPort).
		Str("dir", dir)
	if via != "" {
		eventLog = eventLog.Str("via", via)
	}
	eventLog.
		Str("payload", printablePrefix(payload, 256)).
		Msg(msg)
}

func parseStreamWireEvent(raw []byte) (streamWireEvent, error) {
	if len(raw) < streamWireEventSize {
		return streamWireEvent{}, fmt.Errorf("event too short: %d", len(raw))
	}
	var event streamWireEvent
	if err := binary.Read(bytes.NewReader(raw), binary.LittleEndian, &event); err != nil {
		return streamWireEvent{}, fmt.Errorf("decode event: %w", err)
	}
	return event, nil
}

func http1Prefix(payload []byte) bool {
	for _, prefix := range http1Prefixes {
		if bytes.HasPrefix(payload, prefix) {
			return true
		}
	}
	return false
}

func printablePrefix(payload []byte, n int) string {
	if len(payload) > n {
		payload = payload[:n]
	}
	var result bytes.Buffer
	result.Grow(len(payload))
	for _, b := range payload {
		switch {
		case b == '\r' || b == '\n' || b == '\t' || (b >= 0x20 && b <= 0x7e):
			result.WriteByte(b)
		default:
			result.WriteByte('.')
		}
	}
	return result.String()
}
