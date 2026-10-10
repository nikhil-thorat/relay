package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikhil-thorat/relay/internal/logging"
	"github.com/nikhil-thorat/relay/internal/metrics"
	"github.com/nikhil-thorat/relay/internal/target"
)

func setupMetrics() *metrics.Metrics {
	registry := prometheus.NewRegistry()

	return metrics.New(registry)
}

func TestCheckHealthy(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	address := strings.TrimPrefix(
		server.URL,
		"http://",
	)

	logger, _ := logging.New("off", "text")

	checker := New(
		target.NewPool(),
		setupMetrics(),
		100*time.Millisecond,
		1*time.Second,
		"tcp",
		"/health",
		logger,
	)

	healthy := checker.Check(target.Target{
		Address: address,
	})

	if !healthy {
		t.Fatal("expected target to be healthy")
	}
}

func TestCheckUnhealthy(t *testing.T) {

	logger, _ := logging.New("off", "text")

	checker := New(
		target.NewPool(),
		setupMetrics(),
		100*time.Millisecond,
		1*time.Second,
		"tcp",
		"/health",
		logger,
	)

	healthy := checker.Check(target.Target{
		Address: "localhost:65535",
	})

	if healthy {
		t.Fatal("expected target to be unhealthy")
	}
}

func TestRunUpdatesState(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	address := strings.TrimPrefix(
		server.URL,
		"http://",
	)

	pool := target.NewPool()

	_ = pool.Add(&target.Target{
		ID:      "api_1",
		Address: address,
	})

	metrics := setupMetrics()

	logger, _ := logging.New("off", "text")

	checker := New(
		pool,
		metrics,
		100*time.Millisecond,
		1*time.Second,
		"tcp",
		"/health",
		logger,
	)

	checker.Run()

	state, err := pool.GetState("api_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !state.Healthy {
		t.Fatal("expected target to be healthy")
	}

	healthyTargets := testutil.ToFloat64(
		metrics.HealthyTargets,
	)

	if healthyTargets != 1 {
		t.Fatalf(
			"expected 1 healthy target, got %v",
			healthyTargets,
		)
	}
}

func TestStart(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	address := strings.TrimPrefix(
		server.URL,
		"http://",
	)

	pool := target.NewPool()

	_ = pool.Add(&target.Target{
		ID:      "api_1",
		Address: address,
	})

	metrics := setupMetrics()

	logger, _ := logging.New("off", "text")

	checker := New(
		pool,
		metrics,
		100*time.Millisecond,
		1*time.Second,
		"tcp",
		"/health",
		logger,
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	checker.Start(ctx)

	time.Sleep(
		50 * time.Millisecond,
	)

	state, err := pool.GetState("api_1")
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if !state.Healthy {
		t.Fatal(
			"expected target to be healthy",
		)
	}

	healthyTargets := testutil.ToFloat64(
		metrics.HealthyTargets,
	)

	if healthyTargets != 1 {
		t.Fatalf(
			"expected 1 healthy target, got %v",
			healthyTargets,
		)
	}

	cancel()

	time.Sleep(
		10 * time.Millisecond,
	)
}

func TestHTTPHealthy(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	address := strings.TrimPrefix(server.URL, "http://")
	logger, _ := logging.New("off", "text")

	checker := New(target.NewPool(), setupMetrics(), 100*time.Millisecond, 1*time.Second, "http", "/health", logger)

	if !checker.Check(target.Target{Address: address}) {
		t.Fatal("expected http healthy")
	}
}

func TestHTTPUnhealthy(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer server.Close()

	address := strings.TrimPrefix(server.URL, "http://")
	logger, _ := logging.New("off", "text")

	checker := New(target.NewPool(), setupMetrics(), 100*time.Millisecond, 1*time.Second, "http", "/health", logger)

	if checker.Check(target.Target{Address: address}) {
		t.Fatal("expected http unhealthy on 500")
	}

	checkerDown := New(target.NewPool(), setupMetrics(), 100*time.Millisecond, 200*time.Millisecond, "http", "/health", logger)
	if checkerDown.Check(target.Target{Address: "localhost:1"}) {
		t.Fatal("expected http unhealthy on dial fail")
	}
}

func TestConcurrentRunDoesNotBlock(t *testing.T) {
	pool := target.NewPool()
	for i := range 5 {
		_ = pool.Add(&target.Target{ID: "api_" + string(rune('0'+i)), Address: "localhost:1"})
	}

	logger, _ := logging.New("off", "text")
	m := setupMetrics()
	checker := New(pool, m, 10*time.Millisecond, 50*time.Millisecond, "tcp", "/health", logger)

	done := make(chan struct{})
	go func() {
		checker.Run()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run blocked, expected concurrent checks")
	}
}

func TestTransitionUpdatesGauge(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	address := strings.TrimPrefix(server.URL, "http://")
	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: address})

	m := setupMetrics()
	logger, _ := logging.New("off", "text")
	checker := New(pool, m, 100*time.Millisecond, 1*time.Second, "tcp", "/health", logger)

	checker.Run()
	if v := testutil.ToFloat64(m.HealthyTargets); v != 1 {
		t.Fatalf("expected gauge 1, got %v", v)
	}

	_ = pool.Remove("api_1")
	_ = pool.Add(&target.Target{ID: "api_1", Address: "localhost:1"})
	checker.Run()
	if v := testutil.ToFloat64(m.HealthyTargets); v != 0 {
		t.Fatalf("expected gauge 0 after unhealthy, got %v", v)
	}
}
