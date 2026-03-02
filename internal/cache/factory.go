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

	// Logger receives error reports from cache operations. If nil, errors are silently ignored.
	Logger Logger

	// RedisAddress is the Redis/Valkey server address (e.g., "localhost:6379").
	RedisAddress string

	// RedisPassword is the password for the Redis/Valkey server.
	RedisPassword string

	// RedisDB is the Redis/Valkey database number.
	RedisDB int

	// Group is an optional label value used to namespace Prometheus metrics
	// (cache_hits_total, cache_misses_total, etc.).
	// When non-empty the cache is automatically wrapped with metric instrumentation.
	Group string
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
// When cfg.Group is non-empty the resulting cache is wrapped with metric
// instrumentation: hits, misses, and evictions are tracked with a
// "cache" label equal to Group, and a lazy entries collector is registered
// that queries Len() at scrape time instead of maintaining an in-process counter.
func New(name string, cfg ProviderConfig) (Cache, error) {
	mu.RLock()
	p, ok := providers[name]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("cache: unknown provider %q (registered: %v)", name, RegisteredProviders())
	}

	if cfg.Group == "" {
		return p(cfg)
	}

	group := cfg.Group
	// Wrap OnEvict so the cache layer counts evictions itself.
	original := cfg.OnEvict
	cfg.OnEvict = func(key string, value []byte) {
		EvictionsTotal.WithLabelValues(group).Inc()
		if original != nil {
			original(key, value)
		}
	}

	inner, err := p(cfg)
	if err != nil {
		return nil, err
	}

	return newInstrumentedCache(inner, group), nil
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
