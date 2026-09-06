package ebpf

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cilium/ebpf/ringbuf"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/thread_koder/mochi/agent/internal/metrics"
)

const (
	tcpEstablished = 1
	tcpSynSent     = 2
)

// tcpWireEvent matches struct event in bpf/tcp_state.c.
type tcpWireEvent struct {
	Pid      uint32
	Family   uint16
	Sport    uint16
	Dport    uint16
	Oldstate uint8
	Newstate uint8
	_        [2]byte
	Saddr    [16]byte
	Daddr    [16]byte
	TxBytes  uint64
	RxBytes  uint64
	CgroupID uint64
}

var tcpWireEventSize = binary.Size(tcpWireEvent{})

func (c *Collector) runTCP(ctx context.Context) {
	log := logger.WithComponent("ebpf-tcp")
	go func() {
		<-ctx.Done()
		_ = c.tcpEvents.Close()
	}()

	for {
		record, err := c.tcpEvents.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				return
			}
			log.Error().Err(err).Msg("Failed to read TCP ringbuf")
			continue
		}
		c.handleTCPRecord(record.RawSample)
	}
}

func (c *Collector) handleTCPRecord(raw []byte) {
	event, err := parseTCPWireEvent(raw)
	if err != nil {
		return
	}

	srcAddr, ok := addrFromEvent(event.Family, event.Saddr[:])
	if !ok {
		return
	}
	dstAddr, ok := addrFromEvent(event.Family, event.Daddr[:])
	if !ok {
		return
	}

	tx := float64(event.TxBytes)
	rx := float64(event.RxBytes)

	var kind flowEventKind
	switch {
	case event.Oldstate == tcpSynSent && event.Newstate == tcpEstablished:
		kind = flowOpen
	case event.Oldstate == tcpEstablished && event.Newstate != tcpEstablished:
		kind = flowClose
	default:
		return
	}

	c.observeFlow(
		metrics.ProtocolTCP,
		event.Pid,
		event.CgroupID,
		srcAddr,
		dstAddr,
		event.Sport,
		event.Dport,
		tx,
		rx,
		kind,
	)
}

func parseTCPWireEvent(raw []byte) (tcpWireEvent, error) {
	if len(raw) < tcpWireEventSize {
		return tcpWireEvent{}, fmt.Errorf("event too short: %d", len(raw))
	}
	var event tcpWireEvent
	if err := binary.Read(bytes.NewReader(raw), binary.LittleEndian, &event); err != nil {
		return tcpWireEvent{}, fmt.Errorf("decode event: %w", err)
	}
	return event, nil
}
