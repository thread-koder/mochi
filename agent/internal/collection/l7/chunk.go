package l7

import "net/netip"

const (
	DirRecv = 0
	DirSend = 1

	KindSocket  = 0
	KindOpenSSL = 1
	KindGoTLS   = 2

	streamCap = 1024
)

type Chunk struct {
	Pid      uint32
	CgroupID uint64
	Sport    uint16
	Dport    uint16
	Dir      uint8
	Kind     uint8
	Src      netip.Addr
	Dst      netip.Addr
	Data     []byte
}

func (c Chunk) truncated() bool {
	return len(c.Data) >= streamCap
}

func (c Chunk) fromTLS() bool {
	return c.Kind == KindOpenSSL || c.Kind == KindGoTLS
}
