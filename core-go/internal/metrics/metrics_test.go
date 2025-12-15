package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandler_nilMetrics(t *testing.T) {
	var m *Metrics
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	m.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
	if got := rr.Body.String(); !strings.Contains(got, "metrics unavailable") {
		t.Fatalf("expected body to mention metrics unavailable, got %q", got)
	}
}

func TestHandler_exposesRegisteredMetrics(t *testing.T) {
	m := New()
	m.ObserveHTTPRequest(http.MethodGet, "/readyz", http.StatusOK, 12*time.Millisecond)
	m.IncDiscoveryRun()
	m.ObserveDiscoveryRunDuration(3 * time.Second)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	m.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, "roller_http_requests_total") {
		t.Fatalf("expected http_requests_total metric to be present")
	}
	if !strings.Contains(body, "roller_http_requests_total{method=\"GET\",path=\"/readyz\",status=\"200\"} 1") {
		t.Fatalf("expected labeled request counter to be incremented; body=%s", body)
	}
	if !strings.Contains(body, "roller_discovery_runs_total 1") {
		t.Fatalf("expected discovery runs counter to be incremented; body=%s", body)
	}
	if !strings.Contains(body, "roller_discovery_run_duration_seconds_count 1") {
		t.Fatalf("expected discovery run duration histogram to have one observation; body=%s", body)
	}
}
