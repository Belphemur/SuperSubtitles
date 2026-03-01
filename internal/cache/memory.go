package cache

import (
	lru "github.com/hashicorp/golang-lru/v2/expirable"
)

func init() {
	Register("memory", newMemoryCache)
}

// memoryCache wraps hashicorp/golang-lru/v2/expirable to implement the Cache interface.
type memoryCache struct {
	inner *lru.LRU[string, []byte]
}

func newMemoryCache(cfg ProviderConfig) (Cache, error) {
	var onEvict func(string, []byte)
	if cfg.OnEvict != nil {
		onEvict = func(key string, value []byte) {
			cfg.OnEvict(key, value)
		}
	}
	return &memoryCache{
		inner: lru.NewLRU[string, []byte](cfg.Size, onEvict, cfg.TTL),
	}, nil
}

func (m *memoryCache) Get(key string) ([]byte, bool) {
	return m.inner.Get(key)
}

func (m *memoryCache) Set(key string, value []byte) {
	m.inner.Add(key, value)
}

func (m *memoryCache) Contains(key string) bool {
	return m.inner.Contains(key)
}

func (m *memoryCache) Len() int {
	return m.inner.Len()
}

func (m *memoryCache) Close() error {
	return nil
}
