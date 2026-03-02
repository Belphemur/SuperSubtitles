package cache

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Cache-level Prometheus metrics. All metrics carry a "cache" label whose value
// is the Group set in ProviderConfig, allowing multiple cache instances to be
// distinguished in dashboards and alerts.
var (
	// HitsTotal counts successful cache lookups per group.
	HitsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits.",
		},
		[]string{"cache"},
	)

	// MissesTotal counts failed cache lookups per group.
	MissesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses.",
		},
		[]string{"cache"},
	)

	// EvictionsTotal counts evicted entries per group.
	EvictionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_evictions_total",
			Help: "Total number of entries evicted from the cache.",
		},
		[]string{"cache"},
	)
)

func init() {
	prometheus.MustRegister(
		HitsTotal,
		MissesTotal,
		EvictionsTotal,
	)
}

// cacheEntriesCollector is a Prometheus Collector that lazily reports the current
// number of entries for a single cache group by calling lenFunc at scrape time.
// This avoids stale counts caused by TTL-based eviction in external backends like Redis.
type cacheEntriesCollector struct {
	desc    *prometheus.Desc
	lenFunc func() int
}

func (c *cacheEntriesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *cacheEntriesCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(c.lenFunc()))
}

var (
	entriesCollectorMu sync.Mutex
	entriesCollectors  = make(map[string]*cacheEntriesCollector)
	// entriesReg is the Prometheus registerer used for entries collectors.
	// Exposed as a variable so tests can substitute an isolated registry.
	entriesReg prometheus.Registerer = prometheus.DefaultRegisterer
)

// registerEntriesCollector registers a per-group entries collector that lazily
// reads the cache size at scrape time. If a collector for the same group already
// exists it is replaced, making it safe to call when a new cache instance is
// created for a group that was previously registered (e.g., in tests).
func registerEntriesCollector(group string, lenFunc func() int) *cacheEntriesCollector {
	desc := prometheus.NewDesc(
		"cache_entries",
		"Current number of entries in the cache.",
		nil,
		prometheus.Labels{"cache": group},
	)
	c := &cacheEntriesCollector{desc: desc, lenFunc: lenFunc}

	entriesCollectorMu.Lock()
	defer entriesCollectorMu.Unlock()

	if old, ok := entriesCollectors[group]; ok {
		entriesReg.Unregister(old)
	}
	entriesCollectors[group] = c
	_ = entriesReg.Register(c)
	return c
}

// unregisterEntriesCollector removes the entries collector for the given group.
func unregisterEntriesCollector(group string) {
	entriesCollectorMu.Lock()
	defer entriesCollectorMu.Unlock()

	if c, ok := entriesCollectors[group]; ok {
		entriesReg.Unregister(c)
		delete(entriesCollectors, group)
	}
}
