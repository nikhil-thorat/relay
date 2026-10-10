package strategy

import (
	"sync"
	"testing"

	"github.com/nikhil-thorat/relay/internal/target"
)

func TestWeightedEmptyReturnsError(t *testing.T) {
	w := &WeightedRoundRobin{}

	if _, err := w.Select(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestWeightedSelectDistribution(t *testing.T) {
	w := &WeightedRoundRobin{}

	targets := []target.Target{
		{ID: "api_1", Weight: 2},
		{ID: "api_2", Weight: 1},
	}

	counts := map[string]int{}

	for range 300 {
		s, err := w.Select(targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[s.ID]++
	}

	if counts["api_1"] != 200 {
		t.Fatalf("expected 200 for api_1, got %d", counts["api_1"])
	}

	if counts["api_2"] != 100 {
		t.Fatalf("expected 100 for api_2, got %d", counts["api_2"])
	}
}

func TestWeightedFallsBackToRR(t *testing.T) {
	w := &WeightedRoundRobin{}

	targets := []target.Target{
		{ID: "api_1", Weight: 1},
		{ID: "api_2", Weight: 1},
	}

	got := []string{}
	for range 4 {
		s, _ := w.Select(targets)
		got = append(got, s.ID)
	}

	expected := []string{"api_1", "api_2", "api_1", "api_2"}
	for i, want := range expected {
		if got[i] != want {
			t.Fatalf("expected %s at %d, got %s", want, i, got[i])
		}
	}
}

func TestWeightedConcurrentSelect(t *testing.T) {
	w := &WeightedRoundRobin{}

	targets := []target.Target{
		{ID: "api_1", Weight: 2},
		{ID: "api_2", Weight: 1},
	}

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 50 {
				if _, err := w.Select(targets); err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
			}
		})
	}
	wg.Wait()
}
