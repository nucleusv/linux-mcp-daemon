package cache

import (
	"sync"
	"time"
)

type CacheItem struct {
	Payload   string
	MimeType  string
	ExpiresAt time.Time
}

type TTLCache struct {
	mu    sync.RWMutex
	items map[string]CacheItem
}

func NewTTLCache() *TTLCache {
	return &TTLCache{
		items: make(map[string]CacheItem),
	}
}

// Get returns the cached payload and mime type if it exists and has not expired.
// Returns (payload, mimeType, true) on hit, ("", "", false) on miss or expiration.
func (c *TTLCache) Get(key string) (string, string, bool) {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		return "", "", false
	}

	if time.Now().After(item.ExpiresAt) {
		// Lazily clean up expired items
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return "", "", false
	}

	return item.Payload, item.MimeType, true
}

// Set stores a payload in the cache with a specified TTL.
func (c *TTLCache) Set(key string, payload string, mimeType string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = CacheItem{
		Payload:   payload,
		MimeType:  mimeType,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete explicitly removes an item from the cache.
func (c *TTLCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes every item, e.g. after a config reload changed who may
// read what.
func (c *TTLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]CacheItem)
}
