package dns

import (
	"sync"
	"time"
)

// Floor so short TTLs still correlate to the following connect.
const minHold = time.Hour

type entry struct {
	qname     string
	expires   time.Time
	lastTouch time.Time
}

// Cache maps pod UID -> IP -> dialed QNAME from observed DNS answers.
type Cache struct {
	mu      sync.RWMutex
	maxSize int
	size    int
	byUID   map[string]map[string]entry
}

func NewCache(maxSize int) *Cache {
	return &Cache{
		maxSize: maxSize,
		byUID:   make(map[string]map[string]entry),
	}
}

func (c *Cache) Store(uid, ip, qname string, ttl time.Duration) {
	if uid == "" || ip == "" || qname == "" {
		return
	}
	if ttl < minHold {
		ttl = minHold
	}
	now := time.Now()
	cached := entry{
		qname:     qname,
		expires:   now.Add(ttl),
		lastTouch: now,
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	byIP := c.byUID[uid]
	_, exists := byIP[ip]
	if !exists && c.size >= c.maxSize {
		c.evictOldestLocked()
		byIP = c.byUID[uid]
	}
	if byIP == nil {
		byIP = make(map[string]entry)
		c.byUID[uid] = byIP
	}
	if !exists {
		c.size++
	}
	byIP[ip] = cached
}

func (c *Cache) Lookup(uid, ip string) string {
	if uid == "" || ip == "" {
		return ""
	}
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()
	byIP, ok := c.byUID[uid]
	if !ok {
		return ""
	}
	cached, ok := byIP[ip]
	if !ok {
		return ""
	}
	if now.After(cached.expires) {
		delete(byIP, ip)
		c.size--
		if len(byIP) == 0 {
			delete(c.byUID, uid)
		}
		return ""
	}
	cached.lastTouch = now
	byIP[ip] = cached
	return cached.qname
}

func (c *Cache) DropPod(uid string) {
	if uid == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	byIP, ok := c.byUID[uid]
	if !ok {
		return
	}
	c.size -= len(byIP)
	delete(c.byUID, uid)
}

func (c *Cache) evictOldestLocked() {
	var (
		oldestUID   string
		oldestIP    string
		oldestTouch time.Time
		found       bool
	)
	for uid, byIP := range c.byUID {
		for ip, cached := range byIP {
			if !found || cached.lastTouch.Before(oldestTouch) {
				oldestUID = uid
				oldestIP = ip
				oldestTouch = cached.lastTouch
				found = true
			}
		}
	}
	if !found {
		return
	}
	byIP := c.byUID[oldestUID]
	delete(byIP, oldestIP)
	c.size--
	if len(byIP) == 0 {
		delete(c.byUID, oldestUID)
	}
}
