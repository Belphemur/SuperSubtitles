package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewHTTPServer creates an HTTP server that exposes Prometheus metrics at /metrics.
func NewHTTPServer(address string, port int) *http.Server {
	if port == 0 {
		port = 9090
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", address, port),
		Handler: mux,
	}
}
