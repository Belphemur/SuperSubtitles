package cache

// instrumentedCache wraps a Cache and automatically records Prometheus metrics
// for hits, misses, evictions, and current entry count under the given group label.
// All metric tracking lives in the cache layer so callers do not need to manage it.
type instrumentedCache struct {
	inner Cache
	group string
}

// newInstrumentedCache wraps inner with metric instrumentation for the given group.
// A lazy entries collector is registered that queries inner.Len() at scrape time,
// which is correct for backends (e.g., Redis) where TTL expiry removes entries
// outside the application's control.
func newInstrumentedCache(inner Cache, group string) *instrumentedCache {
	registerEntriesCollector(group, inner.Len)
	return &instrumentedCache{inner: inner, group: group}
}

func (c *instrumentedCache) Get(key string) ([]byte, bool) {
	val, ok := c.inner.Get(key)
	if ok {
		HitsTotal.WithLabelValues(c.group).Inc()
	} else {
		MissesTotal.WithLabelValues(c.group).Inc()
	}
	return val, ok
}

func (c *instrumentedCache) Set(key string, value []byte) {
	c.inner.Set(key, value)
}

func (c *instrumentedCache) Contains(key string) bool {
	return c.inner.Contains(key)
}

func (c *instrumentedCache) Len() int {
	return c.inner.Len()
}

// Close unregisters the entries collector and closes the underlying cache.
func (c *instrumentedCache) Close() error {
	unregisterEntriesCollector(c.group)
	return c.inner.Close()
}
