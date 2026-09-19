package ebpf

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cilium/ebpf/ringbuf"
	"github.com/thread_koder/mochi/agent/internal/collection/http1"
	"github.com/thread_koder/mochi/agent/internal/logger"
)

// Packed stream_hdr in bpf/stream_hdr.h is 56 bytes. STREAM_CAP is 1024.
const (
	streamHdrSize    = 56
	streamPayloadMax = 1024
)

// streamWireEvent matches struct stream_event in bpf/stream_hdr.h.
// Data is a subslice of the ringbuf sample (valid until the next Read).
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
	Data     []byte
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
		c.handleStreamRecord(record.RawSample)
	}
}

func (c *Collector) handleStreamRecord(raw []byte) {
	event, err := parseStreamWireEvent(raw)
	if err != nil {
		return
	}
	c.feedHTTP1(event)
}

func (c *Collector) feedHTTP1(event streamWireEvent) {
	src, ok := addrFromEvent(event.Family, event.Saddr[:])
	if !ok {
		return
	}
	dst, ok := addrFromEvent(event.Family, event.Daddr[:])
	if !ok {
		return
	}
	c.http1.Handle(http1.Chunk{
		Pid:      event.Pid,
		CgroupID: event.CgroupID,
		Family:   event.Family,
		Sport:    event.Sport,
		Dport:    event.Dport,
		Dir:      event.Dir,
		Kind:     event.Kind,
		Src:      src,
		Dst:      dst,
		Data:     event.Data,
	})
}

func parseStreamWireEvent(raw []byte) (streamWireEvent, error) {
	if len(raw) < streamHdrSize {
		return streamWireEvent{}, fmt.Errorf("event too short: %d", len(raw))
	}
	length := binary.LittleEndian.Uint32(raw[4:8])
	if length > streamPayloadMax || streamHdrSize+int(length) > len(raw) {
		return streamWireEvent{}, fmt.Errorf("event payload len %d", length)
	}
	var saddr, daddr [16]byte
	copy(saddr[:], raw[24:40])
	copy(daddr[:], raw[40:56])
	return streamWireEvent{
		Pid:      binary.LittleEndian.Uint32(raw[0:4]),
		Len:      length,
		CgroupID: binary.LittleEndian.Uint64(raw[8:16]),
		Family:   binary.LittleEndian.Uint16(raw[16:18]),
		Sport:    binary.LittleEndian.Uint16(raw[18:20]),
		Dport:    binary.LittleEndian.Uint16(raw[20:22]),
		Dir:      raw[22],
		Kind:     raw[23],
		Saddr:    saddr,
		Daddr:    daddr,
		Data:     raw[streamHdrSize : streamHdrSize+int(length)],
	}, nil
}
