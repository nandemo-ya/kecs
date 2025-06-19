package utils

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"
)

// JSONCache caches JSON marshaling results
type JSONCache struct {
	mu       sync.RWMutex
	cache    map[interface{}]*jsonCacheEntry
	maxSize  int
	ttl      time.Duration
	hits     int64
	misses   int64
	marshals int64
}

type jsonCacheEntry struct {
	data      []byte
	timestamp time.Time
}

// NewJSONCache creates a new JSON cache
func NewJSONCache(maxSize int, ttl time.Duration) *JSONCache {
	cache := &JSONCache{
		cache:   make(map[interface{}]*jsonCacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
	
	// Start cleanup goroutine
	go cache.cleanupLoop()
	
	return cache
}

// Marshal marshals the value to JSON, using cache when possible
func (c *JSONCache) Marshal(v interface{}) ([]byte, error) {
	// Check cache first
	c.mu.RLock()
	entry, exists := c.cache[v]
	c.mu.RUnlock()
	
	if exists && time.Since(entry.timestamp) < c.ttl {
		c.mu.Lock()
		c.hits++
		c.mu.Unlock()
		
		// Return a copy to prevent mutation
		result := make([]byte, len(entry.data))
		copy(result, entry.data)
		return result, nil
	}
	
	// Cache miss - marshal the value
	c.mu.Lock()
	c.misses++
	c.marshals++
	c.mu.Unlock()
	
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Evict if at capacity
	if len(c.cache) >= c.maxSize {
		// Simple eviction - remove oldest entry
		var oldestKey interface{}
		oldestTime := time.Now()
		
		for k, v := range c.cache {
			if v.timestamp.Before(oldestTime) {
				oldestTime = v.timestamp
				oldestKey = k
			}
		}
		
		if oldestKey != nil {
			delete(c.cache, oldestKey)
		}
	}
	
	// Cache the result
	c.cache[v] = &jsonCacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
	
	// Return a copy
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// MarshalIndent marshals with indentation (not cached due to formatting options)
func (c *JSONCache) MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// Invalidate removes an entry from the cache
func (c *JSONCache) Invalidate(v interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.cache, v)
}

// Clear removes all entries from the cache
func (c *JSONCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache = make(map[interface{}]*jsonCacheEntry)
}

// Stats returns cache statistics
func (c *JSONCache) Stats() JSONCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return JSONCacheStats{
		Hits:        c.hits,
		Misses:      c.misses,
		Marshals:    c.marshals,
		Size:        len(c.cache),
		MaxSize:     c.maxSize,
		HitRate:     c.hitRate(),
	}
}

func (c *JSONCache) hitRate() float64 {
	total := float64(c.hits + c.misses)
	if total == 0 {
		return 0
	}
	return float64(c.hits) / total * 100
}

// cleanupLoop periodically removes expired entries
func (c *JSONCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries
func (c *JSONCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	toRemove := []interface{}{}
	
	for k, v := range c.cache {
		if now.Sub(v.timestamp) > c.ttl {
			toRemove = append(toRemove, k)
		}
	}
	
	for _, k := range toRemove {
		delete(c.cache, k)
	}
}

// JSONCacheStats contains cache statistics
type JSONCacheStats struct {
	Hits     int64
	Misses   int64
	Marshals int64
	Size     int
	MaxSize  int
	HitRate  float64
}

// BufferPool provides a pool of bytes.Buffer for JSON encoding
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Get returns a buffer from the pool
func (p *BufferPool) Get() *bytes.Buffer {
	buf := p.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool
func (p *BufferPool) Put(buf *bytes.Buffer) {
	// Don't return very large buffers to the pool
	if buf.Cap() > 1024*1024 {
		return
	}
	buf.Reset()
	p.pool.Put(buf)
}

// Global instances for convenience
var (
	defaultJSONCache = NewJSONCache(1000, 5*time.Minute)
	bufferPool      = NewBufferPool()
)

// MarshalCached uses the global JSON cache
func MarshalCached(v interface{}) ([]byte, error) {
	return defaultJSONCache.Marshal(v)
}

// InvalidateCached invalidates an entry in the global cache
func InvalidateCached(v interface{}) {
	defaultJSONCache.Invalidate(v)
}

// GetJSONCacheStats returns global cache statistics
func GetJSONCacheStats() JSONCacheStats {
	return defaultJSONCache.Stats()
}