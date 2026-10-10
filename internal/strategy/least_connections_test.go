package strategy

import (
	"sync"
	"testing"

	"github.com/nikhil-thorat/relay/internal/target"
)

func TestLeastConnEmptyReturnsError(t *testing.T) {
	l := &LeastConnections{}

	if _, err := l.Select(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestLeastConnSelectDistribution(t *testing.T) {
	l := &LeastConnections{}

	targets := []target.Target{
		{ID: "api_1"},
		{ID: "api_2"},
	}

	l.Inc("api_1")
	l.Inc("api_1")

	for range 10 {
		s, err := l.Select(targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ID != "api_2" {
			t.Fatalf("expected api_2 (less loaded), got %s", s.ID)
		}
	}

	l.Done("api_1")
	l.Done("api_1")
}

func TestLeastConnConcurrentSelect(t *testing.T) {
	l := &LeastConnections{}

	targets := []target.Target{{ID: "api_1"}, {ID: "api_2"}}

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 50 {
				s, err := l.Select(targets)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				l.Inc(s.ID)
				l.Done(s.ID)
			}
		})
	}
	wg.Wait()
}
