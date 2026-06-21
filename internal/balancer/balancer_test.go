package balancer

import (
	"testing"

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
