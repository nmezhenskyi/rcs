// Package cache contains all internal cache implementations for RCS.
package cache

import (
	"sync"
	"time"
)

// CacheMap represents in-memory key-value table safe for concurrent usage.
// Uses strings as keys. Stores items with byte slices and expiration time.
type CacheMap struct {
	cleanupInterval time.Duration
	stop            chan struct{}

	mu    sync.RWMutex
	items map[string]item
}

// NewCacheMap returns pointer to initialized CacheMap without cleanup routine.
func NewCacheMap() *CacheMap {
	return &CacheMap{items: make(map[string]item)}
}

// NewCacheMap returns pointer to initialized CacheMap with cleanup routine.
func NewCacheMapWithCleanup(interval time.Duration) *CacheMap {
	c := &CacheMap{
		cleanupInterval: interval,
		items:           make(map[string]item),
	}
	if c.cleanupInterval > 0 {
		go c.startCleanup()
	}
	return c
}

// Set sets given value for the given key, possibly overwriting it.
func (cm *CacheMap) Set(key string, value []byte) {
	cm.mu.Lock()
	cm.items[key] = item{data: value}
	cm.mu.Unlock()
}

// SetEx sets given value for the given key, and an expiration time.
// Overwrites the previous value for the key.
func (cm *CacheMap) SetEx(key string, value []byte, expires time.Duration) {
	var expirationInNano int64
	if expires > 0 {
		expirationInNano = time.Now().Add(expires).UnixNano()
	}
	cm.mu.Lock()
	cm.items[key] = item{data: value, expires: expirationInNano}
	cm.mu.Unlock()
}

// Get finds the value for given key. The second return value
// is a bool that specifies whether the key is present.
func (cm *CacheMap) Get(key string) ([]byte, bool) {
	cm.mu.RLock()
	value, ok := cm.items[key]
	cm.mu.RUnlock()
	if value.isExpired() {
		return nil, false
	}
	return value.data, ok
}

// Delete removes the key and associated value from the map.
// If key is not present, Delete is a no-op.
func (cm *CacheMap) Delete(key string) {
	cm.mu.Lock()
	delete(cm.items, key)
	cm.mu.Unlock()
}

// Purge removes all keys from the map making it empty.
func (cm *CacheMap) Purge() {
	cm.mu.Lock()
	cm.items = make(map[string]item)
	cm.mu.Unlock()
}

// Length returns number of items stored in the map.
func (cm *CacheMap) Length() int {
	cm.mu.RLock()
	length := len(cm.items)
	cm.mu.RUnlock()
	return length
}

// Keys returns an array of all keys in the map.
func (cm *CacheMap) Keys() []string {
	cm.mu.RLock()
	keys := make([]string, len(cm.items))
	i := 0
	for k := range cm.items {
		keys[i] = k
		i++
	}
	cm.mu.RUnlock()
	return keys
}

// StopCleanup stops the cache's cleanup routine if it was active.
// This is useful for tests and potentially for manually
// controlling cleanup cycles.
func (cm *CacheMap) StopCleanup() {
	if cm.stop != nil {
		cm.stop <- struct{}{}
	}
}

func (cm *CacheMap) startCleanup() {
	cm.stop = make(chan struct{})

	ticker := time.NewTicker(cm.cleanupInterval)
	for {
		select {
		case <-ticker.C:
			cm.deleteExpired()
		case <-cm.stop:
			ticker.Stop()
			cm.stop = nil
			return
		}
	}
}

func (cm *CacheMap) deleteExpired() {
	cm.mu.Lock()
	for k, v := range cm.items {
		if v.isExpired() {
			delete(cm.items, k)
		}
	}
	cm.mu.Unlock()
}
