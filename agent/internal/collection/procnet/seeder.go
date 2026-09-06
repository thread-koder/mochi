package procnet

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thread_koder/mochi/agent/internal/collection/aggregate"
	"github.com/thread_koder/mochi/agent/internal/collection/conntrack"
	"github.com/thread_koder/mochi/agent/internal/collection/dns"
	"github.com/thread_koder/mochi/agent/internal/collection/identity"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/thread_koder/mochi/agent/internal/metrics"
)

const (
	connectedHex = "01"
	tcpListenHex = "0A"
)

type tableKind int

const (
	tableTCP tableKind = iota
	tableUDP
)

type tableSpec struct {
	name string
	v6   bool
	kind tableKind
}

var procTables = []tableSpec{
	{name: "tcp", v6: false, kind: tableTCP},
	{name: "tcp6", v6: true, kind: tableTCP},
	{name: "udp", v6: false, kind: tableUDP},
	{name: "udp6", v6: true, kind: tableUDP},
}

// Seeder walks host PIDs / netns and reconciles active L4 sockets into the store.
type Seeder struct {
	store           *aggregate.Store
	resolver        *identity.Resolver
	conntrackClient *conntrack.Client
	listen          *ListenIndex
	dnsCache        *dns.Cache
}

func NewSeeder(
	store *aggregate.Store,
	resolver *identity.Resolver,
	conntrackClient *conntrack.Client,
	listen *ListenIndex,
	dnsCache *dns.Cache,
) *Seeder {
	return &Seeder{
		store:           store,
		resolver:        resolver,
		conntrackClient: conntrackClient,
		listen:          listen,
		dnsCache:        dnsCache,
	}
}

func (s *Seeder) Start(ctx context.Context, interval time.Duration) {
	s.resync()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.resync()
		}
	}
}

func (s *Seeder) resync() {
	log := logger.WithComponent("procnet")
	sockets, boundUDP, err := s.collectActive()
	if err != nil {
		log.Error().Err(err).Msg("Failed to reconcile /proc L4 sockets")
		return
	}
	s.listen.Replace(boundUDP)
	s.store.ReconcileSeedActives(sockets)
	log.Debug().Int("sockets", len(sockets)).Msg("Reconciled active sockets from /proc")
}

func (s *Seeder) collectActive() ([]aggregate.SeedSocket, map[string]map[uint16]struct{}, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, nil, fmt.Errorf("read /proc: %w", err)
	}

	hostNetNS, err := os.Readlink("/proc/self/ns/net")
	if err != nil {
		return nil, nil, fmt.Errorf("read host netns: %w", err)
	}

	seenNetNS := make(map[string]struct{})
	var sockets []aggregate.SeedSocket
	boundUDP := make(map[string]map[uint16]struct{})

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			continue
		}
		nsPath := filepath.Join("/proc", entry.Name(), "ns", "net")
		nsID, err := os.Readlink(nsPath)
		if err != nil {
			continue
		}
		if nsID == hostNetNS {
			continue
		}
		if _, seen := seenNetNS[nsID]; seen {
			continue
		}

		pod, ok := s.resolver.ResolvePID(uint32(pid))
		if !ok {
			continue
		}
		seenNetNS[nsID] = struct{}{}

		for _, spec := range procTables {
			path := filepath.Join("/proc", entry.Name(), "net", spec.name)
			parsed, listenPorts, _ := s.parseTable(path, spec.v6, spec.kind, pod)
			sockets = append(sockets, parsed...)
			if spec.kind == tableUDP && len(listenPorts) > 0 {
				if boundUDP[pod.UID] == nil {
					boundUDP[pod.UID] = make(map[uint16]struct{})
				}
				for port := range listenPorts {
					boundUDP[pod.UID][port] = struct{}{}
				}
			}
		}
	}
	return sockets, boundUDP, nil
}

func (s *Seeder) parseTable(
	path string,
	v6 bool,
	kind tableKind,
	pod identity.PodInfo,
) ([]aggregate.SeedSocket, map[uint16]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, nil, scanner.Err()
	}
	var rows [][]string
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		rows = append(rows, fields)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	var listenPorts map[uint16]struct{}
	switch kind {
	case tableTCP:
		listenPorts = tcpListenPorts(rows, v6)
	case tableUDP:
		listenPorts = udpListenPorts(rows, v6)
	}

	var sockets []aggregate.SeedSocket
	for _, fields := range rows {
		sock, ok := seedSocketFromRow(fields, v6, kind, listenPorts, pod, s)
		if !ok {
			continue
		}
		sockets = append(sockets, sock)
	}
	return sockets, listenPorts, nil
}

func tcpListenPorts(rows [][]string, v6 bool) map[uint16]struct{} {
	ports := make(map[uint16]struct{})
	for _, fields := range rows {
		if fields[3] != tcpListenHex {
			continue
		}
		_, localPort, err := parseAddr(fields[1], v6)
		if err != nil {
			continue
		}
		ports[localPort] = struct{}{}
	}
	return ports
}

func udpListenPorts(rows [][]string, v6 bool) map[uint16]struct{} {
	ports := make(map[uint16]struct{})
	for _, fields := range rows {
		remoteAddr, remotePort, err := parseAddr(fields[2], v6)
		if err != nil {
			continue
		}
		if remoteAddr.IsValid() && !remoteAddr.IsUnspecified() && remotePort != 0 {
			continue
		}
		_, localPort, err := parseAddr(fields[1], v6)
		if err != nil {
			continue
		}
		ports[localPort] = struct{}{}
	}
	return ports
}

func seedSocketFromRow(
	fields []string,
	v6 bool,
	kind tableKind,
	listenPorts map[uint16]struct{},
	pod identity.PodInfo,
	s *Seeder,
) (aggregate.SeedSocket, bool) {
	if fields[3] != connectedHex {
		return aggregate.SeedSocket{}, false
	}

	localAddr, localPort, err := parseAddr(fields[1], v6)
	if err != nil {
		return aggregate.SeedSocket{}, false
	}
	remoteAddr, remotePort, err := parseAddr(fields[2], v6)
	if err != nil {
		return aggregate.SeedSocket{}, false
	}
	if !remoteAddr.IsValid() || remoteAddr.IsUnspecified() || remotePort == 0 {
		return aggregate.SeedSocket{}, false
	}
	if remoteAddr.Unmap().IsLoopback() {
		return aggregate.SeedSocket{}, false
	}
	if _, listening := listenPorts[localPort]; listening {
		return aggregate.SeedSocket{}, false
	}

	protocol := metrics.ProtocolTCP
	if kind == tableUDP {
		protocol = metrics.ProtocolUDP
	}
	ipProto := conntrack.IPProtocol(protocol)

	actualAddr, actualPort := s.conntrackClient.ActualDst(
		ipProto,
		conntrack.Endpoint{Addr: localAddr, Port: localPort},
		conntrack.Endpoint{Addr: remoteAddr, Port: remotePort},
	)

	key := metrics.NewSeriesKey(
		pod.UID,
		pod.Namespace,
		pod.Name,
		protocol,
		identity.AddrKey(remoteAddr),
		int(remotePort),
		identity.AddrKey(actualAddr),
		int(actualPort),
	)
	dns.StampDest(&key, s.resolver, s.dnsCache, actualAddr)
	return aggregate.SeedSocket{
		Flow:     aggregate.NewFlow(localAddr, remoteAddr, localPort, remotePort, protocol),
		Fallback: key,
	}, true
}

func parseAddr(field string, v6 bool) (netip.Addr, uint16, error) {
	parts := strings.Split(field, ":")
	if len(parts) != 2 {
		return netip.Addr{}, 0, fmt.Errorf("bad addr field %q", field)
	}
	port64, err := strconv.ParseUint(parts[1], 16, 16)
	if err != nil {
		return netip.Addr{}, 0, fmt.Errorf("parse port: %w", err)
	}
	raw, err := hex.DecodeString(parts[0])
	if err != nil {
		return netip.Addr{}, 0, fmt.Errorf("decode ip: %w", err)
	}
	if !v6 && len(raw) == 4 {
		raw[0], raw[3] = raw[3], raw[0]
		raw[1], raw[2] = raw[2], raw[1]
		var b [4]byte
		copy(b[:], raw)
		return netip.AddrFrom4(b), uint16(port64), nil
	}
	if v6 && len(raw) == 16 {
		for i := 0; i < 16; i += 4 {
			raw[i], raw[i+3] = raw[i+3], raw[i]
			raw[i+1], raw[i+2] = raw[i+2], raw[i+1]
		}
		var b [16]byte
		copy(b[:], raw)
		return netip.AddrFrom16(b), uint16(port64), nil
	}
	return netip.Addr{}, 0, fmt.Errorf("unexpected ip length %d", len(raw))
}
