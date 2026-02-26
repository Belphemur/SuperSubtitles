package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Subtitle download metrics
var (
	SubtitleDownloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "subtitle_downloads_total",
			Help: "Total number of subtitle downloads.",
		},
		[]string{"status"},
	)
)

// LRU cache metrics
var (
	CacheHitsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "subtitle_cache_hits_total",
			Help: "Total number of LRU cache hits for downloaded ZIP files.",
		},
	)

	CacheMissesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "subtitle_cache_misses_total",
			Help: "Total number of LRU cache misses for downloaded ZIP files.",
		},
	)

	CacheEvictionsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "subtitle_cache_evictions_total",
			Help: "Total number of entries evicted from the LRU cache.",
		},
	)

	CacheEntries = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "subtitle_cache_entries",
			Help: "Current number of entries in the LRU cache.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		SubtitleDownloadsTotal,
		CacheHitsTotal,
		CacheMissesTotal,
		CacheEvictionsTotal,
		CacheEntries,
	)
}
