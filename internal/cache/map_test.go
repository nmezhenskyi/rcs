package cache

import "testing"

func TestNewCacheMap(t *testing.T) {
	cmap := NewCacheMap()
	if cmap == nil {
		t.Error("Expected pointer to initialized CacheMap, got nil instead")
	}
	if cmap.items == nil {
		t.Error("CacheMap.items field is nil")
	}
}

func TestSet(t *testing.T) {
	cmap := NewCacheMap()
	key := "key1"
	value := []byte("value1")
	cmap.Set(key, value)

	retrieved, ok := cmap.items[key]
	if !ok {
		t.Error("Key has not been set")
	}
	if !equalBytes(retrieved, value) {
		t.Error("Retrieved value is not the same")
	}

	cmap.Set("key2", []byte("value2"))
	cmap.Set("key3", []byte("value3"))
	cmap.Set("key4", []byte("value4"))
	cmap.Set("key5", []byte("value5"))
	cmap.Set("key5", []byte("value5"))

	if len(cmap.items) != 5 {
		t.Errorf("Expected 5 keys, but got %d", len(cmap.items))
	}
}

func TestGet(t *testing.T) {
	cmap := NewCacheMap()
	key := "key1"
	value := []byte("value1")

	_, ok := cmap.Get(key)
	if ok {
		t.Errorf("Found nonexistent key")
	}

	cmap.items = map[string][]byte{key: value}
	retrieved, ok := cmap.Get(key)
	if !ok {
		t.Error("Key not found")
	}
	if !equalBytes(retrieved, value) {
		t.Error("Retrieved value is not the same")
	}
}

func TestDelete(t *testing.T) {
	cmap := NewCacheMap()
	cmap.items = map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
		"key4": []byte("value4"),
		"key5": []byte("value5"),
	}

	cmap.Delete("key2")
	cmap.Delete("key4")

	if len(cmap.items) != 3 {
		t.Errorf("Expected 3 keys, but got %d", len(cmap.items))
	}
}

func TestFlush(t *testing.T) {
	cmap := NewCacheMap()
	cmap.items = map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
		"key4": []byte("value4"),
		"key5": []byte("value5"),
	}

	cmap.Flush()
	if len(cmap.items) != 0 {
		t.Errorf("Expected empty map, instead found %d keys", len(cmap.items))
	}
}

func TestLength(t *testing.T) {
	cmap := NewCacheMap()
	cmap.items = map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
		"key4": []byte("value4"),
		"key5": []byte("value5"),
	}

	length := cmap.Length()
	if len(cmap.items) != length {
		t.Errorf("Expected length %d, got %d instead", len(cmap.items), length)
	}
}

func equalBytes(s1, s2 []byte) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}
