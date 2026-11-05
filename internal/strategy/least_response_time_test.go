package strategy

import (
	"sync"
	"testing"
	"time"

	"github.com/nikhil-thorat/relay/internal/target"
)

func TestLRTEmptyReturnsError(t *testing.T) {
	l := &LeastResponseTime{}

	if _, err := l.Select(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestLRTSelectDistribution(t *testing.T) {
	l := &LeastResponseTime{}

	targets := []target.Target{{ID: "api_1"}, {ID: "api_2"}}

	l.Observe("api_1", 200*time.Millisecond)
	l.Observe("api_2", 20*time.Millisecond)

	for range 10 {
		s, err := l.Select(targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ID != "api_2" {
			t.Fatalf("expected api_2 (faster), got %s", s.ID)
		}
	}
}

func TestLRTAllUnknownRotates(t *testing.T) {
	l := &LeastResponseTime{}

	targets := []target.Target{{ID: "api_1"}, {ID: "api_2"}}

	first, err := l.Select(targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := l.Select(targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.ID == second.ID {
		t.Fatalf("expected rotation, got %s twice", first.ID)
	}
}

func TestLRTConcurrentSelect(t *testing.T) {
	l := &LeastResponseTime{}

	targets := []target.Target{{ID: "api_1"}, {ID: "api_2"}}

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for range 50 {
				if _, err := l.Select(targets); err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				l.Observe("api_1", time.Duration(i)*time.Millisecond)
			}
		}(i)
	}
	wg.Wait()
}

func TestLRTKeepsTemporarilyUnavailableTarget(t *testing.T) {
	l := &LeastResponseTime{}

	full := []target.Target{{ID: "api_1"}, {ID: "api_2"}}
	l.Observe("api_1", 30*time.Millisecond)
	l.Observe("api_2", 10*time.Millisecond)

	if _, err := l.Select([]target.Target{{ID: "api_1"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	selected, err := l.Select(full)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selected.ID != "api_2" {
		t.Fatalf("expected api_2, got %s", selected.ID)
	}
}

func TestLRTPruneRemovesRemovedTarget(t *testing.T) {
	l := &LeastResponseTime{}
	l.Observe("old", time.Millisecond)

	l.Prune([]target.Target{{ID: "new"}})

	if _, ok := l.ewma["old"]; ok {
		t.Fatal("expected removed target to be pruned")
	}
}

func TestLRTExploresUnknownTargets(t *testing.T) {
	l := &LeastResponseTime{}
	l.Observe("api_1", time.Millisecond)

	selected, err := l.Select([]target.Target{{ID: "api_1"}, {ID: "api_2"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selected.ID != "api_2" {
		t.Fatalf("expected api_2, got %s", selected.ID)
	}
}

func TestLRTKnownTiesRotate(t *testing.T) {
	l := &LeastResponseTime{}
	l.Observe("api_1", time.Millisecond)
	l.Observe("api_2", time.Millisecond)

	targets := []target.Target{{ID: "api_1"}, {ID: "api_2"}}

	first, _ := l.Select(targets)
	second, _ := l.Select(targets)

	if first.ID == second.ID {
		t.Fatalf("expected tie rotation, got %s twice", first.ID)
	}
}

func TestLRTThreeWayTieRotates(t *testing.T) {
	l := &LeastResponseTime{}
	targets := []target.Target{{ID: "api_1"}, {ID: "api_2"}, {ID: "api_3"}}

	for _, target := range targets {
		l.Observe(target.ID, time.Millisecond)
	}

	counts := map[string]int{}
	for range 6 {
		selected, err := l.Select(targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[selected.ID]++
	}

	for _, target := range targets {
		if counts[target.ID] != 2 {
			t.Fatalf("expected 2 selections for %s, got %d", target.ID, counts[target.ID])
		}
	}
}
