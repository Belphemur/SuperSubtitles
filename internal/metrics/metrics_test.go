package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

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
