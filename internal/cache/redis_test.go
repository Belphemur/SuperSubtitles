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
