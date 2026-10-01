package webhook

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func mustSeen(t *testing.T, c ReplayCache, id string, ttl time.Duration) bool {
	t.Helper()
	seen, err := c.Seen(id, ttl)
	if err != nil {
		t.Fatal(err)
	}
	return seen
}

func TestMemoryReplayCache(t *testing.T) {
	now := fixedNow
	c := NewMemoryReplayCache()
	c.now = func() time.Time { return now }

	if mustSeen(t, c, "a", time.Minute) {
		t.Error("first sighting reported as seen")
	}
	if !mustSeen(t, c, "a", time.Minute) {
		t.Error("second sighting not reported as seen")
	}
	if mustSeen(t, c, "b", time.Minute) {
		t.Error("other id reported as seen")
	}

	now = now.Add(time.Minute)
	if mustSeen(t, c, "a", time.Minute) {
		t.Error("expired id reported as seen")
	}
	// The sweep removed the expired "b".
	if c.Len() != 1 {
		t.Errorf("Len = %d, want 1 after the sweep", c.Len())
	}
}

func TestMemoryReplayCacheZeroValue(t *testing.T) {
	var c MemoryReplayCache
	if mustSeen(t, &c, "a", time.Minute) || !mustSeen(t, &c, "a", time.Minute) {
		t.Error("zero value cache does not work")
	}
}

func TestMemoryReplayCacheConcurrent(t *testing.T) {
	c := NewMemoryReplayCache()
	var fresh atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if seen, _ := c.Seen("same", time.Minute); !seen {
				fresh.Add(1)
			}
		}()
	}
	wg.Wait()
	if fresh.Load() != 1 {
		t.Errorf("%d goroutines saw the id first, want 1", fresh.Load())
	}
}
