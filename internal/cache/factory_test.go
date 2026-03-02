package cache

import (
	"testing"
	"time"
)

func TestFactory_New_Memory(t *testing.T) {
	c, err := New("memory", ProviderConfig{Size: 100, TTL: time.Hour})
	if err != nil {
		t.Fatalf("New memory: %v", err)
	}
	defer c.Close()

	// Verify it works
	c.Set("test", []byte("data"))
	val, ok := c.Get("test")
	if !ok || string(val) != "data" {
		t.Fatal("Memory cache should work after creation via factory")
	}
}

func TestFactory_New_UnknownProvider(t *testing.T) {
	_, err := New("nonexistent", ProviderConfig{})
	if err == nil {
		t.Fatal("Expected error for unknown provider")
	}
}

func TestFactory_RegisteredProviders(t *testing.T) {
	names := RegisteredProviders()
	if len(names) < 2 {
		t.Fatalf("Expected at least 2 providers (memory, redis), got %d: %v", len(names), names)
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["memory"] {
		t.Error("Expected 'memory' provider to be registered")
	}
	if !found["redis"] {
		t.Error("Expected 'redis' provider to be registered")
	}
}

func TestFactory_RegisteredProviders_Sorted(t *testing.T) {
	names := RegisteredProviders()
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("Providers not sorted: %v", names)
			break
		}
	}
}

func TestFactory_New_Redis_InvalidAddress(t *testing.T) {
	// Redis provider should fail to connect to an invalid address
	_, err := New("redis", ProviderConfig{
		Size:         100,
		TTL:          time.Hour,
		RedisAddress: "localhost:59999", // unlikely to have Redis here
	})
	if err == nil {
		t.Fatal("Expected error when connecting to invalid Redis address")
	}
}

func TestFactory_Register_NilProvider(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected panic when registering nil provider")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("Expected string panic, got %T: %v", r, r)
		}
		if msg != "cache: Register provider is nil" {
			t.Fatalf("Unexpected panic message: %s", msg)
		}
	}()
	Register("nil-test", nil)
}

func TestFactory_Register_Duplicate(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected panic when registering duplicate provider")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("Expected string panic, got %T: %v", r, r)
		}
		expected := `cache: provider "memory" already registered`
		if msg != expected {
			t.Fatalf("Unexpected panic message: %q, want %q", msg, expected)
		}
	}()
	// "memory" is already registered by the memory provider's init()
	Register("memory", func(cfg ProviderConfig) (Cache, error) {
		return nil, nil
	})
}

func TestFactory_New_WithGroup(t *testing.T) {
	c, err := New("memory", ProviderConfig{
		Size:  100,
		TTL:   time.Hour,
		Group: "test-group",
	})
	if err != nil {
		t.Fatalf("New with group: %v", err)
	}
	defer c.Close()

	c.Set("key1", []byte("value1"))
	val, ok := c.Get("key1")
	if !ok || string(val) != "value1" {
		t.Fatal("Instrumented cache should return stored value")
	}
	if !c.Contains("key1") {
		t.Fatal("Instrumented cache Contains should return true for existing key")
	}
	if c.Len() != 1 {
		t.Fatalf("Expected Len()=1, got %d", c.Len())
	}
}

func TestFactory_New_WithGroupAndOnEvict(t *testing.T) {
	evicted := make(chan string, 1)
	c, err := New("memory", ProviderConfig{
		Size:  1, // size=1 forces eviction on second Set
		TTL:   time.Hour,
		Group: "evict-group",
		OnEvict: func(key string, value []byte) {
			evicted <- key
		},
	})
	if err != nil {
		t.Fatalf("New with group+onEvict: %v", err)
	}
	defer c.Close()

	c.Set("first", []byte("1"))
	c.Set("second", []byte("2")) // should evict "first"

	select {
	case key := <-evicted:
		if key != "first" {
			t.Fatalf("Expected evicted key %q, got %q", "first", key)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnEvict callback was not called within timeout")
	}
}
