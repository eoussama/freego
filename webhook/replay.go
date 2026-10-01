package webhook

import (
	"sync"
	"time"
)

// ReplayCache remembers the ids of received messages so that retried or
// replayed deliveries are not processed twice.
//
// The handler records an id before running the callbacks, so delivery is
// at most once per id within the replay window: if the process dies while a
// callback runs, FreeStuff's retry is acknowledged without being processed
// again.
//
// The default, [NewMemoryReplayCache], works for a single process and
// forgets ids on restart. When several instances receive the same webhook,
// implement ReplayCache on top of a shared store (for example Redis SET NX
// with an expiry).
type ReplayCache interface {
	// Seen records id for at least ttl and reports whether it had already
	// been recorded within its ttl. It must be atomic: when called
	// concurrently with the same id, exactly one call returns false. When
	// it returns an error, the delivery is answered with 500 so that
	// FreeStuff retries it.
	Seen(id string, ttl time.Duration) (bool, error)
}

// MemoryReplayCache is an in-memory [ReplayCache]. Expired entries are
// removed lazily. It is safe for concurrent use, and its zero value is
// ready to use.
type MemoryReplayCache struct {
	mu        sync.Mutex
	entries   map[string]time.Time
	lastSweep time.Time
	now       func() time.Time
}

// NewMemoryReplayCache returns an empty in-memory replay cache.
func NewMemoryReplayCache() *MemoryReplayCache {
	return &MemoryReplayCache{}
}

// Seen implements [ReplayCache]. It never returns an error.
func (c *MemoryReplayCache) Seen(id string, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entries == nil {
		c.entries = make(map[string]time.Time)
	}
	now := time.Now()
	if c.now != nil {
		now = c.now()
	}

	if now.Sub(c.lastSweep) >= ttl {
		for k, exp := range c.entries {
			if !now.Before(exp) {
				delete(c.entries, k)
			}
		}
		c.lastSweep = now
	}

	if exp, ok := c.entries[id]; ok && now.Before(exp) {
		return true, nil
	}
	c.entries[id] = now.Add(ttl)
	return false, nil
}

// Len returns the number of ids currently remembered, including expired ones
// not swept yet.
func (c *MemoryReplayCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
