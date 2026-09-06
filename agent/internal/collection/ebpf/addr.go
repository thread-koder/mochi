package ebpf

import (
	"net/netip"
)

const (
	afInet  = 2
	afInet6 = 10
)

func addrFromFamily(family uint8, raw []byte) (netip.Addr, bool) {
	switch family {
	case afInet:
		var b [4]byte
		copy(b[:], raw[:4])
		return netip.AddrFrom4(b), true
	case afInet6:
		var b [16]byte
		copy(b[:], raw[:16])
		return netip.AddrFrom16(b), true
	default:
		return netip.Addr{}, false
	}
}

func addrFromEvent(family uint16, raw []byte) (netip.Addr, bool) {
	return addrFromFamily(uint8(family), raw)
}
