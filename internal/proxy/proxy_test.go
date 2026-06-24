package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/metrics"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

func setupMetrics() *metrics.Metrics {
	registry := prometheus.NewRegistry()

	return metrics.New(registry)
}

func TestNew(t *testing.T) {
	pool := target.NewPool()
	rr := &strategy.RoundRobin{}

	balancer := balancer.New(pool, rr)

	metrics := setupMetrics()

	proxy := New(
		balancer,
		metrics,
	)

	if proxy == nil {
		t.Fatal("expected proxy, got nil")
	}
}

func TestServeHTTPWithNoTargets(t *testing.T) {
	pool := target.NewPool()
	rr := &strategy.RoundRobin{}

	balancer := balancer.New(pool, rr)

	metrics := setupMetrics()

	proxy := New(
		balancer,
		metrics,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf(
			"expected %d, got %d",
			http.StatusBadGateway,
			rec.Code,
		)
	}

	requests := testutil.ToFloat64(
		metrics.RequestsTotal,
	)

	if requests != 1 {
		t.Fatalf(
			"expected 1 request, got %v",
			requests,
		)
	}

	errors := testutil.ToFloat64(
		metrics.RequestErrorsTotal,
	)

	if errors != 1 {
		t.Fatalf(
			"expected 1 error, got %v",
			errors,
		)
	}
}

func TestServeHTTPForwardsRequest(t *testing.T) {
	backend := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)

			_, _ = w.Write(
				[]byte("hello from backend"),
			)
		}),
	)

	defer backend.Close()

	address := strings.TrimPrefix(
		backend.URL,
		"http://",
	)

	pool := target.NewPool()

	_ = pool.Add(&target.Target{
		ID:      "api_1",
		Address: address,
	})

	_ = pool.SetHealthy(
		"api_1",
		true,
	)

	rr := &strategy.RoundRobin{}

	balancer := balancer.New(pool, rr)

	metrics := setupMetrics()

	proxy := New(
		balancer,
		metrics,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if string(body) != "hello from backend" {
		t.Fatalf(
			"expected backend response, got %q",
			string(body),
		)
	}

	requests := testutil.ToFloat64(
		metrics.RequestsTotal,
	)

	if requests != 1 {
		t.Fatalf(
			"expected 1 request, got %v",
			requests,
		)
	}

	targetRequests := testutil.ToFloat64(
		metrics.TargetRequests.
			WithLabelValues("api_1"),
	)

	if targetRequests != 1 {
		t.Fatalf(
			"expected 1 target request, got %v",
			targetRequests,
		)
	}

	errors := testutil.ToFloat64(
		metrics.RequestErrorsTotal,
	)

	if errors != 0 {
		t.Fatalf(
			"expected 0 errors, got %v",
			errors,
		)
	}
}
