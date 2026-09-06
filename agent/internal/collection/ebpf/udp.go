package ebpf

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/thread_koder/mochi/agent/internal/metrics"
	"golang.org/x/sys/unix"
)

const (
	udpIdleTimeout = 30 * time.Second
	udpGCInterval  = 5 * time.Second
)

// udpOpenEvent matches struct open_event in bpf/udp_flow.c.
type udpOpenEvent struct {
	Pid      uint32
	Family   uint16
	Sport    uint16
	Dport    uint16
	_        [2]byte
	Saddr    [16]byte
	Daddr    [16]byte
	CgroupID uint64
}

var udpOpenEventSize = binary.Size(udpOpenEvent{})

func (c *Collector) runUDP(ctx context.Context) {
	log := logger.WithComponent("ebpf-udp")
	go func() {
		<-ctx.Done()
		_ = c.udpEvents.Close()
	}()

	for {
		record, err := c.udpEvents.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				return
			}
			log.Error().Err(err).Msg("Failed to read UDP ringbuf")
			continue
		}
		c.handleUDPOpen(record.RawSample)
	}
}

func (c *Collector) handleUDPOpen(raw []byte) {
	event, err := parseUDPOpenEvent(raw)
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

	c.observeFlow(
		metrics.ProtocolUDP,
		event.Pid,
		event.CgroupID,
		srcAddr,
		dstAddr,
		event.Sport,
		event.Dport,
		0,
		0,
		flowOpen,
	)
}

func parseUDPOpenEvent(raw []byte) (udpOpenEvent, error) {
	if len(raw) < udpOpenEventSize {
		return udpOpenEvent{}, fmt.Errorf("event too short: %d", len(raw))
	}
	var event udpOpenEvent
	if err := binary.Read(bytes.NewReader(raw), binary.LittleEndian, &event); err != nil {
		return udpOpenEvent{}, fmt.Errorf("decode event: %w", err)
	}
	return event, nil
}

func (c *Collector) runUDPIdleGC(ctx context.Context) {
	log := logger.WithComponent("ebpf-udp")
	ticker := time.NewTicker(udpGCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.gcIdleUDPFlows(); err != nil {
				log.Error().Err(err).Msg("UDP idle flow GC failed")
			}
		}
	}
}

func monotonicNowNs() (uint64, error) {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return 0, fmt.Errorf("clock_gettime CLOCK_MONOTONIC: %w", err)
	}
	return uint64(ts.Sec)*1e9 + uint64(ts.Nsec), nil
}

func (c *Collector) gcIdleUDPFlows() error {
	if c.udpObjs.Flows == nil {
		return nil
	}

	nowNs, err := monotonicNowNs()
	if err != nil {
		return err
	}
	cutoff := nowNs - uint64(udpIdleTimeout.Nanoseconds())

	var (
		key     udpflowFlowKey
		val     udpflowFlowVal
		toClose []struct {
			key udpflowFlowKey
			val udpflowFlowVal
		}
	)
	iter := c.udpObjs.Flows.Iterate()
	for iter.Next(&key, &val) {
		if val.LastNs >= cutoff {
			continue
		}
		toClose = append(toClose, struct {
			key udpflowFlowKey
			val udpflowFlowVal
		}{key: key, val: val})
	}
	if err := iter.Err(); err != nil {
		return err
	}

	for _, flow := range toClose {
		c.closeUDPFlow(flow.key, flow.val)
		if err := c.udpObjs.Flows.Delete(&flow.key); err != nil {
			if errors.Is(err, ebpf.ErrKeyNotExist) {
				continue
			}
			return fmt.Errorf("delete idle UDP flow: %w", err)
		}
	}
	return nil
}

func (c *Collector) closeUDPFlow(key udpflowFlowKey, val udpflowFlowVal) {
	srcAddr, ok := addrFromFamily(key.Family, key.Saddr[:])
	if !ok {
		return
	}
	dstAddr, ok := addrFromFamily(key.Family, key.Daddr[:])
	if !ok {
		return
	}

	c.observeFlow(
		metrics.ProtocolUDP,
		val.Pid,
		val.CgroupId,
		srcAddr,
		dstAddr,
		key.Sport,
		key.Dport,
		float64(val.TxBytes),
		0,
		flowClose,
	)
}
