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

func init() {
	prometheus.MustRegister(
		SubtitleDownloadsTotal,
	)
}
