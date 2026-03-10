package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedisCache requires a running Redis/Valkey server.
// Set REDIS_ADDRESS (e.g., "localhost:6379") to enable these tests.
// They are skipped by default.

func skipIfNoRedis(t *testing.T) string {
	t.Helper()
	addr := os.Getenv("REDIS_ADDRESS")
	if addr == "" {
		t.Skip("Skipping Redis tests: set REDIS_ADDRESS to enable")
	}
	return addr
}

// flushTestRedisDB clears all data in DB 15 so tests start with a clean slate.
func flushTestRedisDB(t *testing.T, addr string) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 15})
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("Failed to flush Redis test DB: %v", err)
	}
}

func newTestRedisCache(t *testing.T) Cache {
	t.Helper()
	return newTestRedisCacheWithConfig(t, 100, 10*time.Second, nil)
}

func newTestRedisCacheWithConfig(t *testing.T, size int, ttl time.Duration, onEvict EvictCallback) Cache {
	t.Helper()
	addr := skipIfNoRedis(t)
	flushTestRedisDB(t, addr)
	c, err := New("redis", ProviderConfig{
		Size:         size,
		TTL:          ttl,
		RedisAddress: addr,
		RedisDB:      15, // use a high DB number for tests
		OnEvict:      onEvict,
	})
	if err != nil {
		t.Fatalf("New redis cache: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestRedisCache_GetSet(t *testing.T) {
	c := newTestRedisCache(t)

	val, ok := c.Get("redis-test-key")
	if ok {
		t.Fatal("Expected miss for new key")
	}
	if val != nil {
		t.Fatalf("Expected nil value on miss, got %v", val)
	}

	c.Set("redis-test-key", []byte("hello"))
	val, ok = c.Get("redis-test-key")
	if !ok {
		t.Fatal("Expected hit after Set")
	}
	if string(val) != "hello" {
		t.Fatalf("Expected 'hello', got %q", string(val))
	}
}

func TestRedisCache_Contains(t *testing.T) {
	c := newTestRedisCache(t)

	if c.Contains("redis-absent") {
		t.Fatal("Expected absent key to not be contained")
	}

	c.Set("redis-present", []byte("data"))
	if !c.Contains("redis-present") {
		t.Fatal("Expected present key to be contained")
	}
}

func TestRedisCache_Len(t *testing.T) {
	c := newTestRedisCacheWithConfig(t, 100, 10*time.Second, nil)

	n := c.Len()
	if n != 0 {
		t.Fatalf("Expected Len 0 on clean DB, got %d", n)
	}

	c.Set("redis-len-a", []byte("1"))
	c.Set("redis-len-b", []byte("2"))

	if c.Len() != 2 {
		t.Fatalf("Expected Len 2, got %d", c.Len())
	}
}

func TestRedisCache_LRU_Eviction(t *testing.T) {
	evicted := make([]string, 0)
	onEvict := func(key string, _ []byte) {
		evicted = append(evicted, key)
	}

	// Max size 2 — inserting a third key should evict the oldest.
	c := newTestRedisCacheWithConfig(t, 2, 10*time.Second, onEvict)

	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	c.Set("c", []byte("3")) // should evict "a"

	if c.Contains("a") {
		t.Fatal("Evicted key 'a' should not be present")
	}
	if !c.Contains("b") || !c.Contains("c") {
		t.Fatal("Keys 'b' and 'c' should still be present")
	}
	if len(evicted) != 1 || evicted[0] != "a" {
		t.Fatalf("Expected eviction of 'a', got %v", evicted)
	}
}

func TestRedisCache_LRU_TouchPromotesEntry(t *testing.T) {
	// Max size 2. Insert a, b. Touch a. Insert c. "b" should be evicted (not "a").
	c := newTestRedisCacheWithConfig(t, 2, 10*time.Second, nil)

	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))

	// Touch "a" — promotes it in the LRU ordering.
	_, _ = c.Get("a")

	c.Set("c", []byte("3")) // should evict "b" (oldest untouched)

	if c.Contains("b") {
		t.Fatal("Expected 'b' to be evicted after 'a' was touched")
	}
	if !c.Contains("a") || !c.Contains("c") {
		t.Fatal("Keys 'a' and 'c' should still be present")
	}
}

func TestRedisCache_MultipleEvictions(t *testing.T) {
	evicted := make([]string, 0)
	onEvict := func(key string, _ []byte) {
		evicted = append(evicted, key)
	}

	// Max size 3 — inserting 6 items should evict the first 3.
	c := newTestRedisCacheWithConfig(t, 3, 10*time.Second, onEvict)

	c.Set("key1", []byte("val1"))
	c.Set("key2", []byte("val2"))
	c.Set("key3", []byte("val3"))
	c.Set("key4", []byte("val4"))
	c.Set("key5", []byte("val5"))
	c.Set("key6", []byte("val6"))

	// Verify the first 3 keys were evicted.
	if c.Contains("key1") || c.Contains("key2") || c.Contains("key3") {
		t.Fatal("Expected key1, key2, key3 to be evicted")
	}

	// Verify the last 3 keys remain.
	if !c.Contains("key4") || !c.Contains("key5") || !c.Contains("key6") {
		t.Fatal("Expected key4, key5, key6 to remain")
	}

	// Verify eviction callback was invoked for all evicted keys.
	if len(evicted) != 3 {
		t.Fatalf("Expected 3 evictions, got %d: %v", len(evicted), evicted)
	}
	expectedEvicted := map[string]bool{"key1": true, "key2": true, "key3": true}
	for _, key := range evicted {
		if !expectedEvicted[key] {
			t.Fatalf("Unexpected eviction of key %q", key)
		}
	}

	// Verify final cache size.
	if c.Len() != 3 {
		t.Fatalf("Expected Len 3, got %d", c.Len())
	}
}

func TestRedisCache_EvictionWithLRUOrdering(t *testing.T) {
	evicted := make([]string, 0)
	onEvict := func(key string, _ []byte) {
		evicted = append(evicted, key)
	}

	// Max size 3.
	c := newTestRedisCacheWithConfig(t, 3, 10*time.Second, onEvict)

	// Insert 3 items: a, b, c (LRU order: a -> b -> c, oldest to newest).
	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	c.Set("c", []byte("3"))

	// Touch "a" and "b" to promote them (LRU order: c -> a -> b).
	_, _ = c.Get("a")
	_, _ = c.Get("b")

	// Insert "d" — should evict "c" (oldest).
	c.Set("d", []byte("4"))

	if c.Contains("c") {
		t.Fatal("Expected 'c' to be evicted (oldest after touches)")
	}
	if !c.Contains("a") || !c.Contains("b") || !c.Contains("d") {
		t.Fatal("Expected 'a', 'b', 'd' to remain")
	}
	if len(evicted) != 1 || evicted[0] != "c" {
		t.Fatalf("Expected eviction of 'c', got %v", evicted)
	}

	evicted = evicted[:0] // Reset

	// Touch "a" again (LRU order: b -> d -> a).
	_, _ = c.Get("a")

	// Insert "e" — should evict "b" (oldest).
	c.Set("e", []byte("5"))

	if c.Contains("b") {
		t.Fatal("Expected 'b' to be evicted")
	}
	if !c.Contains("a") || !c.Contains("d") || !c.Contains("e") {
		t.Fatal("Expected 'a', 'd', 'e' to remain")
	}
	if len(evicted) != 1 || evicted[0] != "b" {
		t.Fatalf("Expected eviction of 'b', got %v", evicted)
	}
}

func TestRedisCache_EvictionCallbackInvoked(t *testing.T) {
	callbackInvocations := make(map[string]int)
	onEvict := func(key string, value []byte) {
		callbackInvocations[key]++
		// Redis cache doesn't retrieve evicted values, so value should be nil.
		if value != nil {
			t.Errorf("Expected nil value for evicted key %q, got %v", key, value)
		}
	}

	// Max size 2.
	c := newTestRedisCacheWithConfig(t, 2, 10*time.Second, onEvict)

	c.Set("first", []byte("1"))
	c.Set("second", []byte("2"))
	c.Set("third", []byte("3"))

	// Verify "first" was evicted and callback was invoked exactly once.
	if invocations, exists := callbackInvocations["first"]; !exists || invocations != 1 {
		t.Fatalf("Expected callback for 'first' to be invoked once, got %d", invocations)
	}

	// Verify no callback for still-present keys.
	if _, exists := callbackInvocations["second"]; exists {
		t.Fatal("Expected no callback for 'second' (still present)")
	}
	if _, exists := callbackInvocations["third"]; exists {
		t.Fatal("Expected no callback for 'third' (still present)")
	}
}

func TestRedisCache_TTLExpiration(t *testing.T) {
	// Short TTL to test expiration.
	c := newTestRedisCacheWithConfig(t, 100, 1*time.Second, nil)

	c.Set("expires-soon", []byte("data"))

	// Immediately after Set, key should exist.
	if !c.Contains("expires-soon") {
		t.Fatal("Expected key to exist immediately after Set")
	}

	val, ok := c.Get("expires-soon")
	if !ok || string(val) != "data" {
		t.Fatal("Expected to retrieve value immediately after Set")
	}

	// Wait for TTL to expire (add buffer for timing variance).
	time.Sleep(1500 * time.Millisecond)

	// After TTL, key should be gone (Redis HPEXPIRE removes expired fields).
	if c.Contains("expires-soon") {
		t.Fatal("Expected key to be expired and removed by Redis")
	}

	val, ok = c.Get("expires-soon")
	if ok {
		t.Fatalf("Expected miss after TTL expiration, got %v", val)
	}
}

func TestRedisCache_StaleLRUCleanup(t *testing.T) {
	// This test verifies that stale LRU entries (whose hash field has expired)
	// are cleaned up during eviction, as documented in the redisCache implementation.

	evicted := make([]string, 0)
	onEvict := func(key string, _ []byte) {
		evicted = append(evicted, key)
	}

	// Max size 3, short TTL.
	c := newTestRedisCacheWithConfig(t, 3, 1*time.Second, onEvict)

	// Insert 3 items.
	c.Set("stale1", []byte("v1"))
	c.Set("stale2", []byte("v2"))
	c.Set("fresh", []byte("v3"))

	// Wait for stale1 and stale2 to expire (their hash fields will be removed by Redis).
	time.Sleep(1500 * time.Millisecond)

	// Verify expired keys are gone from the hash.
	if c.Contains("stale1") || c.Contains("stale2") {
		t.Fatal("Expected stale1 and stale2 to be expired")
	}

	// The LRU sorted set still contains stale1 and stale2 as members, but
	// their hash fields are gone. Now insert a new item to trigger eviction.
	// The Lua script should clean up stale LRU members during this operation.
	c.Set("trigger-cleanup", []byte("v4"))

	// Verify fresh and trigger-cleanup remain (both are active).
	if !c.Contains("fresh") || !c.Contains("trigger-cleanup") {
		t.Fatal("Expected 'fresh' and 'trigger-cleanup' to remain")
	}

	// The cache should now have 2 items (stale entries removed).
	// Depending on timing, the eviction callback may or may not be invoked for
	// stale entries. The important thing is that the cache size is correct.
	if c.Len() != 2 {
		t.Fatalf("Expected Len 2 after stale cleanup, got %d", c.Len())
	}
}

func TestRedisCache_EvictionAtExactCapacity(t *testing.T) {
	evicted := make([]string, 0)
	onEvict := func(key string, _ []byte) {
		evicted = append(evicted, key)
	}

	// Max size 1 — every Set after the first should evict.
	c := newTestRedisCacheWithConfig(t, 1, 10*time.Second, onEvict)

	c.Set("first", []byte("v1"))
	if len(evicted) != 0 {
		t.Fatalf("Expected no evictions yet, got %v", evicted)
	}

	c.Set("second", []byte("v2"))
	if len(evicted) != 1 || evicted[0] != "first" {
		t.Fatalf("Expected 'first' to be evicted, got %v", evicted)
	}

	evicted = evicted[:0]

	c.Set("third", []byte("v3"))
	if len(evicted) != 1 || evicted[0] != "second" {
		t.Fatalf("Expected 'second' to be evicted, got %v", evicted)
	}

	// Only "third" should remain.
	if !c.Contains("third") || c.Contains("first") || c.Contains("second") {
		t.Fatal("Expected only 'third' to remain")
	}

	if c.Len() != 1 {
		t.Fatalf("Expected Len 1, got %d", c.Len())
	}
}

func TestRedisCache_Close(t *testing.T) {
	addr := skipIfNoRedis(t)
	flushTestRedisDB(t, addr)
	c, err := New("redis", ProviderConfig{
		Size:         10,
		TTL:          time.Minute,
		RedisAddress: addr,
		RedisDB:      15,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
