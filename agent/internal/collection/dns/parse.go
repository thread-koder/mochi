package dns

import (
	"net/netip"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// Presentation-form FQDN max length (RFC 1035).
const maxHostnameLen = 253

// Answer is one dialed QNAME with the A/AAAA addresses from a DNS response.
type Answer struct {
	QName string
	IPs   []netip.Addr
	TTL   time.Duration
}

// ParseResponse unpacks a client DNS response and returns QNAME + A/AAAA when useful.
// TCP payloads must already have the 2-byte length prefix stripped.
func ParseResponse(payload []byte) (Answer, bool) {
	if len(payload) < 12 {
		return Answer{}, false
	}

	var msg dns.Msg
	if err := msg.Unpack(payload); err != nil {
		return Answer{}, false
	}
	if !msg.Response || msg.Rcode != dns.RcodeSuccess {
		return Answer{}, false
	}
	if len(msg.Question) == 0 {
		return Answer{}, false
	}

	qname := normalizeHostname(msg.Question[0].Name)
	if qname == "" {
		return Answer{}, false
	}

	var (
		ips    []netip.Addr
		minTTL uint32
	)
	for _, rr := range msg.Answer {
		switch a := rr.(type) {
		case *dns.A:
			addr, ok := addrFromIP(a.A)
			if !ok {
				continue
			}
			ips = append(ips, addr)
			if len(ips) == 1 || a.Hdr.Ttl < minTTL {
				minTTL = a.Hdr.Ttl
			}
		case *dns.AAAA:
			addr, ok := addrFromIP(a.AAAA)
			if !ok {
				continue
			}
			ips = append(ips, addr)
			if len(ips) == 1 || a.Hdr.Ttl < minTTL {
				minTTL = a.Hdr.Ttl
			}
		}
	}
	if len(ips) == 0 {
		return Answer{}, false
	}

	return Answer{
		QName: qname,
		IPs:   ips,
		TTL:   time.Duration(minTTL) * time.Second,
	}, true
}

// StripTCPLength validates TCP DNS framing (RFC 1035) and returns the message bytes.
func StripTCPLength(payload []byte) ([]byte, bool) {
	if len(payload) < 2 {
		return nil, false
	}
	msgLen := int(payload[0])<<8 | int(payload[1])
	if msgLen <= 0 || len(payload) < 2+msgLen {
		return nil, false
	}
	return payload[2 : 2+msgLen], true
}

func addrFromIP(ip []byte) (netip.Addr, bool) {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return netip.Addr{}, false
	}
	addr = addr.Unmap()
	if !addr.IsValid() || addr.IsUnspecified() || addr.IsLoopback() {
		return netip.Addr{}, false
	}
	return addr, true
}

func normalizeHostname(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".")
	name = strings.ToLower(name)
	if name == "" || len(name) > maxHostnameLen {
		return ""
	}
	if strings.ContainsAny(name, " \t\r\n") {
		return ""
	}
	return name
}
