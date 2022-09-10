package cache

import "sync"

// CacheMap represents in-memory key-value table safe for concurrent usage.
// Uses strings as keys. Stores byte slices.
type CacheMap struct {
	mu    sync.RWMutex
	items map[string][]byte
}

// NewCacheMap returns pointer to initialized CacheMap.
func NewCacheMap() *CacheMap {
	return &CacheMap{items: make(map[string][]byte)}
}

// Set sets given value for the given key, possible overwriting it.
func (cm *CacheMap) Set(key string, value []byte) {
	cm.mu.Lock()
	cm.items[key] = value
	cm.mu.Unlock()
}

// Get finds the value for given key. The second return value
// is a bool that specifies whether the key is present.
func (cm *CacheMap) Get(key string) (value []byte, ok bool) {
	cm.mu.RLock()
	value, ok = cm.items[key]
	cm.mu.RUnlock()
	return value, ok
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
	cm.items = make(map[string][]byte)
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
