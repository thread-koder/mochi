package procnet

import "sync"

// ListenIndex tracks bound UDP server ports per pod UID from /proc snapshots.
// UDP fexit hooks fire on server replies. eBPF drops sendmsg when the local port
// is bound or unconnected in that pod netns. TCP collection does not use this index.
type ListenIndex struct {
	mu    sync.RWMutex
	byPod map[string]map[uint16]struct{}
}

func NewListenIndex() *ListenIndex {
	return &ListenIndex{byPod: make(map[string]map[uint16]struct{})}
}

func (i *ListenIndex) Replace(snapshot map[string]map[uint16]struct{}) {
	i.mu.Lock()
	i.byPod = snapshot
	i.mu.Unlock()
}

func (i *ListenIndex) IsBound(podUID string, port uint16) bool {
	i.mu.RLock()
	ports, ok := i.byPod[podUID]
	i.mu.RUnlock()
	if !ok {
		return false
	}
	_, ok = ports[port]
	return ok
}
