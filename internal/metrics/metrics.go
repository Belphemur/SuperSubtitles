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

	// CircuitBreakerState reports the current state of the HTTP client circuit
	// breaker protecting calls to feliratok.eu: 0 = closed, 1 = half-open, 2 = open.
	CircuitBreakerState = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_client_circuit_breaker_state",
			Help: "Current state of the HTTP client circuit breaker (0=closed, 1=half-open, 2=open).",
		},
	)
)

func init() {
	prometheus.MustRegister(
		SubtitleDownloadsTotal,
		CircuitBreakerState,
	)
}
