package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics exposes application metrics that are safe to scrape via Prometheus.
type Metrics struct {
	registry             *prometheus.Registry
	httpRequests         *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	discoveryRunsTotal   prometheus.Counter
	discoveryRunDuration prometheus.Histogram
}

// New creates a fresh Metrics registry with HTTP and discovery metrics registered.
func New() *Metrics {
	registry := prometheus.NewRegistry()

	httpRequests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "roller",
		Name:      "http_requests_total",
		Help:      "Count of HTTP requests processed by core-go",
	}, []string{"method", "path", "status"})

	httpRequestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "roller",
		Name:      "http_request_duration_seconds",
		Help:      "Duration of HTTP requests served by core-go",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	discoveryRunsTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "roller",
		Name:      "discovery_runs_total",
		Help:      "Total number of discovery runs processed",
	})

	discoveryRunDuration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "roller",
		Name:      "discovery_run_duration_seconds",
		Help:      "Duration of discovery runs from start to finish",
		Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600, 1200},
	})

	registry.MustRegister(
		httpRequests,
		httpRequestDuration,
		discoveryRunsTotal,
		discoveryRunDuration,
	)

	return &Metrics{
		registry:             registry,
		httpRequests:         httpRequests,
		httpRequestDuration:  httpRequestDuration,
		discoveryRunsTotal:   discoveryRunsTotal,
		discoveryRunDuration: discoveryRunDuration,
	}
}

// ObserveHTTPRequest records a single HTTP request/response cycle.
func (m *Metrics) ObserveHTTPRequest(method, path string, status int, duration time.Duration) {
	if m == nil {
		return
	}
	labels := prometheus.Labels{
		"method": method,
		"path":   path,
		"status": strconv.Itoa(status),
	}
	m.httpRequests.With(labels).Inc()
	m.httpRequestDuration.With(labels).Observe(duration.Seconds())
}

// IncDiscoveryRun increments the discovery run counter.
func (m *Metrics) IncDiscoveryRun() {
	if m == nil {
		return
	}
	m.discoveryRunsTotal.Inc()
}

// ObserveDiscoveryRunDuration observes a discovery run duration.
func (m *Metrics) ObserveDiscoveryRunDuration(duration time.Duration) {
	if m == nil {
		return
	}
	m.discoveryRunDuration.Observe(duration.Seconds())
}

// Handler exposes the Prometheus registry over HTTP.
func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("metrics unavailable"))
		})
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
