package balancer

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikhil-thorat/relay/internal/metrics"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

func TestNewBalancer(t *testing.T) {
	pool := target.NewPool()
	rr := &strategy.RoundRobin{}

	balancer := New(pool, rr)

	if balancer == nil {
		t.Fatal("expected balancer, got nil")
	}
}

func TestNextSingleHealthyTarget(t *testing.T) {
	pool := target.NewPool()

	_ = pool.Add(&target.Target{
		ID:      "api_1",
		Address: "localhost:9001",
		Weight:  1,
	})

	_ = pool.SetHealthy("api_1", true)

	rr := &strategy.RoundRobin{}
	balancer := New(pool, rr)

	selected, err := balancer.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selected.ID != "api_1" {
		t.Fatalf("expected api_1, got %s", selected.ID)
	}
}

func TestNextNoHealthyTargets(t *testing.T) {
	pool := target.NewPool()

	_ = pool.Add(&target.Target{
		ID: "api_1",
	})

	rr := &strategy.RoundRobin{}
	balancer := New(pool, rr)

	_, err := balancer.Next()

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRoundRobinIntegration(t *testing.T) {
	pool := target.NewPool()

	_ = pool.Add(&target.Target{ID: "api_1"})
	_ = pool.Add(&target.Target{ID: "api_2"})
	_ = pool.Add(&target.Target{ID: "api_3"})

	_ = pool.SetHealthy("api_1", true)
	_ = pool.SetHealthy("api_2", true)
	_ = pool.SetHealthy("api_3", true)

	rr := &strategy.RoundRobin{}
	balancer := New(pool, rr)

	expected := []string{
		"api_1",
		"api_2",
		"api_3",
		"api_1",
	}

	for _, want := range expected {
		selected, err := balancer.Next()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if selected.ID != want {
			t.Fatalf("expected %s, got %s", want, selected.ID)
		}
	}
}

func TestNextConcurrent(t *testing.T) {
	pool := target.NewPool()

	_ = pool.Add(&target.Target{ID: "api_1"})
	_ = pool.Add(&target.Target{ID: "api_2"})
	_ = pool.Add(&target.Target{ID: "api_3"})

	_ = pool.SetHealthy("api_1", true)
	_ = pool.SetHealthy("api_2", true)
	_ = pool.SetHealthy("api_3", true)

	rr := &strategy.RoundRobin{}
	balancer := New(pool, rr)

	var wg sync.WaitGroup

	for range 20 {
		wg.Go(func() {
			for range 50 {
				if _, err := balancer.Next(); err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
			}
		})
	}

	wg.Wait()
}

func TestConcurrentSetStrategy(t *testing.T) {
	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: "localhost:9001"})
	_ = pool.Add(&target.Target{ID: "api_2", Address: "localhost:9002"})
	_ = pool.SetHealthy("api_1", true)
	_ = pool.SetHealthy("api_2", true)

	b := New(pool, &strategy.RoundRobin{})

	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			b.SetStrategy(&strategy.RoundRobin{})
		}
	})

	for range 10 {
		wg.Go(func() {
			for range 100 {
				if _, err := b.SelectForRequest("key", nil); err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
			}
		})
	}

	wg.Wait()
}

func TestSelectForRequestExcludesTarget(t *testing.T) {
	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: "localhost:9001"})
	_ = pool.Add(&target.Target{ID: "api_2", Address: "localhost:9002"})
	_ = pool.SetHealthy("api_1", true)
	_ = pool.SetHealthy("api_2", true)

	b := New(pool, &strategy.ConsistentHash{})

	first, err := b.SelectForRequest("key", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := b.SelectForRequest("key", map[string]struct{}{first.Target.ID: {}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.Target.ID == second.Target.ID {
		t.Fatalf("expected a different target, got %s", second.Target.ID)
	}
}

func TestReportFailureUpdatesHealthyGauge(t *testing.T) {
	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: "localhost:9001"})
	_ = pool.SetHealthy("api_1", true)
	pool.SetPassiveRecovery(true)

	m := metrics.New(prometheus.NewRegistry())
	b := New(pool, &strategy.RoundRobin{})
	b.SetMetrics(m)

	for range 3 {
		b.ReportFailure("api_1")
	}

	if got := testutil.ToFloat64(m.HealthyTargets); got != 0 {
		t.Fatalf("expected 0 healthy targets, got %v", got)
	}

	b.ReportSuccess("api_1")

	if got := testutil.ToFloat64(m.HealthyTargets); got != 1 {
		t.Fatalf("expected 1 healthy target, got %v", got)
	}
}
