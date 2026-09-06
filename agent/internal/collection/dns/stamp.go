package dns

import (
	"net/netip"

	"github.com/thread_koder/mochi/agent/internal/collection/identity"
	"github.com/thread_koder/mochi/agent/internal/metrics"
)

// StampDest fills dest pod identity from the IP index, else dst_hostname from DNS cache.
// Hostname is skipped when dest UID is known so cluster series stay unsplit.
func StampDest(key *metrics.SeriesKey, resolver *identity.Resolver, cache *Cache, actual netip.Addr) {
	if dest, ok := resolver.LookupByIP(actual); ok {
		key.DstPodUID = dest.UID
		key.DstNamespace = dest.Namespace
		key.DstPod = dest.Name
		return
	}
	key.DstHostname = cache.Lookup(key.SrcPodUID, identity.AddrKey(actual))
}
