package cache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// MemoryCache provides an in-memory cache with TTL and LRU eviction
type MemoryCache struct {
	mu         sync.RWMutex
	items      map[string]*cacheItem
	evictList  *list.List
	maxItems   int
	defaultTTL time.Duration

	// Statistics
	hits      int64
	misses    int64
	evictions int64
	sets      int64
}

type cacheItem struct {
	key        string
	value      interface{}
	expiration time.Time
	element    *list.Element
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache(maxItems int, defaultTTL time.Duration) *MemoryCache {
	cache := &MemoryCache{
		items:      make(map[string]*cacheItem),
		evictList:  list.New(),
		maxItems:   maxItems,
		defaultTTL: defaultTTL,
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	// Check expiration
	if time.Now().After(item.expiration) {
		c.mu.Lock()
		c.removeItem(item)
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	// Move to front (LRU)
	c.mu.Lock()
	c.evictList.MoveToFront(item.element)
	c.hits++
	c.mu.Unlock()

	return item.value, true
}

// Set stores a value in the cache with default TTL
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}) {
	c.SetWithTTL(ctx, key, value, c.defaultTTL)
}

// SetWithTTL stores a value in the cache with custom TTL
func (c *MemoryCache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sets++

	// Check if key already exists
	if item, exists := c.items[key]; exists {
		// Update existing item
		item.value = value
		item.expiration = time.Now().Add(ttl)
		c.evictList.MoveToFront(item.element)
		return
	}

	// Evict if at capacity
	if len(c.items) >= c.maxItems {
		c.evictOldest()
	}

	// Create new item
	item := &cacheItem{
		key:        key,
		value:      value,
		expiration: time.Now().Add(ttl),
	}

	// Add to front of list
	item.element = c.evictList.PushFront(item)
	c.items[key] = item
}

// Delete removes a value from the cache
func (c *MemoryCache) Delete(ctx context.Context, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		c.removeItem(item)
	}
}

// DeleteWithPrefix removes all items whose keys start with the given prefix
func (c *MemoryCache) DeleteWithPrefix(ctx context.Context, prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var toRemove []*cacheItem
	for key, item := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			toRemove = append(toRemove, item)
		}
	}

	for _, item := range toRemove {
		c.removeItem(item)
	}
}

// Clear removes all items from the cache
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	c.evictList.Init()
}

// Stats returns cache statistics
func (c *MemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Hits:      c.hits,
		Misses:    c.misses,
		Evictions: c.evictions,
		Sets:      c.sets,
		Size:      len(c.items),
		MaxSize:   c.maxItems,
	}
}

// removeItem removes an item from the cache (must be called with lock held)
func (c *MemoryCache) removeItem(item *cacheItem) {
	c.evictList.Remove(item.element)
	delete(c.items, item.key)
}

// evictOldest removes the oldest item from the cache (must be called with lock held)
func (c *MemoryCache) evictOldest() {
	oldest := c.evictList.Back()
	if oldest != nil {
		item := oldest.Value.(*cacheItem)
		c.removeItem(item)
		c.evictions++
	}
}

// cleanupLoop periodically removes expired items
func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired items
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var toRemove []*cacheItem

	// Find expired items
	for _, item := range c.items {
		if now.After(item.expiration) {
			toRemove = append(toRemove, item)
		}
	}

	// Remove expired items
	for _, item := range toRemove {
		c.removeItem(item)
	}
}

// CacheStats contains cache statistics
type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
	Sets      int64
	Size      int
	MaxSize   int
}

// HitRate returns the cache hit rate as a percentage
func (s CacheStats) HitRate() float64 {
	total := float64(s.Hits + s.Misses)
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / total * 100
}

// MultiLevelCache provides a two-level cache with memory and storage backend
type MultiLevelCache struct {
	memory  *MemoryCache
	storage CacheBackend
}

// CacheBackend interface for secondary cache storage
type CacheBackend interface {
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
}

// NewMultiLevelCache creates a new multi-level cache
func NewMultiLevelCache(memory *MemoryCache, storage CacheBackend) *MultiLevelCache {
	return &MultiLevelCache{
		memory:  memory,
		storage: storage,
	}
}

// Get retrieves from memory first, then storage
func (m *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, bool) {
	// Check memory cache first
	if value, found := m.memory.Get(ctx, key); found {
		return value, true
	}

	// Check storage backend
	if m.storage != nil {
		if value, found := m.storage.Get(ctx, key); found {
			// Populate memory cache
			m.memory.Set(ctx, key, value)
			return value, true
		}
	}

	return nil, false
}

// Set stores in both memory and storage
func (m *MultiLevelCache) Set(ctx context.Context, key string, value interface{}) error {
	// Set in memory
	m.memory.Set(ctx, key, value)

	// Set in storage backend
	if m.storage != nil {
		return m.storage.Set(ctx, key, value)
	}

	return nil
}
