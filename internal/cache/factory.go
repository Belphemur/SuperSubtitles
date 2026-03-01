package cache

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ProviderConfig holds the configuration needed to create a cache instance.
type ProviderConfig struct {
	// Size is the maximum number of entries for LRU caches.
	Size int

	// TTL is the time-to-live for cache entries.
	TTL time.Duration

	// OnEvict is called when an entry is evicted. Not all providers support this.
	OnEvict EvictCallback

	// RedisAddress is the Redis/Valkey server address (e.g., "localhost:6379").
	RedisAddress string

	// RedisPassword is the password for the Redis/Valkey server.
	RedisPassword string

	// RedisDB is the Redis/Valkey database number.
	RedisDB int
}

// Provider is a constructor function that creates a Cache from config.
type Provider func(cfg ProviderConfig) (Cache, error)

var (
	mu        sync.RWMutex
	providers = make(map[string]Provider)
)

// Register registers a cache provider under the given name.
// It panics if the name is already registered or the provider is nil.
func Register(name string, p Provider) {
	mu.Lock()
	defer mu.Unlock()

	if p == nil {
		panic("cache: Register provider is nil")
	}
	if _, exists := providers[name]; exists {
		panic(fmt.Sprintf("cache: provider %q already registered", name))
	}
	providers[name] = p
}

// New creates a new Cache using the named provider and the given config.
func New(name string, cfg ProviderConfig) (Cache, error) {
	mu.RLock()
	p, ok := providers[name]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("cache: unknown provider %q (registered: %v)", name, RegisteredProviders())
	}
	return p(cfg)
}

// RegisteredProviders returns a sorted list of registered provider names.
func RegisteredProviders() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
