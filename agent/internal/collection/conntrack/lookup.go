package conntrack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"

	"github.com/mdlayher/netlink"
	"github.com/ti-mo/conntrack"
	"github.com/ti-mo/netfilter"
	"golang.org/x/sys/unix"
)

const (
	lookupRecvBuffer = 64 << 10 // 64KB
	netlinkErrnoSize = 4        // nlmsgerr.error is int32
	ctGetMessage     = 1        // IPCTNL_MSG_CT_GET
)

// ti-mo leaves uapi attribute numbers unexported, so lookup marshals them here.
const (
	ctaTupleOrig  = 1
	ctaTupleIP    = 1
	ctaTupleProto = 2
	ctaIPv4Src    = 1
	ctaIPv4Dst    = 2
	ctaIPv6Src    = 3
	ctaIPv6Dst    = 4
	ctaProtoNum   = 1
	ctaProtoSport = 2
	ctaProtoDport = 3
)

// lookupFlow does a one-shot unicast CT_GET. ti-mo Conn.Get hangs: the kernel
// sets NLM_F_MULTI on successful unicast replies without NLMSG_DONE, and
// mdlayher Execute waits forever for DONE. Send with NLM_F_REQUEST only, then
// one recvmsg. Orig tuple only: the kernel looks up via CTA_TUPLE_ORIG.
// NewFlow would invent a fake inverted reply.
func (c *Client) lookupFlow(proto uint8, src, dst Endpoint) (conntrack.Flow, bool, error) {
	c.socketMu.Lock()
	conn := c.lookup
	c.socketMu.Unlock()
	if conn == nil {
		return conntrack.Flow{}, false, fmt.Errorf("conntrack lookup closed")
	}

	req, err := marshalOrigGet(proto, src.Addr, dst.Addr, src.Port, dst.Port)
	if err != nil {
		return conntrack.Flow{}, false, err
	}

	c.lookupMu.Lock()
	defer c.lookupMu.Unlock()

	c.socketMu.Lock()
	conn = c.lookup
	c.socketMu.Unlock()
	if conn == nil {
		return conntrack.Flow{}, false, fmt.Errorf("conntrack lookup closed")
	}

	sent, err := conn.Send(req)
	if err != nil {
		return conntrack.Flow{}, false, fmt.Errorf("send conntrack get: %w", err)
	}

	raw, err := recvOne(conn)
	if err != nil {
		return c.redialAndReturn(fmt.Errorf("recv conntrack get: %w", err))
	}

	var msg netlink.Message
	if err := msg.UnmarshalBinary(raw); err != nil {
		return c.redialAndReturn(fmt.Errorf("unmarshal conntrack get: %w", err))
	}
	if msg.Header.Sequence != 0 && sent.Header.Sequence != 0 && msg.Header.Sequence != sent.Header.Sequence {
		return c.redialAndReturn(fmt.Errorf("conntrack get sequence mismatch"))
	}

	if msg.Header.Type == netlink.Error {
		errno, err := netlinkErrno(msg)
		if err != nil {
			return c.redialAndReturn(err)
		}
		if errno == unix.ENOENT {
			return conntrack.Flow{}, false, nil
		}
		return conntrack.Flow{}, false, fmt.Errorf("conntrack get: %w", errno)
	}

	if msg.Header.Type == netlink.Done {
		return c.redialAndReturn(fmt.Errorf("conntrack get: unexpected NLMSG_DONE"))
	}

	var ev conntrack.Event
	if err := ev.Unmarshal(msg); err != nil {
		return c.redialAndReturn(fmt.Errorf("unmarshal conntrack flow: %w", err))
	}
	if ev.Flow == nil {
		return c.redialAndReturn(fmt.Errorf("conntrack get: empty flow"))
	}
	return *ev.Flow, true, nil
}

// failLookupRedial redials after a desynced reply. Caller holds lookupMu.
func (c *Client) redialAndReturn(err error) (conntrack.Flow, bool, error) {
	if redialErr := c.redialLookupLocked(); redialErr != nil {
		return conntrack.Flow{}, false, errors.Join(err, redialErr)
	}
	return conntrack.Flow{}, false, err
}

// redialLookupLocked replaces a desynced lookup socket. Caller holds lookupMu.
func (c *Client) redialLookupLocked() error {
	c.socketMu.Lock()
	old := c.lookup
	c.lookup = nil
	c.socketMu.Unlock()
	var closeErr error
	if old != nil {
		closeErr = old.Close()
	}
	conn, err := netlink.Dial(unix.NETLINK_NETFILTER, &netlink.Config{})
	if err != nil {
		return errors.Join(fmt.Errorf("redial conntrack lookup: %w", err), closeErr)
	}
	c.socketMu.Lock()
	c.lookup = conn
	c.socketMu.Unlock()
	return closeErr
}

func marshalOrigGet(proto uint8, src, dst netip.Addr, srcPort, dstPort uint16) (netlink.Message, error) {
	src = src.Unmap()
	dst = dst.Unmap()
	if !src.IsValid() || !dst.IsValid() {
		return netlink.Message{}, fmt.Errorf("invalid lookup addresses")
	}

	var ipChildren []netfilter.Attribute
	family := netfilter.ProtoIPv4
	switch {
	case src.Is4() && dst.Is4():
		ipChildren = []netfilter.Attribute{
			{Type: ctaIPv4Src, Data: src.AsSlice()},
			{Type: ctaIPv4Dst, Data: dst.AsSlice()},
		}
	case src.Is6() && dst.Is6():
		family = netfilter.ProtoIPv6
		ipChildren = []netfilter.Attribute{
			{Type: ctaIPv6Src, Data: src.AsSlice()},
			{Type: ctaIPv6Dst, Data: dst.AsSlice()},
		}
	default:
		return netlink.Message{}, fmt.Errorf("mixed address families")
	}

	srcPortAttr := netfilter.Attribute{Type: ctaProtoSport}
	srcPortAttr.PutUint16(srcPort)
	dstPortAttr := netfilter.Attribute{Type: ctaProtoDport}
	dstPortAttr.PutUint16(dstPort)

	attrs := []netfilter.Attribute{{
		Type:   ctaTupleOrig,
		Nested: true,
		Children: []netfilter.Attribute{
			{Type: ctaTupleIP, Nested: true, Children: ipChildren},
			{
				Type:   ctaTupleProto,
				Nested: true,
				Children: []netfilter.Attribute{
					{Type: ctaProtoNum, Data: []byte{proto}},
					srcPortAttr,
					dstPortAttr,
				},
			},
		},
	}}

	return netfilter.MarshalNetlink(netfilter.Header{
		SubsystemID: netfilter.NFSubsysCTNetlink,
		MessageType: netfilter.MessageType(ctGetMessage),
		Family:      family,
		Flags:       netlink.Request,
	}, attrs)
}

func recvOne(conn *netlink.Conn) ([]byte, error) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, lookupRecvBuffer)
	var (
		bytesRead int
		recvErr   error
	)
	if err := rawConn.Read(func(fd uintptr) bool {
		var errno error
		bytesRead, _, errno = unix.Recvfrom(int(fd), buf, 0)
		if errno == unix.EAGAIN || errno == unix.EWOULDBLOCK {
			return false
		}
		if errno != nil {
			recvErr = errno
			return true
		}
		return true
	}); err != nil {
		return nil, err
	}
	if recvErr != nil {
		return nil, recvErr
	}
	if bytesRead <= 0 {
		return nil, fmt.Errorf("empty recv")
	}
	return buf[:bytesRead], nil
}

func netlinkErrno(msg netlink.Message) (unix.Errno, error) {
	if len(msg.Data) < netlinkErrnoSize {
		return 0, fmt.Errorf("short netlink error")
	}
	code := int32(binary.NativeEndian.Uint32(msg.Data[:netlinkErrnoSize]))
	if code == 0 {
		return 0, nil
	}
	return unix.Errno(-code), nil
}
