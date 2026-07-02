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

	logger, _ := logging.New("off")

	checker := New(
		target.NewPool(),
		setupMetrics(),
		100*time.Millisecond,
		1*time.Second,
		logger,
	)

	healthy := checker.Check(&target.Target{
		Address: address,
	})

	if !healthy {
		t.Fatal("expected target to be healthy")
	}
}

func TestCheckUnhealthy(t *testing.T) {

	logger, _ := logging.New("off")

	checker := New(
		target.NewPool(),
		setupMetrics(),
		100*time.Millisecond,
		1*time.Second,
		logger,
	)

	healthy := checker.Check(&target.Target{
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

	logger, _ := logging.New("off")

	checker := New(
		pool,
		metrics,
		100*time.Millisecond,
		1*time.Second,
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

	logger, _ := logging.New("off")

	checker := New(
		pool,
		metrics,
		100*time.Millisecond,
		1*time.Second,
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
