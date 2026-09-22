package memcachettl

import (
	"container/heap"
	"context"
	log "log/slog"
	"sync"
	"time"
)

// CacheItem represents an item in the cache.
type CacheItem struct {
	Value      interface{}
	Expiration time.Time
	seq        uint64
}

const reapInterval = 30 * time.Second

const limitWarnInterval = 10 * time.Second

type expireItem struct {
	key    string
	expiry time.Time
	seq    uint64
}

type expireHeap []expireItem

func (h expireHeap) Len() int            { return len(h) }
func (h expireHeap) Less(i, j int) bool  { return h[i].expiry.Before(h[j].expiry) }
func (h expireHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *expireHeap) Push(x interface{}) { *h = append(*h, x.(expireItem)) }
func (h *expireHeap) Pop() interface{} {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]
	return it
}

// Cache is a thread-safe in-memory cache.
type Cache struct {
	items      map[string]CacheItem
	expiryHeap expireHeap
	nextSeq    uint64
	maxEntries int
	lastLog    time.Time
	mu         sync.RWMutex
	reaperMu   sync.Mutex
	done       chan struct{}
	stopped    bool
	wg         sync.WaitGroup
}

func (c *Cache) GetName() string {
	return "CacheTTL"
}
func (c *Cache) Start(ctx context.Context) error {
	c.reaperMu.Lock()
	defer c.reaperMu.Unlock()
	if c.done != nil {
		return nil
	}
	c.done = make(chan struct{})
	c.stopped = false
	c.wg.Add(1)
	go c.reap(c.done)
	return nil
}
func (c *Cache) Stop() error {
	c.reaperMu.Lock()
	if c.stopped {
		c.reaperMu.Unlock()
		return nil
	}
	c.stopped = true
	done := c.done
	c.done = nil
	c.reaperMu.Unlock()
	if done != nil {
		close(done)
	}
	c.wg.Wait()
	return nil
}

func (c *Cache) reap(done chan struct{}) {
	defer c.wg.Done()
	ticker := time.NewTicker(reapInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			c.mu.Lock()
			c.popExpiredLocked(time.Now())
			c.mu.Unlock()
		}
	}
}

// NewCache creates a new cache.
func NewCache() *Cache {
	c := &Cache{
		items: make(map[string]CacheItem),
	}
	c.Start(context.Background())
	return c
}

// Set adds an item to the cache with a TTL.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxEntries > 0 {
		if _, exists := c.items[key]; !exists && len(c.items) >= c.maxEntries {
			c.popExpiredLocked(time.Now())
			if len(c.items) >= c.maxEntries {
				c.warnLimitLocked()
				return
			}
		}
	}

	expiry := time.Now().Add(ttl)
	c.nextSeq++
	c.items[key] = CacheItem{Value: value, Expiration: expiry, seq: c.nextSeq}
	heap.Push(&c.expiryHeap, expireItem{key: key, expiry: expiry, seq: c.nextSeq})
}

func (c *Cache) SetMaxEntries(n int) {
	c.mu.Lock()
	c.maxEntries = n
	c.mu.Unlock()
}

func (c *Cache) warnLimitLocked() {
	now := time.Now()
	if now.Sub(c.lastLog) >= limitWarnInterval {
		c.lastLog = now
		log.Warn("cache limit reached, new entry skipped", log.Int("maxEntries", c.maxEntries))
	}
}

func (c *Cache) popExpiredLocked(now time.Time) {
	for len(c.expiryHeap) > 0 && !c.expiryHeap[0].expiry.After(now) {
		it := heap.Pop(&c.expiryHeap).(expireItem)
		item, found := c.items[it.key]
		if found && item.seq == it.seq && item.Expiration.Equal(it.expiry) {
			delete(c.items, it.key)
		}
	}
}

// Get retrieves an item from the cache. Expired items are not returned.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, found := c.items[key]
	expired := found && time.Now().After(item.Expiration)
	c.mu.RUnlock()
	if !expired {
		if !found {
			return nil, false
		}
		return item.Value, true
	}

	// expired: re-check under the write lock and delete
	c.mu.Lock()
	var (
		value interface{}
		valid bool
	)
	if item, found = c.items[key]; found {
		if time.Now().After(item.Expiration) {
			delete(c.items, key)
		} else {
			value, valid = item.Value, true
		}
	}
	c.mu.Unlock()
	return value, valid
}

// Delete removes an item from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// CheckTTL clean old records
func (c *Cache) CheckTTL() {
	c.mu.Lock()
	prevSize := len(c.items)
	c.popExpiredLocked(time.Now())
	newSize := len(c.items)
	c.mu.Unlock()
	log.Debug("CheckTTL", log.Int("old", prevSize), log.Int("new", newSize))
}

// Size count of records
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// GetAll invokes the callback for each cached item. The callback runs after
// the cache is unlocked, so it may safely call Set/Delete/Get on this cache.
func (c *Cache) GetAll(callback func(string, CacheItem)) {
	c.mu.RLock()
	keys := make([]string, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}
	c.mu.RUnlock()
	for _, key := range keys {
		if value, ok := c.Get(key); ok {
			callback(key, CacheItem{Value: value})
		}
	}
}
