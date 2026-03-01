package cache

import (
	"testing"
	"time"
)

func TestMemoryCache_GetSet(t *testing.T) {
	c, err := New("memory", ProviderConfig{Size: 10, TTL: time.Hour})
	if err != nil {
		t.Fatalf("New memory cache: %v", err)
	}
	defer c.Close()

	// Miss
	val, ok := c.Get("key1")
	if ok {
		t.Fatal("Expected miss for key1")
	}
	if val != nil {
		t.Fatalf("Expected nil value on miss, got %v", val)
	}

	// Set + hit
	c.Set("key1", []byte("value1"))
	val, ok = c.Get("key1")
	if !ok {
		t.Fatal("Expected hit for key1")
	}
	if string(val) != "value1" {
		t.Fatalf("Expected value1, got %s", string(val))
	}
}

func TestMemoryCache_Contains(t *testing.T) {
	c, err := New("memory", ProviderConfig{Size: 10, TTL: time.Hour})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	if c.Contains("absent") {
		t.Fatal("Expected absent key to not be contained")
	}

	c.Set("present", []byte("data"))
	if !c.Contains("present") {
		t.Fatal("Expected present key to be contained")
	}
}

func TestMemoryCache_Len(t *testing.T) {
	c, err := New("memory", ProviderConfig{Size: 10, TTL: time.Hour})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	if c.Len() != 0 {
		t.Fatalf("Expected Len 0, got %d", c.Len())
	}

	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	if c.Len() != 2 {
		t.Fatalf("Expected Len 2, got %d", c.Len())
	}
}

func TestMemoryCache_Eviction(t *testing.T) {
	evictedKeys := make([]string, 0)
	onEvict := func(key string, _ []byte) {
		evictedKeys = append(evictedKeys, key)
	}

	c, err := New("memory", ProviderConfig{Size: 2, TTL: time.Hour, OnEvict: onEvict})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	c.Set("c", []byte("3")) // should evict "a"

	if len(evictedKeys) != 1 {
		t.Fatalf("Expected 1 eviction, got %d", len(evictedKeys))
	}
	if evictedKeys[0] != "a" {
		t.Fatalf("Expected evicted key 'a', got %q", evictedKeys[0])
	}

	if c.Contains("a") {
		t.Fatal("Evicted key 'a' should not be present")
	}
	if !c.Contains("b") || !c.Contains("c") {
		t.Fatal("Keys 'b' and 'c' should still be present")
	}
}

func TestMemoryCache_Overwrite(t *testing.T) {
	c, err := New("memory", ProviderConfig{Size: 10, TTL: time.Hour})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	c.Set("key", []byte("v1"))
	c.Set("key", []byte("v2"))

	val, ok := c.Get("key")
	if !ok {
		t.Fatal("Expected hit")
	}
	if string(val) != "v2" {
		t.Fatalf("Expected v2, got %s", string(val))
	}

	if c.Len() != 1 {
		t.Fatalf("Expected Len 1 after overwrite, got %d", c.Len())
	}
}

func TestMemoryCache_Close(t *testing.T) {
	c, err := New("memory", ProviderConfig{Size: 10, TTL: time.Hour})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
