package cache

import (
	"bytes"
	"sort"
	"testing"
	"time"
)

func TestNewCacheMap(t *testing.T) {
	cmap := NewCacheMap()
	if cmap == nil {
		t.Error("Expected pointer to initialized CacheMap, got nil instead")
	}
	if cmap != nil && cmap.items == nil {
		t.Error("CacheMap.items field is nil")
	}
	if cmap.cleanupInterval != 0 {
		t.Errorf("CacheMap.cleanupInterval is not 0")
	}
}

func TestNewCacheMapWithCleanup(t *testing.T) {
	interval := 5 * time.Minute
	cmap := NewCacheMapWithCleanup(interval)
	if cmap == nil {
		t.Error("Expected pointer to initialized CacheMap, got nil instead")
	}
	if cmap != nil && cmap.items == nil {
		t.Error("CacheMap.items field is nil")
	}
	if cmap.cleanupInterval != interval {
		t.Errorf("Expected CacheMap.cleanupInterval to be %s, got %s instead",
			interval.String(), cmap.cleanupInterval.String())
	}
	cmap.StopCleanup()
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
	if !bytes.Equal(retrieved.data, value) {
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

func TestSetEx(t *testing.T) {
	cmap := NewCacheMap()
	key := "key1"
	value := []byte("value1")
	expires := int64(1668819947000)
	cmap.SetEx(key, value, expires)

	retrieved, ok := cmap.items[key]
	if !ok {
		t.Error("Key has not been set")
	}
	if !bytes.Equal(retrieved.data, value) {
		t.Error("Retrieved value is not the same")
	}
	if retrieved.expires != expires {
		t.Error("Stored expires time does not match the given value")
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

	// Valid item:
	cmap.items = map[string]item{key: {data: value}}
	retrieved, ok := cmap.Get(key)
	if !ok {
		t.Error("Key not found")
	}
	if !bytes.Equal(retrieved, value) {
		t.Error("Retrieved value is not the same")
	}

	// Expired item:
	cmap.items = map[string]item{key: {data: value, expires: -100}}
	retrieved, ok = cmap.Get(key)
	if ok {
		t.Errorf("Expected ok to be false, got %v instead", ok)
	}
	if retrieved != nil {
		t.Errorf("Expected retrieved value to be nil, got %s instead",
			string(retrieved))
	}
}

func TestDelete(t *testing.T) {
	cmap := NewCacheMap()
	cmap.items = map[string]item{
		"key1": {data: []byte("value1")},
		"key2": {data: []byte("value2")},
		"key3": {data: []byte("value3")},
		"key4": {data: []byte("value4")},
		"key5": {data: []byte("value5")},
	}

	cmap.Delete("key2")
	cmap.Delete("key4")

	if len(cmap.items) != 3 {
		t.Errorf("Expected 3 keys, but got %d", len(cmap.items))
	}
}

func TestPurge(t *testing.T) {
	cmap := NewCacheMap()
	cmap.items = map[string]item{
		"key1": {data: []byte("value1")},
		"key2": {data: []byte("value2")},
		"key3": {data: []byte("value3")},
		"key4": {data: []byte("value4")},
		"key5": {data: []byte("value5")},
	}

	cmap.Purge()
	if len(cmap.items) != 0 {
		t.Errorf("Expected empty map, instead found %d keys", len(cmap.items))
	}
}

func TestLength(t *testing.T) {
	cmap := NewCacheMap()
	cmap.items = map[string]item{
		"key1": {data: []byte("value1")},
		"key2": {data: []byte("value2")},
		"key3": {data: []byte("value3")},
		"key4": {data: []byte("value4")},
		"key5": {data: []byte("value5")},
	}

	length := cmap.Length()
	if len(cmap.items) != length {
		t.Errorf("Expected length %d, got %d instead", len(cmap.items), length)
	}
}

func TestKeys(t *testing.T) {
	cmap := NewCacheMap()
	cmap.items = map[string]item{
		"key1": {data: []byte("value1")},
		"key2": {data: []byte("value2")},
		"key3": {data: []byte("value3")},
		"key4": {data: []byte("value4")},
		"key5": {data: []byte("value5")},
	}
	expectedKeys := []string{"key1", "key2", "key3", "key4", "key5"}

	keys := cmap.Keys()
	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d instead", len(expectedKeys), len(keys))
	}
	if len(keys) != len(cmap.items) {
		t.Error("Number of keys does not match the length of the map")
	}

	sort.Strings(keys)
	for i := range keys {
		if keys[i] != expectedKeys[i] {
			t.Errorf("Keys do not match the expected ones")
		}
	}
}
