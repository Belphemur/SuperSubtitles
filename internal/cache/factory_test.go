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
