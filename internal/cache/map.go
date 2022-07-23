package cache

import "sync"

type CacheMap struct {
	sync.RWMutex
	items map[string][]byte
}

func NewCacheMap() *CacheMap {
	return &CacheMap{items: make(map[string][]byte)}
}

func (cm *CacheMap) Set(key string, value []byte) {
	cm.Lock()
	cm.items[key] = value
	cm.Unlock()
}

func (cm *CacheMap) Get(key string) (value []byte, ok bool) {
	cm.RLock()
	value, ok = cm.items[key]
	cm.RUnlock()
	return value, ok
}

func (cm *CacheMap) Delete(key string) {
	cm.Lock()
	delete(cm.items, key)
	cm.Unlock()
}

func (cm *CacheMap) Purge() {
	cm.Lock()
	cm.items = make(map[string][]byte)
	cm.Unlock()
}

func (cm *CacheMap) Length() int {
	cm.RLock()
	length := len(cm.items)
	cm.RUnlock()
	return length
}

func (cm *CacheMap) Keys() []string {
	cm.RLock()
	keys := make([]string, len(cm.items))
	i := 0
	for k := range cm.items {
		keys[i] = k
		i++
	}
	cm.RUnlock()
	return keys
}
