package strategy

import (
	"sync"
	"testing"

	"github.com/nikhil-thorat/relay/internal/target"
)

func TestRoundRobinNilTargets(t *testing.T) {
	rr := &RoundRobin{}

	_, err := rr.Select(nil)

	if err == nil {
		t.Fatal("expected error")
	}

}

func TestRoundRobinEmptyTargets(t *testing.T) {
	rr := &RoundRobin{}

	_, err := rr.Select([]target.Target{})

	if err == nil {
		t.Fatal("expected error")
	}

}

func TestRoundRobinSingleTarget(t *testing.T) {
	rr := &RoundRobin{}

	targets := []target.Target{
		{
			ID:      "api_1",
			Address: "localhost:9001",
			Weight:  1,
		},
	}

	selected, err := rr.Select(targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selected.ID != "api_1" {
		t.Fatalf("expected api_1, got %s", selected.ID)
	}

}

func TestRoundRobinRotation(t *testing.T) {
	rr := &RoundRobin{}

	targets := []target.Target{
		{ID: "api_1"},
		{ID: "api_2"},
		{ID: "api_3"},
	}

	expected := []string{
		"api_1",
		"api_2",
		"api_3",
		"api_1",
		"api_2",
		"api_3",
	}

	for _, want := range expected {
		selected, err := rr.Select(targets)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if selected.ID != want {
			t.Fatalf("expected %s, got %s", want, selected.ID)
		}
	}
}

func TestConcurrentSelect(t *testing.T) {
	rr := &RoundRobin{}

	targets := []target.Target{
		{ID: "api_1"},
		{ID: "api_2"},
		{ID: "api_3"},
	}

	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() {
			for range 100 {
				selected, err := rr.Select(targets)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if selected.ID == "" {
					t.Errorf("expected id, got empty")
					return
				}
			}
		})
	}

	wg.Wait()
}

func TestShrinkingPoolNoPanic(t *testing.T) {
	rr := &RoundRobin{}

	three := []target.Target{{ID: "api_1"}, {ID: "api_2"}, {ID: "api_3"}}
	two := []target.Target{{ID: "api_1"}, {ID: "api_2"}}
	one := []target.Target{{ID: "api_1"}}

	for range 5 {
		if _, err := rr.Select(three); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	for range 5 {
		selected, err := rr.Select(two)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if selected.ID != "api_1" && selected.ID != "api_2" {
			t.Fatalf("unexpected target %s", selected.ID)
		}
	}

	for range 5 {
		selected, err := rr.Select(one)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if selected.ID != "api_1" {
			t.Fatalf("expected api_1, got %s", selected.ID)
		}
	}
}
