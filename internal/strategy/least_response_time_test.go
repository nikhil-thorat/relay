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
