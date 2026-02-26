package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func getCounterValue(c prometheus.Counter) float64 {
	var m dto.Metric
	if err := c.(prometheus.Metric).Write(&m); err != nil {
		return 0
	}
	return m.GetCounter().GetValue()
}

func getGaugeValue(g prometheus.Gauge) float64 {
	var m dto.Metric
	if err := g.(prometheus.Metric).Write(&m); err != nil {
		return 0
	}
	return m.GetGauge().GetValue()
}

func getCounterVecValue(cv *prometheus.CounterVec, labels ...string) float64 {
	c, err := cv.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0
	}
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		return 0
	}
	return m.GetCounter().GetValue()
}

func TestMetrics_SubtitleDownloadsTotal(t *testing.T) {
	before := getCounterVecValue(SubtitleDownloadsTotal, "success")
	SubtitleDownloadsTotal.WithLabelValues("success").Inc()
	after := getCounterVecValue(SubtitleDownloadsTotal, "success")

	if after != before+1 {
		t.Errorf("Expected success counter to increment by 1, got diff %.0f", after-before)
	}
}

func TestMetrics_SubtitleDownloadsTotal_Error(t *testing.T) {
	before := getCounterVecValue(SubtitleDownloadsTotal, "error")
	SubtitleDownloadsTotal.WithLabelValues("error").Inc()
	after := getCounterVecValue(SubtitleDownloadsTotal, "error")

	if after != before+1 {
		t.Errorf("Expected error counter to increment by 1, got diff %.0f", after-before)
	}
}

func TestMetrics_CacheHitsTotal(t *testing.T) {
	before := getCounterValue(CacheHitsTotal)
	CacheHitsTotal.Inc()
	after := getCounterValue(CacheHitsTotal)

	if after != before+1 {
		t.Errorf("Expected cache hits to increment by 1, got diff %.0f", after-before)
	}
}

func TestMetrics_CacheMissesTotal(t *testing.T) {
	before := getCounterValue(CacheMissesTotal)
	CacheMissesTotal.Inc()
	after := getCounterValue(CacheMissesTotal)

	if after != before+1 {
		t.Errorf("Expected cache misses to increment by 1, got diff %.0f", after-before)
	}
}

func TestMetrics_CacheEvictionsTotal(t *testing.T) {
	before := getCounterValue(CacheEvictionsTotal)
	CacheEvictionsTotal.Inc()
	after := getCounterValue(CacheEvictionsTotal)

	if after != before+1 {
		t.Errorf("Expected cache evictions to increment by 1, got diff %.0f", after-before)
	}
}

func TestMetrics_CacheEntries(t *testing.T) {
	CacheEntries.Set(42)
	val := getGaugeValue(CacheEntries)

	if val != 42 {
		t.Errorf("Expected cache entries to be 42, got %.0f", val)
	}

	CacheEntries.Set(0)
}

func TestMetrics_NewHTTPServer(t *testing.T) {
	srv := NewHTTPServer("localhost", 9090)

	if srv.Addr != "localhost:9090" {
		t.Errorf("Expected address 'localhost:9090', got '%s'", srv.Addr)
	}

	if srv.Handler == nil {
		t.Error("Expected handler to be set")
	}
}

func TestMetrics_NewHTTPServer_DefaultPort(t *testing.T) {
	srv := NewHTTPServer("0.0.0.0", 0)

	if srv.Addr != "0.0.0.0:9090" {
		t.Errorf("Expected address '0.0.0.0:9090', got '%s'", srv.Addr)
	}
}
