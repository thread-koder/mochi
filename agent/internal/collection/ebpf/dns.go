package ebpf

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cilium/ebpf/ringbuf"
	"github.com/thread_koder/mochi/agent/internal/collection/dns"
	"github.com/thread_koder/mochi/agent/internal/collection/identity"
	"github.com/thread_koder/mochi/agent/internal/logger"
)

// Must match DNS_PAYLOAD_MAX in bpf/dns_recv.c (RFC 9715: 1280−40−8).
const dnsPayloadMax = 1232

// dnsWireEvent matches struct event in bpf/dns_recv.c.
type dnsWireEvent struct {
	Pid      uint32
	Len      uint32
	CgroupID uint64
	IsTCP    uint8
	_        [3]byte
	Data     [dnsPayloadMax]byte
}

var dnsWireEventSize = binary.Size(dnsWireEvent{})

func (c *Collector) runDNS(ctx context.Context) {
	log := logger.WithComponent("ebpf-dns")
	go func() {
		<-ctx.Done()
		_ = c.dnsEvents.Close()
	}()

	for {
		record, err := c.dnsEvents.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				return
			}
			log.Error().Err(err).Msg("Failed to read DNS ringbuf")
			continue
		}
		c.handleDNSRecord(record.RawSample)
	}
}

func (c *Collector) handleDNSRecord(raw []byte) {
	event, err := parseDNSWireEvent(raw)
	if err != nil {
		return
	}
	if int(event.Len) > len(event.Data) {
		return
	}

	// Resolve before unpack so host / non-pod DNS skips miekg.
	pod, ok := c.resolver.Resolve(event.Pid, event.CgroupID)
	if !ok {
		return
	}

	payload := event.Data[:event.Len]
	if event.IsTCP != 0 {
		payload, ok = dns.StripTCPLength(payload)
		if !ok {
			return
		}
	}

	answer, ok := dns.ParseResponse(payload)
	if !ok {
		return
	}

	for _, ip := range answer.IPs {
		c.dnsCache.Store(pod.UID, identity.AddrKey(ip), answer.QName, answer.TTL)
	}
}

func parseDNSWireEvent(raw []byte) (dnsWireEvent, error) {
	if len(raw) < dnsWireEventSize {
		return dnsWireEvent{}, fmt.Errorf("event too short: %d", len(raw))
	}
	var event dnsWireEvent
	if err := binary.Read(bytes.NewReader(raw), binary.LittleEndian, &event); err != nil {
		return dnsWireEvent{}, fmt.Errorf("decode event: %w", err)
	}
	return event, nil
}
