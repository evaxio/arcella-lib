// Package wcache is a simple in-memory cache without TTL. The package name is kept for compatibility; the directory is memcache.
package wcache

import (
	"sync"
)

// Cache is a thread-safe in-memory cache. Entries are never evicted
// until explicitly deleted; use memcachettl for TTL-based expiration.
type Cache[K comparable, V any] struct {
	items map[K]*V
	mu    sync.RWMutex
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		items: make(map[K]*V),
	}
}

func (c *Cache[K, V]) Get(key K) (*V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, exists := c.items[key]
	if !exists {
		return nil, false
	}
	return val, true
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	c.items[key] = &value
	c.mu.Unlock()
}

func (c *Cache[K, V]) Del(key K) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}
