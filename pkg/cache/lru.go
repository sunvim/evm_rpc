package cache

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Value      interface{}
	Expiration time.Time
}

// IsExpired checks if the item has expired
func (i *CacheItem) IsExpired() bool {
	if i.Expiration.IsZero() {
		return false // Never expires
	}
	return time.Now().After(i.Expiration)
}

// Cache is a thread-safe LRU cache with TTL support
type Cache struct {
	cache *lru.Cache[string, *CacheItem]
	mu    sync.RWMutex
	
	hits   uint64
	misses uint64
}

// NewCache creates a new cache with specified size
func NewCache(size int) (*Cache, error) {
	cache, err := lru.New[string, *CacheItem](size)
	if err != nil {
		return nil, err
	}

	return &Cache{
		cache: cache,
	}, nil
}

// Get retrieves a value from cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.cache.Get(key)
	if !ok {
		c.misses++
		return nil, false
	}

	if item.IsExpired() {
		c.misses++
		go c.Delete(key) // Async cleanup
		return nil, false
	}

	c.hits++
	return item.Value, true
}

// Set stores a value in cache with optional TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	item := &CacheItem{
		Value:      value,
		Expiration: expiration,
	}

	c.cache.Add(key, item)
}

// Delete removes a value from cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Remove(key)
}

// Clear clears all items from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Purge()
}

// Len returns the number of items in cache
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache.Len()
}

// HitRate returns the cache hit rate
func (c *Cache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	if total == 0 {
		return 0
	}

	return float64(c.hits) / float64(total)
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Hits:    c.hits,
		Misses:  c.misses,
		Size:    c.cache.Len(),
		HitRate: c.HitRate(),
	}
}

// CacheStats contains cache statistics
type CacheStats struct {
	Hits    uint64
	Misses  uint64
	Size    int
	HitRate float64
}
