package cache

import "sync"

type CacheMap struct {
	mu    sync.RWMutex
	items map[string][]byte
}

func NewCacheMap() *CacheMap {
	return &CacheMap{items: make(map[string][]byte)}
}

func (cm *CacheMap) Set(key string, value []byte) {
	cm.mu.Lock()
	cm.items[key] = value
	cm.mu.Unlock()
}

func (cm *CacheMap) Get(key string) (value []byte, ok bool) {
	cm.mu.RLock()
	value, ok = cm.items[key]
	cm.mu.RUnlock()
	return value, ok
}

func (cm *CacheMap) Delete(key string) {
	cm.mu.Lock()
	delete(cm.items, key)
	cm.mu.Unlock()
}

func (cm *CacheMap) Purge() {
	cm.mu.Lock()
	cm.items = make(map[string][]byte)
	cm.mu.Unlock()
}

func (cm *CacheMap) Length() int {
	cm.mu.RLock()
	length := len(cm.items)
	cm.mu.RUnlock()
	return length
}

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
