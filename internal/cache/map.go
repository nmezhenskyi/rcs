// Package cache contains all internal cache implementations for RCS.
package cache

import "sync"

// CacheMap represents in-memory key-value table safe for concurrent usage.
// Uses strings as keys. Stores items with byte slices and expiration time.
type CacheMap struct {
	mu    sync.RWMutex
	items map[string]item
}

// NewCacheMap returns pointer to initialized CacheMap.
func NewCacheMap() *CacheMap {
	return &CacheMap{items: make(map[string]item)}
}

// Set sets given value for the given key, possibly overwriting it.
func (cm *CacheMap) Set(key string, value []byte) {
	cm.mu.Lock()
	cm.items[key] = item{data: value}
	cm.mu.Unlock()
}

// SetEx sets given value for the given key, and an expiration time.
// Overwrites the previous value for the key.
func (cm *CacheMap) SetEx(key string, value []byte, expires int64) {
	cm.mu.Lock()
	cm.items[key] = item{data: value, expires: expires}
	cm.mu.Unlock()
}

// Get finds the value for given key. The second return value
// is a bool that specifies whether the key is present.
func (cm *CacheMap) Get(key string) ([]byte, bool) {
	cm.mu.RLock()
	value, ok := cm.items[key]
	cm.mu.RUnlock()
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
